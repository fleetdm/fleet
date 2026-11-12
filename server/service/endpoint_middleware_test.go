package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/token"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
	"github.com/fleetdm/fleet/v4/server/platform/middleware/authzcheck"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO update this test for new patterns

// TestEndpointPermissions tests that
// the endpoint.Middleware correctly grants or denies
// permissions to access or modify resources
// func TestEndpointPermissions(t *testing.T) {
// 	req := struct{}{}
// 	ds, err := inmem.New(config.TestConfig())
// 	assert.Nil(t, err)

// 	createTestUsers(t, ds)

// 	admin1, err := ds.User("admin1")
// 	assert.Nil(t, err)
// 	admin1Session, err := ds.NewSession(&fleet.Session{
// 		UserID: admin1.ID,
// 		Key:    "admin1",
// 	})
// 	assert.Nil(t, err)

// 	user1, err := ds.User("user1")
// 	assert.Nil(t, err)
// 	user1Session, err := ds.NewSession(&fleet.Session{
// 		UserID: user1.ID,
// 		Key:    "user1",
// 	})
// 	assert.Nil(t, err)

// 	user2, err := ds.User("user2")
// 	assert.Nil(t, err)
// 	user2Session, err := ds.NewSession(&fleet.Session{
// 		UserID: user2.ID,
// 		Key:    "user2",
// 	})
// 	assert.Nil(t, err)
// 	user2.Enabled = false

// 	e := endpoint.Nop // a test endpoint
// 	var endpointTests = []struct {
// 		endpoint endpoint.Endpoint
// 		// who is making the request
// 		vc *viewer.Viewer
// 		// what resource are we editing
// 		requestID uint
// 		// what error to expect
// 		wantErr interface{}
// 		// custom request struct
// 		request interface{}
// 	}{
// 		{
// 			endpoint: canReadUser(e),
// 			wantErr:  fleet.ErrNoContext,
// 		},
// 		{
// 			endpoint: canModifyUser(e),
// 			wantErr:  fleet.ErrNoContext,
// 		},
// 		{
// 			endpoint: canModifyUser(e),
// 			vc:       &viewer.Viewer{User: admin1, Session: admin1Session},
// 		},
// 		{
// 			endpoint: canModifyUser(e),
// 			vc:       &viewer.Viewer{User: user1, Session: user1Session},
// 			wantErr:  permissionError{message: "no write permissions on user"},
// 		},
// 		{
// 			endpoint:  canModifyUser(e),
// 			vc:        &viewer.Viewer{User: user1, Session: user1Session},
// 			requestID: admin1.ID,
// 			wantErr:   permissionError{message: "no write permissions on user"},
// 		},
// 		{
// 			endpoint:  canReadUser(e),
// 			vc:        &viewer.Viewer{User: user1, Session: user1Session},
// 			requestID: admin1.ID,
// 		},
// 		{
// 			endpoint:  canReadUser(e),
// 			vc:        &viewer.Viewer{User: user2, Session: user2Session},
// 			requestID: admin1.ID,
// 			wantErr:   permissionError{message: "no read permissions on user"},
// 		},
// 	}

// 	for _, tt := range endpointTests {
// 		tt := tt
// 		t.Run("", func(st *testing.T) {
// 			st.Parallel()
// 			ctx := context.Background()
// 			if tt.vc != nil {
// 				ctx = viewer.NewContext(ctx, *tt.vc)
// 			}
// 			if tt.requestID != 0 {
// 				ctx = context.WithValue(ctx, "request-id", tt.requestID)
// 			}
// 			var request interface{}
// 			if tt.request != nil {
// 				request = tt.request
// 			} else {
// 				request = req
// 			}
// 			_, eerr := tt.endpoint(ctx, request)
// 			assert.IsType(st, tt.wantErr, eerr)
// 			if ferr, ok := eerr.(permissionError); ok {
// 				assert.Equal(st, tt.wantErr.(permissionError).message, ferr.Error())
// 			}
// 		})
// 	}
// }

