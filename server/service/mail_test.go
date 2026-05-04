package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

var testMailpitSMTPPort = getTestMailpitSMTPPort()
var testMailpitWebURL = getTestMailpitWebURL()

func getTestMailpitSMTPPort() uint {
	if port := os.Getenv("FLEET_MAILPIT_SMTP_PORT"); port != "" {
		if p, err := strconv.ParseUint(port, 10, 32); err == nil && p > 0 {
			return uint(p)
		}
	}
	return 1026
}

func getTestMailpitWebURL() string {
	if port := os.Getenv("FLEET_MAILPIT_WEB_PORT"); port != "" {
		return "http://127.0.0.1:" + port
	}
	return "http://127.0.0.1:8026"
}

type notTestFoundError struct{}

func (e *notTestFoundError) Error() string {
	return "not found"
}

func (e *notTestFoundError) IsNotFound() bool {
	return true
}

func newTestNotFoundError() *notTestFoundError {
	return &notTestFoundError{}
}

// Is is implemented so that errors.Is(err, sql.ErrNoRows) returns true for an
// error of type *notFoundError, without having to wrap sql.ErrNoRows
// explicitly.
func (e *notTestFoundError) Is(other error) bool {
	return other == sql.ErrNoRows
}

func TestMailService(t *testing.T) {
	// This mail test requires mailpit (ports read from env vars FLEET_MAILPIT_SMTP_PORT, FLEET_MAILPIT_WEB_PORT).
	if _, ok := os.LookupEnv("MAIL_TEST"); !ok {
		t.Skip("Mail tests are disabled")
	}

	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		UseMailService: true,
	})

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			SMTPSettings: &fleet.SMTPSettings{
				SMTPEnabled:              true,
				SMTPConfigured:           true,
				SMTPAuthenticationType:   fleet.AuthTypeNameUserNamePassword,
				SMTPAuthenticationMethod: fleet.AuthMethodNamePlain,
				SMTPUserName:             "mailpit-username",
				SMTPPassword:             "mailpit-password",
				SMTPEnableTLS:            false,
				SMTPVerifySSLCerts:       false,
				SMTPPort:                 testMailpitSMTPPort,
				SMTPServer:               "localhost",
				SMTPSenderAddress:        "foobar@example.com",
			},
		}, nil
	}

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return nil, newTestNotFoundError()
	}

	var invite *fleet.Invite
	ds.NewInviteFunc = func(ctx context.Context, i *fleet.Invite) (*fleet.Invite, error) {
		invite = i
		return invite, nil
	}

	ds.SaveAppConfigFunc = func(ctx context.Context, info *fleet.AppConfig) error {
		return nil
	}

	ds.InviteFunc = func(ctx context.Context, id uint) (*fleet.Invite, error) {
		return invite, nil
	}

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}

	ctx = test.UserContext(ctx, test.UserAdmin)

	// (1) Modifying the app config `sender_address` field to trigger a test e-mail send.
	_, err := svc.ModifyAppConfig(ctx, fmt.Appendf(nil, `{
  "org_info": {
	"org_name": "Acme"
  },
  "server_settings": {
	"server_url": "http://someurl"
  },
  "smtp_settings": {
    "enable_smtp": true,
    "configured": true,
    "authentication_type": "authtype_username_password",
    "authentication_method": "authmethod_plain",
    "user_name": "mailpit-username",
    "password": "mailpit-password",
    "enable_ssl_tls": false,
    "verify_ssl_certs": false,
    "port": %d,
    "server": "127.0.0.1",
    "sender_address": "foobar_updated@example.com"
  }
}`, testMailpitSMTPPort), fleet.ApplySpecOptions{})
	require.NoError(t, err)

	getLastMailPitMessage := func() map[string]interface{} {
		resp, err := http.Get(testMailpitWebURL + "/api/v1/messages?limit=1")
		require.NoError(t, err)
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		var m map[string]interface{}
		err = json.Unmarshal(b, &m)
		require.NoError(t, err)
		require.NotNil(t, m["messages"])
		require.Len(t, m["messages"], 1)
		lm := (m["messages"]).([]interface{})[0]
		require.NotNil(t, lm)
		lastMessage := lm.(map[string]interface{})
		fmt.Printf("%+v\n", lastMessage)
		return lastMessage
	}

	lastMessage := getLastMailPitMessage()
	require.Equal(t, "Hello from Fleet", lastMessage["Subject"])

	// (2) Inviting a user should send an e-mail to join.
	_, err = svc.InviteNewUser(ctx, fleet.InvitePayload{
		Email:      ptr.String("foobar_recipient@example.com"),
		Name:       ptr.String("Foobar"),
		GlobalRole: null.NewString("observer", true),
	})
	require.NoError(t, err)

	lastMessage = getLastMailPitMessage()
	require.Equal(t, "You have been invited to Fleet!", lastMessage["Subject"])

	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		if id == 1 {
			return test.UserAdmin, nil
		}
		return nil, newNotFoundError()
	}
	ds.InviteByEmailFunc = func(ctx context.Context, email string) (*fleet.Invite, error) {
		return nil, newTestNotFoundError()
	}
	ds.PendingEmailChangeFunc = func(ctx context.Context, userID uint, newEmail, token string) error {
		return nil
	}
	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		return nil
	}

	// (3) Changing e-mail address should send an e-mail for confirmation.
	_, err = svc.ModifyUser(ctx, 1, fleet.UserPayload{
		Email: ptr.String("useradmin_2@example.com"),
	})
	require.NoError(t, err)

	lastMessage = getLastMailPitMessage()
	require.Equal(t, "Confirm Fleet Email Change", lastMessage["Subject"])
}
