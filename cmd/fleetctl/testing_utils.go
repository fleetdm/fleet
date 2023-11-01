package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/cached_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// runServerWithMockedDS runs the fleet server with several mocked DS methods.
//
// NOTE: Assumes the current session is always from the admin user (see ds.SessionByKeyFunc below).
func runServerWithMockedDS(t *testing.T, opts ...*service.TestServerOpts) (*httptest.Server, *mock.Store) {
	ds := new(mock.Store)
	var users []*fleet.User
	var admin *fleet.User
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
			admin = user
		}
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              1,
			AccessedAt:      time.Now(),
			UserID:          admin.ID,
			Key:             key,
		}, nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return admin, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return users, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	var cachedDS fleet.Datastore
	if len(opts) > 0 && opts[0].NoCacheDatastore {
		cachedDS = ds
	} else {
		cachedDS = cached_mysql.New(ds)
	}
	_, server := service.RunServerForTestsWithDS(t, cachedDS, opts...)
	os.Setenv("FLEET_SERVER_ADDRESS", server.URL)

	return server, ds
}

func runAppForTest(t *testing.T, args []string) string {
	w, err := runAppNoChecks(args)
	require.NoError(t, err)
	return w.String()
}

func runAppCheckErr(t *testing.T, args []string, errorMsg string) string {
	w, err := runAppNoChecks(args)
	require.Error(t, err)
	require.Equal(t, errorMsg, err.Error())
	return w.String()
}

func runAppNoChecks(args []string) (*bytes.Buffer, error) {
	// first arg must be the binary name. Allow tests to omit it.
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, os.Stderr, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func runWithErrWriter(args []string, errWriter io.Writer) (*bytes.Buffer, error) {
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, errWriter, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}

func serveMDMBootstrapPackage(t *testing.T, pkgPath, pkgName string) (*httptest.Server, int) {
	pkgBytes, err := os.ReadFile(pkgPath)
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(pkgBytes)))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, pkgName))
		if n, err := w.Write(pkgBytes); err != nil {
			require.NoError(t, err)
			require.Equal(t, len(pkgBytes), n)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, len(pkgBytes)
}
