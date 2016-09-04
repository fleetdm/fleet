package kitserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func login(svc kolide.Service, logger kitlog.Logger) http.HandlerFunc {
	ctx := context.Background()
	logger = kitlog.NewContext(logger).With("method", "login")
	return func(w http.ResponseWriter, r *http.Request) {
		var loginRequest = struct {
			Username *string
			Password *string
		}{}
		if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
			encodeResponse(ctx, w, getUserResponse{
				Err: err,
			})
			logger.Log("err", err)
			return
		}
		var username, password string
		{
			if loginRequest.Username != nil {
				username = *loginRequest.Username
			}
			if loginRequest.Password != nil {
				password = *loginRequest.Password
			}
		}

		// retrieve user or respond with error
		user, err := svc.Authenticate(ctx, username, password)
		switch err.(type) {
		case nil:
			logger.Log("msg", "authenticated", "user", username, "id", user.ID)
		case authError:
			encodeResponse(ctx, w, getUserResponse{
				Err: err,
			})
			logger.Log("err", err, "user", username)
			return
		default:
			encodeResponse(ctx, w, getUserResponse{
				Err: errors.New("unknown error, try again later"),
			})
			logger.Log("err", err, "user", username)
			return
		}

		// create session here
		sm := svc.NewSessionManager(ctx, w, r)

		// TODO it feels awkward to create and then save the session in two steps.
		// the session manager should just call Save on it's own?
		if err := sm.MakeSessionForUserID(user.ID); err != nil {
			encodeResponse(ctx, w, getUserResponse{
				Err: errors.New("error creating new user session"),
			})
			logger.Log("err", err, "user", username)
			return
		}

		if err := sm.Save(); err != nil {
			encodeResponse(ctx, w, getUserResponse{
				Err: errors.New("error saving new user session"),
			})
			logger.Log("err", err, "user", username)
			return
		}

		encodeResponse(ctx, w, getUserResponse{
			ID:                 user.ID,
			Username:           user.Username,
			Name:               user.Name,
			Admin:              user.Admin,
			Enabled:            user.Enabled,
			NeedsPasswordReset: user.NeedsPasswordReset,
		})

	}
}

const noAuthRedirect = "/"

func logout(svc kolide.Service, logger kitlog.Logger) http.HandlerFunc {
	logger = kitlog.NewContext(logger).With("method", "logout")
	ctx := context.Background()
	return func(w http.ResponseWriter, r *http.Request) {
		sm := svc.NewSessionManager(ctx, w, r)
		if err := sm.Destroy(); err != nil {
			encodeResponse(ctx, w, getUserResponse{
				Err: errors.New("error deleting session"),
			})
			logger.Log("err", err)
			return
		}

		// redirect
		http.Redirect(w, r, noAuthRedirect, http.StatusFound)
	}
}

func authMiddleware(svc kolide.Service, logger kitlog.Logger, next http.Handler) http.Handler {
	logger = kitlog.NewContext(logger).With("method", "authMiddleware")
	ctx := context.Background()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sm := svc.NewSessionManager(ctx, w, r)
		session, err := sm.Session()
		if err != nil {
			http.Error(w,
				"failed to retrieve user session. is there a user logged in?",
				http.StatusUnauthorized)
			logger.Log("err", err)
			return
		}

		user, err := svc.User(ctx, session.UserID)
		if err != nil {
			http.Error(w,
				"failed to get user from db", http.StatusUnauthorized)
			logger.Log("err", err, "user", session.UserID)
			return
		}

		if !user.Enabled {
			http.Error(w, "user disabled", http.StatusUnauthorized)
		}

		// all good to pass
		next.ServeHTTP(w, r)
	})
}

// authentication error
type authError struct {
	message string
}

func (e authError) Error() string {
	if e.message == "" {
		return "unauthorized"
	}
	return fmt.Sprintf("unauthorized: %s", e.message)
}

// forbidden, set when user is authenticated, but not allowd to perform action
type forbiddenError struct {
	message string
}

func (e forbiddenError) Error() string {
	if e.message == "" {
		return "unauthorized"
	}
	return fmt.Sprintf("unauthorized: %s", e.message)
}

// ViewerContext is a struct which represents the ability for an execution
// context to participate in certain actions. Most often, a ViewerContext is
// associated with an application user, but a ViewerContext can represent a
// variety of other execution contexts as well (script, test, etc). The main
// purpose of a ViewerContext is to assist in the authorization of sensitive
// actions.
type viewerContext struct {
	user *kolide.User
}

// IsAdmin indicates whether or not the current user can perform administrative
// actions.
func (vc *viewerContext) IsAdmin() bool {
	if vc.user != nil {
		return vc.user.Admin && vc.user.Enabled
	}
	return false
}

// UserID is a helper that enables quick access to the user ID of the current
// user.
func (vc *viewerContext) UserID() uint {
	if vc.user != nil {
		return vc.user.ID
	}
	return 0
}

// IsLoggedIn determines whether or not the current VC is attached to a user
// account
func (vc *viewerContext) IsLoggedIn() bool {
	return vc.user != nil && vc.user.Enabled
}

// CanPerformActions returns a bool indicating the current user's ability to
// perform the most basic actions on the site
func (vc *viewerContext) CanPerformActions() bool {
	return vc.IsLoggedIn() && !vc.user.NeedsPasswordReset
}

// CanPerformReadActionsOnUser returns a bool indicating the current user's
// ability to perform read actions on the given user
func (vc *viewerContext) CanPerformReadActionOnUser(uid uint) bool {
	return vc.CanPerformActions() || (vc.IsLoggedIn() && vc.IsUserID(uid))
}

// CanPerformWriteActionOnUser returns a bool indicating the current user's
// ability to perform write actions on the given user
func (vc *viewerContext) CanPerformWriteActionOnUser(uid uint) bool {
	return vc.CanPerformActions() && (vc.IsUserID(uid) || vc.IsAdmin())
}

// IsUserID returns true if the given user id the same as the user which is
// represented by this ViewerContext
func (vc *viewerContext) IsUserID(id uint) bool {
	if vc.UserID() == id {
		return true
	}
	return false
}

// newViewerContext generates a ViewerContext given a user struct
func newViewerContext(user *kolide.User) *viewerContext {
	return &viewerContext{
		user: user,
	}
}

// emptyVC is a utility which generates an empty ViewerContext. This is often
// used to represent users which are not logged in.
func emptyVC() *viewerContext {
	return &viewerContext{}
}

func vcFromID(ds kolide.UserStore, id uint) (*viewerContext, error) {
	user, err := ds.UserByID(id)
	if err != nil {
		return nil, err
	}
	return &viewerContext{user: user}, nil
}
