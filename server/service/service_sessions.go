package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/sso"
	"github.com/pkg/errors"
)

func (svc service) SSOSettings(ctx context.Context) (*kolide.SSOSettings, error) {
	appConfig, err := svc.ds.AppConfig()
	if err != nil {
		return nil, errors.Wrap(err, "SSOSettings getting app config")
	}
	settings := &kolide.SSOSettings{
		IDPName:     appConfig.IDPName,
		IDPImageURL: appConfig.IDPImageURL,
		SSOEnabled:  appConfig.EnableSSO,
	}
	return settings, nil
}

func (svc service) InitiateSSO(ctx context.Context, redirectURL string) (string, error) {
	appConfig, err := svc.ds.AppConfig()
	if err != nil {
		return "", errors.Wrap(err, "InitiateSSO getting app config")
	}

	metadata, err := svc.getMetadata(appConfig)
	if err != nil {
		return "", errors.Wrap(err, "InitiateSSO getting metadata")
	}

	settings := sso.Settings{
		Metadata: metadata,
		// Construct call back url to send to idp
		AssertionConsumerServiceURL: appConfig.KolideServerURL + svc.config.Server.URLPrefix + "/api/v1/kolide/sso/callback",
		SessionStore:                svc.ssoSessionStore,
		OriginalURL:                 redirectURL,
	}

	// If issuer is not explicitly set, default to host name.
	var issuer string
	if appConfig.EntityID == "" {
		u, err := url.Parse(appConfig.KolideServerURL)
		if err != nil {
			return "", errors.Wrap(err, "parsing kolide server url")
		}
		issuer = u.Hostname()
	} else {
		issuer = appConfig.EntityID
	}
	idpURL, err := sso.CreateAuthorizationRequest(&settings, issuer)
	if err != nil {
		return "", errors.Wrap(err, "InitiateSSO creating authorization")
	}

	return idpURL, nil
}

func (svc service) getMetadata(config *kolide.AppConfig) (*sso.Metadata, error) {
	if config.MetadataURL != "" {
		metadata, err := sso.GetMetadata(config.MetadataURL, svc.metaDataClient)
		if err != nil {
			return nil, err
		}
		return metadata, nil
	}
	if config.Metadata != "" {
		metadata, err := sso.ParseMetadata(config.Metadata)
		if err != nil {
			return nil, err
		}
		return metadata, nil
	}
	return nil, errors.Errorf("missing metadata for idp %s", config.IDPName)
}

func (svc service) CallbackSSO(ctx context.Context, auth kolide.Auth) (*kolide.SSOSession, error) {
	// The signature and validity of auth response has been checked already in
	// validation middleware.
	sess, err := svc.ssoSessionStore.Get(auth.RequestID())
	if err != nil {
		return nil, errors.Wrap(err, "fetching sso session in callback")
	}
	// Remove session to so that is can't be reused before it expires.
	err = svc.ssoSessionStore.Expire(auth.RequestID())
	if err != nil {
		return nil, errors.Wrap(err, "expiring sso session in callback")
	}
	user, err := svc.userByEmailOrUsername(auth.UserID())
	if err != nil {
		return nil, errors.Wrap(err, "finding user in sso callback")
	}
	// if user is not active they are not authorized to use the application
	if !user.Enabled {
		return nil, errors.New("user authorization failed")
	}
	// if the user is not sso enabled they are not authorized
	if !user.SSOEnabled {
		return nil, errors.New("user not configured to use sso")
	}
	token, err := svc.makeSession(user.ID)
	if err != nil {
		return nil, errors.Wrap(err, "making user session in sso callback")
	}
	result := &kolide.SSOSession{
		Token:       token,
		RedirectURL: sess.OriginalURL,
	}
	if !strings.HasPrefix(result.RedirectURL, "/") {
		result.RedirectURL = svc.config.Server.URLPrefix + result.RedirectURL
	}
	return result, nil
}

