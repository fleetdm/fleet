package service

import (
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/contexts/viewer"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/kolide/kolide-ose/server/websocket"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Create Distributed Query Campaign
////////////////////////////////////////////////////////////////////////////////

type createDistributedQueryCampaignRequest struct {
	UserID   uint
	Query    string `json:"query"`
	Selected struct {
		Labels []uint `json:"labels"`
		Hosts  []uint `json:"hosts"`
	} `json:"selected"`
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
			return createQueryResponse{Err: err}, nil
		}
		return createDistributedQueryCampaignResponse{campaign, nil}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Stream Distributed Query Campaign Results and Metadata
////////////////////////////////////////////////////////////////////////////////

func makeStreamDistributedQueryCampaignResultsHandler(svc kolide.Service, jwtKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Upgrade to websocket connection
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			return
		}
		defer conn.Close()

		// Receive the auth bearer token
		token, err := conn.ReadAuthToken()
		if err != nil {
			return
		}

		// Authenticate with the token
		vc, err := authViewer(context.Background(), jwtKey, string(token), svc)
		if err != nil || !vc.CanPerformActions() {
			conn.WriteJSONError("unauthorized")
			return
		}

		ctx := viewer.NewContext(context.Background(), *vc)

		campaignID, err := idFromRequest(r, "id")
		if err != nil {
			conn.WriteJSONError("invalid campaign ID")
			return
		}

		svc.StreamCampaignResults(ctx, conn, campaignID)

	}
}
