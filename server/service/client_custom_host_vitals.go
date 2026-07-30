package service

import (
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) SaveCustomHostVitals(customHostVitals []fleet.CustomHostVital, dryRun bool) error {
	verb, path := "PUT", "/api/latest/fleet/spec/custom_host_vitals"
	params := fleet.UpsertCustomHostVitalsRequest{
		CustomHostVitals: customHostVitals,
		DryRun:           dryRun,
	}
	var responseBody fleet.UpsertCustomHostVitalsResponse
	return c.authenticatedRequest(params, verb, path, &responseBody)
}

// ListCustomHostVitals returns a page of custom host vital definitions.
func (c *Client) ListCustomHostVitals(query string) ([]fleet.CustomHostVital, error) {
	verb, path := "GET", "/api/latest/fleet/custom_host_vitals"
	var responseBody fleet.ListCustomHostVitalsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	if err != nil {
		return nil, err
	}
	return responseBody.CustomHostVitals, nil
}

// listAllCustomHostVitals pages through ListCustomHostVitals to return every definition.
func (c *Client) listAllCustomHostVitals() ([]fleet.CustomHostVital, error) {
	const perPage = 1000
	var all []fleet.CustomHostVital
	for page := 0; ; page++ {
		pageVitals, err := c.ListCustomHostVitals(fmt.Sprintf("per_page=%d&page=%d", perPage, page))
		if err != nil {
			return nil, err
		}
		all = append(all, pageVitals...)
		if len(pageVitals) < perPage {
			return all, nil
		}
	}
}