type testNodeKeyRequest struct {
	NodeKey string
}

func (r *testNodeKeyRequest) hostNodeKey() string { return r.NodeKey }

func TestAuthenticatedHost(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expectedHost := fleet.Host{Hostname: "foo!", HasHostIdentityCert: ptr.Bool(false)}
	goodNodeKey := "foo bar baz bing bang boom"

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.LoadHostByNodeKeyFunc = func(ctx context.Context, nodeKey string) (*fleet.Host, error) {
		switch nodeKey {
		case goodNodeKey:
			return &expectedHost, nil
		default:
			return nil, errors.New("no host found")

		}
	}

	endpoint := authenticatedHost(
		svc,
		slog.New(slog.DiscardHandler),
		func(ctx context.Context, request interface{}) (interface{}, error) {
			host, ok := hostctx.FromContext(ctx)
			assert.True(t, ok)
			assert.Equal(t, &expectedHost, host)
			return nil, nil
		},
	)

	authenticatedHostTests := []struct {
		nodeKey   string
		shouldErr bool
	}{
		{
			nodeKey:   "invalid",
			shouldErr: true,
		},
		{
			nodeKey:   "",
			shouldErr: true,
		},
		{
			nodeKey:   goodNodeKey,
			shouldErr: false,
		},
	}

	for _, tt := range authenticatedHostTests {
		t.Run("", func(t *testing.T) {
			r := &testNodeKeyRequest{NodeKey: tt.nodeKey}
			_, err := endpoint(ctx, r)
			if tt.shouldErr {
				assert.IsType(t, &OsqueryError{}, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestAuthenticatedUserMW(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	authenticatedUserTests := []struct {
		user      *fleet.User
		shouldErr bool
	}{
		{
			user: &fleet.User{
				ID:                       32,
				Name:                     "name",
				Email:                    "em@il.com",
				AdminForcedPasswordReset: true,
				SSOEnabled:               true,
			},
			shouldErr: false,
		},
		{
			user: &fleet.User{
				ID:                       32,
				Name:                     "name",
				Email:                    "em@il.com",
				AdminForcedPasswordReset: true,
				SSOEnabled:               false,
			},
			shouldErr: true,
		},
	}

	for _, tt := range authenticatedUserTests {
		t.Run("", func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			nextCalled := false
			endpoint := auth.AuthenticatedUser(svc, func(ctx context.Context, request interface{}) (response interface{}, err error) {
				nextCalled = true
				return nil, nil
			})
			_, err := endpoint(ctx, nil)
			if tt.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.True(t, nextCalled)
			}
		})
	}
}

func TestAuthenticatedUserDatastoreErrors(t *testing.T) {
	dbErr := &net.OpError{Op: "dial", Net: "tcp", Err: errors.New("connect: connection refused")}
	user := &fleet.User{ID: 7, Email: "user@example.com"}

	validSession := func() *fleet.Session {
		return &fleet.Session{ID: 1, UserID: user.ID, Key: "key", AccessedAt: time.Now()}
	}
	expiredSession := func() *fleet.Session {
		return &fleet.Session{ID: 1, UserID: user.ID, Key: "key", AccessedAt: time.Now().Add(-365 * 24 * time.Hour)}
	}

	cases := []struct {
		name           string
		sessionByKey   func() (*fleet.Session, error)
		destroySession error
		markAccessed   error
		userByID       func() (*fleet.User, error)
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name:         "session lookup db error",
			sessionByKey: func() (*fleet.Session, error) { return nil, dbErr },
			wantStatus:   http.StatusInternalServerError,
		},
		{
			name:         "session not found",
			sessionByKey: func() (*fleet.Session, error) { return nil, &notFoundError{} },
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "client canceled",
			sessionByKey: func() (*fleet.Session, error) { return nil, context.Canceled },
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:         "expired session",
			sessionByKey: func() (*fleet.Session, error) { return expiredSession(), nil },
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:           "expired session destroy db error",
			sessionByKey:   func() (*fleet.Session, error) { return expiredSession(), nil },
			destroySession: dbErr,
			wantStatus:     http.StatusInternalServerError,
		},
		{
			name:         "mark session accessed db error",
			sessionByKey: func() (*fleet.Session, error) { return validSession(), nil },
			markAccessed: dbErr,
			wantStatus:   http.StatusInternalServerError,
		},
		{
			name:         "user lookup db error",
			sessionByKey: func() (*fleet.Session, error) { return validSession(), nil },
			userByID:     func() (*fleet.User, error) { return nil, dbErr },
			wantStatus:   http.StatusInternalServerError,
		},
		{
			name:         "user not found",
			sessionByKey: func() (*fleet.Session, error) { return validSession(), nil },
			userByID:     func() (*fleet.User, error) { return nil, &notFoundError{} },
			wantStatus:   http.StatusUnauthorized,
		},
		{
			name:           "valid session and user",
			sessionByKey:   func() (*fleet.Session, error) { return validSession(), nil },
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			svc, ctx := newTestService(t, ds, nil, nil)
			ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
				return tc.sessionByKey()
			}
			ds.DestroySessionFunc = func(ctx context.Context, session *fleet.Session) error {
				return tc.destroySession
			}
			ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error {
				return tc.markAccessed
			}
			ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
				if tc.userByID != nil {
					return tc.userByID()
				}
				return user, nil
			}
			ctx = token.NewContext(ctx, "key")

			t.Run("endpoint", func(t *testing.T) {
				nextCalled := false
				ep := authzcheck.NewMiddleware().AuthzCheck()(auth.AuthenticatedUser(svc, func(ctx context.Context, request any) (any, error) {
					nextCalled = true
					if ac, ok := authz_ctx.FromContext(ctx); ok {
						ac.SetChecked()
					}
					return nil, nil
				}))
				_, err := ep(ctx, nil)
				rec := httptest.NewRecorder()
				var logs bytes.Buffer
				if err != nil {
					encodeError(ctx, err, rec)
					(&endpointer.ErrorHandler{Logger: slog.New(slog.NewTextHandler(&logs, nil))}).Handle(ctx, err)
				}
				assert.Equal(t, tc.wantStatus, rec.Code)
				assert.Equal(t, tc.wantNextCalled, nextCalled)
				switch {
				case tc.wantStatus == http.StatusUnauthorized:
					assert.Contains(t, rec.Body.String(), "Authentication required")
				case tc.wantStatus >= http.StatusInternalServerError:
					var jsonErr endpointer.JsonError
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jsonErr))
					assert.Equal(t, platform_http.GenericErrorMessage, jsonErr.Message)
					assert.NotContains(t, rec.Body.String(), "connection refused")
					assert.Contains(t, logs.String(), "connection refused")
				}
			})

			t.Run("http middleware", func(t *testing.T) {
				var gotDetail string
				errHandler := func(w http.ResponseWriter, detail string, status int) {
					gotDetail = detail
					w.WriteHeader(status)
				}
				nextCalled := false
				h := auth.AuthenticatedUserMiddleware(svc, errHandler, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					nextCalled = true
				}))
				lc := &logging.LoggingContext{}
				req := httptest.NewRequestWithContext(logging.NewContext(ctx, lc), http.MethodGet, "/api/v1/fleet/scim/Users", nil)
				rec := httptest.NewRecorder()
				h.ServeHTTP(rec, req)

				assert.Equal(t, tc.wantStatus, rec.Code)
				assert.Equal(t, tc.wantNextCalled, nextCalled)
				if tc.wantStatus >= http.StatusInternalServerError {
					assert.Equal(t, platform_http.GenericErrorMessage, gotDetail)
					var logs bytes.Buffer
					lc.Log(req.Context(), slog.New(slog.NewTextHandler(&logs, nil)))
					assert.Contains(t, logs.String(), "connection refused")
				}
			})
		})
	}
}
