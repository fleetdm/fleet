package service

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

// DebugPprof calls the /debug/pprof/ endpoints.
func (c *Client) DebugPprof(name string) ([]byte, error) {
	endpoint := "/debug/pprof/" + name
	response, err := c.AuthenticatedDo("GET", endpoint, "", nil)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", endpoint, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"get pprof received status %d",
			response.StatusCode,
		)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read pprof response body: %w", err)
	}

	return body, nil
}
