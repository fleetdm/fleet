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

type orbitRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
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
		return "", orbitError{
			message:     err.Error(),
			nodeInvalid: true,
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var resp enrollOrbitResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", orbitError{
			message:     err.Error(),
			nodeInvalid: true,
		}
	}
	return resp.OrbitNodeKey, nil
}

func (c *Client) GetFlags(orbitNodeKey string) (json.RawMessage, error) {
	verb, path := "POST", "/api/latest/fleet/orbit/flags"
	params := orbitRequest{OrbitNodeKey: orbitNodeKey}
	response, err := c.Do(verb, path, "", params)

	if err != nil {
		return nil, fmt.Errorf("POST /api/latest/fleet/orbit/enroll: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
