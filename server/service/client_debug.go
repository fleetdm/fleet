package service

import (
	"fmt"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (c *Client) getRawBody(endpoint string) ([]byte, error) {
	response, err := c.AuthenticatedDo("GET", endpoint, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := io.ReadAll(response.Body)
		if err == nil && len(body) > 0 {
			return nil, fmt.Errorf("get %s received status %d: %s", endpoint, response.StatusCode, body)
		}
		return nil, fmt.Errorf("get %s received status %d", endpoint, response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s response body: %w", endpoint, err)
	}

	return body, nil
}

// DebugPprof calls the /debug/pprof/ endpoints.
func (c *Client) DebugPprof(name string) ([]byte, error) {
	return c.getRawBody("/debug/pprof/" + name)
}

func (c *Client) DebugMigrations() (*fleet.MigrationStatus, error) {
	var migrationStatus fleet.MigrationStatus
	err := c.authenticatedRequest(nil, "GET", "/debug/migrations", &migrationStatus)
	if err != nil {
		return nil, err
	}
	return &migrationStatus, nil
}

// DebugErrors calls the /debug/errors endpoint and on success writes its
// (potentially large) response body to w.
func (c *Client) DebugErrors(w io.Writer, flush bool) error {
	endpoint := "/debug/errors"
	rawQuery := ""
	if flush {
		rawQuery = "flush=true"
	}
	response, err := c.AuthenticatedDo("GET", endpoint, rawQuery, nil)
	if err != nil {
		return fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("get errors received status %d", response.StatusCode)
	}

	if _, err := io.Copy(w, response.Body); err != nil {
		return fmt.Errorf("read errors response body: %w", err)
	}
	return nil
}

// DebugDBLocks calls the /debug/db/locks endpoint and on success returns its
// response body data.
func (c *Client) DebugDBLocks() ([]byte, error) {
	return c.getRawBody("/debug/db/locks")
}

func (c *Client) DebugInnoDBStatus() ([]byte, error) {
	return c.getRawBody("/debug/db/innodb-status")
}

func (c *Client) DebugProcessList() ([]byte, error) {
	return c.getRawBody("/debug/db/process-list")
}
