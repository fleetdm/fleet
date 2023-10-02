package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) RunHostScriptSync(hostID uint, scriptContents []byte) (*fleet.HostScriptResult, error) {
	verb, path := "POST", "/api/latest/fleet/scripts/run/sync"

	req := fleet.HostScriptRequestPayload{
		HostID:         hostID,
		ScriptContents: string(scriptContents),
	}

	var result fleet.HostScriptResult

	res, err := c.AuthenticatedDo(verb, path, "", &req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("reading %s %s response: %w", verb, path, err)
		}
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, fmt.Errorf("decoding %s %s response: %w, body: %s", verb, path, err, b)
		}
	case http.StatusForbidden:
		return nil, errors.New(fleet.RunScriptForbiddenErrMsg)

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
