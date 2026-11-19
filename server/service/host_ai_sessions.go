package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type listHostAISessionsRequest struct {
	ID          uint              `url:"id"`
	ListOptions fleet.ListOptions `url:"list_options"`
	// OrderDirection is order_direction as sent, since the direction defaults to descending and ListOptions can't
	// tell an unset direction from an explicit "asc".
	OrderDirection string `query:"order_direction,optional"`
	Agent          string `query:"agent,optional"`
	Category       string `query:"category,optional"`
}

type listHostAISessionsResponse struct {
	AISessions []*fleet.HostAISession    `json:"ai_sessions"`
	Count      uint                      `json:"count"`
	Meta       *fleet.PaginationMetadata `json:"meta,omitempty"`
	Err        error                     `json:"error,omitempty"`
}

func (r listHostAISessionsResponse) Error() error { return r.Err }

func listHostAISessionsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listHostAISessionsRequest)
	opts := req.ListOptions
	if req.OrderDirection == "" {
		opts.OrderDirection = fleet.OrderDescending
	}
	sessions, meta, err := svc.ListHostAISessions(ctx, req.ID, fleet.HostAISessionListOptions{
		ListOptions: opts,
		Agent:       req.Agent,
		Category:    fleet.AISessionCategory(req.Category),
	})
	if err != nil {
		return listHostAISessionsResponse{Err: err}, nil
	}
	resp := listHostAISessionsResponse{AISessions: sessions, Meta: meta}
	if meta != nil {
		resp.Count = meta.TotalResults
	}
	return resp, nil
}

func (svc *Service) ListHostAISessions(ctx context.Context, hostID uint, opts fleet.HostAISessionListOptions) ([]*fleet.HostAISession, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

type getHostAISessionRequest struct {
	ID          uint `url:"id"`
	AISessionID uint `url:"ai_session_id"`
}

type getHostAISessionResponse struct {
	AISession *fleet.HostAISession `json:"ai_session"`
	Err       error                `json:"error,omitempty"`
}

func (r getHostAISessionResponse) Error() error { return r.Err }

func getHostAISessionEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getHostAISessionRequest)
	session, err := svc.GetHostAISession(ctx, req.ID, req.AISessionID)
	if err != nil {
		return getHostAISessionResponse{Err: err}, nil
	}
	return getHostAISessionResponse{AISession: session}, nil
}

func (svc *Service) GetHostAISession(ctx context.Context, hostID, aiSessionID uint) (*fleet.HostAISession, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

type listHostAISessionEventsRequest struct {
	ID          uint              `url:"id"`
	AISessionID uint              `url:"ai_session_id"`
	ListOptions fleet.ListOptions `url:"list_options"`
	Category    string            `query:"category,optional"`
}

type listHostAISessionEventsResponse struct {
	Events []*fleet.HostAISessionEvent `json:"events"`
	Count  uint                        `json:"count"`
	Meta   *fleet.PaginationMetadata   `json:"meta,omitempty"`
	Err    error                       `json:"error,omitempty"`
}

func (r listHostAISessionEventsResponse) Error() error { return r.Err }

func listHostAISessionEventsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listHostAISessionEventsRequest)
	events, meta, err := svc.ListHostAISessionEvents(ctx, req.ID, req.AISessionID, fleet.HostAISessionEventListOptions{
		ListOptions: req.ListOptions,
		Category:    fleet.AISessionCategory(req.Category),
	})
	if err != nil {
		return listHostAISessionEventsResponse{Err: err}, nil
	}
	resp := listHostAISessionEventsResponse{Events: events, Meta: meta}
	if meta != nil {
		resp.Count = meta.TotalResults
	}
	return resp, nil
}

func (svc *Service) ListHostAISessionEvents(ctx context.Context, hostID, aiSessionID uint, opts fleet.HostAISessionEventListOptions) ([]*fleet.HostAISessionEvent, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}
