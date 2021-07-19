package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestPasswordResetRequests(t *testing.T) {
	db := CreateMySQLDS(t)
	defer db.Close()

	createTestUsers(t, db)
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
		req, err := db.NewPasswordResetRequest(r)
		assert.Nil(t, err)
		assert.Equal(t, tt.userID, req.UserID)
	}
}
