package conditional_access_google

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/str"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/cryptoutil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/cloudidentity/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

// EndpointOverrideEnv points the API and the token exchange at a mock server. Dev only.
const EndpointOverrideEnv = "FLEET_DEV_GOOGLE_CLOUD_IDENTITY_ENDPOINT"

// ClientStateSuffix follows the customer ID, without its "C", in Fleet's client state ID.
const ClientStateSuffix = "-fleet"

const maxGoogleResponseBytes = 1 << 20

func NewCloudIdentityFactory(logger *slog.Logger) CloudIdentityFactory {
	return func(ctx context.Context, creds Credentials) (CloudIdentity, error) {
		c, err := newCloudIdentity(ctx, creds, logger)
		if err != nil {
			return nil, err
		}
		return c, nil
	}
}

type cloudIdentityClient struct {
	svc            *cloudidentity.Service
	bareCustomerID string
}

func newCloudIdentity(ctx context.Context, creds Credentials, logger *slog.Logger) (*cloudIdentityClient, error) {
	// Google's device APIs take the ID without its "C".
	bareCustomerID := strings.TrimPrefix(creds.CustomerID, "C")
	key, err := parseServiceAccountKey(creds.ServiceAccountKey)
	if err != nil {
		return nil, err
	}

	endpoint := dev_mode.Env(EndpointOverrideEnv)
	conf := &jwt.Config{
		Email:        key.ClientEmail,
		PrivateKey:   []byte(key.PrivateKey),
		PrivateKeyID: key.PrivateKeyID,
		Scopes:       []string{cloudidentity.CloudIdentityDevicesScope},
		TokenURL:     tokenURL(endpoint),
		Subject:      creds.AdminEmail,
	}

	// The token exchange and the API calls share this client's timeout and response cap.
	httpClient := fleethttp.NewClient(fleethttp.WithMaxResponseSize(maxGoogleResponseBytes))
	opts := []option.ClientOption{option.WithHTTPClient(conf.Client(context.WithValue(ctx, oauth2.HTTPClient, httpClient)))}
	if endpoint != "" {
		logger.WarnContext(ctx, "using Google Cloud Identity endpoint override; do not use in production",
			"env", EndpointOverrideEnv, "endpoint", endpoint)
		opts = append(opts, option.WithEndpoint(endpoint))
	}
	svc, err := cloudidentity.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create cloud identity service: %w", err)
	}
	return &cloudIdentityClient{svc: svc, bareCustomerID: bareCustomerID}, nil
}

func (c *cloudIdentityClient) CheckDelegation(ctx context.Context) error {
	// Only authorization matters; fetch no device data.
	if _, err := c.svc.Devices.List().Customer(c.customer()).PageSize(1).Fields("nextPageToken").Context(ctx).Do(); err != nil {
		return &Error{err: err}
	}
	return nil
}

func (c *cloudIdentityClient) SetClientState(ctx context.Context, deviceUserName string, state fleet.GoogleConditionalAccessClientState) error {
	var body cloudidentity.GoogleAppsCloudidentityDevicesV1ClientState
	var mask []string
	if state.Managed != nil {
		body.Managed = "UNMANAGED"
		if *state.Managed {
			body.Managed = "MANAGED"
		}
		mask = append(mask, "managed")
	}
	if state.CustomID != nil {
		body.CustomId = *state.CustomID
		mask = append(mask, "customId")
	}
	if len(mask) == 0 {
		return errors.New("client state has no field to write")
	}
	name := deviceUserName + "/clientStates/" + c.bareCustomerID + ClientStateSuffix
	if _, err := c.svc.Devices.DeviceUsers.ClientStates.Patch(name, &body).
		Customer(c.customer()).UpdateMask(strings.Join(mask, ",")).Context(ctx).Do(); err != nil {
		return &Error{err: err, resourceCall: true}
	}
	return nil
}

func (c *cloudIdentityClient) customer() string {
	return "customers/" + c.bareCustomerID
}

type serviceAccountKey struct {
	ClientEmail  string `json:"client_email"`
	PrivateKeyID string `json:"private_key_id"`
	PrivateKey   string `json:"private_key"`
}

