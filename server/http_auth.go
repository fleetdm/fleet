package server

import (
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
	kitlog "github.com/go-kit/kit/log"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

// authentication error
type authError struct {
	reason string
	// client reason is used to provide
	// a different error message to the client
	// when security is a concern
	clientReason string
}

func (e authError) Error() string {
	return e.reason
}

func (e authError) AuthError() string {
	if e.clientReason != "" {
		return e.clientReason
	}
	return "authentication error"
}

// permissionError, set when user is authenticated, but not allowed to perform action
type permissionError struct {
	message string
	badArgs []invalidArgument
}

func (e permissionError) Error() string {
	switch len(e.badArgs) {
	case 0:
	case 1:
		e.message = fmt.Sprintf("unauthorized: %s",
			e.badArgs[0].reason,
		)
	default:
		e.message = fmt.Sprintf("unauthorized: %s and %d other errors",
			e.badArgs[0].reason,
			len(e.badArgs),
		)
	}
	if e.message == "" {
		return "unauthorized"
	}
	return e.message
}

func (e permissionError) PermissionError() []map[string]string {
	var forbidden []map[string]string
	if len(e.badArgs) == 0 {
		forbidden = append(forbidden, map[string]string{"reason": e.Error()})
		return forbidden
	}
	for _, arg := range e.badArgs {
		forbidden = append(forbidden, map[string]string{
			"name":   arg.name,
			"reason": arg.reason,
		})
	}
	return forbidden

}

// ViewerContext is a struct which represents the ability for an execution
// context to participate in certain actions. Most often, a ViewerContext is
// associated with an application user, but a ViewerContext can represent a
// variety of other execution contexts as well (script, test, etc). The main
// purpose of a ViewerContext is to assist in the authorization of sensitive
// actions.
type viewerContext struct {
	user    *kolide.User
	session *kolide.Session
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

func (vc *viewerContext) SessionID() uint {
	if vc.session != nil {
		return vc.session.ID
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
	return vc.IsLoggedIn() && !vc.user.AdminForcedPasswordReset
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

// newViewerContext generates a ViewerContext given a session
func newViewerContext(user *kolide.User, session *kolide.Session) *viewerContext {
	return &viewerContext{
		user:    user,
		session: session,
	}
}

// emptyVC is a utility which generates an empty ViewerContext. This is often
// used to represent users which are not logged in.
func emptyVC() *viewerContext {
	return &viewerContext{}
}

func osqueryHostFromContext(ctx context.Context) (*kolide.Host, error) {
	host, ok := ctx.Value("osqueryHost").(*kolide.Host)
	if !ok {
		return nil, errNoContext
	}
	return host, nil
}

func viewerContextFromContext(ctx context.Context) (*viewerContext, error) {
	vc, ok := ctx.Value("viewerContext").(*viewerContext)
	if !ok {
		return nil, errNoContext
	}
	return vc, nil
}

// setViewerContext updates the context with a viewerContext,
// which holds the currently logged in user
func setViewerContext(svc kolide.Service, ds kolide.Datastore, jwtKey string, logger kitlog.Logger) kithttp.RequestFunc {
	return func(ctx context.Context, r *http.Request) context.Context {
		token, err := request.ParseFromRequest(r, request.AuthorizationHeaderExtractor, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtKey), nil
		})
		if err != nil {
			if err != request.ErrNoTokenInRequest {
				// all unauthenticated requests (login,logout,passwordreset) result in the
				// request.ErrNoTokenInRequest error. we can ignore logging it
				logger.Log("err", err, "error-source", "setViewerContext")
			}
			return context.WithValue(ctx, "viewerContext", emptyVC())
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return context.WithValue(ctx, "viewerContext", emptyVC())
		}

		sessionKeyClaim, ok := claims["session_key"]
		if !ok {
			return context.WithValue(ctx, "viewerContext", emptyVC())
		}

		sessionKey, ok := sessionKeyClaim.(string)
		if !ok {
			return context.WithValue(ctx, "viewerContext", emptyVC())
		}

		session, err := svc.GetSessionByKey(ctx, sessionKey)
		if err != nil {
			switch err {
			case kolide.ErrNoActiveSession:
				// If the code path got this far, it's likely that the user was logged
				// in some time in the past, but their session has been expired since
				// their last usage of the application
				return context.WithValue(ctx, "viewerContext", emptyVC())
			default:
				return context.WithValue(ctx, "viewerContext", emptyVC())
			}
		}

		user, err := svc.User(ctx, session.UserID)
		if err != nil {
			logger.Log("err", err, "error-source", "setViewerContext")
			return context.WithValue(ctx, "viewerContext", emptyVC())
		}

		ctx = context.WithValue(ctx, "viewerContext", newViewerContext(user, session))
		logger.Log("msg", "viewer context set", "user", user.ID)
		// get the user-id for request
		if strings.Contains(r.URL.Path, "users/") {
			ctx = withUserIDFromRequest(r, ctx)
		}
		return ctx
	}
}

func withUserIDFromRequest(r *http.Request, ctx context.Context) context.Context {
	id, _ := idFromRequest(r, "id")
	return context.WithValue(ctx, "request-id", id)
}
