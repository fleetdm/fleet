package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/websocket"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/igm/sockjs-go/v3/sockjs"
)

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignRequest struct {
	QuerySQL string            `json:"query"`
	QueryID  *uint             `json:"query_id"`
	Selected fleet.HostTargets `json:"selected"`
}

type createDistributedQueryCampaignResponse struct {
	Campaign *fleet.DistributedQueryCampaign `json:"campaign,omitempty"`
	Err      error                           `json:"error,omitempty"`
}

func (r createDistributedQueryCampaignResponse) error() error { return r.Err }

func makeCreateDistributedQueryCampaignEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createDistributedQueryCampaignRequest)
		campaign, err := svc.NewDistributedQueryCampaign(ctx, req.QuerySQL, req.QueryID, req.Selected)
		if err != nil {
			return createDistributedQueryCampaignResponse{Err: err}, nil
		}
		return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign By Names
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignByNamesRequest struct {
	QuerySQL string                                 `json:"query"`
	QueryID  *uint                                  `json:"query_id"`
	Selected distributedQueryCampaignTargetsByNames `json:"selected"`
}

type distributedQueryCampaignTargetsByNames struct {
	Labels []string `json:"labels"`
	Hosts  []string `json:"hosts"`
}

func makeCreateDistributedQueryCampaignByNamesEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createDistributedQueryCampaignByNamesRequest)
		campaign, err := svc.NewDistributedQueryCampaignByNames(ctx, req.QuerySQL, req.QueryID, req.Selected.Hosts, req.Selected.Labels)
		if err != nil {
			return createDistributedQueryCampaignResponse{Err: err}, nil
		}
		return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Stream Distributed Query Campaign Results and Metadata
////////////////////////////////////////////////////////////////////////////////

func makeStreamDistributedQueryCampaignResultsHandler(svc fleet.Service, logger kitlog.Logger) http.Handler {
	opt := sockjs.DefaultOptions
	opt.Websocket = true
	opt.RawWebsocket = true
	return sockjs.NewHandler("/api/v1/fleet/results", opt, func(session sockjs.Session) {
		conn := &websocket.Conn{Session: session}
		defer func() {
			if p := recover(); p != nil {
				logger.Log("err", p, "msg", "panic in result handler")
				conn.WriteJSONError("panic in result handler")
			}
			session.Close(0, "none")
		}()

		// Receive the auth bearer token
		token, err := conn.ReadAuthToken()
		if err != nil {
			logger.Log("err", err, "msg", "failed to read auth token")
			return
		}

		// Authenticate with the token
		vc, err := authViewer(context.Background(), string(token), svc)
		if err != nil || !vc.CanPerformActions() {
			logger.Log("err", err, "msg", "unauthorized viewer")
			conn.WriteJSONError("unauthorized")
			return
		}

		ctx := viewer.NewContext(context.Background(), *vc)

		msg, err := conn.ReadJSONMessage()
		if err != nil {
			logger.Log("err", err, "msg", "reading select_campaign JSON")
			conn.WriteJSONError("error reading select_campaign")
			return
		}
		if msg.Type != "select_campaign" {
			logger.Log("err", "unexpected msg type, expected select_campaign", "msg-type", msg.Type)
			conn.WriteJSONError("expected select_campaign")
			return
		}

		var info struct {
			CampaignID uint `json:"campaign_id"`
		}
		err = json.Unmarshal(*(msg.Data.(*json.RawMessage)), &info)
		if err != nil {
			logger.Log("err", err, "msg", "unmarshaling select_campaign data")
			conn.WriteJSONError("error unmarshaling select_campaign data")
			return
		}
		if info.CampaignID == 0 {
			logger.Log("err", "campaign ID not set")
			conn.WriteJSONError("0 is not a valid campaign ID")
			return
		}

		svc.StreamCampaignResults(ctx, conn, info.CampaignID)
	})
}
