// Package googlecas talks to Google Cloud Certificate Authority Service (CAS) on behalf of a Google CAS certificate authority configured
// in Fleet, authenticating with the service account key stored on the CA.
package googlecas

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/privateca/v1"
)

const defaultTimeout = 20 * time.Second

type Service struct {
	logger   *slog.Logger
	timeout  time.Duration
	endpoint string
	tokenURL string

	// Created once and shared by every CA's client, like the EST client, so connections to Google are reused.
	transport   *http.Transport
	tokenClient *http.Client
}

// Compile-time check to ensure that Service implements fleet.GoogleCASService.
var _ fleet.GoogleCASService = (*Service)(nil)

type Opt func(*Service)

func WithLogger(logger *slog.Logger) Opt {
	return func(s *Service) {
		s.logger = logger
	}
}

func WithTimeout(timeout time.Duration) Opt {
	return func(s *Service) {
		s.timeout = timeout
	}
}

// WithEndpoint sends API requests to endpoint instead of Google, so tests can stand in for it. Requests are still authenticated with
// the CA's key.
func WithEndpoint(endpoint string) Opt {
	return func(s *Service) {
		s.endpoint = endpoint
	}
}

// WithTokenURL gets OAuth tokens from tokenURL instead of Google, so tests can stand in for it.
func WithTokenURL(tokenURL string) Opt {
	return func(s *Service) {
		s.tokenURL = tokenURL
	}
}

