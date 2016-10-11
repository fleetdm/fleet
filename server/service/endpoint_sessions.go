package service

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

////////////////////////////////////////////////////////////////////////////////
// Login
////////////////////////////////////////////////////////////////////////////////

type loginRequest struct {
	Username string // can be username or email
	Password string
}

type loginResponse struct {
	User  *kolide.User `json:"user,omitempty"`
	Token string       `json:"token,omitempty"`
	Err   error        `json:"error,omitempty"`
}

func (r loginResponse) error() error { return r.Err }

func makeLoginEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loginRequest)
		user, token, err := svc.Login(ctx, req.Username, req.Password)
		if err != nil {
			return loginResponse{Err: err}, nil
		}
		return loginResponse{user, token, nil}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Logout
////////////////////////////////////////////////////////////////////////////////

type logoutResponse struct {
	Err error `json:"error,omitempty"`
}

func (r logoutResponse) error() error { return r.Err }

func makeLogoutEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		err := svc.Logout(ctx)
		if err != nil {
			return logoutResponse{Err: err}, nil
		}
		return logoutResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Info About Session
////////////////////////////////////////////////////////////////////////////////

type getInfoAboutSessionRequest struct {
	ID uint
}

type getInfoAboutSessionResponse struct {
	SessionID uint      `json:"session_id"`
	UserID    uint      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Err       error     `json:"error,omitempty"`
}

func (r getInfoAboutSessionResponse) error() error { return r.Err }

func makeGetInfoAboutSessionEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getInfoAboutSessionRequest)
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
}

////////////////////////////////////////////////////////////////////////////////
// Get Info About Sessions For User
////////////////////////////////////////////////////////////////////////////////

type getInfoAboutSessionsForUserRequest struct {
	ID uint
}

type getInfoAboutSessionsForUserResponse struct {
	Sessions []getInfoAboutSessionResponse `json:"sessions"`
	Err      error                         `json:"error,omitempty"`
}

func (r getInfoAboutSessionsForUserResponse) error() error { return r.Err }

func makeGetInfoAboutSessionsForUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getInfoAboutSessionsForUserRequest)
		sessions, err := svc.GetInfoAboutSessionsForUser(ctx, req.ID)
		if err != nil {
			return getInfoAboutSessionsForUserResponse{Err: err}, nil
		}
		var resp getInfoAboutSessionsForUserResponse
		for _, session := range sessions {
			resp.Sessions = append(resp.Sessions, getInfoAboutSessionResponse{
				SessionID: session.ID,
				UserID:    session.UserID,
				CreatedAt: session.CreatedAt,
			})
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Session
////////////////////////////////////////////////////////////////////////////////

type deleteSessionRequest struct {
	ID uint
}

type deleteSessionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSessionResponse) error() error { return r.Err }

func makeDeleteSessionEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteSessionRequest)
		err := svc.DeleteSession(ctx, req.ID)
		if err != nil {
			return deleteSessionResponse{Err: err}, nil
		}
		return deleteSessionResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Sessions For User
////////////////////////////////////////////////////////////////////////////////

type deleteSessionsForUserRequest struct {
	ID uint
}

type deleteSessionsForUserResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSessionsForUserResponse) error() error { return r.Err }

func makeDeleteSessionsForUserEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteSessionsForUserRequest)
		err := svc.DeleteSessionsForUser(ctx, req.ID)
		if err != nil {
			return deleteSessionsForUserResponse{Err: err}, nil
		}
		return deleteSessionsForUserResponse{}, nil
	}
}