func (svc service) Login(ctx context.Context, username, password string) (*kolide.User, string, error) {
	user, err := svc.userByEmailOrUsername(username)
	if _, ok := err.(kolide.NotFoundError); ok {
		return nil, "", authError{reason: "no such user"}
	}
	if err != nil {
		return nil, "", err
	}
	if !user.Enabled {
		return nil, "", authError{reason: "account disabled", clientReason: "account disabled"}
	}
	if user.SSOEnabled {
		const errMessage = "password login not allowed for single sign on users"
		return nil, "", authError{reason: errMessage, clientReason: errMessage}
	}
	if err = user.ValidatePassword(password); err != nil {
		return nil, "", authError{reason: "bad password"}
	}
	token, err := svc.makeSession(user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (svc service) userByEmailOrUsername(username string) (*kolide.User, error) {
	if strings.Contains(username, "@") {
		return svc.ds.UserByEmail(username)
	}
	return svc.ds.User(username)
}

// makeSession is a helper that creates a new session after authentication
func (svc service) makeSession(id uint) (string, error) {
	sessionKeySize := svc.config.Session.KeySize
	key := make([]byte, sessionKeySize)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}

	session := &kolide.Session{
		UserID:     id,
		Key:        base64.StdEncoding.EncodeToString(key),
		AccessedAt: time.Now().UTC(),
	}

	session, err = svc.ds.NewSession(session)
	if err != nil {
		return "", errors.Wrap(err, "creating new session")
	}

	tokenString, err := generateJWT(session.Key, svc.config.Auth.JwtKey)
	if err != nil {
		return "", errors.Wrap(err, "generating JWT token")
	}

	return tokenString, nil
}

func (svc service) Logout(ctx context.Context) error {
	// this should not return an error if the user wasn't logged in
	return svc.DestroySession(ctx)
}

func (svc service) DestroySession(ctx context.Context) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return errNoContext
	}

	session, err := svc.ds.SessionByID(vc.SessionID())
	if err != nil {
		return err
	}

	return svc.ds.DestroySession(session)
}

func (svc service) GetInfoAboutSessionsForUser(ctx context.Context, id uint) ([]*kolide.Session, error) {
	var validatedSessions []*kolide.Session

	sessions, err := svc.ds.ListSessionsForUser(id)
	if err != nil {
		return validatedSessions, err
	}

	for _, session := range sessions {
		if svc.validateSession(session) == nil {
			validatedSessions = append(validatedSessions, session)
		}
	}

	return validatedSessions, nil
}

func (svc service) DeleteSessionsForUser(ctx context.Context, id uint) error {
	return svc.ds.DestroyAllSessionsForUser(id)
}

func (svc service) GetInfoAboutSession(ctx context.Context, id uint) (*kolide.Session, error) {
	session, err := svc.ds.SessionByID(id)
	if err != nil {
		return nil, err
	}

	err = svc.validateSession(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (svc service) GetSessionByKey(ctx context.Context, key string) (*kolide.Session, error) {
	session, err := svc.ds.SessionByKey(key)
	if err != nil {
		return nil, err
	}

	err = svc.validateSession(session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func (svc service) DeleteSession(ctx context.Context, id uint) error {
	session, err := svc.ds.SessionByID(id)
	if err != nil {
		return err
	}
	return svc.ds.DestroySession(session)
}

func (svc service) validateSession(session *kolide.Session) error {
	if session == nil {
		return authError{
			reason:       "active session not present",
			clientReason: "session error",
		}
	}

	sessionDuration := svc.config.Session.Duration
	// duration 0 = unlimited
	if sessionDuration != 0 && time.Since(session.AccessedAt) >= sessionDuration {
		err := svc.ds.DestroySession(session)
		if err != nil {
			return errors.Wrap(err, "destroying session")
		}
		return authError{
			reason:       "expired session",
			clientReason: "session error",
		}
	}

	return svc.ds.MarkSessionAccessed(session)
}

// Given a session key create a JWT to be delivered to the client
func generateJWT(sessionKey, jwtKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"session_key": sessionKey,
	})

	return token.SignedString([]byte(jwtKey))
}
