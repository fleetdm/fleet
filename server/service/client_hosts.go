package service

import (
	"encoding/csv"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GetHosts retrieves the list of all Hosts
func (c *Client) GetHosts(query string) ([]fleet.HostResponse, error) {
	verb, path := "GET", "/api/latest/fleet/hosts"
	var responseBody listHostsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query)
	return responseBody.Hosts, err
}

func (c *Client) GetHost(id uint) (*fleet.HostDetailResponse, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", id)
	var responseBody getHostResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	return responseBody.Host, err
}

// HostByIdentifier retrieves a host by the hardware_serial, uuid, hostname, osquery_host_id, or node_key. It doesn't match a host ID.
func (c *Client) HostByIdentifier(identifier string) (*fleet.HostDetailResponse, error) {
	verb, path := "GET", "/api/latest/fleet/hosts/"+identifier
	var responseBody getHostResponse
	if err := c.authenticatedRequest(nil, verb, path, &responseBody); err != nil {
		return nil, err
	}
	// An identifier that is also a literal route name, like "count", reaches that route instead, and its response has no host.
	if responseBody.Host == nil || matchesOnlyByHostID(identifier, responseBody.Host.Hostname, responseBody.Host.UUID,
		responseBody.Host.HardwareSerial) {
		return nil, &notFoundErr{Msg: fmt.Sprintf("host %q not found", identifier)}
	}
	return responseBody.Host, nil
}

// matchesOnlyByHostID reports whether a host returned for a numeric identifier matched it only as a host ID. GET /hosts/:id tries a
// numeric value as a host ID first, so a hostname or serial number made of digits can resolve to a different host.
func matchesOnlyByHostID(identifier, hostname, uuid, hardwareSerial string) bool {
	if _, err := strconv.ParseUint(identifier, 10, 0); err != nil {
		return false
	}
	return !strings.EqualFold(identifier, hostname) && !strings.EqualFold(identifier, uuid) && !strings.EqualFold(identifier, hardwareSerial)
}

func (c *Client) translateTransferHostsToIDs(hosts []string, label string, team string) ([]uint, uint, uint, error) {
	verb, path := "POST", "/api/latest/fleet/translate"
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

	if team != "" {
		translatedPayload, err := encodeTranslatedPayload(fleet.TranslatorTypeTeam, team)
		if err != nil {
			return nil, 0, 0, err
		}
		translatePayloads = append(translatePayloads, translatedPayload)
	}

	var hostIDs []uint
	var labelID uint
	var teamID uint

	if len(translatePayloads) == 0 {
		return hostIDs, labelID, teamID, nil
	}
	params := translatorRequest{List: translatePayloads}

	err := c.authenticatedRequest(&params, verb, path, &responseBody)
	if err != nil {
		return nil, 0, 0, err
	}

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

	var teamIDPtr *uint
	if teamID != 0 {
		teamIDPtr = &teamID
	}
	if len(hosts) != 0 {
		verb, path := "POST", "/api/latest/fleet/hosts/transfer"
		var responseBody addHostsToTeamResponse
		params := addHostsToTeamRequest{TeamID: teamIDPtr, HostIDs: hostIDs}
		return c.authenticatedRequest(params, verb, path, &responseBody)
	}

	filter := make(map[string]interface{})

	if label != "" {
		filter["label_id"] = labelID
	}

	if status != "" {
		filter["status"] = fleet.HostStatus(status)
	}

	if searchQuery != "" {
		filter["query"] = searchQuery
	}

	verb, path := "POST", "/api/latest/fleet/hosts/transfer/filter"
	var responseBody addHostsToTeamByFilterResponse
	params := addHostsToTeamByFilterRequest{
		TeamID:  teamIDPtr,
		Filters: &filter,
	}

	return c.authenticatedRequest(params, verb, path, &responseBody)
}

// GetHostsReport returns a report of all hosts.
//
// The first row holds the name of the columns and each subsequent row are
// the column values for each host.
func (c *Client) GetHostsReport(columns ...string) ([][]string, error) {
	verb, path := "GET", "/api/latest/fleet/hosts/report"
	query := make(url.Values)
	query.Add("format", "csv")
	if len(columns) > 0 {
		query.Add("columns", strings.Join(columns, ","))
	}
	response, err := c.AuthenticatedDo(verb, path, query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	csvReader := csv.NewReader(response.Body)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}
