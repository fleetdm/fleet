// Command throttle repeatedly transfers a single host back and forth between
// two teams to exercise Apple's DEP profile-assignment rate limiting, while
// polling the host's DEP assignment status and reporting when Apple returns
// THROTTLED.
// The team transferring between needs to have End User Auth enabled and disabled respectively so the assigned profile differs.
//
// Each team transfer forces Fleet to (re)assign the host's DEP enrollment
// profile in Apple Business Manager. Doing this rapidly is what provokes a
// THROTTLED response, which then surfaces on:
//
//	GET /api/latest/fleet/hosts/{id}/dep_assignment
//	  -> host_dep_assignment.assign_profile_response == "THROTTLED"
//
// The host is transferred on every `interval`, and the DEP assignment is polled
// on every `interval + 1s` so the poll drifts relative to the transfers and
// samples the status at varying points in the assign cycle.
//
// Usage:
//
//	go run ./tools/mdm/apple/throttle.go \
//	  -token "$FLEET_API_TOKEN" \
//	  -url https://fleet.example.com \
//	  -host 42 \
//	  -interval 2s \
//	  -from 1 \
//	  -to 2
//
// Use -from=-1 or -to=-1 to target "No team" (team_id: null).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
)

func main() {
	var (
		token    = flag.String("token", "", "Fleet API token (required)")
		baseURL  = flag.String("url", "", "Fleet base URL, e.g. https://fleet.example.com (required)")
		hostID   = flag.Uint("host", 0, "host ID to transfer back and forth (required)")
		interval = flag.Duration("interval", 2*time.Second, "transfer interval; DEP status is polled every interval+1s")
		fromTeam = flag.Int("from", -1, "team ID to transfer FROM (-1 = No team)")
		toTeam   = flag.Int("to", -1, "team ID to transfer TO (-1 = No team)")
	)
	flag.Parse()

	if *token == "" || *baseURL == "" || *hostID == 0 {
		flag.Usage()
		log.Fatal("-token, -url and -host are required")
	}
	if *fromTeam == *toTeam {
		log.Fatal("-from and -to must be different teams")
	}

	c := &client{
		http:    fleethttp.NewClient(),
		baseURL: strings.TrimRight(*baseURL, "/"),
		token:   *token,
		hostID:  *hostID,
	}

	// Trap Ctrl-C / SIGTERM so we can stop cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("transferring host %d between team %s and team %s every %s; polling DEP status every %s",
		*hostID, teamLabel(*fromTeam), teamLabel(*toTeam), *interval, *interval+time.Second)
	log.Printf("watching for assign_profile_response == %q (Ctrl-C to stop)", "THROTTLED")

	var throttledSeen atomic.Int64

	// Transfer loop: flip the host between the two teams every interval.
	go func() {
		targets := []int{*fromTeam, *toTeam}
		i := 0
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				target := targets[i%2]
				i++
				if err := c.transfer(ctx, teamPtr(target)); err != nil {
					log.Printf("transfer -> team %s: ERROR: %v", teamLabel(target), err)
					continue
				}
				log.Printf("transfer -> team %s: ok", teamLabel(target))
			}
		}
	}()

	// Poll loop: check the DEP assignment status every interval+1s.
	go func() {
		ticker := time.NewTicker(*interval + time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status, updatedAt, err := c.depAssignmentStatus(ctx)
				if err != nil {
					log.Printf("dep_assignment: ERROR: %v", err)
					continue
				}
				if status == "THROTTLED" {
					n := throttledSeen.Add(1)
					log.Printf("!!! THROTTLED (updated %s) — observed %d time(s)", updatedAt, n)
					continue
				}
				log.Printf("dep_assignment: %s", statusLabel(status))
			}
		}
	}()

	<-ctx.Done()
	log.Printf("stopping; THROTTLED observed %d time(s)", throttledSeen.Load())
}

type client struct {
	http    *http.Client
	baseURL string
	token   string
	hostID  uint
}

// transfer moves the host to the given team. A nil team means "No team".
func (c *client) transfer(ctx context.Context, teamID *uint) error {
	body, err := json.Marshal(struct {
		TeamID  *uint  `json:"team_id"` // nolint:apiparamcheck // custom tool
		HostIDs []uint `json:"hosts"`
	}{TeamID: teamID, HostIDs: []uint{c.hostID}})
	if err != nil {
		return err
	}
	_, err = c.do(ctx, http.MethodPost, "/api/latest/fleet/hosts/transfer", body)
	return err
}

// depAssignmentStatus returns the host's current assign_profile_response
// (e.g. SUCCESS, FAILED, NOT_ACCESSIBLE, THROTTLED) and when it was updated.
func (c *client) depAssignmentStatus(ctx context.Context) (status, updatedAt string, err error) {
	respBody, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/latest/fleet/hosts/%d/dep_assignment", c.hostID), nil)
	if err != nil {
		return "", "", err
	}

	var resp struct {
		HostDEPAssignment *struct {
			AssignProfileResponse *string `json:"assign_profile_response"`
			ResponseUpdatedAt     *string `json:"response_updated_at"`
		} `json:"host_dep_assignment"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", fmt.Errorf("decode dep_assignment response: %w", err)
	}
	if resp.HostDEPAssignment == nil {
		return "", "", nil
	}
	if resp.HostDEPAssignment.AssignProfileResponse != nil {
		status = *resp.HostDEPAssignment.AssignProfileResponse
	}
	if resp.HostDEPAssignment.ResponseUpdatedAt != nil {
		updatedAt = *resp.HostDEPAssignment.ResponseUpdatedAt
	}
	return status, updatedAt, nil
}

func (c *client) do(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s: unexpected status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return respBody, nil
}

// teamPtr converts a team-ID flag to the *uint the API expects: a negative
// value means "No team" (team_id: null).
func teamPtr(id int) *uint {
	if id < 0 {
		return nil
	}
	u := uint(id)
	return &u
}

func teamLabel(id int) string {
	if id < 0 {
		return "No team"
	}
	return fmt.Sprintf("%d", id)
}

func statusLabel(status string) string {
	if status == "" {
		return "(no assign_profile_response yet)"
	}
	return status
}
