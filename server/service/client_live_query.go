package service

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"

	ws "github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// LiveQueryResultsHandler provides access to all of the information about an
// incoming stream of live query results.
type LiveQueryResultsHandler struct {
	errors  chan error
	results chan fleet.DistributedQueryResult
	totals  atomic.Value // real type: targetTotals
	status  atomic.Value // real type: campaignStatus

	conn *websocket.Conn
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

func (h *LiveQueryResultsHandler) Close() error {
	if h.conn != nil {
		return h.conn.Close()
	}
	return nil
}

// LiveQuery creates a new live query and begins streaming results.
func (c *Client) LiveQuery(query string, labels []string, hosts []string) (*LiveQueryResultsHandler, error) {
	req := createDistributedQueryCampaignByNamesRequest{
		QuerySQL: query,
		Selected: distributedQueryCampaignTargetsByNames{Labels: labels, Hosts: hosts},
	}
	response, err := c.AuthenticatedDo("POST", "/api/v1/fleet/queries/run_by_names", "", req)
	if err != nil {
		return nil, errors.Wrap(err, "POST /api/v1/fleet/queries/run_by_names")
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"create live query received status %d %s",
			response.StatusCode,
			extractServerErrorText(response.Body),
		)
	}

	var responseBody createDistributedQueryCampaignResponse
	err = json.NewDecoder(response.Body).Decode(&responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "decode create live query response")
	}
	if responseBody.Err != nil {
		return nil, errors.Errorf("create live query: %s", responseBody.Err)
	}

	// Copy default dialer but skip cert verification if set.
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: c.insecureSkipVerify},
	}

	wssURL := *c.baseURL
	wssURL.Scheme = "wss"
	wssURL.Path = c.urlPrefix + "/api/v1/fleet/results/websocket"
	conn, _, err := dialer.Dial(wssURL.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "upgrade live query result websocket")
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
		return nil, errors.Wrap(err, "auth for results")
	}

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "select_campaign",
		Data: map[string]interface{}{"campaign_id": responseBody.Campaign.ID},
	})
	if err != nil {
		_ = conn.Close()
		return nil, errors.Wrap(err, "auth for results")
	}

	resHandler := NewLiveQueryResultsHandler()
	resHandler.conn = conn
	go func() {
		defer conn.Close()
		for {
			msg := struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}{}
			err := conn.ReadJSON(&msg)
			if err != nil {
				resHandler.errors <- errors.Wrap(err, "receive ws message")
			}

			switch msg.Type {
			case "result":
				var res fleet.DistributedQueryResult
				if err := json.Unmarshal(msg.Data, &res); err != nil {
					resHandler.errors <- errors.Wrap(err, "unmarshal results")
				}
				resHandler.results <- res

			case "totals":
				var totals targetTotals
				if err := json.Unmarshal(msg.Data, &totals); err != nil {
					resHandler.errors <- errors.Wrap(err, "unmarshal totals")
				}
				resHandler.totals.Store(&totals)

			case "status":
				var status campaignStatus
				if err := json.Unmarshal(msg.Data, &status); err != nil {
					resHandler.errors <- errors.Wrap(err, "unmarshal status")
				}
				resHandler.status.Store(&status)

			default:
				resHandler.errors <- errors.Errorf("unknown msg type %s", msg.Type)
			}
		}
	}()

	return resHandler, nil
}
