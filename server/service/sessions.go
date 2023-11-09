package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/sso"
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

func getInfoAboutSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
		svc.authz.SkipAuthorization(ctx)
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

func deleteSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
		svc.authz.SkipAuthorization(ctx)
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

func loginEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*loginRequest)
	req.Email = strings.ToLower(req.Email)

	user, session, err := svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		return loginResponse{Err: err}, nil
	}
	// Add viewer to context to allow access to service teams for list of available teams.
	ctx = viewer.NewContext(ctx, viewer.Viewer{
		User:    user,
		Session: session,
	})
	availableTeams, err := svc.ListAvailableTeamsForUser(ctx, user)
	if err != nil {
		if errors.Is(err, fleet.ErrMissingLicense) {
			availableTeams = []*fleet.TeamSummary{}
		} else {
			return loginResponse{Err: err}, nil
		}
	}
	return loginResponse{user, availableTeams, session.Key, nil}, nil
}

func (svc *Service) Login(ctx context.Context, email, password string) (*fleet.User, *fleet.Session, error) {
	// skipauth: No user context available yet to authorize against.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithExtras(logging.WithNoUser(ctx),
		"op", "login",
		"email", email,
		"public_ip", publicip.FromContext(ctx),
	), level.Info)

	// If there is an error, sleep until the request has taken at least 1
	// second. This means that generally a login failure for any reason will
	// take ~1s and frustrate a timing attack.
	var err error
	defer func(start time.Time) {
		if err != nil {
			if err := svc.ds.NewActivity(ctx, nil, fleet.ActivityTypeUserFailedLogin{
				Email:    email,
				PublicIP: publicip.FromContext(ctx),
			}); err != nil {
				logging.WithExtras(logging.WithNoUser(ctx),
					"msg", "failed to generate failed login activity",
				)
			}
			time.Sleep(time.Until(start.Add(1 * time.Second)))
		}
	}(time.Now())

	user, err := svc.ds.UserByEmail(ctx, email)
	var nfe fleet.NotFoundError
	if errors.As(err, &nfe) {
		return nil, nil, fleet.NewAuthFailedError("user not found")
	}
	if err != nil {
		return nil, nil, fleet.NewAuthFailedError(err.Error())
	}

	if err = user.ValidatePassword(password); err != nil {
		return nil, nil, fleet.NewAuthFailedError("invalid password")
	}

	if user.SSOEnabled {
		return nil, nil, fleet.NewAuthFailedError("password login disabled for sso users")
	}

	session, err := svc.makeSession(ctx, user.ID)
	if err != nil {
		return nil, nil, fleet.NewAuthFailedError(err.Error())
	}

	if err := svc.ds.NewActivity(ctx, user, fleet.ActivityTypeUserLoggedIn{
		PublicIP: publicip.FromContext(ctx),
	}); err != nil {
		return nil, nil, err
	}
	return user, session, nil
}

////////////////////////////////////////////////////////////////////////////////
// Logout
////////////////////////////////////////////////////////////////////////////////

type logoutResponse struct {
	Err error `json:"error,omitempty"`
}

func (r logoutResponse) error() error { return r.Err }

func logoutEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

	return svc.DestroySession(ctx)
}

func (svc *Service) DestroySession(ctx context.Context) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.NewAuthRequiredError(fleet.ErrNoContext.Error())
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

////////////////////////////////////////////////////////////////////////////////
// Initiate SSO
////////////////////////////////////////////////////////////////////////////////

type initiateSSORequest struct {
	// RelayURL is the URL path that the IdP will redirect to once authenticated
	// (e.g. "/dashboard").
	RelayURL string `json:"relay_url"`
}

type initiateSSOResponse struct {
	URL string `json:"url,omitempty"`
	Err error  `json:"error,omitempty"`
}

func (r initiateSSOResponse) error() error { return r.Err }

func initiateSSOEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*initiateSSORequest)
	idProviderURL, err := svc.InitiateSSO(ctx, req.RelayURL)
	if err != nil {
		return initiateSSOResponse{Err: err}, nil
	}
	return initiateSSOResponse{URL: idProviderURL}, nil
}

// InitiateSSO initiates a Single Sign-On flow for a request to visit the
// protected URL identified by redirectURL. It returns the URL of the identity
// provider to make a request to to proceed with the authentication via that
// external service, and stores ephemeral session state to validate the
// callback from the identity provider to finalize the SSO flow.
func (svc *Service) InitiateSSO(ctx context.Context, redirectURL string) (string, error) {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// initiate SSO.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "InitiateSSO getting app config")
	}

	if appConfig.SSOSettings == nil || !appConfig.SSOSettings.EnableSSO {
		err := &fleet.BadRequestError{Message: "organization not configured to use sso"}
		return "", ctxerr.Wrap(ctx, newSSOError(err, ssoOrgDisabled), "initiate sso")
	}

	metadata, err := sso.GetMetadata(&appConfig.SSOSettings.SSOProviderSettings)
	if err != nil {
		return "", ctxerr.Wrap(ctx, badRequestErr("Could not get SSO Metadata. Check your SSO settings.", err))
	}

	serverURL := appConfig.ServerSettings.ServerURL
	settings := sso.Settings{
		Metadata: metadata,
		// Construct call back url to send to idp
		AssertionConsumerServiceURL: serverURL + svc.config.Server.URLPrefix + "/api/v1/fleet/sso/callback",
		SessionStore:                svc.ssoSessionStore,
		OriginalURL:                 redirectURL,
	}

	// If issuer is not explicitly set, default to host name.
	var issuer string
	entityID := appConfig.SSOSettings.EntityID
	if entityID == "" {
		u, err := url.Parse(serverURL)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "parse server url")
		}
		issuer = u.Hostname()
	} else {
		issuer = entityID
	}

	idpURL, err := sso.CreateAuthorizationRequest(&settings, issuer)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "InitiateSSO creating authorization")
	}

	return idpURL, nil
}

