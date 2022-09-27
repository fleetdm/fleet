package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type OrbitClient struct {
	*baseClient
	enrollSecret string
	hardwareUUID string
}

func (oc *OrbitClient) request(verb string, path string, params interface{}, resp interface{}) error {
	var bodyBytes []byte
	var err error
	if params != nil {
		bodyBytes, err = json.Marshal(params)
		if err != nil {
			return fmt.Errorf("making requst json marshalling : %w", err)
		}
	}

	request, err := http.NewRequest(
		verb,
		oc.url(path, "").String(),
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return err
	}
	oc.setClientCapabilitiesHeader(request)
	response, err := oc.http.Do(request)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	return oc.parseResponse(verb, path, response, resp)
}

func NewOrbitClient(addr string, rootCA string, insecureSkipVerify bool, enrollSecret, hardwareUUID string, capabilities fleet.CapabilityMap) (*OrbitClient, error) {
	bc, err := newBaseClient(addr, insecureSkipVerify, rootCA, "", capabilities)
	if err != nil {
		return nil, err
	}

	return &OrbitClient{
		baseClient:   bc,
		enrollSecret: enrollSecret,
		hardwareUUID: hardwareUUID,
	}, nil
}

func (oc *OrbitClient) DoEnroll() (string, error) {
	verb, path := "POST", "/api/fleet/orbit/enroll"
	params := enrollOrbitRequest{EnrollSecret: oc.enrollSecret, HardwareUUID: oc.hardwareUUID}
	var resp enrollOrbitResponse
	err := oc.request(verb, path, params, &resp)
	if err != nil {
		return "", err
	}
	return resp.OrbitNodeKey, nil
}

func (oc *OrbitClient) GetConfig(orbitNodeKey string) (json.RawMessage, error) {
	verb, path := "POST", "/api/fleet/orbit/config"
	params := orbitGetConfigRequest{OrbitNodeKey: orbitNodeKey}
	var resp orbitGetConfigResponse
	err := oc.request(verb, path, params, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Flags, nil
}

func (oc *OrbitClient) Ping() error {
	verb, path := "HEAD", "/api/fleet/orbit/ping"
	err := oc.request(verb, path, nil, nil)

	if err == nil || errors.Is(err, notFoundErr{}) {
		// notFound is ok, it means an old server without the capabilities header
		return nil
	}

	return err
}
