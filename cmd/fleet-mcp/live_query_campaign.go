package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// createCampaignRequest is the body for POST /api/v1/fleet/reports/run. Selected
// targets the campaign by already-resolved host IDs.
type createCampaignRequest struct {
	Query    string          `json:"query"`
	Selected campaignTargets `json:"selected"`
}

type campaignTargets struct {
	Hosts []uint `json:"hosts"`
}

// createCampaignResponse captures the campaign ID returned when the campaign is
// created. Only the ID is needed to subscribe to the result stream.
type createCampaignResponse struct {
	Campaign struct {
		ID uint `json:"id"`
	} `json:"campaign"`
}

// wsJSONMessage mirrors server/websocket.JSONMessage on the read side. Data is
// left raw and decoded per Type.
type wsJSONMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// wsDistributedResult mirrors fleet.DistributedQueryResult (the "result" frame).
type wsDistributedResult struct {
	Host struct {
		ID          uint   `json:"id"`
		Hostname    string `json:"hostname"`
		DisplayName string `json:"display_name"`
	} `json:"host"`
	Rows  []map[string]string `json:"rows"`
	Error *string             `json:"error,omitempty"`
}

// wsTotals mirrors the service targetTotals struct (the "totals" frame).
type wsTotals struct {
	Total           uint `json:"count"`
	Online          uint `json:"online"`
	Offline         uint `json:"offline"`
	MissingInAction uint `json:"missing_in_action"`
}

// wsStatus mirrors the service campaignStatus struct (the "status" frame).
// ActualResults is the authoritative count of hosts that have reported
// (with or without rows).
type wsStatus struct {
	ExpectedResults uint   `json:"expected_results"`
	ActualResults   uint   `json:"actual_results"`
	Status          string `json:"status"`
}

// runMultiHostCampaign runs raw SQL against the given hosts via an ad-hoc live
// query campaign and returns the aggregated results. It is the multi-host live
// query path for every role.
func (fc *FleetClient) runMultiHostCampaign(ctx context.Context, hostIDs []uint, sql string, endpointByID map[uint]Endpoint) (*LiveQueryResult, error) {
	resp, err := fc.makeFleetRequest(ctx, "POST", "/api/v1/fleet/reports/run", createCampaignRequest{
		Query:    sql,
		Selected: campaignTargets{Hosts: hostIDs},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create live query campaign: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("live query campaign creation failed: %s", fleetErrMsg(resp.StatusCode, body))
	}

	var camp createCampaignResponse
	if err := json.NewDecoder(resp.Body).Decode(&camp); err != nil {
		return nil, fmt.Errorf("failed to decode campaign response: %w", err)
	}
	if camp.Campaign.ID == 0 {
		return nil, fmt.Errorf("live query campaign creation returned no campaign id")
	}

	logrus.Infof("Created live query campaign ID=%d, streaming results for %d hosts", camp.Campaign.ID, len(hostIDs))
	return fc.streamCampaignResults(ctx, camp.Campaign.ID, hostIDs, endpointByID)
}

// streamCampaignResults opens the results websocket, subscribes to the campaign,
// and reads frames until all online hosts have reported or the live-query
// deadline elapses. Partial results gathered before the deadline are returned.
func (fc *FleetClient) streamCampaignResults(ctx context.Context, campaignID uint, hostIDs []uint, endpointByID map[uint]Endpoint) (*LiveQueryResult, error) {
	wsURL, err := fc.campaignWebsocketURL()
	if err != nil {
		return nil, err
	}

	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  fc.tlsClientConfig(),
	}
	conn, _, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open live query results websocket: %w", err)
	}
	defer conn.Close()

	// Authenticate with the same token used for REST calls, then subscribe to
	// the campaign. WriteJSON marshals an arbitrary value, matching the server's
	// JSONMessage{Type, Data} envelope.
	if err := conn.WriteJSON(map[string]interface{}{"type": "auth", "data": map[string]string{"token": fc.apiKey}}); err != nil {
		return nil, fmt.Errorf("failed to authenticate results websocket: %w", err)
	}
	if err := conn.WriteJSON(map[string]interface{}{"type": "select_campaign", "data": map[string]interface{}{"campaign_id": campaignID}}); err != nil {
		return nil, fmt.Errorf("failed to subscribe to campaign: %w", err)
	}

	// Bound the read loop by the same deadline the synchronous REST endpoints
	// use, so a wedged or offline-heavy fleet can't pin us indefinitely.
	deadline := time.Now().Add(liveQueryDeadline())
	_ = conn.SetReadDeadline(deadline)

	// Closing the connection unblocks the blocking ReadJSON below, so a caller
	// cancellation (MCP client hangs up) interrupts the stream immediately
	// rather than waiting out the read deadline. The done channel stops this
	// watcher when the function returns normally.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	results := make([]map[string]interface{}, 0, len(hostIDs))
	var totals wsTotals
	var status wsStatus
	var gotTotals, gotStatus bool

	for {
		var msg wsJSONMessage
		if err := conn.ReadJSON(&msg); err != nil {
			// A cancelled context means the read was interrupted on purpose.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			// The read deadline elapsing is the expected upper bound — return
			// the partial results gathered so far rather than failing.
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				break
			}
			// A clean server-side close after it has streamed its frames is a
			// normal end of stream, not a failure.
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			// Anything else — auth rejection, protocol violation, dropped
			// connection — is a real error that must not masquerade as an empty
			// successful result.
			return nil, fmt.Errorf("live query campaign %d: results stream read failed: %w", campaignID, err)
		}

		switch msg.Type {
		case "result":
			var r wsDistributedResult
			if err := json.Unmarshal(msg.Data, &r); err != nil {
				logrus.Warnf("live query campaign %d: failed to decode result frame: %v", campaignID, err)
				continue
			}
			results = append(results, fc.campaignResultRow(r, endpointByID))
		case "totals":
			if err := json.Unmarshal(msg.Data, &totals); err != nil {
				logrus.Warnf("live query campaign %d: failed to decode totals frame: %v", campaignID, err)
				continue
			}
			gotTotals = true
		case "status":
			if err := json.Unmarshal(msg.Data, &status); err != nil {
				logrus.Warnf("live query campaign %d: failed to decode status frame: %v", campaignID, err)
				continue
			}
			gotStatus = true
		case "error":
			// The server reports post-subscription failures (campaign not found,
			// unauthorized, pubsub error) as an error frame, then closes the
			// stream. Surface it instead of returning an empty success.
			var em string
			if err := json.Unmarshal(msg.Data, &em); err != nil {
				em = string(msg.Data)
			}
			return nil, fmt.Errorf("live query campaign %d failed: %s", campaignID, em)
		}

		// The server reports a terminal "finished" status once the campaign
		// completes. Treat that as authoritative so termination never depends
		// solely on the count arithmetic below.
		if gotStatus && status.Status == "finished" {
			break
		}

		// Stop as soon as every online host has reported. Offline hosts never
		// report, so waiting for them would just burn the deadline. The server
		// only refreshes status on a 5s ticker, so also count result frames
		// directly — that lets the common all-hosts-return-rows case finish
		// promptly instead of idling until the next status tick.
		if gotTotals && totals.Online == 0 {
			break
		}
		if gotTotals && uint(len(results)) >= totals.Online {
			break
		}
		if gotTotals && gotStatus && status.ActualResults >= totals.Online {
			break
		}
	}

	targeted := len(hostIDs)
	if gotTotals {
		targeted = int(totals.Total)
	}
	// Count both result frames (hosts that returned rows) and the server's
	// status tally (which also counts hosts that responded with no rows), and
	// take the larger — status can lag behind the result frames we just read.
	responded := len(results)
	if gotStatus && int(status.ActualResults) > responded {
		responded = int(status.ActualResults)
	}

	return &LiveQueryResult{
		TargetedHostCount:  targeted,
		RespondedHostCount: responded,
		Results:            results,
	}, nil
}

