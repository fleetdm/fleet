package service

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/rs/zerolog/log"
)

// Device client is used consume the `device/...` endpoints and meant to be used by Fleet Desktop
type DeviceClient struct {
	*baseClient

	// fleetAlternativeBrowserHost is an alternative host to use for the Fleet Desktop URLs generated for the browser.
	//
	// This is needed when the host that Orbit will connect to is different from the host that will connect via the browser.
	fleetAlternativeBrowserHost string

	// if set and a request fails with ErrUnauthenticated, the client will call
	// this function to get a fresh token and retry if it returns a different,
	// non-empty token.
	invalidTokenRetryFunc func() string
}

// NewDeviceClient instantiates a new client to perform requests against device endpoints.
func NewDeviceClient(addr string, insecureSkipVerify bool, rootCA string, fleetClientCrt *tls.Certificate, fleetAlternativeBrowserHost string) (*DeviceClient, error) {
	capabilities := fleet.CapabilityMap{}
	baseClient, err := newBaseClient(addr, insecureSkipVerify, rootCA, "", fleetClientCrt, capabilities)
	if err != nil {
		return nil, err
	}

	return &DeviceClient{
		baseClient:                  baseClient,
		fleetAlternativeBrowserHost: fleetAlternativeBrowserHost,
	}, nil
}

// WithInvalidTokenRetry sets the function to call if a request fails with
// ErrUnauthenticated. The client will call this function to get a fresh token
// and retry if it returns a different, non-empty token.
func (dc *DeviceClient) WithInvalidTokenRetry(fn func() string) {
	log.Debug().Msg("setting invalid token retry hook")
	dc.invalidTokenRetryFunc = fn
}

// request performs the request, resolving the pathFmt that should contain a %s
// verb to be replaced with the token, or no verb at all if the token is "-"
// (the pathFmt is used as-is as path). It will retry if the request fails due
// to an invalid token and the invalidTokenRetryFunc field is set.
func (dc *DeviceClient) request(verb, pathFmt, token, query string, params interface{}, responseDest interface{}) error {
	const maxAttempts = 4
	var attempt int
	for {
		attempt++

		path := pathFmt
		if token != "-" {
			path = fmt.Sprintf(pathFmt, token)
		}
		reqErr := dc.requestAttempt(verb, path, query, params, responseDest)
		if attempt >= maxAttempts || dc.invalidTokenRetryFunc == nil || token == "-" || !errors.Is(reqErr, ErrUnauthenticated) {
			// no retry possible, return the result
			if reqErr != nil {
				log.Debug().Msgf("not retrying API error; attempt=%d, hook set=%t, token unset=%t, error is auth=%t",
					attempt, dc.invalidTokenRetryFunc != nil, token == "-", errors.Is(reqErr, ErrUnauthenticated))
			}
			return reqErr
		}

		delay := time.Duration(attempt) * time.Second
		log.Debug().Msgf("retrying API error in %s", delay)
		time.Sleep(delay)
		newToken := dc.invalidTokenRetryFunc()
		log.Debug().Msgf("retrying API error; token is different=%t", newToken != "" && newToken != token)
		if newToken != "" {
			token = newToken
		}
	}
}

func (dc *DeviceClient) requestAttempt(verb string, path string, query string, params interface{}, responseDest interface{}) error {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("making request json marshalling : %w", err)
		}
	}
	request, err := http.NewRequest(
		verb,
		dc.url(path, query).String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}

	dc.setClientCapabilitiesHeader(request)
	response, err := dc.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	return dc.parseResponse(verb, path, response, responseDest)
}

// BrowserTransparencyURL returns a URL for the browser that the server
// will use to redirect to the transparency URL configured by the user.
func (dc *DeviceClient) BrowserTransparencyURL(token string) string {
	transparencyURL := dc.baseClient.url("/api/latest/fleet/device/"+token+"/transparency", "")
	if dc.fleetAlternativeBrowserHost != "" {
		transparencyURL.Host = dc.fleetAlternativeBrowserHost
	}
	return transparencyURL.String()
}

