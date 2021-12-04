package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// GetHosts retrieves the list of all Hosts
func (c *Client) GetHosts(query string) ([]HostResponse, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/hosts", query, nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/hosts: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get hosts received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}
	var responseBody listHostsResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode list hosts response: %w", err)
	}
	if responseBody.Err != nil {
		return nil, fmt.Errorf("list hosts: %s", responseBody.Err)
	}

	return responseBody.Hosts, nil
}

// HostByIdentifier retrieves a host by the uuid, osquery_host_id, hostname, or
// node_key.
func (c *Client) HostByIdentifier(identifier string) (*HostDetailResponse, error) {
	response, err := c.AuthenticatedDo("GET", "/api/v1/fleet/hosts/identifier/"+identifier, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET /api/v1/fleet/hosts/identifier: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get host by identifier received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}
	var responseBody getHostResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, fmt.Errorf("decode host response: %w", err)
	}
	if responseBody.Err != nil {
		return nil, fmt.Errorf("get host by identifier: %s", responseBody.Err)
	}

	return responseBody.Host, nil
}

// DeleteHost deletes the host with the matching id.
func (c *Client) DeleteHost(id uint) error {
	verb := "DELETE"
	path := fmt.Sprintf("/api/v1/fleet/hosts/%d", id)
	response, err := c.AuthenticatedDo(verb, path, "", nil)
	if err != nil {
		return fmt.Errorf("%s %s: %w", verb, path, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNotFound:
		return notFoundErr{}
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"delete host received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody deleteHostResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return fmt.Errorf("decode delete host response: %w", err)
	}

	if responseBody.Err != nil {
		return fmt.Errorf("delete host: %s", responseBody.Err)
	}

	return nil
}

func (c *Client) translateTransferHostsToIDs(hosts []string, label string, team string) ([]uint, uint, uint, error) {
	verb, path := "POST", "/api/v1/fleet/translate"
	var responseBody translatorResponse

	var translatePayloads []fleet.TranslatePayload
	for _, host := range hosts {
		translatedPayload, err := encodeTranslatedPayload(fleet.TranslatorTypeHost, host)
		if err != nil {
			return nil, 0, 0, err
		}
		translatePayloads = append(translatePayloads, translatedPayload)
	}

	if label != "" {
		translatedPayload, err := encodeTranslatedPayload(fleet.TranslatorTypeLabel, label)
		if err != nil {
			return nil, 0, 0, err
		}
		translatePayloads = append(translatePayloads, translatedPayload)
	}

	translatedPayload, err := encodeTranslatedPayload(fleet.TranslatorTypeTeam, team)
	if err != nil {
		return nil, 0, 0, err
	}
	translatePayloads = append(translatePayloads, translatedPayload)

	params := translatorRequest{List: translatePayloads}

	err = c.authenticatedRequest(&params, verb, path, &responseBody)
	if err != nil {
		return nil, 0, 0, err
	}

	var hostIDs []uint
	var labelID uint
	var teamID uint

	for _, payload := range responseBody.List {
		switch payload.Type {
		case fleet.TranslatorTypeLabel:
			labelID = payload.Payload.ID
		case fleet.TranslatorTypeTeam:
			teamID = payload.Payload.ID
		case fleet.TranslatorTypeHost:
			hostIDs = append(hostIDs, payload.Payload.ID)
		}
	}
	return hostIDs, labelID, teamID, nil
}

func encodeTranslatedPayload(translatorType string, identifier string) (fleet.TranslatePayload, error) {
	translatedPayload := fleet.TranslatePayload{
		Type:    translatorType,
		Payload: fleet.StringIdentifierToIDPayload{Identifier: identifier},
	}
	return translatedPayload, nil
}

func (c *Client) TransferHosts(hosts []string, label string, status, searchQuery string, team string) error {
	hostIDs, labelID, teamID, err := c.translateTransferHostsToIDs(hosts, label, team)
	if err != nil {
		return err
	}

	if len(hosts) != 0 {
		verb, path := "POST", "/api/v1/fleet/hosts/transfer"
		var responseBody addHostsToTeamResponse
		params := addHostsToTeamRequest{TeamID: ptr.Uint(teamID), HostIDs: hostIDs}
		return c.authenticatedRequest(params, verb, path, &responseBody)
	}

	var labelIDPtr *uint
	if label != "" {
		labelIDPtr = &labelID
	}

	verb, path := "POST", "/api/v1/fleet/hosts/transfer/filter"
	var responseBody addHostsToTeamByFilterResponse
	params := addHostsToTeamByFilterRequest{TeamID: ptr.Uint(teamID), Filters: struct {
		MatchQuery string           `json:"query"`
		Status     fleet.HostStatus `json:"status"`
		LabelID    *uint            `json:"label_id"`
	}{MatchQuery: searchQuery, Status: fleet.HostStatus(status), LabelID: labelIDPtr}}
	return c.authenticatedRequest(params, verb, path, &responseBody)
}
