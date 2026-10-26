// Package googlecas talks to Google Cloud Certificate Authority Service (CAS) on behalf of a Google CAS certificate authority configured
// in Fleet, authenticating with the service account key stored on the CA.
package googlecas

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/auth"
	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/privateca/v1"
)

const (
	defaultTimeout = 20 * time.Second
	// certificateLifetime is the lifetime Fleet requests. The CA pool's issuance policy and the issuing CA's own expiry can shorten it.
	certificateLifetime = 365 * 24 * time.Hour
	// issueRetryAfterSeconds is the Retry-After for a certificate request Google CAS couldn't serve for a reason expected to clear.
	issueRetryAfterSeconds = 30
)

type Service struct {
	logger   *slog.Logger
	timeout  time.Duration
	endpoint string
	tokenURL string

	// Created once and shared by every CA's client, like the EST client, so connections to Google are reused.
	transport   *http.Transport
	tokenClient *http.Client

	// issueClients holds one client per CA, so certificate requests reuse its cached access token instead of exchanging the key for a
	// new one on every request. A CA's entry is replaced when its key changes; entries for deleted CAs stay until the server restarts.
	mu           sync.Mutex
	issueClients map[uint]issueClient
}

type issueClient struct {
	key    fleet.GoogleCASKeyJSON
	client *privateca.Service
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
	s := &Service{timeout: defaultTimeout, tokenURL: google.JWTTokenURL, issueClients: make(map[uint]issueClient)}
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

func (s *Service) IssueCertificate(ctx context.Context, ca fleet.GoogleCASCA, csr *x509.CertificateRequest) (*x509.Certificate, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	client, err := s.issueClient(ca)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating Google CAS client")
	}
	request := &privateca.Certificate{
		PemCsr:   string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr.Raw})),
		Lifetime: fmt.Sprintf("%ds", int64(certificateLifetime/time.Second)),
	}
	// Enterprise-tier pools require a certificate ID that is unique in the location; DevOps-tier pools ignore it.
	issued, err := client.Projects.Locations.CaPools.Certificates.Create(ca.CAPoolResourceName(), request).
		CertificateId("fleet-" + uuid.NewString()).Context(ctx).Do()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, s.describeIssueError(ctx, err, ca), "issuing Google CAS certificate")
	}
	// A certificate Google issued but Fleet doesn't return stays valid, so it's logged for revocation. DevOps-tier pools keep no
	// record of their certificates and return no name, so the serial identifies it there.
	var serial string
	refuse := func(err error) (*x509.Certificate, error) {
		s.logger.ErrorContext(ctx, "Not returning the certificate Google CAS issued", "ca_pool", ca.CAPoolResourceName(),
			"certificate_name", issued.Name, "serial", serial, "err", err)
		return nil, err
	}
	block, _ := pem.Decode([]byte(issued.PemCertificate))
	if block == nil || block.Type != "CERTIFICATE" {
		return refuse(ctxerr.New(ctx, "Google CAS returned no certificate"))
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return refuse(ctxerr.Wrap(ctx, err, "parsing the certificate Google CAS issued"))
	}
	serial = cert.SerialNumber.String()
	// The private key stays on the host, so a certificate for any other key is useless to it and points at a misbehaving CA.
	if issuedKey, ok := cert.PublicKey.(interface{ Equal(crypto.PublicKey) bool }); !ok || !issuedKey.Equal(csr.PublicKey) {
		return refuse(ctxerr.New(ctx, "Google CAS issued a certificate for a different public key than the CSR's"))
	}
	// Fleet only asks for client certificates, so one that can sign certificates, CRLs or OCSP responses means the pool added it.
	if cert.IsCA || cert.KeyUsage&(x509.KeyUsageCertSign|x509.KeyUsageCRLSign) != 0 || slices.Contains(cert.ExtKeyUsage, x509.ExtKeyUsageOCSPSigning) {
		return refuse(ctxerr.New(ctx, "Google CAS issued a certificate that can sign other certificates. Check the CA pool's issuance policy."))
	}
	s.logger.InfoContext(ctx, "Google CAS issued a certificate", "ca_pool", ca.CAPoolResourceName(),
		"certificate_name", issued.Name, "serial", serial)
	return cert, nil
}

// issueClient returns the CA's cached client, creating it on first use and again when the CA's key changes.
func (s *Service) issueClient(ca fleet.GoogleCASCA) (*privateca.Service, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cached, ok := s.issueClients[ca.ID]; ok && cached.key == ca.APIKeyJSON {
		return cached.client, nil
	}
	// Built without the request's context, which is cancelled when the request ends, because the client outlives it. Token requests
	// still use each API request's context, so they respect its timeout.
	client, err := s.newClient(context.Background(), ca)
	if err != nil {
		return nil, err
	}
	s.issueClients[ca.ID] = issueClient{key: ca.APIKeyJSON, client: client}
	return client, nil
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

// describeIssueError is describeError for a certificate request, where a failure that should clear on its own is transient, so the
// caller gets a 503 with Retry-After and can try again.
func (s *Service) describeIssueError(ctx context.Context, err error, ca fleet.GoogleCASCA) error {
	s.logger.ErrorContext(ctx, "Google CAS certificate request failed", "ca_pool", ca.CAPoolResourceName(), "err", err)
	if apiErr, ok := errors.AsType[*googleapi.Error](err); ok {
		switch {
		case apiErr.Code == http.StatusTooManyRequests || apiErr.Code == http.StatusRequestTimeout || apiErr.Code >= http.StatusInternalServerError:
			return transientError(fmt.Sprintf("Google CAS couldn't issue the certificate (HTTP %d). Try again later.", apiErr.Code))
		case apiErr.Code == http.StatusUnauthorized || apiErr.Code == http.StatusForbidden:
			return errors.New("The service account can't issue certificates from the CA pool. Check the certificate authority's project ID, location and CA pool, and that the service account has the CA Service Certificate Manager role on the pool.")
		case apiErr.Code == http.StatusNotFound:
			// The requester may be a host, so the pool's project and location are left to the server log.
			return errors.New("The certificate authority's CA pool wasn't found. Check its Google Cloud project ID, location and CA pool.")
		case apiErr.Code == http.StatusBadRequest:
			// Google's reason is in the server log: it names the failed policy or field and the CA's resource names.
			return errors.New("Google CAS rejected the certificate request. Check that the CSR's subject and SANs are allowed by the CA pool's issuance policy and that the pool has an enabled CA.")
		}
		return fmt.Errorf("Google CAS returned an error (HTTP %d).", apiErr.Code)
	}
	if status, ok := tokenServiceUnavailable(err); ok {
		return transientError(fmt.Sprintf("Google couldn't issue an access token (HTTP %d). Try again later.", status))
	}
	if isConnectionError(err) {
		return transientError("Couldn't connect to Google CAS. Try again later.")
	}
	return errAuthentication
}

// transientError carries only a message, not the cause: wrapping a *net.OpError makes the HTTP encoder answer 408 instead of 503.
func transientError(message string) error {
	return fleet.CertificateAuthorityTransientError{Message: message, RetryAfterSeconds: issueRetryAfterSeconds}
}
