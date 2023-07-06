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

func (c *Client) EnqueueCommand(deviceIDs []string, rawPlist []byte) (*fleet.CommandEnqueueResult, error) {
	var commandPayload map[string]interface{}
	if _, err := plist.Unmarshal(rawPlist, &commandPayload); err != nil {
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

	request := enqueueMDMAppleCommandRequest{
		Command:   base64.RawStdEncoding.EncodeToString(b),
		DeviceIDs: deviceIDs,
	}
	var response enqueueMDMAppleCommandResponse
	if err := c.authenticatedRequest(request, "POST", "/api/latest/fleet/mdm/apple/enqueue", &response); err != nil {
		return nil, fmt.Errorf("run command request: %w", err)
	}
	return response.CommandEnqueueResult, nil
}

func (c *Client) MDMAppleGetCommandResults(commandUUID string) ([]*fleet.MDMAppleCommandResult, error) {
	verb, path := http.MethodGet, "/api/latest/fleet/mdm/apple/commandresults"

	query := url.Values{}
	query.Set("command_uuid", commandUUID)

	var responseBody getMDMAppleCommandResultsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Results, nil
}

func (c *Client) MDMAppleListCommands() ([]*fleet.MDMAppleCommand, error) {
	const defaultCommandsPerPage = 1000

	verb, path := http.MethodGet, "/api/latest/fleet/mdm/apple/commands"

	query := url.Values{}
	query.Set("per_page", fmt.Sprint(defaultCommandsPerPage))
	query.Set("order_key", "updated_at")
	query.Set("order_direction", "desc")

	var responseBody listMDMAppleCommandsResponse
	err := c.authenticatedRequestWithQuery(nil, verb, path, &responseBody, query.Encode())
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Results, nil
}
