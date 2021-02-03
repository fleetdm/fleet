package service

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/websocket"
	"github.com/go-kit/kit/endpoint"
	kitlog "github.com/go-kit/kit/log"
	"github.com/igm/sockjs-go/v3/sockjs"
)

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignRequest struct {
	Query    string                          `json:"query"`
	Selected distributedQueryCampaignTargets `json:"selected"`
}

type distributedQueryCampaignTargets struct {
	Labels []uint `json:"labels"`
	Hosts  []uint `json:"hosts"`
}

type createDistributedQueryCampaignResponse struct {
	Campaign *kolide.DistributedQueryCampaign `json:"campaign,omitempty"`
	Err      error                            `json:"error,omitempty"`
}

func (r createDistributedQueryCampaignResponse) error() error { return r.Err }

func makeCreateDistributedQueryCampaignEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createDistributedQueryCampaignRequest)
		campaign, err := svc.NewDistributedQueryCampaign(ctx, req.Query, req.Selected.Hosts, req.Selected.Labels)
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
	Query    string                                 `json:"query"`
	Selected distributedQueryCampaignTargetsByNames `json:"selected"`
}

type distributedQueryCampaignTargetsByNames struct {
	Labels []string `json:"labels"`
	Hosts  []string `json:"hosts"`
}

func makeCreateDistributedQueryCampaignByNamesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createDistributedQueryCampaignByNamesRequest)
		campaign, err := svc.NewDistributedQueryCampaignByNames(ctx, req.Query, req.Selected.Hosts, req.Selected.Labels)
		if err != nil {
			return createDistributedQueryCampaignResponse{Err: err}, nil
		}
		return createDistributedQueryCampaignResponse{Campaign: campaign}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Stream Distributed Query Campaign Results and Metadata
////////////////////////////////////////////////////////////////////////////////

func makeStreamDistributedQueryCampaignResultsHandler(svc kolide.Service, jwtKey string, logger kitlog.Logger) http.Handler {
	opt := sockjs.DefaultOptions
	opt.Websocket = true
	opt.RawWebsocket = true
	return sockjs.NewHandler("/api/v1/kolide/results", opt, func(session sockjs.Session) {
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
		vc, err := authViewer(context.Background(), jwtKey, token, svc)
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
