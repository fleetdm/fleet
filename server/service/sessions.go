package service

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/publicip"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mail"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/go-kit/log/level"
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

func (r getInfoAboutSessionResponse) Error() error { return r.Err }

func getInfoAboutSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

func (r deleteSessionResponse) Error() error { return r.Err }

func deleteSessionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

type loginResponse struct {
	User           *fleet.User          `json:"user,omitempty"`
	AvailableTeams []*fleet.TeamSummary `json:"available_teams"`
	Token          string               `json:"token,omitempty"`
	Err            error                `json:"error,omitempty"`
}

func (r loginResponse) Error() error { return r.Err }

type loginMfaResponse struct {
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

func (r loginMfaResponse) Status() int { return http.StatusAccepted }

func (r loginMfaResponse) Error() error { return r.Err }

func loginEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*contract.LoginRequest)
	req.Email = strings.ToLower(req.Email)

	user, session, err := svc.Login(ctx, req.Email, req.Password, req.SupportsEmailVerification)
	if err != nil {
		if errors.Is(err, sendingMFAEmail) {
			return loginMfaResponse{Message: "We sent an email to you. Please click the magic link in the email to sign in."}, nil
		}

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

var (
	//goland:noinspection GoErrorStringFormat
	sendingMFAEmail = errors.New("sending MFA email")

	noMFASupported           = errors.New("client with no MFA email support")
	mfaNotSupportedForClient = endpoint_utils.BadRequestErr(
		"Your login client does not support MFA. Please log in via the web, then use an API token to authenticate.",
		noMFASupported,
	)
)

func (svc *Service) Login(ctx context.Context, email, password string, supportsEmailVerification bool) (*fleet.User, *fleet.Session, error) {
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
		if err != nil && !errors.Is(err, sendingMFAEmail) && !errors.Is(err, mfaNotSupportedForClient) {
			if err := svc.NewActivity(
				ctx, nil, fleet.ActivityTypeUserFailedLogin{
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
	} else if user.MFAEnabled {
		if !supportsEmailVerification {
			return nil, nil, mfaNotSupportedForClient
		}

		if err = svc.makeMFAEmail(ctx, *user); err != nil {
			return nil, nil, fleet.NewAuthFailedError(err.Error())
		}

		return nil, nil, sendingMFAEmail
	}

	session, err := svc.makeSession(ctx, user.ID)
	if err != nil {
		return nil, nil, fleet.NewAuthFailedError(err.Error())
	}

	if err := svc.NewActivity(
		ctx, user, fleet.ActivityTypeUserLoggedIn{
			PublicIP: publicip.FromContext(ctx),
		}); err != nil {
		return nil, nil, err
	}
	return user, session, nil
}

func (svc *Service) makeSession(ctx context.Context, userID uint) (*fleet.Session, error) {
	return svc.ds.NewSession(ctx, userID, svc.config.Session.KeySize)
}

////////////////////////////////////////////////////////////////////////////////
// Session create (second step of MFA)
////////////////////////////////////////////////////////////////////////////////

type sessionCreateRequest struct {
	Token string `json:"token,omitempty"`
}

func sessionCreateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*sessionCreateRequest)
	session, user, err := svc.CompleteMFA(ctx, req.Token)
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

func (svc *Service) CompleteMFA(ctx context.Context, token string) (*fleet.Session, *fleet.User, error) {
	// skipauth: No user context available yet to authorize against.
	svc.authz.SkipAuthorization(ctx)

	var err error
	defer func(start time.Time) { // force MFA failures to take at least a second for brute force/timing attack resistance
		if err != nil {
			time.Sleep(time.Until(start.Add(1 * time.Second)))
		}
	}(time.Now())

	session, user, err := svc.ds.SessionByMFAToken(ctx, token, svc.config.Session.KeySize)
	if err != nil {
		return nil, nil, fleet.NewAuthFailedError(err.Error())
	}

	if err := svc.NewActivity(
		ctx, user, fleet.ActivityTypeUserLoggedIn{
			PublicIP: publicip.FromContext(ctx),
		}); err != nil {
		return nil, nil, err
	}
	return session, user, nil
}

////////////////////////////////////////////////////////////////////////////////
// Logout
////////////////////////////////////////////////////////////////////////////////

type logoutResponse struct {
	Err error `json:"error,omitempty"`
}

func (r logoutResponse) Error() error { return r.Err }

func logoutEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

	sessionID              string
	sessionDurationSeconds int
}

const cookieNameSSOSession = "__Host-FLEETSSOSESSIONID"

func (r initiateSSOResponse) Error() error { return r.Err }

// cookieSecure is defined as a variable for testing purposes.
var cookieSecure = true

func setSSOCookie(w http.ResponseWriter, sessionID string, cookieDurationSeconds int) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameSSOSession,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   cookieDurationSeconds,
		Secure:   cookieSecure,
		HttpOnly: true,
		// SameSite: Strict or Lax do not work with SSO.
	})
}

func deleteSSOCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameSSOSession,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   cookieSecure,
		HttpOnly: true,
	})
}

