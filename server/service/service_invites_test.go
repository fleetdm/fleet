package service

import (
	"context"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteNewUserMock(t *testing.T) {
	svc, mockStore, mailer := setupInviteTest(t)
	ctx := context.Background()

	payload := kolide.InvitePayload{
		Email:     stringPtr("user@acme.co"),
		InvitedBy: &adminUser.ID,
		Admin:     boolPtr(false),
	}

	// happy path
	invite, err := svc.InviteNewUser(ctx, payload)
	require.Nil(t, err)
	assert.Equal(t, invite.ID, validInvite.ID)
	assert.True(t, mockStore.NewInviteFuncInvoked)
	assert.True(t, mockStore.AppConfigFuncInvoked)
	assert.True(t, mailer.Invoked)

	mockStore.UserByEmailFunc = mock.UserByEmailWithUser(new(kolide.User))
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

	ms.InviteByTokenFunc = mock.ReturnFakeInviteByToken(expiredInvite)
	wantErr := &invalidArgumentError{{name: "invite_token", reason: "Invite token has expired."}}
	_, err := svc.VerifyInvite(ctx, expiredInvite.Token)
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

func setupInviteTest(t *testing.T) (kolide.Service, *mock.Store, *mockMailService) {

	ms := new(mock.Store)
	ms.UserByEmailFunc = mock.UserWithEmailNotFound()
	ms.UserByIDFunc = mock.UserWithID(adminUser)
	ms.NewInviteFunc = mock.ReturnNewInivite(validInvite)
	ms.AppConfigFunc = mock.ReturnFakeAppConfig(&kolide.AppConfig{
		KolideServerURL: "https://acme.co",
	})
	mailer := &mockMailService{SendEmailFn: func(e kolide.Email) error { return nil }}
	svc := validationMiddleware{&service{
		ds:          ms,
		config:      config.TestConfig(),
		mailService: mailer,
		clock:       clock.NewMockClock(),
	}, ms, nil}
	return svc, ms, mailer
}

var adminUser = &kolide.User{
	ID:       1,
	Email:    "admin@acme.co",
	Username: "admin",
	Name:     "Administrator",
}

var validInvite = &kolide.Invite{
	ID:    1,
	Token: "abcd",
}

var expiredInvite = &kolide.Invite{
	ID:    1,
	Token: "abcd",
	UpdateCreateTimestamps: kolide.UpdateCreateTimestamps{
		CreateTimestamp: kolide.CreateTimestamp{
			CreatedAt: time.Now().AddDate(-1, 0, 0),
		},
	},
}
