package service

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// Device client is used consume the `device/...` endpoints and meant to be used by Fleet Desktop
type DeviceClient struct {
	*baseClient

	// fleetAlternativeBrowserHost is an alternative host to use for the Fleet Desktop URLs generated for the browser.
	//
	// This is needed when the host that Orbit will connect to is different from the host that will connect via the browser.
	fleetAlternativeBrowserHost string
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

func (dc *DeviceClient) request(verb string, path string, query string, responseDest interface{}) error {
	var bodyBytes []byte
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
	_, err := dc.DesktopSummary(token)
	return err
}

// Ping sends a ping to the server using the device/ping endpoint
func (dc *DeviceClient) Ping() error {
	verb, path := "HEAD", "/api/fleet/device/ping"
	err := dc.request(verb, path, "", nil)

	if err == nil || errors.Is(err, notFoundErr{}) {
		// notFound is ok, it means an old server without the ping endpoint +
		// capabilities header
		return nil
	}

	return err
}

func (dc *DeviceClient) getListDevicePolicies(token string) ([]*fleet.HostPolicy, error) {
	verb, path := "GET", "/api/latest/fleet/device/"+token+"/policies"
	var responseBody listDevicePoliciesResponse
	err := dc.request(verb, path, "", &responseBody)
	return responseBody.Policies, err
}

func (dc *DeviceClient) getMinDesktopPayload(token string) (fleetDesktopResponse, error) {
	verb, path := "GET", "/api/latest/fleet/device/"+token+"/desktop"
	var r fleetDesktopResponse
	err := dc.request(verb, path, "", &r)
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
	verb, path := "POST", "/api/latest/fleet/device/"+token+"/migrate_mdm"
	return dc.request(verb, path, "", nil)
}
