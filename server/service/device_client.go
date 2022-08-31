package service

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Device client is used to consume `/device/...` endpoints,
// and meant to be used by Fleet Desktop
type DeviceClient struct {
	*baseClient
	token string
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

	response, err := dc.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	return dc.parseResponse(verb, path, response, responseDest)
}

// NewDeviceClient instantiates a new client to perform requests against device endpoints
func NewDeviceClient(addr, token string, insecureSkipVerify bool, rootCA string) (*DeviceClient, error) {
	baseClient, err := newBaseClient(addr, insecureSkipVerify, rootCA, "")
	if err != nil {
		return nil, err
	}

	return &DeviceClient{
		baseClient: baseClient,
		token:      token,
	}, nil
}

func (dc *DeviceClient) SetToken(token string) {
	dc.token = token
}

func (dc *DeviceClient) DeviceURL() string {
	return dc.baseClient.url("/device/"+dc.token, "").String()
}

func (dc *DeviceClient) TransparencyURL() string {
	return dc.baseClient.url("/api/latest/fleet/device/"+dc.token+"/transparency", "").String()
}

// ListDevicePolicies fetches all policies for the device with the provided token
func (dc *DeviceClient) ListDevicePolicies() ([]*fleet.HostPolicy, error) {
	verb, path := "GET", "/api/latest/fleet/device/"+dc.token+"/policies"
	var responseBody listDevicePoliciesResponse
	err := dc.request(verb, path, "", &responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody.Policies, nil
}
