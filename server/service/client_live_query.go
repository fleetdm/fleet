package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"

	ws "github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/gorilla/websocket"
)

// LiveQueryResultsHandler provides access to all of the information about an
// incoming stream of live query results.
type LiveQueryResultsHandler struct {
	errors  chan error
	results chan fleet.DistributedQueryResult
	totals  atomic.Value // real type: targetTotals
	status  atomic.Value // real type: campaignStatus
}

func NewLiveQueryResultsHandler() *LiveQueryResultsHandler {
	return &LiveQueryResultsHandler{
		errors:  make(chan error),
		results: make(chan fleet.DistributedQueryResult),
	}
}

// Errors returns a read channel that includes any errors returned by the
// server or receiving the results.
func (h *LiveQueryResultsHandler) Errors() <-chan error {
	return h.errors
}

// Results returns a read channel including any received results
func (h *LiveQueryResultsHandler) Results() <-chan fleet.DistributedQueryResult {
	return h.results
}

// Totals returns the current metadata of hosts targeted by the query
func (h *LiveQueryResultsHandler) Totals() *targetTotals {
	t := h.totals.Load()
	if t != nil {
		return t.(*targetTotals)
	}
	return nil
}

func (h *LiveQueryResultsHandler) Status() *campaignStatus {
	s := h.status.Load()
	if s != nil {
		return s.(*campaignStatus)
	}
	return nil
}

// LiveQuery creates a new live query and begins streaming results.
func (c *Client) LiveQuery(query string, queryID *uint, labels []string, hostIdentifiers []string) (*LiveQueryResultsHandler, error) {
	return c.LiveQueryWithContext(context.Background(), query, queryID, labels, hostIdentifiers)
}

func (c *Client) LiveQueryWithContext(
	ctx context.Context, query string, queryID *uint, labels []string, hostIdentifiers []string,
) (*LiveQueryResultsHandler, error) {
	req := createDistributedQueryCampaignByIdentifierRequest{
		QueryID:  queryID,
		QuerySQL: query,
		Selected: distributedQueryCampaignTargetsByIdentifiers{Labels: labels, Hosts: hostIdentifiers},
	}
	verb, path := "POST", "/api/latest/fleet/queries/run_by_identifiers"
	var responseBody createDistributedQueryCampaignResponse
	err := c.authenticatedRequest(req, verb, path, &responseBody)
	if err != nil {
		return nil, ctxerr.Errorf(ctx, "create live query: %v", err)
	}

	// Copy default dialer but skip cert verification if set.
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: c.insecureSkipVerify},
	}

	wssURL := *c.baseURL
	wssURL.Scheme = "wss"
	if flag.Lookup("test.v") != nil {
		wssURL.Scheme = "ws"
	}
	wssURL.Path = c.urlPrefix + "/api/latest/fleet/results/websocket"
	// Ensure custom headers (set by config) are added to websocket request
	headers := make(http.Header)
	for k, v := range c.customHeaders {
		headers.Set(k, v)
	}
	conn, _, err := dialer.Dial(wssURL.String(), headers)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "upgrade live query result websocket")
	}
	// Cannot defer connection closing here because we need it to remain
	// open for the goroutine below. Manually close for the couple of error
	// cases below until we enter that goroutine.

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "auth",
		Data: map[string]interface{}{"token": c.token},
	})
	if err != nil {
		_ = conn.Close()
		return nil, ctxerr.Wrap(ctx, err, "auth for results")
	}

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "select_campaign",
		Data: map[string]interface{}{"campaign_id": responseBody.Campaign.ID},
	})
	if err != nil {
		_ = conn.Close()
		return nil, ctxerr.Wrap(ctx, err, "selecting results")
	}

	resHandler := NewLiveQueryResultsHandler()
	go func() {
		defer conn.Close()
		for {
			msg := struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}{}

			doneReadingChan := make(chan error)

			go func() {
				doneReadingChan <- conn.ReadJSON(&msg)
			}()

			select {
			case <-ctx.Done():
				return
			case err := <-doneReadingChan:
				if err != nil {
					resHandler.errors <- ctxerr.Wrap(ctx, err, "receive ws message")
					if errors.Is(err, websocket.ErrCloseSent) {
						return
					}
				}
			}
			close(doneReadingChan)

			switch msg.Type {
			case "result":
				var res fleet.DistributedQueryResult
				if err := json.Unmarshal(msg.Data, &res); err != nil {
					resHandler.errors <- ctxerr.Wrap(ctx, err, "unmarshal results")
				}
				resHandler.results <- res

			case "totals":
				var totals targetTotals
				if err := json.Unmarshal(msg.Data, &totals); err != nil {
					resHandler.errors <- ctxerr.Wrap(ctx, err, "unmarshal totals")
				}
				resHandler.totals.Store(&totals)

			case "status":
				var status campaignStatus
				if err := json.Unmarshal(msg.Data, &status); err != nil {
					resHandler.errors <- ctxerr.Wrap(ctx, err, "unmarshal status")
				}
				resHandler.status.Store(&status)

			default:
				resHandler.errors <- ctxerr.Errorf(ctx, "unknown msg type %s", msg.Type)
			}
		}
	}()

	return resHandler, nil
}
