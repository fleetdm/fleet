package service

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Device client is used consume the `device/...` endpoints and meant to be used by Fleet Desktop
type DeviceClient struct {
	*baseClient
}

// NewDeviceClient instantiates a new client to perform requests against device
// endpoints
func NewDeviceClient(addr string, insecureSkipVerify bool, rootCA string) (*DeviceClient, error) {
	capabilities := fleet.CapabilityMap{}
	baseClient, err := newBaseClient(addr, insecureSkipVerify, rootCA, "", capabilities)
	if err != nil {
		return nil, err
	}

	return &DeviceClient{
		baseClient: baseClient,
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

// TransparencyURL returns an URL that the server will use to redirect to the
// transparency URL configured by the user
func (dc *DeviceClient) TransparencyURL(token string) string {
	return dc.baseClient.url("/api/latest/fleet/device/"+token+"/transparency", "").String()
}

// DeviceURL returns the public device URL for the given token
func (dc *DeviceClient) DeviceURL(token string) string {
	return dc.baseClient.url("/device/"+token, "").String()
}

// CheckToken checks if a token is valid by making an authenticated request to
// the server
func (dc *DeviceClient) CheckToken(token string) error {
	_, err := dc.NumberOfFailingPolicies(token)
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

func (dc *DeviceClient) NumberOfFailingPolicies(token string) (uint, error) {
	r, err := dc.getMinDesktopPayload(token)
	if err == nil {
		return uintValueOrZero(r.FailingPolicies), nil
	}

	if errors.Is(err, notFoundErr{}) {
		policies, err := dc.getListDevicePolicies(token)
		if err != nil {
			return 0, err
		}

		var failingPolicies uint
		for _, policy := range policies {
			if policy.Response != "pass" {
				failingPolicies++
			}
		}
		return failingPolicies, nil
	}

	return 0, err
}
