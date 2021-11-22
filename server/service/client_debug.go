package service

import (
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
	var migrationStatus fleet.MigrationStatus
	err := c.authenticatedRequest(nil, "GET", "/debug/migrations", &migrationStatus)
	if err != nil {
		return nil, err
	}
	return &migrationStatus, nil
}