func parseServiceAccountKey(b []byte) (*serviceAccountKey, error) {
	var key serviceAccountKey
	if err := json.Unmarshal(b, &key); err != nil {
		return nil, errors.New("read the service account key: not a JSON object")
	}
	// The token source reports a key it can't use the way it reports a failed token
	// request, which IsUnavailable takes for an outage, so the key is checked here.
	privateKey, err := cryptoutil.ParsePrivateKey([]byte(key.PrivateKey), "private_key")
	if err != nil {
		return nil, fmt.Errorf("read the service account key: %w", err)
	}
	if _, ok := privateKey.(*rsa.PrivateKey); !ok {
		return nil, errors.New("read the service account key: private_key is not an RSA key")
	}
	return &key, nil
}

// tokenURL ignores the key's token_uri, which would receive the signed assertion.
func tokenURL(endpointOverride string) string {
	if endpointOverride != "" {
		return strings.TrimSuffix(endpointOverride, "/") + "/token"
	}
	return google.JWTTokenURL
}

// Error is a failed Google call whose message is safe to show the admin.
type Error struct {
	err error
	// resourceCall is set when a 404 means the device resource is gone.
	resourceCall bool
}

func (e *Error) Unwrap() error { return e.err }

func (e *Error) Error() string {
	return str.TruncateErrorResponse(e.message())
}

func (e *Error) message() string {
	if apiErr, ok := errors.AsType[*googleapi.Error](e.err); ok {
		if apiErr.Message != "" {
			return apiErr.Message
		}
		// Without a JSON reason the body is a proxy page; don't show it.
		return fmt.Sprintf("Google returned HTTP %d", apiErr.Code)
	}
	if tokenErr, ok := errors.AsType[*oauth2.RetrieveError](e.err); ok {
		return tokenErrorMessage(tokenErr)
	}
	if _, ok := errors.AsType[*http.MaxBytesError](e.err); ok || errors.Is(e.err, fleethttp.ErrMaxSizeExceeded) {
		return "Google's response is too large"
	}
	if urlErr, ok := errors.AsType[*url.Error](e.err); ok {
		return "request to Google failed: " + urlErr.Err.Error()
	}
	return e.err.Error()
}

func tokenErrorMessage(e *oauth2.RetrieveError) string {
	code, desc := e.ErrorCode, e.ErrorDescription
	if code == "" {
		// The jwt token source leaves these empty.
		var body struct {
			Code        string `json:"error"`
			Description string `json:"error_description"`
		}
		if json.Unmarshal(e.Body, &body) == nil {
			code, desc = body.Code, body.Description
		}
	}
	switch {
	case code != "" && desc != "":
		return code + ": " + desc
	case code != "":
		return code
	case e.Response != nil:
		return fmt.Sprintf("Google token request failed: HTTP %d", e.Response.StatusCode)
	}
	return "Google token request failed"
}

// IsNotFound reports a 404 on a call naming a device resource.
func (e *Error) IsNotFound() bool {
	if !e.resourceCall {
		return false
	}
	apiErr, ok := errors.AsType[*googleapi.Error](e.err)
	return ok && apiErr.Code == http.StatusNotFound
}

var _ fleet.NotFoundError = (*Error)(nil)

// IsUnavailable reports a failure on Google's side or on the way to Google, as opposed to
// settings or credentials Google rejects. Only a 4xx answer other than a 429 is a
// rejection: the token source flattens a failed token request into a string, so a
// failure without an answer can't be told apart by its cause.
func (e *Error) IsUnavailable() bool {
	status, answered := e.httpStatus()
	return !answered || status >= http.StatusInternalServerError || status == http.StatusTooManyRequests
}

// httpStatus is the status of Google's answer, when it answered with an error.
func (e *Error) httpStatus() (int, bool) {
	if apiErr, ok := errors.AsType[*googleapi.Error](e.err); ok {
		return apiErr.Code, true
	}
	if tokenErr, ok := errors.AsType[*oauth2.RetrieveError](e.err); ok && tokenErr.Response != nil {
		return tokenErr.Response.StatusCode, true
	}
	return 0, false
}

// IsUnavailable reports whether err is a Google call that failed on Google's side or on
// the way there (a 5xx, a 429, a timeout or a dropped connection), so retrying later can
// succeed with the same settings.
func IsUnavailable(err error) bool {
	u, ok := errors.AsType[interface {
		error
		IsUnavailable() bool
	}](err)
	return ok && u.IsUnavailable()
}