////////////////////////////////////////////////////////////////////////////////
// Callback SSO
////////////////////////////////////////////////////////////////////////////////

type callbackSSORequest struct{}

func (callbackSSORequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to parse form",
			InternalErr: err,
		}, "decode sso callback")
	}
	authResponse, err := sso.DecodeAuthResponse(r.FormValue("SAMLResponse"))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to decode SAMLResponse",
			InternalErr: err,
		}, "decoding sso callback")
	}
	return authResponse, nil
}

type callbackSSOResponse struct {
	content string
	Err     error `json:"error,omitempty"`
}

func (r callbackSSOResponse) error() error { return r.Err }

// If html is present we return a web page
func (r callbackSSOResponse) html() string { return r.content }

func makeCallbackSSOEndpoint(urlPrefix string) handlerFunc {
	return func(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
		authResponse := request.(fleet.Auth)
		session, err := getSSOSession(ctx, svc, authResponse)
		var resp callbackSSOResponse
		if err != nil {
			if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeUserFailedLogin{
				Email:    authResponse.UserID(),
				PublicIP: publicip.FromContext(ctx),
			}); err != nil {
				logging.WithLevel(logging.WithExtras(logging.WithNoUser(ctx),
					"msg", "failed to generate failed login activity",
				), level.Info)
			}

			var ssoErr *ssoError

			status := ssoOtherError
			if errors.As(err, &ssoErr) {
				status = ssoErr.code
			}
			// redirect to login page on front end if there was some problem,
			// errors should still be logged
			session = &fleet.SSOSession{
				RedirectURL: urlPrefix + "/login?status=" + string(status),
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
     Redirecting to Fleet at {{ .RedirectURL }} ...
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

func getSSOSession(ctx context.Context, svc fleet.Service, auth fleet.Auth) (*fleet.SSOSession, error) {
	redirectURL, err := svc.InitSSOCallback(ctx, auth)
	if err != nil {
		return nil, err
	}

	user, err := svc.GetSSOUser(ctx, auth)
	if err != nil {
		return nil, err
	}

	return svc.LoginSSOUser(ctx, user, redirectURL)
}

func (svc *Service) InitSSOCallback(ctx context.Context, auth fleet.Auth) (string, error) {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// hit the SSO callback.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get config for sso")
	}

	if appConfig.SSOSettings == nil || !appConfig.SSOSettings.EnableSSO {
		err := ctxerr.New(ctx, "organization not configured to use sso")
		return "", ctxerr.Wrap(ctx, newSSOError(err, ssoOrgDisabled), "callback sso")
	}

	// Load the request metadata if available.
	var metadata *sso.Metadata
	var redirectURL string
	if appConfig.SSOSettings.EnableSSOIdPLogin && auth.RequestID() == "" {
		// Missing request ID indicates this was IdP-initiated. Only allow if
		// configured to do so.
		metadata, err = sso.GetMetadata(&appConfig.SSOSettings.SSOProviderSettings)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "get sso metadata")
		}
		redirectURL = "/"
	} else {
		var session *sso.Session
		session, metadata, err = svc.ssoSessionStore.Fullfill(auth.RequestID())
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "validate request in session")
		}
		redirectURL = session.OriginalURL
	}

	// Validate response
	err = sso.ValidateAudiences(
		*metadata,
		auth,
		appConfig.SSOSettings.EntityID,
		appConfig.ServerSettings.ServerURL,
		appConfig.ServerSettings.ServerURL+svc.config.Server.URLPrefix+"/api/v1/fleet/sso/callback", // ACS
	)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "validating sso response")
	}

	return redirectURL, nil
}

func (svc *Service) GetSSOUser(ctx context.Context, auth fleet.Auth) (*fleet.User, error) {
	user, err := svc.ds.UserByEmail(ctx, auth.UserID())
	if err != nil {
		var nfe notFoundErrorInterface
		if errors.As(err, &nfe) {
			return nil, ctxerr.Wrap(ctx, newSSOError(err, ssoAccountInvalid))
		}
		return nil, ctxerr.Wrap(ctx, err, "find user in sso callback")
	}
	return user, nil
}

