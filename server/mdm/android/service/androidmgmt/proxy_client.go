package androidmgmt

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-json-experiment/json"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const defaultProxyEndpoint = "https://fleetdm.com/api/android/"

// ProxyClient connects to Google's Android Management API via a proxy, which is hosted at fleetdm.com by default.
type ProxyClient struct {
	logger            kitlog.Logger
	mgmt              *androidmanagement.Service
	licenseKey        string
	proxyEndpoint     string
	fleetServerSecret string
}

// Compile-time check to ensure that ProxyClient implements Client.
var _ Client = &ProxyClient{}

func NewProxyClient(ctx context.Context, logger kitlog.Logger, licenseKey string, getenv func(string) string) Client {
	proxyEndpoint := getenv("FLEET_DEV_ANDROID_PROXY_ENDPOINT")
	if proxyEndpoint == "" {
		proxyEndpoint = defaultProxyEndpoint
	}

	slogLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// We use the same client that we use to directly connect to Google to minimize issues/maintenance.
	// But we point it to our proxy endpoint instead of Google.
	mgmt, err := androidmanagement.NewService(ctx,
		option.WithEndpoint(proxyEndpoint),
		option.WithLogger(slogLogger),
		// The API key is required to exist but not used by this client. Instead, we use the FleetServerSecret as a bearer token.
		option.WithAPIKey("not_used"),
		option.WithHTTPClient(fleethttp.NewClient()),
	)
	if err != nil {
		level.Error(logger).Log("msg", "creating android management service", "err", err)
		return nil
	}
	return &ProxyClient{
		logger:        logger,
		mgmt:          mgmt,
		licenseKey:    licenseKey,
		proxyEndpoint: proxyEndpoint,
	}
}

func (p *ProxyClient) SetAuthenticationSecret(secret string) error {
	p.fleetServerSecret = secret
	return nil
}

// SignupURLsCreate hits the unauthenticated endpoint of the proxy. If a record already exists for this serverURL,
// then the proxy will return a conflict error.
func (p *ProxyClient) SignupURLsCreate(ctx context.Context, serverURL, callbackURL string) (*android.SignupDetails, error) {
	if p == nil || p.mgmt == nil {
		return nil, errors.New("android management service not initialized")
	}
	call := p.mgmt.SignupUrls.Create().CallbackUrl(callbackURL).Context(ctx)
	call.Header().Set("Origin", serverURL)
	signupURL, err := call.Do()
	switch {
	case isErrorCode(err, http.StatusConflict):
		// The frontend looks for the text in this error. Please update the frontend code if modifying this error.
		return nil, android.NewConflictError(fmt.Errorf("android enterprise already exists. Contact Fleet support for help: %w", err))
	case err != nil:
		return nil, fmt.Errorf("creating signup url: %w", err)
	}
	return &android.SignupDetails{
		Url:  signupURL.Url,
		Name: signupURL.Name,
	}, nil
}

// EnterprisesCreate hits a custom endpoint of the proxy that does not exactly match the Google API's counterpart.
// The reason is that we are passing additional information such as license key, pubSubURL, etc. Because of that,
// we use a separate HTTP client in this method.
func (p *ProxyClient) EnterprisesCreate(ctx context.Context, req EnterprisesCreateRequest) (EnterprisesCreateResponse, error) {
	if p == nil || p.mgmt == nil {
		return EnterprisesCreateResponse{}, errors.New("android management service not initialized")
	}

	type proxyEnterprise struct {
		FleetLicenseKey string                       `json:"fleetLicenseKey"`
		PubSubPushURL   string                       `json:"pubsubPushUrl"`
		EnterpriseToken string                       `json:"enterpriseToken"`
		SignupURLName   string                       `json:"signupUrlName"`
		Enterprise      androidmanagement.Enterprise `json:"enterprise"`
	}

	client := fleethttp.NewClient()
	pe := proxyEnterprise{
		FleetLicenseKey: p.licenseKey,
		PubSubPushURL:   req.PubSubPushURL,
		EnterpriseToken: req.EnterpriseToken,
		SignupURLName:   req.SignupURLName,
		Enterprise: androidmanagement.Enterprise{
			EnabledNotificationTypes: req.EnabledNotificationTypes,
		},
	}

	reqBody, err := json.Marshal(pe)
	if err != nil {
		return EnterprisesCreateResponse{}, fmt.Errorf("marshaling enterprise request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.proxyEndpoint+"v1/enterprises", bytes.NewBuffer(reqBody))
	if err != nil {
		return EnterprisesCreateResponse{}, fmt.Errorf("creating enterprise request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Origin", req.ServerURL)

	resp, err := client.Do(httpReq)
	if err != nil {
		return EnterprisesCreateResponse{}, fmt.Errorf("sending enterprise request: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotModified:
		return EnterprisesCreateResponse{}, fmt.Errorf("android enterprise %s was already created", req.SignupURLName)
	case resp.StatusCode != http.StatusOK:
		return EnterprisesCreateResponse{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	type proxyEnterpriseResponse struct {
		FleetServerSecret string `json:"fleetServerSecret"`
		Name              string `json:"name"`
	}
	var per proxyEnterpriseResponse
	if err := json.UnmarshalRead(resp.Body, &per); err != nil {
		return EnterprisesCreateResponse{}, fmt.Errorf("decoding enterprise response: %w", err)
	}
	return EnterprisesCreateResponse{
		EnterpriseName:    per.Name,
		FleetServerSecret: per.FleetServerSecret,
	}, nil
}

func (p *ProxyClient) EnterprisesPoliciesPatch(ctx context.Context, policyName string, policy *androidmanagement.Policy) error {
	call := p.mgmt.Enterprises.Policies.Patch(policyName, policy).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+p.fleetServerSecret)
	_, err := call.Do()
	switch {
	case googleapi.IsNotModified(err):
		p.logger.Log("msg", "Android policy not modified", "policy_name", policyName)
	case err != nil:
		return fmt.Errorf("patching policy %s: %w", policyName, err)
	}
	return nil
}

func (p *ProxyClient) EnterprisesEnrollmentTokensCreate(ctx context.Context, enterpriseName string,
	token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
	if p == nil || p.mgmt == nil {
		return nil, errors.New("android management service not initialized")
	}
	call := p.mgmt.Enterprises.EnrollmentTokens.Create(enterpriseName, token).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+p.fleetServerSecret)
	token, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("creating enrollment token: %w", err)
	}
	return token, nil
}

func (p *ProxyClient) EnterpriseDelete(ctx context.Context, enterpriseName string) error {
	if p == nil || p.mgmt == nil {
		return errors.New("android management service not initialized")
	}

	call := p.mgmt.Enterprises.Delete(enterpriseName).Context(ctx)
	call.Header().Set("Authorization", "Bearer "+p.fleetServerSecret)
	_, err := call.Do()
	switch {
	case googleapi.IsNotModified(err) || isErrorCode(err, http.StatusNotFound):
		level.Info(p.logger).Log("msg", "enterprise was already deleted", "enterprise_name", enterpriseName)
		return nil
	case err != nil:
		return fmt.Errorf("deleting enterprise %s: %w", enterpriseName, err)
	}

	return nil
}

func isErrorCode(err error, code int) bool {
	if err == nil {
		return false
	}
	var ae *googleapi.Error
	ok := errors.As(err, &ae)
	return ok && ae.Code == code
}