func (r initiateSSOResponse) SetCookies(_ context.Context, w http.ResponseWriter) {
	setSSOCookie(w, r.sessionID, r.sessionDurationSeconds)
}

func initiateSSOEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*initiateSSORequest)
	sessionID, sessionDurationSeconds, idProviderURL, err := svc.InitiateSSO(ctx, req.RelayURL)
	if err != nil {
		return initiateSSOResponse{Err: err}, nil
	}
	return initiateSSOResponse{
		URL:                    idProviderURL,
		sessionID:              sessionID,
		sessionDurationSeconds: sessionDurationSeconds,
	}, nil
}

// InitiateSSO initiates a Single Sign-On flow for a request to visit the
// protected URL identified by redirectURL. It returns the URL of the identity
// provider to make a request to to proceed with the authentication via that
// external service, and stores ephemeral session state to validate the
// callback from the identity provider to finalize the SSO flow.
func (svc *Service) InitiateSSO(ctx context.Context, redirectURL string) (sessionID string, sessionDurationSeconds int, idpURL string, err error) {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// initiate SSO.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", 0, "", ctxerr.Wrap(ctx, err, "InitiateSSO getting app config")
	}

	if appConfig.SSOSettings == nil || !appConfig.SSOSettings.EnableSSO {
		err := &fleet.BadRequestError{Message: "organization not configured to use sso"}
		return "", 0, "", ctxerr.Wrap(ctx, newSSOError(err, ssoOrgDisabled), "initiate sso")
	}

	parsedUrl, err := url.Parse(redirectURL)
	if err != nil {
		return "", 0, "", ctxerr.Wrap(ctx, badRequest("invalid sso redirect url"))
	}
	if slices.Contains([]string{"javascript", "vbscript", "data"}, parsedUrl.Scheme) {
		return "", 0, "", ctxerr.Wrap(ctx, badRequest("invalid sso redirect url scheme: "+parsedUrl.Scheme))
	}

	serverURL := appConfig.ServerSettings.ServerURL
	acsURL := serverURL + svc.config.Server.URLPrefix + "/api/v1/fleet/sso/callback"

	// If entityID is not explicitly set, default to host name.
	//
	// NOTE(lucas): This code may be required if SSO was configured with an older version of Fleet
	// where EntityID wasn't required to configure SSO.
	entityID := appConfig.SSOSettings.EntityID
	if entityID == "" {
		u, err := url.Parse(serverURL)
		if err != nil {
			return "", 0, "", ctxerr.Wrap(ctx, err, "parse server url")
		}
		entityID = u.Hostname()
	}

	samlProvider, err := sso.SAMLProviderFromConfiguredMetadata(ctx,
		entityID,
		acsURL,
		&appConfig.SSOSettings.SSOProviderSettings,
	)
	if err != nil {
		return "", 0, "", ctxerr.Wrap(ctx, err, "failed to create provider from configured metadata")
	}

	sessionDurationSeconds = int(svc.config.Auth.SsoSessionValidityPeriod.Seconds())
	sessionID, idpURL, err = sso.CreateAuthorizationRequest(
		ctx, samlProvider, svc.ssoSessionStore, redirectURL,
		uint(sessionDurationSeconds), //nolint:gosec // dismiss G115
	)
	if err != nil {
		return "", 0, "", ctxerr.Wrap(ctx, err, "InitiateSSO creating authorization")
	}

	return sessionID, sessionDurationSeconds, idpURL, nil
}

////////////////////////////////////////////////////////////////////////////////
// Callback SSO
////////////////////////////////////////////////////////////////////////////////

type callbackSSORequest struct {
	sessionID    string
	samlResponse []byte
}

func (c callbackSSORequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	sessionID, samlResponse, err := decodeCallbackRequest(ctx, r)
	if err != nil {
		return nil, err
	}
	return &callbackSSORequest{
		sessionID:    sessionID,
		samlResponse: samlResponse,
	}, nil
}

func decodeCallbackRequest(ctx context.Context, r *http.Request) (
	sessionID string,
	decodedSAMLResponse []byte,
	err error,
) {
	cs, err := r.Cookie(cookieNameSSOSession)
	switch {
	case err == nil:
		sessionID = cs.Value
	case errors.Is(err, http.ErrNoCookie):
		// SessionID cookie will be empty on IdP-initiated logins.
	default:
		return "", nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: "failed to read SSO cookie session ID",
		}, "cookie session ID in SSO callback")
	}

	if err := r.ParseForm(); err != nil {
		return "", nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to parse form",
			InternalErr: err,
		}, "parse form in SSO callback")
	}

	samlResponseValue := r.FormValue("SAMLResponse")
	if samlResponseValue == "" {
		return "", nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: "missing SAMLResponse",
		}, "missing SAMLResponse in SSO callback")
	}
	decodedSAMLResponseValue, err := sso.DecodeSAMLResponse(samlResponseValue)
	if err != nil {
		return "", nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to decode SAMLResponse",
			InternalErr: err,
		}, "decode SAMLResponse in SSO callback")
	}
	return sessionID, decodedSAMLResponseValue, nil
}

