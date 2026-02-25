package fleet

import (
	"context"
	"net/http"
	"time"
)

type GetInfoAboutSessionRequest struct {
	ID uint `url:"id"`
}

type GetInfoAboutSessionResponse struct {
	SessionID uint      `json:"session_id"`
	UserID    uint      `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Err       error     `json:"error,omitempty"`
}

func (r GetInfoAboutSessionResponse) Error() error { return r.Err }

type DeleteSessionRequest struct {
	ID uint `url:"id"`
}

type DeleteSessionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteSessionResponse) Error() error { return r.Err }

type LoginResponse struct {
	User           *User          `json:"user,omitempty"`
	AvailableTeams []*TeamSummary `json:"available_teams" renameto:"available_fleets"`
	Token          string         `json:"token,omitempty"`
	Err            error          `json:"error,omitempty"`
}

func (r LoginResponse) Error() error { return r.Err }

type LoginMfaResponse struct {
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

func (r LoginMfaResponse) Status() int { return http.StatusAccepted }

func (r LoginMfaResponse) Error() error { return r.Err }

type SessionCreateRequest struct {
	Token string `json:"token,omitempty"`
}

type LogoutResponse struct {
	Err error `json:"error,omitempty"`
}

func (r LogoutResponse) Error() error { return r.Err }

type InitiateSSORequest struct {
	// RelayURL is the URL path that the IdP will redirect to once authenticated
	// (e.g. "/dashboard").
	RelayURL string `json:"relay_url"`
}

type InitiateSSOResponse struct {
	URL          string                                     `json:"url,omitempty"`
	Err          error                                      `json:"error,omitempty"`
	SetCookiesFn func(context.Context, http.ResponseWriter) `json:"-"`
}

func (r InitiateSSOResponse) Error() error { return r.Err }

func (r InitiateSSOResponse) SetCookies(ctx context.Context, w http.ResponseWriter) {
	if r.SetCookiesFn != nil {
		r.SetCookiesFn(ctx, w)
	}
}

type CallbackSSORequest struct {
	SessionID    string
	SAMLResponse []byte
}

type CallbackSSOResponse struct {
	Content      string                                     `json:"-"`
	Err          error                                      `json:"error,omitempty"`
	SetCookiesFn func(context.Context, http.ResponseWriter) `json:"-"`
}

func (r CallbackSSOResponse) Error() error { return r.Err }

func (r CallbackSSOResponse) Html() string { return r.Content }

func (r CallbackSSOResponse) SetCookies(ctx context.Context, w http.ResponseWriter) {
	if r.SetCookiesFn != nil {
		r.SetCookiesFn(ctx, w)
	}
}

type SsoSettingsResponse struct {
	Settings *SessionSSOSettings `json:"settings,omitempty"`
	Err      error               `json:"error,omitempty"`
}

func (r SsoSettingsResponse) Error() error { return r.Err }