func (svc *Service) LoginSSOUser(ctx context.Context, user *fleet.User, redirectURL string) (*fleet.SSOSession, error) {
	logging.WithExtras(ctx, "email", user.Email)

	// if the user is not sso enabled they are not authorized
	if !user.SSOEnabled {
		err := ctxerr.New(ctx, "user not configured to use sso")
		return nil, ctxerr.Wrap(ctx, newSSOError(err, ssoAccountDisabled))
	}
	session, err := svc.makeSession(ctx, user.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "make session in sso callback")
	}
	result := &fleet.SSOSession{
		Token:       session.Key,
		RedirectURL: redirectURL,
	}
	err = svc.ds.NewActivity(
		ctx,
		user,
		fleet.ActivityTypeUserLoggedIn{
			PublicIP: publicip.FromContext(ctx),
		},
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create activity in sso callback")
	}
	return result, nil
}

////////////////////////////////////////////////////////////////////////////////
// SSO Settings
////////////////////////////////////////////////////////////////////////////////

type ssoSettingsResponse struct {
	Settings *fleet.SessionSSOSettings `json:"settings,omitempty"`
	Err      error                     `json:"error,omitempty"`
}

func (r ssoSettingsResponse) error() error { return r.Err }

func settingsSSOEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (errorer, error) {
	settings, err := svc.SSOSettings(ctx)
	if err != nil {
		return ssoSettingsResponse{Err: err}, nil
	}
	return ssoSettingsResponse{Settings: settings}, nil
}

// SSOSettings returns a subset of the Single Sign-On settings as configured in
// the app config. Those can be exposed e.g. via the response to an HTTP request,
// and as such should not contain sensitive information.
func (svc *Service) SSOSettings(ctx context.Context) (*fleet.SessionSSOSettings, error) {
	// skipauth: Basic SSO settings are available to unauthenticated users (so
	// that they have the necessary information to initiate SSO).
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "SessionSSOSettings getting app config")
	}

	var ssoSettings fleet.SSOSettings
	if appConfig.SSOSettings != nil {
		ssoSettings = *appConfig.SSOSettings
	}

	settings := &fleet.SessionSSOSettings{
		IDPName:     ssoSettings.IDPName,
		IDPImageURL: ssoSettings.IDPImageURL,
		SSOEnabled:  ssoSettings.EnableSSO,
	}
	return settings, nil
}

// makeSession creates a new session for the given user.
func (svc *Service) makeSession(ctx context.Context, userID uint) (*fleet.Session, error) {
	sessionKeySize := svc.config.Session.KeySize
	key := make([]byte, sessionKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	session, err := svc.ds.NewSession(ctx, userID, base64.StdEncoding.EncodeToString(key))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating new session")
	}
	return session, nil
}

func (svc *Service) GetSessionByKey(ctx context.Context, key string) (*fleet.Session, error) {
	session, err := svc.ds.SessionByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	err = svc.validateSession(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (svc *Service) validateSession(ctx context.Context, session *fleet.Session) error {
	if session == nil {
		return fleet.NewAuthRequiredError("active session not present")
	}

	sessionDuration := svc.config.Session.Duration
	if session.APIOnly != nil && *session.APIOnly {
		sessionDuration = 0 // make API-only tokens unlimited
	}

	// duration 0 = unlimited
	if sessionDuration != 0 && time.Since(session.AccessedAt) >= sessionDuration {
		err := svc.ds.DestroySession(ctx, session)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "destroying session")
		}
		return fleet.NewAuthRequiredError("expired session")
	}

	return svc.ds.MarkSessionAccessed(ctx, session)
}

////////////////////////////////////////////////////////////////////////////////
// Demo Login
////////////////////////////////////////////////////////////////////////////////

// This is a special kind of login where the username and password come in form values rather than
// JSON as is typical for API requests. This is intended to support logins from demo environments,
// when users are being redirected from fleetdm.com.

type demologinRequest struct {
	Email    string
	Password string
}

func (demologinRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to parse form",
			InternalErr: err,
		}, "decode demo login")
	}

	return demologinRequest{
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}, nil
}

type demologinResponse struct {
	content string
	Err     error `json:"error,omitempty"`
}

func (r demologinResponse) error() error { return r.Err }

// If html is present we return a web page
func (r demologinResponse) html() string { return r.content }

func makeDemologinEndpoint(urlPrefix string) handlerFunc {
	return func(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
		req := request.(demologinRequest)

		if !svc.SandboxEnabled() {
			return nil, errors.New("this endpoint only enabled in demo mode")
		}

		_, sess, err := svc.Login(ctx, req.Email, req.Password)

		// This endpoint handles errors slightly differently in that we want to still return the
		// HTML page redirect to login if there was some error, so we can't just return the response
		// error without doing the rest of the logic.

		session := struct {
			Token string
		}{}
		var resp demologinResponse
		if err != nil {
			resp.Err = err
		}
		if sess != nil {
			session.Token = sess.Key
		}

		relayStateLoadPage := `<!DOCTYPE html>
     <script type='text/javascript'>
     window.localStorage.setItem('FLEET::auth_token', '{{ .Token }}');
     window.location = "/";
     </script>
     <body>
     Redirecting to Fleet...
     </body>
     </html>
    `
		tmpl, err := template.New("demologin").Parse(relayStateLoadPage)
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
