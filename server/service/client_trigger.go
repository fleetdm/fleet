package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// TriggerCronSchedule attempts to trigger an ad-hoc run of the named cron schedule.
func (c *Client) TriggerCronSchedule(name string) error {
	verb, path := http.MethodPost, "/api/latest/fleet/trigger"

	query := url.Values{}
	query.Set("name", name)

	response, err := c.AuthenticatedDo(verb, path, query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("%s %s: %s", verb, path, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusConflict:
		msg, err := extractServerErrMsg(verb, path, response)
		if err != nil {
			return err
		}
		return conflictErr{msg: msg}
	case http.StatusNotFound:
		msg, err := extractServerErrMsg(verb, path, response)
		if err != nil {
			return err
		}
		return notFoundErr{msg: msg}
	default:
		return c.parseResponse(verb, path, response, nil)
	}
}

func extractServerErrMsg(verb string, path string, res *http.Response) (string, error) {
	var decoded serverError
	if err := json.NewDecoder(res.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("%s %s: decode server error: %s", verb, path, err)
	}
	if len(decoded.Errors) > 0 {
		return decoded.Errors[0].Reason, nil
	}
	return decoded.Message, nil
}
