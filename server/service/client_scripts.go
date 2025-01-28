package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const pollWaitTime = 5 * time.Second

func (c *Client) RunHostScriptSync(hostID uint, scriptContents []byte, scriptName string, teamID uint) (*fleet.HostScriptResult, error) {
	verb, path := "POST", "/api/latest/fleet/scripts/run"
	res, err := c.runHostScript(verb, path, hostID, scriptContents, scriptName, teamID, http.StatusAccepted)
	if err != nil {
		return nil, err
	}

	if res.ExecutionID == "" {
		return nil, errors.New("missing execution id in response")
	}

	return c.pollForResult(res.ExecutionID)
}

func (c *Client) RunHostScriptAsync(hostID uint, scriptContents []byte, scriptName string, teamID uint) (*fleet.HostScriptResult, error) {
	verb, path := "POST", "/api/latest/fleet/scripts/run"
	return c.runHostScript(verb, path, hostID, scriptContents, scriptName, teamID, http.StatusAccepted)
}

func (c *Client) runHostScript(verb, path string, hostID uint, scriptContents []byte, scriptName string, teamID uint, successStatusCode int) (*fleet.HostScriptResult, error) {
	req := fleet.HostScriptRequestPayload{
		HostID:     hostID,
		ScriptName: scriptName,
		TeamID:     teamID,
	}
	if len(scriptContents) > 0 {
		req.ScriptContents = string(scriptContents)
	}

	var result fleet.HostScriptResult

	res, err := c.AuthenticatedDo(verb, path, "", &req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case successStatusCode:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("reading %s %s response: %w", verb, path, err)
		}
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, fmt.Errorf("decoding %s %s response: %w, body: %s", verb, path, err, b)
		}
	case http.StatusForbidden:
		errMsg, err := extractServerErrMsg(verb, path, res)
		if err != nil {
			return nil, err
		}
		if strings.Contains(errMsg, fleet.RunScriptScriptsDisabledGloballyErrMsg) {
			return nil, errors.New(fleet.RunScriptScriptsDisabledGloballyErrMsg)
		}
		return nil, errors.New(fleet.RunScriptForbiddenErrMsg)
	// It's possible we get a GatewayTimeout error message from nginx or another
	// proxy server, so we want to return a more helpful error message in that
	// case.
	case http.StatusGatewayTimeout:
		return nil, errors.New(fleet.RunScriptGatewayTimeoutErrMsg)
	case http.StatusPaymentRequired:
		if teamID > 0 {
			return nil, errors.New("Team id parameter requires Fleet Premium license.")
		}
		fallthrough // if no team id, fall through to default error message
	default:
		msg, err := extractServerErrMsg(verb, path, res)
		if err != nil {
			return nil, err
		}
		if msg == "" {
			msg = fmt.Sprintf("decoding %d response is missing expected message.", res.StatusCode)
		}
		return nil, errors.New(msg)
	}

	return &result, nil
}

func (c *Client) pollForResult(id string) (*fleet.HostScriptResult, error) {
	verb, path := "GET", fmt.Sprintf("/api/latest/fleet/scripts/results/%s", id)
	var result *fleet.HostScriptResult
	for {
		res, err := c.AuthenticatedDo(verb, path, "", nil)
		if err != nil {
			return nil, fmt.Errorf("polling for result: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {

			msg, err := extractServerErrMsg(verb, path, res)
			if err != nil {
				return nil, fmt.Errorf("extracting error message: %w", err)
			}
			if msg == "" {
				msg = fmt.Sprintf("decoding %d response is missing expected message.", res.StatusCode)
			}
			return nil, errors.New(msg)
		}

		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decoding response: %w", err)
		}

		if result.ExitCode != nil {
			break
		}

		time.Sleep(pollWaitTime)

	}

	return result, nil
}

// ApplyNoTeamScripts sends the list of scripts to be applied for the hosts in
// no team.
func (c *Client) ApplyNoTeamScripts(scripts []fleet.ScriptPayload, opts fleet.ApplySpecOptions) ([]fleet.ScriptResponse, error) {
	verb, path := "POST", "/api/latest/fleet/scripts/batch"
	var resp batchSetScriptsResponse
	err := c.authenticatedRequestWithQuery(map[string]interface{}{"scripts": scripts}, verb, path, &resp, opts.RawQuery())

	return resp.Scripts, err
}

func (c *Client) validateMacOSSetupScript(fileName string) ([]byte, error) {
	if err := c.CheckAppleMDMEnabled(); err != nil {
		return nil, err
	}

	b, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (c *Client) deleteMacOSSetupScript(teamID *uint) error {
	var query string
	if teamID != nil {
		query = fmt.Sprintf("team_id=%d", *teamID)
	}

	verb, path := "DELETE", "/api/latest/fleet/setup_experience/script"
	var delResp deleteSetupExperienceScriptResponse
	return c.authenticatedRequestWithQuery(nil, verb, path, &delResp, query)
}

func (c *Client) uploadMacOSSetupScript(filename string, data []byte, teamID *uint) error {
	// there is no "replace setup experience script" endpoint, and none was
	// planned, so to avoid delaying the feature I'm doing DELETE then SET, but
	// that's not ideal (will always re-create the script when apply/gitops is
	// run with the same yaml). Note though that we also redo software installers
	// downloads on each run, so the churn of this one is minor in comparison.
	if err := c.deleteMacOSSetupScript(teamID); err != nil {
		return err
	}

	verb, path := "POST", "/api/latest/fleet/setup_experience/script"

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("script", filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(fw, bytes.NewBuffer(data)); err != nil {
		return err
	}

	// add the team_id field
	if teamID != nil {
		if err := w.WriteField("team_id", fmt.Sprint(*teamID)); err != nil {
			return err
		}
	}
	w.Close()

	response, err := c.doContextWithBodyAndHeaders(context.Background(), verb, path, "",
		b.Bytes(),
		map[string]string{
			"Content-Type":  w.FormDataContentType(),
			"Accept":        "application/json",
			"Authorization": fmt.Sprintf("Bearer %s", c.token),
		},
	)
	if err != nil {
		return fmt.Errorf("do multipart request: %w", err)
	}
	defer response.Body.Close()

	var resp setSetupExperienceScriptResponse
	if err := c.parseResponse(verb, path, response, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}
