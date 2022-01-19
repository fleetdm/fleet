package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get Info About Session
////////////////////////////////////////////////////////////////////////////////

type getInfoAboutSessionRequest struct {
	ID uint `url:"id"`
}

type getInfoAboutSessionResponse struct {
	SessionID uint      `json:"session_id"`
	UserID    uint      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Err       error     `json:"error,omitempty"`
}

func (r getInfoAboutSessionResponse) error() error { return r.Err }

func getInfoAboutSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getInfoAboutSessionRequest)
	session, err := svc.GetInfoAboutSession(ctx, req.ID)
	if err != nil {
		return getInfoAboutSessionResponse{Err: err}, nil
	}

	return getInfoAboutSessionResponse{
		SessionID: session.ID,
		UserID:    session.UserID,
		CreatedAt: session.CreatedAt,
	}, nil
}

func (svc *Service) GetInfoAboutSession(ctx context.Context, id uint) (*fleet.Session, error) {
	session, err := svc.ds.SessionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// TODO(mna): this seemed incorrect, the `id` is a session ID, used to retrieve the session above,
	// but here we use it as user id to check auth?
	if err := svc.authz.Authorize(ctx, session, fleet.ActionRead); err != nil {
		return nil, err
	}

	err = svc.validateSession(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Session
////////////////////////////////////////////////////////////////////////////////

type deleteSessionRequest struct {
	ID uint `url:"id"`
}

type deleteSessionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSessionResponse) error() error { return r.Err }

func deleteSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteSessionRequest)
	err := svc.DeleteSession(ctx, req.ID)
	if err != nil {
		return deleteSessionResponse{Err: err}, nil
	}
	return deleteSessionResponse{}, nil
}

func (svc *Service) DeleteSession(ctx context.Context, id uint) error {
	session, err := svc.ds.SessionByID(ctx, id)
	if err != nil {
		return err
	}

	if err := svc.authz.Authorize(ctx, session, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DestroySession(ctx, session)
}