// campaignResultRow converts a websocket result frame into the same row shape
// produced by the single-host ad-hoc path, preferring the locally-resolved host
// name and falling back to the name the server reports.
func (fc *FleetClient) campaignResultRow(r wsDistributedResult, endpointByID map[uint]Endpoint) map[string]interface{} {
	// A result frame means the host reported, so it is online — mirror the
	// single-host path, which always carries a "status" field. The server
	// injects host_hostname/host_display_name into every result row; strip them
	// so the campaign rows match the single-host shape (host identity lives in
	// the row's own host_id/host_name keys, not inside the osquery columns).
	for _, qr := range r.Rows {
		delete(qr, "host_hostname")
		delete(qr, "host_display_name")
	}

	row := map[string]interface{}{
		"host_id": r.Host.ID,
		"status":  "online",
		"rows":    r.Rows,
	}

	name := ""
	if ep, ok := endpointByID[r.Host.ID]; ok {
		name = ep.DisplayName
		if name == "" {
			name = ep.Name
		}
	}
	if name == "" {
		name = r.Host.DisplayName
	}
	if name == "" {
		name = r.Host.Hostname
	}
	if name != "" {
		row["host_name"] = name
	}

	if r.Error != nil {
		row["error"] = *r.Error
	}
	return row
}

// campaignWebsocketURL derives the results websocket URL from the configured
// base URL, mapping http→ws and https→wss.
func (fc *FleetClient) campaignWebsocketURL() (string, error) {
	u, err := url.Parse(fc.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid Fleet base URL %q: %w", fc.baseURL, err)
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("unsupported Fleet base URL scheme %q", u.Scheme)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/fleet/results/websocket"
	return u.String(), nil
}

// tlsClientConfig returns the TLS settings configured on the REST HTTP client
// (skip-verify / custom CA) so the websocket dialer trusts the same way. Returns
// nil when the transport isn't a *http.Transport (e.g. test clients), in which
// case the dialer uses its defaults.
func (fc *FleetClient) tlsClientConfig() *tls.Config {
	if tr, ok := fc.httpClient.Transport.(*http.Transport); ok {
		return tr.TLSClientConfig
	}
	return nil
}

// liveQueryDeadline is how long to wait for hosts to report. It mirrors the
// synchronous REST endpoints' FLEET_LIVE_QUERY_REST_PERIOD (default 25s).
func liveQueryDeadline() time.Duration {
	period := os.Getenv("FLEET_LIVE_QUERY_REST_PERIOD")
	if period == "" {
		return 25 * time.Second
	}
	d, err := time.ParseDuration(period)
	if err != nil {
		logrus.Warnf("invalid FLEET_LIVE_QUERY_REST_PERIOD %q, defaulting to 25s: %v", period, err)
		return 25 * time.Second
	}
	return d
}
