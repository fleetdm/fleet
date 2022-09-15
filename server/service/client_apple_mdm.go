package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

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

func (c *Client) EnqueueCommand(deviceIDs []string, rawPlist []byte) (*NanoMDMAPIResult, error) {
	var commandPayload map[string]interface{}
	_, err := plist.Unmarshal(rawPlist, &commandPayload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal command plist: %w", err)
	}

	// generate a random command UUID
	commandPayload["CommandUUID"] = uuid.New().String()

	b, err := plist.Marshal(commandPayload, plist.XMLFormat)
	if err != nil {
		return nil, fmt.Errorf("marshal command plist: %w", err)
	}

	// comma separated device IDs in path
	path := "/mdm/apple/mdm/api/v1/enqueue/" + strings.Join(deviceIDs, ",")

	u := c.url(path, "")
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, u.String(), bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	var result NanoMDMAPIResult
	err = json.Unmarshal(b, &result)
	if err != nil {
		return nil, fmt.Errorf("decode nano api response: %w", err)
	}

	return &result, nil
}

// These following types are copied from nanomdm.

// NanoMDMEnrolledAPIResult is a per-enrollment API result.
type NanoMDMEnrolledAPIResult struct {
	PushError    string `json:"push_error,omitempty"`
	PushResult   string `json:"push_result,omitempty"`
	CommandError string `json:"command_error,omitempty"`
}

// NanoMDMEnrolledAPIResults is a map of enrollments to a per-enrollment API result.
type NanoMDMEnrolledAPIResults map[string]*NanoMDMEnrolledAPIResult

// NanoMDMAPIResult is the JSON reply returned from either pushing or queuing commands.
type NanoMDMAPIResult struct {
	Status       NanoMDMEnrolledAPIResults `json:"status,omitempty"`
	NoPush       bool                      `json:"no_push,omitempty"`
	PushError    string                    `json:"push_error,omitempty"`
	CommandError string                    `json:"command_error,omitempty"`
	CommandUUID  string                    `json:"command_uuid,omitempty"`
	RequestType  string                    `json:"request_type,omitempty"`
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

	var installerResponse uploadMacOSInstallerResponse
	if err := c.parseResponse(verb, path, response, &installerResponse); err != nil {
		return 0, fmt.Errorf("parse response: %w", err)
	}
	return installerResponse.ID, nil
}
