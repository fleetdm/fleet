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
)

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
			UserID:          1,
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
	w := new(bytes.Buffer)
	r, _, _ := os.Pipe()
	app := createApp(r, w, nil)
	err := app.Run(append([]string{""}, args...))
	return w, err
}
