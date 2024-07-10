package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) RunHostScriptSync(hostID uint, scriptContents []byte, scriptName string, teamID uint) (*fleet.HostScriptResult, error) {
	verb, path := "POST", "/api/latest/fleet/scripts/run/sync"
	return c.runHostScript(verb, path, hostID, scriptContents, scriptName, teamID, http.StatusOK)
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

// ApplyNoTeamScripts sends the list of scripts to be applied for the hosts in
// no team.
func (c *Client) ApplyNoTeamScripts(scripts []fleet.ScriptPayload, opts fleet.ApplySpecOptions) error {
	verb, path := "POST", "/api/latest/fleet/scripts/batch"
	return c.authenticatedRequestWithQuery(map[string]interface{}{"scripts": scripts}, verb, path, nil, opts.RawQuery())
}
