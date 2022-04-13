package main

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// runServerWithMockedDS runs the fleet server with several mocked DS methods.
//
// NOTE: Assumes the current session is always from the admin user (see ds.SessionByKeyFunc below).
func runServerWithMockedDS(t *testing.T, opts ...service.TestServerOpts) (*httptest.Server, *mock.Store) {
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
	_, server := service.RunServerForTestsWithDS(t, ds, opts...)
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
	require.Equal(t, errorMsg, err.Error())
	return w.String()
}

func runAppNoChecks(args []string) (*bytes.Buffer, error) {
	// first arg must be the binary name. Allow tests to omit it.
	args = append([]string{""}, args...)

	w := new(bytes.Buffer)
	app := createApp(nil, w, noopExitErrHandler)
	err := app.Run(args)
	return w, err
}

func noopExitErrHandler(c *cli.Context, err error) {}