type callbackSSOResponse struct {
	content string
	Err     error `json:"error,omitempty"`
}

func (r callbackSSOResponse) Error() error { return r.Err }

// If html is present we return a web page
func (r callbackSSOResponse) Html() string { return r.content }

func (r callbackSSOResponse) SetCookies(_ context.Context, w http.ResponseWriter) {
	deleteSSOCookie(w)
}

func makeCallbackSSOEndpoint(urlPrefix string) endpoint_utils.HandlerFunc {
	return func(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
		callbackRequest := request.(*callbackSSORequest)
		session, userID, err := getSSOSession(ctx, svc, callbackRequest)
		var resp callbackSSOResponse
		if err != nil {
			if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeUserFailedLogin{
				Email:    userID,
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

func getSSOSession(
	ctx context.Context,
	svc fleet.Service,
	callbackRequest *callbackSSORequest,
) (session *fleet.SSOSession, userID string, err error) {
	auth, redirectURL, err := svc.InitSSOCallback(ctx, callbackRequest.sessionID, callbackRequest.samlResponse)
	if err != nil {
		return nil, "", err
	}

	user, err := svc.GetSSOUser(ctx, auth)
	if err != nil {
		return nil, auth.UserID(), err
	}

	session, err = svc.LoginSSOUser(ctx, user, redirectURL)
	if err != nil {
		return nil, auth.UserID(), err
	}

	return session, auth.UserID(), nil
}

func (svc *Service) InitSSOCallback(
	ctx context.Context,
	sessionID string,
	samlResponse []byte,
) (auth fleet.Auth, redirectURL string, err error) {
	// skipauth: User context does not yet exist. Unauthenticated users may
	// hit the SSO callback.
	svc.authz.SkipAuthorization(ctx)

	logging.WithLevel(logging.WithNoUser(ctx), level.Info)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "get config for sso")
	}

	if appConfig.SSOSettings == nil || !appConfig.SSOSettings.EnableSSO {
		err := ctxerr.New(ctx, "organization not configured to use sso")
		return nil, "", ctxerr.Wrap(ctx, newSSOError(err, ssoOrgDisabled), "callback sso")
	}

	serverURL := appConfig.ServerSettings.ServerURL
	acsURL, err := url.Parse(serverURL + svc.config.Server.URLPrefix + "/api/v1/fleet/sso/callback")
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "failed to parse ACS URL")
	}

	expectedAudiences := []string{
		appConfig.SSOSettings.EntityID,
		appConfig.ServerSettings.ServerURL,
		appConfig.ServerSettings.ServerURL + svc.config.Server.URLPrefix + "/api/v1/fleet/sso/callback", // ACS
	}
	samlProvider, requestID, redirectURL, err := sso.SAMLProviderFromSessionOrConfiguredMetadata(
		ctx, sessionID, svc.ssoSessionStore, acsURL, appConfig.SSOSettings, expectedAudiences,
	)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "failed to create provider from metadata")
	}

	// Parse and verify SAMLResponse (verifies fields, expected IDs and signature).
	auth, err = sso.ParseAndVerifySAMLResponse(samlProvider, samlResponse, requestID, acsURL)
	if err != nil {
		// We actually don't return 401 to clients and instead return an HTML page with /login?status=error,
		// but to be consistent we will return fleet.AuthFailedError which is used for unauthorized access.
		return nil, "", ctxerr.Wrap(ctx, fleet.NewAuthFailedError(err.Error()))
	}

	return auth, redirectURL, nil
}

func (svc *Service) GetSSOUser(ctx context.Context, auth fleet.Auth) (*fleet.User, error) {
	user, err := svc.ds.UserByEmail(ctx, auth.UserID())
	if err != nil {
		var nfe endpoint_utils.NotFoundErrorInterface
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
	err = svc.NewActivity(
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

func (r ssoSettingsResponse) Error() error { return r.Err }

func settingsSSOEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (fleet.Errorer, error) {
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

// makeMFAEmail sends an MFA email to the given user
func (svc *Service) makeMFAEmail(ctx context.Context, user fleet.User) error {
	token, err := svc.ds.NewMFAToken(ctx, user.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating MFA token")
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	var smtpSettings fleet.SMTPSettings
	if config.SMTPSettings != nil {
		smtpSettings = *config.SMTPSettings
	}
	email := fleet.Email{
		Subject:      "Log in to Fleet",
		To:           []string{user.Email},
		ServerURL:    config.ServerSettings.ServerURL,
		SMTPSettings: smtpSettings,
		Mailer: &mail.MFAMailer{
			FullName: user.Name,
			Token:    token,
			BaseURL:  template.URL(config.ServerSettings.ServerURL + svc.config.Server.URLPrefix), //nolint:gosec // dismiss G203
			AssetURL: getAssetURL(),
		},
	}

	return svc.mailService.SendEmail(ctx, email)
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
