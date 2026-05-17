package connectivity

import "time"

// Status is the classified outcome of a single probe.
type Status string

const (
	// StatusReachable means the endpoint is exposed, routable, and (when the
	// check opts in) responded with a Fleet-recognizable fingerprint. For
	// authenticated orbit endpoints this also means Fleet accepted the orbit
	// node key.
	StatusReachable Status = "reachable"
	// StatusForbidden means the server responded with 403. Unauthenticated
	// probes should not reach Fleet's authorization layer, so a 403 almost
	// always indicates a reverse proxy, WAF, or CDN rule blocking the path.
	StatusForbidden Status = "forbidden"
	// StatusNotFound means the server responded with 404. The path is not
	// wired up on the target — typically because the feature is disabled or
	// the reverse proxy rewrote the path.
	StatusNotFound Status = "not-found"
	// StatusBlocked means no HTTP response was received (DNS, TCP, TLS, or
	// timeout). This is the failure mode the tool is designed to surface.
	StatusBlocked Status = "blocked"
	// StatusNotFleet means an HTTP response came back but did not match any
	// Fleet-specific fingerprint the check required. Usually caused by a
	// reverse proxy, captive portal, or unrelated service intercepting the
	// path and returning its own page.
	StatusNotFleet Status = "not-fleet"
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
	Forbidden int `json:"forbidden"`
	NotFound  int `json:"not_found"`
	Blocked   int `json:"blocked"`
	NotFleet  int `json:"not_fleet"`
}

// Summarize counts results by status.
func Summarize(results []Result) Summary {
	s := Summary{Total: len(results)}
	for _, r := range results {
		switch r.Status {
		case StatusReachable:
			s.Reachable++
		case StatusForbidden:
			s.Forbidden++
		case StatusNotFound:
			s.NotFound++
		case StatusBlocked:
			s.Blocked++
		case StatusNotFleet:
			s.NotFleet++
		}
	}
	return s
}
