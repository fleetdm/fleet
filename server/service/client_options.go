package service

import (
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

// ApplyOptions sends the osquery options to be applied to the Fleet instance.
func (c *Client) ApplyOptions(spec *kolide.OptionsSpec) error {
	req := applyOsqueryOptionsSpecRequest{Spec: spec}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/spec/osquery_options", "", req)
	if err != nil {
		return errors.Wrap(err, "POST /api/v1/fleet/spec/osquery_options")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.Errorf(
			"apply options received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody applyOsqueryOptionsSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return errors.Wrap(err, "decode apply options spec response")
	}

	if responseBody.Err != nil {
		return errors.Errorf("apply options spec: %s", responseBody.Err)
	}

	return nil
}

// GetOptions retrieves the configured osquery options.
func (c *Client) GetOptions() (*kolide.OptionsSpec, error) {
	verb, path := "GET", "/api/v1/fleet/spec/osquery_options"
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return nil, errors.Wrap(err, verb+" "+path)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return nil, notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get options received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody getOsqueryOptionsSpecResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode get options spec response")
	}

	if responseBody.Err != nil {
		return nil, errors.Errorf("get options spec: %s", responseBody.Err)
	}

	return responseBody.Spec, nil
}
