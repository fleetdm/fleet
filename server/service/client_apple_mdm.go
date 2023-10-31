package service

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"howett.net/plist"
)

func (c *Client) RunMDMCommand(hostUUIDs []string, rawCmd []byte, forPlatform string) (*fleet.CommandEnqueueResult, error) {
	var prepareFn func([]byte) ([]byte, error)
	switch forPlatform {
	case "darwin":
		prepareFn = c.prepareAppleMDMCommand
	case "windows":
		prepareFn = c.prepareWindowsMDMCommand
	default:
		return nil, fmt.Errorf("Invalid platform %q. You can only run MDM commands on Windows or macOS hosts.", forPlatform)
	}

	rawCmd, err := prepareFn(rawCmd)
	if err != nil {
		return nil, err
	}

	request := runMDMCommandRequest{
		Command:   base64.RawStdEncoding.EncodeToString(rawCmd),
		HostUUIDs: hostUUIDs,
	}
	var response runMDMCommandResponse
	if err := c.authenticatedRequest(request, "POST", "/api/latest/fleet/mdm/commands/run", &response); err != nil {
		return nil, fmt.Errorf("run command request: %w", err)
	}
	return response.CommandEnqueueResult, nil
}

func (c *Client) prepareWindowsMDMCommand(rawCmd []byte) ([]byte, error) {
	if _, err := fleet.ParseWindowsMDMCommand(rawCmd); err != nil {
		return nil, err
	}
	return rawCmd, nil
}

func (c *Client) prepareAppleMDMCommand(rawCmd []byte) ([]byte, error) {
	var commandPayload map[string]interface{}
	if _, err := plist.Unmarshal(rawCmd, &commandPayload); err != nil {
		return nil, fmt.Errorf("The payload isn't valid XML. Please provide a file with valid XML: %w", err)
	}
	if commandPayload == nil {
		return nil, errors.New("The payload isn't valid. Please provide a valid MDM command in the form of a plist-encoded XML file.")
	}

	// generate a random command UUID
	commandPayload["CommandUUID"] = uuid.New().String()

	b, err := plist.Marshal(commandPayload, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("marshal command plist: %w", err)
	}
	return b, nil
}

func (c *Client) MDMGetCommandResults(commandUUID string) ([]*fleet.MDMCommandResult, error) {
	verb, path := http.MethodGet, "/api/latest/fleet/mdm/commandresults"

	query := url.Values{}
	query.Set("command_uuid", commandUUID)

	var responseBody getMDMCommandResultsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Results, nil
}

func (c *Client) MDMListCommands() ([]*fleet.MDMCommand, error) {
	const defaultCommandsPerPage = 1000

	verb, path := http.MethodGet, "/api/latest/fleet/mdm/commands"

	query := url.Values{}
	query.Set("per_page", fmt.Sprint(defaultCommandsPerPage))
	query.Set("order_key", "updated_at")
	query.Set("order_direction", "desc")

	var responseBody listMDMCommandsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Results, nil
}
