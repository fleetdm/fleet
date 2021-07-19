package main

import (
	"bytes"
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

func runServerWithMockedDS(t *testing.T) (*httptest.Server, *mock.Store) {
	ds := new(mock.Store)
	var users []*fleet.User
	ds.NewUserFunc = func(user *fleet.User) (*fleet.User, error) {
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              1,
			AccessedAt:      time.Now(),
			UserID:          1,
			Key:             key,
		}, nil
	}
	ds.MarkSessionAccessedFunc = func(session *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(id uint) (*fleet.User, error) {
		return users[0], nil
	}
	ds.ListUsersFunc = func(opt fleet.UserListOptions) ([]*fleet.User, error) {
		return users, nil
	}
	_, server := service.RunServerForTestsWithDS(t, ds)
	os.Setenv("FLEET_SERVER_ADDRESS", server.URL)

	return server, ds
}

func runAppForTest(t *testing.T, args []string) string {
	w := new(bytes.Buffer)
	r, _, _ := os.Pipe()
	var exitErr error
	app := createApp(r, w, func(context *cli.Context, err error) {
		exitErr = err
	})
	err := app.Run(append([]string{""}, args...))
	require.Nil(t, err)
	require.Nil(t, exitErr)
	return w.String()
}
