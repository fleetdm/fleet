package service

import (
	"context"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInviteNewUserMock(t *testing.T) {
	ms := new(mock.Store)
	ms.UserByEmailFunc = mock.UserWithEmailNotFound()
	ms.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "https://acme.co"}}, nil
	}

	ms.NewInviteFunc = func(ctx context.Context, i *fleet.Invite) (*fleet.Invite, error) {
		return i, nil
	}
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error { return nil }}

	svc := validationMiddleware{&Service{
		ds:          ms,
		config:      config.TestConfig(),
		mailService: mailer,
		clock:       clock.NewMockClock(),
		authz:       authz.Must(),
	}, ms, nil}

	payload := fleet.InvitePayload{
		Email: ptr.String("user@acme.co"),
	}

	// happy path
	invite, err := svc.InviteNewUser(test.UserContext(test.UserAdmin), payload)
	require.Nil(t, err)
	assert.Equal(t, test.UserAdmin.ID, invite.InvitedBy)
	assert.True(t, ms.NewInviteFuncInvoked)
	assert.True(t, ms.AppConfigFuncInvoked)
	assert.True(t, mailer.Invoked)

	ms.UserByEmailFunc = mock.UserByEmailWithUser(new(fleet.User))
	_, err = svc.InviteNewUser(test.UserContext(test.UserAdmin), payload)
	require.NotNil(t, err, "should err if the user we're inviting already exists")
}

func TestVerifyInvite(t *testing.T) {
	ms := new(mock.Store)
	svc := newTestService(ms, nil, nil)

	ms.InviteByTokenFunc = func(ctx context.Context, token string) (*fleet.Invite, error) {
		return &fleet.Invite{
			ID:    1,
			Token: "abcd",
			UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
				CreateTimestamp: fleet.CreateTimestamp{
					CreatedAt: time.Now().AddDate(-1, 0, 0),
				},
			},
		}, nil
	}
	wantErr := fleet.NewInvalidArgumentError("invite_token", "Invite token has expired.")
	_, err := svc.VerifyInvite(test.UserContext(test.UserAdmin), "abcd")
	assert.Equal(t, err, wantErr)

	wantErr = fleet.NewInvalidArgumentError("invite_token", "Invite Token does not match Email Address.")

	_, err = svc.VerifyInvite(test.UserContext(test.UserAdmin), "bad_token")
	assert.Equal(t, err, wantErr)
}

func TestDeleteInvite(t *testing.T) {
	ms := new(mock.Store)
	svc := newTestService(ms, nil, nil)

	ms.DeleteInviteFunc = func(context.Context, uint) error { return nil }
	err := svc.DeleteInvite(test.UserContext(test.UserAdmin), 1)
	require.Nil(t, err)
	assert.True(t, ms.DeleteInviteFuncInvoked)
}

func TestListInvites(t *testing.T) {
	ms := new(mock.Store)
	svc := newTestService(ms, nil, nil)

	ms.ListInvitesFunc = func(context.Context, fleet.ListOptions) ([]*fleet.Invite, error) {
		return nil, nil
	}
	_, err := svc.ListInvites(test.UserContext(test.UserAdmin), fleet.ListOptions{})
	require.Nil(t, err)
	assert.True(t, ms.ListInvitesFuncInvoked)
}
