package service

import (
	"errors"
	"testing"

	"golang.org/x/net/context"

	"github.com/WatchBeam/clock"
	"github.com/kolide/kolide-ose/server/config"
	"github.com/kolide/kolide-ose/server/datastore/inmem"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func TestInviteNewUser(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	createTestUsers(t, ds)
	createTestAppConfig(t, ds)
	assert.Nil(t, err)
	nosuchAdminID := uint(999)
	adminID := uint(1)
	mailer := &mockMailService{SendEmailFn: func(e kolide.Email) error { return nil }}
	svc := validationMiddleware{service{
		ds:          ds,
		config:      config.TestConfig(),
		mailService: mailer,
		clock:       clock.NewMockClock(),
	}}

	var inviteTests = []struct {
		payload kolide.InvitePayload
		wantErr error
	}{
		{
			wantErr: &invalidArgumentError{
				{name: "email", reason: "missing required argument"},
				{name: "invited_by", reason: "missing required argument"},
				{name: "admin", reason: "missing required argument"},
			},
		},
		{
			payload: kolide.InvitePayload{
				Email:     stringPtr("nosuchuser@example.com"),
				InvitedBy: &nosuchAdminID,
				Admin:     boolPtr(false),
			},
			wantErr: errors.New("User 999 was not found in the datastore"),
		},
		{
			payload: kolide.InvitePayload{
				Email:     stringPtr("admin1@example.com"),
				InvitedBy: &adminID,
				Admin:     boolPtr(false),
			},
			wantErr: &invalidArgumentError{
				{name: "email", reason: "a user with this account already exists"}},
		},
		{
			payload: kolide.InvitePayload{
				Email:     stringPtr("nosuchuser@example.com"),
				InvitedBy: &adminID,
				Admin:     boolPtr(false),
			},
		},
	}

	for _, tt := range inviteTests {
		t.Run("", func(t *testing.T) {
			invite, err := svc.InviteNewUser(context.Background(), tt.payload)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr.Error(), err.Error())
				return
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, *tt.payload.InvitedBy, invite.InvitedBy)
		})
	}
}