func NewService(opts ...Opt) *Service {
	s := &Service{timeout: defaultTimeout, tokenURL: google.JWTTokenURL}
	for _, opt := range opts {
		opt(s)
	}
	if s.logger == nil {
		s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	s.transport = fleethttp.NewTransport()
	s.tokenClient = fleethttp.NewClient(fleethttp.WithFollowRedir(false))
	return s
}

// serviceAccountKey is the part of a Google service account key file that the Google auth library needs to sign requests.
type serviceAccountKey struct {
	Type        string `json:"type"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// ValidateServiceAccountKey checks that keyJSON is a Google service account key with the fields needed to authenticate. It makes no
// network call, so the service layer can reject a malformed key before storing it or contacting Google.
func ValidateServiceAccountKey(keyJSON string) error {
	var key serviceAccountKey
	if err := json.Unmarshal([]byte(keyJSON), &key); err != nil {
		return errors.New("The service account key must be a JSON object.")
	}
	if key.Type != "service_account" {
		return errors.New(`The service account key must have "type" set to "service_account".`)
	}
	if strings.TrimSpace(key.ClientEmail) == "" || strings.TrimSpace(key.PrivateKey) == "" {
		return errors.New(`The service account key must include "client_email" and "private_key".`)
	}
	return nil
}

func (s *Service) VerifyCAPool(ctx context.Context, ca fleet.GoogleCASCA) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	client, err := s.newClient(ctx, ca)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating Google CAS client")
	}
	if _, err := client.Projects.Locations.CaPools.Get(ca.CAPoolResourceName()).Context(ctx).Do(); err != nil {
		return ctxerr.Wrap(ctx, s.describeError(ctx, err, ca), "getting Google CAS CA pool")
	}
	return nil
}

func (s *Service) newClient(ctx context.Context, ca fleet.GoogleCASCA) (*privateca.Service, error) {
	if err := ValidateServiceAccountKey(string(ca.APIKeyJSON)); err != nil {
		return nil, err
	}
	keyJSON, err := s.withTokenURL(ca.APIKeyJSON)
	if err != nil {
		return nil, err
	}
	// The token exchange and API calls both use fleethttp, so they get the same outbound restrictions as other integrations.
	creds, err := credentials.NewCredentialsFromJSON(credentials.ServiceAccount, keyJSON, &credentials.DetectOptions{
		Scopes: []string{privateca.CloudPlatformScope},
		Client: s.tokenClient,
	})
	if err != nil {
		return nil, s.describeError(ctx, err, ca)
	}
	httpClient, err := httptransport.NewClient(&httptransport.Options{
		Credentials:      creds,
		BaseRoundTripper: s.transport,
	})
	if err != nil {
		return nil, s.describeError(ctx, err, ca)
	}
	// Google doesn't redirect these API calls, and a redirect would carry the CA's access token, so it isn't followed.
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	opts := []option.ClientOption{option.WithHTTPClient(httpClient)}
	if s.endpoint != "" {
		opts = append(opts, option.WithEndpoint(s.endpoint))
	}
	client, err := privateca.NewService(ctx, opts...)
	if err != nil {
		return nil, s.describeError(ctx, err, ca)
	}
	return client, nil
}

// withTokenURL returns the key with its token_uri set to s.tokenURL, Google's token endpoint outside tests. The auth library sends the
// signed assertion to the key's token_uri, and every genuine key names Google's, so the field is not trusted. Case variants go too, since
// encoding/json matches field names case-insensitively.
func (s *Service) withTokenURL(keyJSON fleet.GoogleCASKeyJSON) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(keyJSON), &fields); err != nil {
		return nil, errors.New("The service account key must be a JSON object.")
	}
	for name := range fields {
		if strings.EqualFold(name, "token_uri") {
			delete(fields, name)
		}
	}
	tokenURL, err := json.Marshal(s.tokenURL)
	if err != nil {
		return nil, err
	}
	fields["token_uri"] = tokenURL
	return json.Marshal(fields)
}

// describeError turns a Google API failure into a message that tells the user which of their settings to check. The underlying error
// is logged because the Google message can include the request details.
func (s *Service) describeError(ctx context.Context, err error, ca fleet.GoogleCASCA) error {
	s.logger.ErrorContext(ctx, "Google CAS request failed", "ca_pool", ca.CAPoolResourceName(), "err", err)
	if apiErr, ok := errors.AsType[*googleapi.Error](err); ok {
		switch apiErr.Code {
		// Google answers 403, not 404, for a pool the service account has no role on, which includes a mistyped project, location
		// or pool, so the message names those too.
		case http.StatusUnauthorized, http.StatusForbidden:
			return errors.New("The service account can't access the CA pool. Check the project ID, location and CA pool, and that the service account has the CA Service Certificate Manager role on the pool.")
		case http.StatusNotFound:
			return fmt.Errorf("CA pool %q wasn't found in project %q, location %q.", ca.CAPool, ca.Project, ca.Location)
		}
		return fmt.Errorf("Google CAS returned an error (HTTP %d).", apiErr.Code)
	}
	if status, ok := tokenServiceUnavailable(err); ok {
		return fmt.Errorf("Google couldn't issue an access token (HTTP %d). Try again later.", status)
	}
	if isConnectionError(err) {
		return errors.New("Couldn't connect to Google CAS. Try again later.")
	}
	return errAuthentication
}

// A malformed, revoked or deleted key fails while getting a token, before any API status is available.
var errAuthentication = errors.New("Couldn't authenticate with Google CAS. Check that the service account key is valid.")

// tokenServiceUnavailable reports a token request Google turned away for a reason expected to clear, such as an outage or rate
// limiting, and its HTTP status. Those say nothing about the key.
func tokenServiceUnavailable(err error) (int, bool) {
	authErr, ok := errors.AsType[*auth.Error](err)
	if !ok || authErr.Response == nil {
		return 0, false
	}
	status := authErr.Response.StatusCode
	return status, status >= http.StatusInternalServerError || status == http.StatusTooManyRequests || status == http.StatusRequestTimeout
}

// isConnectionError reports a failure to reach Google, to verify its TLS certificate, or to get a whole response from it, which says
// nothing about the key.
func isConnectionError(err error) bool {
	_, opErr := errors.AsType[*net.OpError](err)
	_, certErr := errors.AsType[*tls.CertificateVerificationError](err)
	_, recordErr := errors.AsType[tls.RecordHeaderError](err)
	return opErr || certErr || recordErr || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
}
