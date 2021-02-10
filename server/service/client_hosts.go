package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// GetHosts retrieves the list of all Hosts
func (c *Client) GetHosts() ([]HostResponse, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/hosts", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/fleet/hosts")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get hosts received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}
	var responseBody listHostsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode list hosts response")
	}
	if responseBody.Err != nil {
		return nil, errors.Errorf("list hosts: %s", responseBody.Err)
	}

	return responseBody.Hosts, nil
}

// HostByIdentifier retrieves a host by the uuid, osquery_host_id, hostname, or
// node_key.
func (c *Client) HostByIdentifier(identifier string) (*HostDetailResponse, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/hosts/identifier/"+identifier, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GET /api/v1/fleet/hosts/identifier")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get host by identifier received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}
	var responseBody getHostResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode host response")
	}
	if responseBody.Err != nil {
		return nil, errors.Errorf("get host by identifier: %s", responseBody.Err)
	}

	return responseBody.Host, nil
}

// DeleteHost deletes the host with the matching id.
func (c *Client) DeleteHost(id uint) error {
	verb := "DELETE"
	path := fmt.Sprintf("/api/v1/fleet/hosts/%d", id)
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return errors.Wrapf(err, "%s %s", verb, path)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return errors.Errorf(
			"delete host received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody deleteHostResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode delete host response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("delete host: %s", responseBody.Err)
	}

	return nil
}
