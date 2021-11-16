package service

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

// DebugPprof calls the /debug/pprof/ endpoints.
func (c *Client) DebugPprof(name string) ([]byte, error) {
	endpoint := "/debug/pprof/" + name
	response, err := c.AuthenticatedDo("GET", endpoint, "", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GET %s", endpoint)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"get pprof received status %d",
			response.StatusCode,
		)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read pprof response body")
	}

	return body, nil
}

func (c *Client) DebugMigrations() (*fleet.MigrationStatus, error) {
	response, err := c.AuthenticatedDo("GET", "/debug/migrations", "", nil)
	if err != nil {
		return nil, errors.Wrap(err, "POST /api/v1/fleet/spec/labels")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"debug migrations received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}
	var migrationStatus fleet.MigrationStatus
	err = json.NewDecoder(response.Body).Decode(&migrationStatus)
	if err != nil {
		return nil, errors.Wrap(err, "decode debug migrations response")
	}
	return &migrationStatus, nil
}
