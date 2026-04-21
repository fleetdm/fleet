package connectivity

import "time"

// Status is the classified outcome of a single probe.
type Status string

const (
	// StatusReachable means an HTTP response was received from the server for
	// this path. Any status code other than 404 qualifies — we treat routed
	// 4xx/5xx as reachability, since the guide is about network exposure, not
	// feature configuration.
	StatusReachable Status = "reachable"
	// StatusNotFound means the server responded with 404. The path is not
	// wired up on the target — typically because the feature is disabled or
	// the reverse proxy rewrote the path.
	StatusNotFound Status = "not-found"
	// StatusBlocked means no HTTP response was received (DNS, TCP, TLS, or
	// timeout). This is the failure mode the tool is designed to surface.
	StatusBlocked Status = "blocked"
)

// Result is the outcome of running one Check.
type Result struct {
	Check      Check         `json:"check"`
	Status     Status        `json:"status"`
	HTTPStatus int           `json:"http_status,omitempty"`
	Latency    time.Duration `json:"latency_ns"`
	Error      string        `json:"error,omitempty"`
}

// Summary tallies results by status for quick reporting.
type Summary struct {
	Total     int `json:"total"`
	Reachable int `json:"reachable"`
	NotFound  int `json:"not_found"`
	Blocked   int `json:"blocked"`
}

// Summarize counts results by status.
func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch r.Status {
		case StatusReachable:
			s.Reachable++
		case StatusNotFound:
			s.NotFound++
		case StatusBlocked:
			s.Blocked++
		}
	}
	return s
}
