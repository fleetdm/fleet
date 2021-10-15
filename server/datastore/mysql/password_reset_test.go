package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestPasswordReset(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Requests", testPasswordResetRequests},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPasswordResetRequests(t *testing.T, ds *Datastore) {
	createTestUsers(t, ds)
	now := time.Now().UTC()
	tomorrow := now.Add(time.Hour * 24)
	var passwordResetTests = []struct {
		userID  uint
		expires time.Time
		token   string
	}{
		{userID: 1, expires: tomorrow, token: "abcd"},
	}

	for _, tt := range passwordResetTests {
		r := &fleet.PasswordResetRequest{
			UserID:    tt.userID,
			ExpiresAt: tt.expires,
			Token:     tt.token,
		}
		req, err := ds.NewPasswordResetRequest(context.Background(), r)
		assert.Nil(t, err)
		assert.Equal(t, tt.userID, req.UserID)
	}
}
