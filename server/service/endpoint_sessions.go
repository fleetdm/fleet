package service

import (
	"bytes"
	"context"
	"html/template"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Login
////////////////////////////////////////////////////////////////////////////////

type loginRequest struct {
	Email    string
	Password string
}

type loginResponse struct {
	User  *fleet.User `json:"user,omitempty"`
	Token string      `json:"token,omitempty"`
	Err   error       `json:"error,omitempty"`
}

func (r loginResponse) error() error { return r.Err }

func makeLoginEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(loginRequest)
		user, token, err := svc.Login(ctx, req.Email, req.Password)
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

func makeLogoutEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeGetInfoAboutSessionEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeGetInfoAboutSessionsForUserEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeDeleteSessionEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeDeleteSessionsForUserEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteSessionsForUserRequest)
		err := svc.DeleteSessionsForUser(ctx, req.ID)
		if err != nil {
			return deleteSessionsForUserResponse{Err: err}, nil
		}
		return deleteSessionsForUserResponse{}, nil
	}
}

type initiateSSORequest struct {
	RelayURL string `json:"relay_url"`
}

type initiateSSOResponse struct {
	URL string `json:"url,omitempty"`
	Err error  `json:"error,omitempty"`
}

func (r initiateSSOResponse) error() error { return r.Err }

func makeInitiateSSOEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(initiateSSORequest)
		idProviderURL, err := svc.InitiateSSO(ctx, req.RelayURL)
		if err != nil {
			return initiateSSOResponse{Err: err}, nil
		}
		return initiateSSOResponse{URL: idProviderURL}, nil
	}
}

type callbackSSOResponse struct {
	content string
	Err     error `json:"error,omitempty"`
}

func (r callbackSSOResponse) error() error { return r.Err }

// If html is present we return a web page
func (r callbackSSOResponse) html() string { return r.content }

func makeCallbackSSOEndpoint(svc fleet.Service, urlPrefix string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		authResponse := request.(fleet.Auth)
		session, err := svc.CallbackSSO(ctx, authResponse)
		var resp callbackSSOResponse
		if err != nil {
			// redirect to login page on front end if there was some problem,
			// errors should still be logged
			session = &fleet.SSOSession{
				RedirectURL: urlPrefix + "/login",
				Token:       "",
			}
			resp.Err = err
		}
		relayStateLoadPage := ` <html>
     <script type='text/javascript'>
     var redirectURL = {{ .RedirectURL }};
     window.localStorage.setItem('FLEET::auth_token', '{{ .Token }}');
     window.location = redirectURL;
     </script>
     <body>
     Redirecting to Fleet...
     </body>
     </html>
    `
		tmpl, err := template.New("relayStateLoader").Parse(relayStateLoadPage)
		if err != nil {
			return nil, err
		}
		var writer bytes.Buffer
		err = tmpl.Execute(&writer, session)
		if err != nil {
			return nil, err
		}
		resp.content = writer.String()
		return resp, nil
	}
}

type ssoSettingsResponse struct {
	Settings *fleet.SessionSSOSettings `json:"settings,omitempty"`
	Err      error                     `json:"error,omitempty"`
}

func (r ssoSettingsResponse) error() error { return r.Err }

func makeSSOSettingsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, unused interface{}) (interface{}, error) {
		settings, err := svc.SSOSettings(ctx)
		if err != nil {
			return ssoSettingsResponse{Err: err}, nil
		}
		return ssoSettingsResponse{Settings: settings}, nil
	}
}
