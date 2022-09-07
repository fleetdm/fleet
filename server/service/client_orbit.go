package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type enrollOrbitRequest struct {
	EnrollSecret string `json:"enroll_secret"`
	HardwareUUID string `json:"hardware_uuid"`
}

type enrollOrbitResponse struct {
	OrbitNodeKey string `json:"orbit_node_key,omitempty"`
	Err          error  `json:"error,omitempty"`
}

type orbitGetConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

func (r *orbitGetConfigRequest) orbitHostNodeKey() string {
	return r.OrbitNodeKey
}

type orbitGetConfigResponse struct {
	Flags json.RawMessage `json:"command_line_startup_flags,omitempty"`
}

func NewOrbitClient(addr string, rootCA string, insecureSkipVerify bool) (*Client, error) {
	return NewClient(addr, insecureSkipVerify, rootCA, "")
}

func (c *Client) DoEnroll(enrollSecret string, hardwareUUID string) (string, error) {
	verb, path := "POST", "/api/latest/fleet/orbit/enroll"
	params := enrollOrbitRequest{EnrollSecret: enrollSecret, HardwareUUID: hardwareUUID}
	response, err := c.Do(verb, path, "", params)

	if err != nil {
		return "", fmt.Errorf("POST /api/latest/fleet/orbit/enroll: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Error POST /api/latest/fleet/orbit/enroll: %w", err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("Error POST /api/latest/fleet/orbit/enroll: %w", err)
	}

	var resp enrollOrbitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("Error POST /api/latest/fleet/orbit/enroll: %w", err)
	}
	return resp.OrbitNodeKey, nil
}

func (c *Client) GetConfig(orbitNodeKey string) (json.RawMessage, error) {
	verb, path := "POST", "/api/latest/fleet/orbit/flags"
	params := orbitGetConfigRequest{OrbitNodeKey: orbitNodeKey}
	response, err := c.Do(verb, path, "", params)

	if err != nil {
		return nil, fmt.Errorf("POST /api/latest/fleet/orbit/flags: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("POST /api/latest/fleet/orbit/flags: %w", err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("POST /api/latest/fleet/orbit/flags: %w", err)
	}

	resp := &orbitGetConfigResponse{
		Flags: body,
	}

	return resp.Flags, nil
}