// BrowserSelfServiceURL returns the "Self-service" URL for the browser.
func (dc *DeviceClient) BrowserSelfServiceURL(token string) string {
	selfServiceURL := dc.baseClient.url("/device/"+token+"/self-service", "")
	if dc.fleetAlternativeBrowserHost != "" {
		selfServiceURL.Host = dc.fleetAlternativeBrowserHost
	}
	return selfServiceURL.String()
}

// BrowserDeviceURL returns the "My device" URL for the browser.
func (dc *DeviceClient) BrowserDeviceURL(token string) string {
	deviceURL := dc.baseClient.url("/device/"+token, "")
	if dc.fleetAlternativeBrowserHost != "" {
		deviceURL.Host = dc.fleetAlternativeBrowserHost
	}
	return deviceURL.String()
}

// CheckToken checks if a token is valid by making an authenticated request to
// the server
func (dc *DeviceClient) CheckToken(token string) error {
	verb, path := "HEAD", "/api/latest/fleet/device/%s/ping"
	err := dc.request(verb, path, token, "", nil, nil)

	if errors.As(err, &notFoundErr{}) {
		// notFound is ok, it means an old server without the auth ping endpoint,
		// so we fall back to previously-used endpoint
		_, err = dc.DesktopSummary(token)
	}
	return err
}

// Ping sends a ping to the server using the device/ping endpoint
func (dc *DeviceClient) Ping() error {
	verb, path := "HEAD", "/api/fleet/device/ping"
	err := dc.request(verb, path, "-", "", nil, nil)

	if err == nil || errors.Is(err, notFoundErr{}) {
		// notFound is ok, it means an old server without the ping endpoint +
		// capabilities header
		return nil
	}

	return err
}

func (dc *DeviceClient) getListDevicePolicies(token string) ([]*fleet.HostPolicy, error) {
	verb, path := "GET", "/api/latest/fleet/device/%s/policies"
	var responseBody listDevicePoliciesResponse
	err := dc.request(verb, path, token, "", nil, &responseBody)
	return responseBody.Policies, err
}

func (dc *DeviceClient) getMinDesktopPayload(token string) (fleetDesktopResponse, error) {
	verb, path := "GET", "/api/latest/fleet/device/%s/desktop"
	var r fleetDesktopResponse
	err := dc.request(verb, path, token, "", nil, &r)
	return r, err
}

func (dc *DeviceClient) DesktopSummary(token string) (*fleetDesktopResponse, error) {
	r, err := dc.getMinDesktopPayload(token)
	if err == nil {
		r.FailingPolicies = ptr.Uint(uintValueOrZero(r.FailingPolicies))
		return &r, nil
	}

	if errors.Is(err, notFoundErr{}) {
		policies, err := dc.getListDevicePolicies(token)
		if err != nil {
			return nil, err
		}

		var failingPolicies uint
		for _, policy := range policies {
			if policy.Response != "pass" {
				failingPolicies++
			}
		}
		return &fleetDesktopResponse{
			DesktopSummary: fleet.DesktopSummary{
				FailingPolicies: ptr.Uint(failingPolicies),
			},
		}, nil
	}

	return nil, err
}

func (dc *DeviceClient) MigrateMDM(token string) error {
	verb, path := "POST", "/api/latest/fleet/device/%s/migrate_mdm"
	return dc.request(verb, path, token, "", nil, nil)
}

func (dc *DeviceClient) ReportError(token string, fleetdErr fleet.FleetdError) error {
	verb, path := "POST", "/api/latest/fleet/device/%s/debug/errors"
	req := fleetdErrorRequest{FleetdError: fleetdErr}
	return retry.Do(
		func() error {
			err := dc.request(verb, path, token, "", req, nil)
			if err != nil {
				return err
			}

			return nil
		},
		retry.WithMaxAttempts(3),
		retry.WithInterval(15*time.Second),
	)
}
