package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"howett.net/plist"
)

func (c *Client) CreateEnrollment(name string, depConfig *json.RawMessage) (*fleet.MDMAppleEnrollment, string, error) {
	request := createMDMAppleEnrollmentRequest{
		Name:      name,
		DEPConfig: depConfig,
	}
	var response createMDMAppleEnrollmentResponse
	if err := c.authenticatedRequest(request, "POST", "/api/latest/fleet/mdm/apple/enrollments", &response); err != nil {
		return nil, "", fmt.Errorf("request: %w", err)
	}
	return &fleet.MDMAppleEnrollment{
		ID:        response.ID,
		Name:      name,
		DEPConfig: depConfig,
	}, response.URL, nil
}

func (c *Client) EnqueueCommand(deviceIDs []string, rawPlist []byte) (*fleet.CommandEnqueueResult, error) {
	var commandPayload map[string]interface{}
	if _, err := plist.Unmarshal(rawPlist, &commandPayload); err != nil {
		return nil, fmt.Errorf("unmarshal command plist: %w", err)
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
		NoPush:    false,
	}
	var response enqueueMDMAppleCommandResponse
	if err := c.authenticatedRequest(request, "POST", "/api/latest/fleet/mdm/apple/enqueue", &response); err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	return &response.Result, nil
}

func (c *Client) MDMAppleGetCommandResults(commandUUID string) (map[string]*fleet.MDMAppleCommandResult, error) {
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

func (c *Client) UploadMDMAppleInstaller(ctx context.Context, name string, installer io.Reader) (uint, error) {
	if c.token == "" {
		return 0, errors.New("authentication token is empty")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, err := writer.CreateFormFile("installer", name)
	if err != nil {
		return 0, fmt.Errorf("create form file: %w", err)
	}
	_, err = io.Copy(fw, installer)
	if err != nil {
		return 0, fmt.Errorf("write form file: %w", err)
	}
	writer.Close()

	var (
		verb = "POST"
		path = "/api/latest/fleet/mdm/apple/installers"
	)
	response, err := c.doContextWithBodyAndHeaders(ctx, verb, path, "",
		body.Bytes(),
		map[string]string{
			"Content-Type":  writer.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", c.token),
		},
	)
	if err != nil {
		return 0, fmt.Errorf("do multipart request: %w", err)
	}

	var installerResponse uploadAppleInstallerResponse
	if err := c.parseResponse(verb, path, response, &installerResponse); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	return installerResponse.ID, nil
}

func (c *Client) MDMAppleGetInstallerDetails(id uint) (*fleet.MDMAppleInstaller, error) {
	verb, path := http.MethodGet, fmt.Sprintf("/api/latest/fleet/mdm/apple/installers/%d", id)

	var responseBody getAppleInstallerDetailsResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Installer, nil
}

func (c *Client) MDMAppleListDevices() ([]fleet.MDMAppleDevice, error) {
	verb, path := http.MethodGet, "/api/latest/fleet/mdm/apple/devices"

	var responseBody listMDMAppleDevicesResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Devices, nil
}

func (c *Client) DEPListDevices() ([]fleet.MDMAppleDEPDevice, error) {
	verb, path := http.MethodGet, "/api/latest/fleet/mdm/apple/dep/devices"

	var responseBody listMDMAppleDEPDevicesResponse
	err := c.authenticatedRequest(nil, verb, path, &responseBody)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return responseBody.Devices, nil
}
