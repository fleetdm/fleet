// Package apnsmock reads the status of a local mock APNs server
// (cmd/apple-apns-mock), which Hangar runs per Fleet server so osquery-perf's
// simulated Apple devices can receive MDM pushes.
package apnsmock

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Status is what the Server and osquery-perf tabs show for one mock.
type Status struct {
	Up  bool   `json:"up"`
	URL string `json:"url"`
	// ActiveConnections is the number of simulated devices holding a push
	// stream; TotalPushes counts pushes Fleet has sent since the mock started.
	ActiveConnections int64 `json:"active_connections"`
	TotalPushes       int64 `json:"total_pushes"`
}

// URL is the base URL Fleet pushes to and osquery-perf listens on for a mock on
// port. 127.0.0.1 rather than localhost because the mock listens on IPv4 only.
func URL(port uint16) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// Probe reads GET /stats, which only answers once the mock is serving and
// connected to Redis.
func Probe(port uint16) Status {
	base := URL(port)
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(base + "/stats") //nolint:noctx // short local readiness probe
	if err != nil {
		return Status{URL: base}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Status{URL: base}
	}
	var stats struct {
		NodeID string `json:"node_id"`
		Node   struct {
			ActiveConnections int64 `json:"active_connections"`
			TotalPushes       int64 `json:"total_pushes"`
		} `json:"node"`
	}
	// node_id tells the mock apart from some other server that happens to
	// answer /stats with JSON.
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&stats); err != nil || stats.NodeID == "" {
		return Status{URL: base}
	}
	return Status{
		Up:                true,
		URL:               base,
		ActiveConnections: stats.Node.ActiveConnections,
		TotalPushes:       stats.Node.TotalPushes,
	}
}
