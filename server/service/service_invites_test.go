package service

import (
	"context"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteNewUserMock(t *testing.T) {
	ms := new(mock.Store)
	ms.UserByEmailFunc = mock.UserWithEmailNotFound()
	ms.AppConfigFunc = mock.ReturnFakeAppConfig(&kolide.AppConfig{
		KolideServerURL: "https://acme.co",
	})
	ms.NewInviteFunc = func(i *kolide.Invite) (*kolide.Invite, error) {
		return i, nil
	}
	mailer := &mockMailService{SendEmailFn: func(e kolide.Email) error { return nil }}
	svc := validationMiddleware{service{
		ds:          ms,
		config:      config.TestConfig(),
		mailService: mailer,
		clock:       clock.NewMockClock(),
	}, ms, nil}

	ctx := context.Background()
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &kolide.User{ID: 3}})
	payload := kolide.InvitePayload{
		Email: stringPtr("user@acme.co"),
		Admin: boolPtr(false),
	}

	// happy path
	invite, err := svc.InviteNewUser(ctx, payload)
	require.Nil(t, err)
	assert.Equal(t, uint(3), invite.InvitedBy)
	assert.True(t, ms.NewInviteFuncInvoked)
	assert.True(t, ms.AppConfigFuncInvoked)
	assert.True(t, mailer.Invoked)

	ms.UserByEmailFunc = mock.UserByEmailWithUser(new(kolide.User))
	_, err = svc.InviteNewUser(ctx, payload)
	require.NotNil(t, err, "should err if the user we're inviting already exists")
}

func TestVerifyInvite(t *testing.T) {
	ms := new(mock.Store)
	svc := service{
		ds:     ms,
		config: config.TestConfig(),
		clock:  clock.NewMockClock(),
	}
	ctx := context.Background()

	ms.InviteByTokenFunc = func(token string) (*kolide.Invite, error) {
		return &kolide.Invite{
			ID:    1,
			Token: "abcd",
			UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
				CreateTimestamp: kolide.CreateTimestamp{
					CreatedAt: time.Now().AddDate(-1, 0, 0),
				},
			},
		}, nil
	}
	wantErr := &invalidArgumentError{{name: "invite_token", reason: "Invite token has expired."}}
	_, err := svc.VerifyInvite(ctx, "abcd")
	assert.Equal(t, err, wantErr)

	wantErr = &invalidArgumentError{{name: "invite_token",
		reason: "Invite Token does not match Email Address."}}

	_, err = svc.VerifyInvite(ctx, "bad_token")
	assert.Equal(t, err, wantErr)
}

func TestDeleteInvite(t *testing.T) {
	ms := new(mock.Store)
	svc := service{ds: ms}

	ms.DeleteInviteFunc = func(uint) error { return nil }
	err := svc.DeleteInvite(context.Background(), 1)
	require.Nil(t, err)
	assert.True(t, ms.DeleteInviteFuncInvoked)
}

func TestListInvites(t *testing.T) {
	ms := new(mock.Store)
	svc := service{ds: ms}

	ms.ListInvitesFunc = func(kolide.ListOptions) ([]*kolide.Invite, error) {
		return nil, nil
	}
	_, err := svc.ListInvites(context.Background(), kolide.ListOptions{})
	require.Nil(t, err)
	assert.True(t, ms.ListInvitesFuncInvoked)
}
