package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
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

////////////////////////////////////////////////////////////////////////////////
// Login
////////////////////////////////////////////////////////////////////////////////

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	User           *fleet.User          `json:"user,omitempty"`
	AvailableTeams []*fleet.TeamSummary `json:"available_teams"`
	Token          string               `json:"token,omitempty"`
	Err            error                `json:"error,omitempty"`
}

func (r loginResponse) error() error { return r.Err }

func loginEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*loginRequest)
	req.Email = strings.ToLower(req.Email)

	user, token, err := svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		return loginResponse{Err: err}, nil
	}
	// Add viewer context allow access to service teams for list of available teams
	v, err := authViewer(ctx, token, svc)
	if err != nil {
		return loginResponse{Err: err}, nil
	}
	ctx = viewer.NewContext(ctx, *v)
	availableTeams, err := svc.ListAvailableTeamsForUser(ctx, user)
	if err != nil {
		if errors.Is(err, fleet.ErrMissingLicense) {
			availableTeams = []*fleet.TeamSummary{}
		} else {
			return loginResponse{Err: err}, nil
		}
	}
	return loginResponse{user, availableTeams, token, nil}, nil
}

func (svc *Service) Login(ctx context.Context, email, password string) (*fleet.User, string, error) {
	// skipauth: No user context available yet to authorize against.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	// If there is an error, sleep until the request has taken at least 1
	// second. This means that generally a login failure for any reason will
	// take ~1s and frustrate a timing attack.
	var err error
	defer func(start time.Time) {
		if err != nil {
			time.Sleep(time.Until(start.Add(1 * time.Second)))
		}
	}(time.Now())

	user, err := svc.ds.UserByEmail(ctx, email)
	var nfe fleet.NotFoundError
	if errors.As(err, &nfe) {
		return nil, "", fleet.NewAuthFailedError("user not found")
	}
	if err != nil {
		return nil, "", fleet.NewAuthFailedError(err.Error())
	}

	if err = user.ValidatePassword(password); err != nil {
		return nil, "", fleet.NewAuthFailedError("invalid password")
	}

	if user.SSOEnabled {
		return nil, "", fleet.NewAuthFailedError("password login disabled for sso users")
	}

	token, err := svc.makeSession(ctx, user.ID)
	if err != nil {
		return nil, "", fleet.NewAuthFailedError(err.Error())
	}

	return user, token, nil
}

////////////////////////////////////////////////////////////////////////////////
// Logout
////////////////////////////////////////////////////////////////////////////////

type logoutResponse struct {
	Err error `json:"error,omitempty"`
}

func (r logoutResponse) error() error { return r.Err }

func logoutEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	err := svc.Logout(ctx)
	if err != nil {
		return logoutResponse{Err: err}, nil
	}
	return logoutResponse{}, nil
}

func (svc *Service) Logout(ctx context.Context) error {
	// skipauth: Any user can always log out of their own session.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(ctx, level.Info)

	// TODO: this should not return an error if the user wasn't logged in
	return svc.DestroySession(ctx)
}

func (svc *Service) DestroySession(ctx context.Context) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	session, err := svc.ds.SessionByID(ctx, vc.SessionID())
	if err != nil {
		return err
	}

	if err := svc.authz.Authorize(ctx, session, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DestroySession(ctx, session)
}
