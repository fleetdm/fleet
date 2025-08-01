package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.ListSessionsForUserFunc = func(ctx context.Context, id uint) ([]*fleet.Session, error) {
		if id == 999 {
			return []*fleet.Session{
				{ID: 1, UserID: id, AccessedAt: time.Now()},
			}, nil
		}
		return nil, nil
	}
	ds.SessionByIDFunc = func(ctx context.Context, id uint) (*fleet.Session, error) {
		return &fleet.Session{ID: id, UserID: 999, AccessedAt: time.Now()}, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{ID: 111, GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{ID: 111, GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			true,
		},
		{
			"global observer",
			&fleet.User{ID: 111, GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"owner user",
			&fleet.User{ID: 999},
			false,
			false,
		},
		{
			"non-owner user",
			&fleet.User{ID: 888},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.GetInfoAboutSessionsForUser(ctx, 999)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.GetInfoAboutSession(ctx, 1)
			checkAuthErr(t, tt.shouldFailRead, err)

			err = svc.DeleteSession(ctx, 1)
			checkAuthErr(t, tt.shouldFailWrite, err)
		})
	}
}

func TestAuthenticate(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc, ctx := newTestService(t, ds, nil, nil)
	createTestUsers(t, ds)

	loginTests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "admin1",
			email:    testUsers["admin1"].Email,
			password: testUsers["admin1"].PlaintextPassword,
		},
		{
			name:     "user1",
			email:    testUsers["user1"].Email,
			password: testUsers["user1"].PlaintextPassword,
		},
	}

	for _, tt := range loginTests {
		t.Run(tt.email, func(st *testing.T) {
			loggedIn, token, err := svc.Login(test.UserContext(ctx, test.UserAdmin), tt.email, tt.password, false)
			require.Nil(st, err, "login unsuccessful")
			assert.Equal(st, tt.email, loggedIn.Email)
			assert.NotEmpty(st, token)

			sessions, err := svc.GetInfoAboutSessionsForUser(test.UserContext(ctx, test.UserAdmin), loggedIn.ID)
			require.Nil(st, err)
			require.Len(st, sessions, 1, "user should have one session")
			session := sessions[0]
			assert.NotZero(st, session.UserID)
			assert.WithinDuration(st, time.Now(), session.AccessedAt, 3*time.Second,
				"access time should be set with current time at session creation")
		})
	}
}

func TestMFA(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	user := &fleet.User{MFAEnabled: true, Name: "Bob Smith", Email: "foo@example.com"}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return user, nil
	}
	_, _, err := svc.Login(ctx, "foo@example.com", test.GoodPassword, false)
	require.Equal(t, err, mfaNotSupportedForClient)

	var sentMail fleet.Email
	mailer := &mockMailService{SendEmailFn: func(e fleet.Email) error {
		sentMail = e
		return nil
	}}
	mfaToken := "foovalidate"
	ds.NewMFATokenFunc = func(ctx context.Context, userID uint) (string, error) {
		return mfaToken, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	svcForMailing := validationMiddleware{&Service{
		ds:          ds,
		config:      config.TestConfig(),
		mailService: mailer,
	}, ds, nil}
	_, _, err = svcForMailing.Login(ctx, "foo@example.com", test.GoodPassword, true)
	require.Equal(t, err, sendingMFAEmail)
	require.Equal(t, "foo@example.com", sentMail.To[0])
	require.Equal(t, "Log in to Fleet", sentMail.Subject)

	var session *fleet.Session
	var mfaUser *fleet.User
	ds.SessionByMFATokenFunc = func(ctx context.Context, token string, sessionKeySize int) (*fleet.Session, *fleet.User, error) {
		if token == mfaToken {
			return session, mfaUser, nil
		}
		return nil, nil, notFoundErr{}
	}
	resp, err := sessionCreateEndpoint(ctx, &sessionCreateRequest{Token: "foo"}, svc)
	require.NoError(t, err)
	require.NotNil(t, resp.Error())

	session = &fleet.Session{}
	mfaUser = user
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time) error {
		require.Equal(t, mfaUser, user)
		require.Equal(t, fleet.ActivityTypeUserLoggedIn{}.ActivityName(), activity.ActivityName())
		return nil
	}
	resp, err = sessionCreateEndpoint(ctx, &sessionCreateRequest{Token: mfaToken}, svc)
	require.NoError(t, err)
	require.Nil(t, resp.Error())
	require.True(t, ds.NewActivityFuncInvoked)
}

func TestGetSessionByKey(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	cfg := config.TestConfig()

	theSession := &fleet.Session{UserID: 123, Key: "abc"}

	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return theSession, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}

	cases := []struct {
		desc     string
		accessed time.Duration
		apiOnly  bool
		fail     bool
	}{
		{"real user, accessed recently", -1 * time.Hour, false, false},
		{"real user, accessed too long ago", -(cfg.Session.Duration + time.Hour), false, true},
		{"api-only, accessed recently", -1 * time.Hour, true, false},
		{"api-only, accessed long ago", -(cfg.Session.Duration + time.Hour), true, false},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authErr *fleet.AuthRequiredError
			ds.SessionByKeyFuncInvoked, ds.DestroySessionFuncInvoked, ds.MarkSessionAccessedFuncInvoked = false, false, false

			theSession.AccessedAt = time.Now().Add(tc.accessed)
			theSession.APIOnly = ptr.Bool(tc.apiOnly)
			_, err := svc.GetSessionByKey(ctx, theSession.Key)
			if tc.fail {
				require.Error(t, err)
				require.ErrorAs(t, err, &authErr)
				require.True(t, ds.SessionByKeyFuncInvoked)
				require.True(t, ds.DestroySessionFuncInvoked)
				require.False(t, ds.MarkSessionAccessedFuncInvoked)
			} else {
				require.NoError(t, err)
				require.True(t, ds.SessionByKeyFuncInvoked)
				require.False(t, ds.DestroySessionFuncInvoked)
				require.True(t, ds.MarkSessionAccessedFuncInvoked)
			}
		})
	}
}

type testAuth struct {
	userID              string
	userDisplayName     string
	requestID           string
	assertionAttributes []fleet.SAMLAttribute
}

var _ fleet.Auth = (*testAuth)(nil)

func (a *testAuth) UserID() string {
	return a.userID
}

func (a *testAuth) UserDisplayName() string {
	return a.userDisplayName
}

func (a *testAuth) RequestID() string {
	return a.requestID
}

func (a *testAuth) AssertionAttributes() []fleet.SAMLAttribute {
	return a.assertionAttributes
}

func (a *testAuth) RawResponse() []byte {
	return nil
}

func TestGetSSOUser(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
	})

	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			SSOSettings: &fleet.SSOSettings{
				EnableSSO:             true,
				EnableSSOIdPLogin:     true,
				EnableJITProvisioning: true,
			},
		}, nil
	}

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return nil, newNotFoundError()
	}

	var newUser *fleet.User
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		newUser = user
		return user, nil
	}

	auth := &testAuth{
		userID:          "foo@example.com",
		userDisplayName: "foo@example.com",
		requestID:       "foobar",
		assertionAttributes: []fleet.SAMLAttribute{
			{
				Name: "FLEET_JIT_USER_ROLE_GLOBAL",
				Values: []fleet.SAMLAttributeValue{
					{Value: "admin"},
				},
			},
		},
	}

	// Test SSO login with a non-existent user.
	_, err := svc.GetSSOUser(ctx, auth)
	require.NoError(t, err)

	require.NotNil(t, newUser)
	require.NotNil(t, newUser.GlobalRole)
	require.Equal(t, "admin", *newUser.GlobalRole)
	require.Empty(t, newUser.Teams)

	// Test SSO login with the same (now existing) user (should update roles).

	// (1) Check that when a user's role attributes are unchanged then SavedUser is not called.

	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		return nil
	}

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		return newUser, nil
	}

	_, err = svc.GetSSOUser(ctx, auth)
	require.NoError(t, err)

	require.False(t, ds.SaveUserFuncInvoked)

	// (2) Test SSO login with the same user with roles updated in its attributes.

	var savedUser *fleet.User
	ds.SaveUserFunc = func(ctx context.Context, user *fleet.User) error {
		savedUser = user
		return nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return &fleet.Team{ID: tid}, nil
	}

	auth.assertionAttributes = []fleet.SAMLAttribute{
		{
			Name: "FLEET_JIT_USER_ROLE_TEAM_2",
			Values: []fleet.SAMLAttributeValue{
				{Value: "maintainer"},
			},
		},
	}

	_, err = svc.GetSSOUser(ctx, auth)
	require.NoError(t, err)

	require.NotNil(t, savedUser)
	require.Nil(t, savedUser.GlobalRole)
	require.Len(t, savedUser.Teams, 1)
	require.Equal(t, uint(2), savedUser.Teams[0].ID)
	require.Equal(t, "maintainer", savedUser.Teams[0].Role)

	require.True(t, ds.SaveUserFuncInvoked)

	// (3) Test existing user's role is not changed after a new login if EnableJITProvisioning is false.

	ds.SaveUserFuncInvoked = false

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			SSOSettings: &fleet.SSOSettings{
				EnableSSO:             true,
				EnableSSOIdPLogin:     true,
				EnableJITProvisioning: false,
			},
		}, nil
	}

	auth.assertionAttributes = []fleet.SAMLAttribute{
		{
			Name: "FLEET_JIT_USER_ROLE_TEAM_2",
			Values: []fleet.SAMLAttributeValue{
				{Value: "admin"},
			},
		},
	}

	_, err = svc.GetSSOUser(ctx, auth)
	require.NoError(t, err)

	require.False(t, ds.SaveUserFuncInvoked)

	// (4) Test with invalid team ID in the attributes

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			SSOSettings: &fleet.SSOSettings{
				EnableSSO:             true,
				EnableSSOIdPLogin:     true,
				EnableJITProvisioning: true,
			},
		}, nil
	}

	ds.TeamFunc = func(ctx context.Context, tid uint) (*fleet.Team, error) {
		return nil, newNotFoundError()
	}

	auth.assertionAttributes = []fleet.SAMLAttribute{
		{
			Name: "FLEET_JIT_USER_ROLE_TEAM_3",
			Values: []fleet.SAMLAttributeValue{
				{Value: "maintainer"},
			},
		},
	}

	_, err = svc.GetSSOUser(ctx, auth)
	require.Error(t, err)
}

func TestInitiateSSOWithSSOServerURL(t *testing.T) {
	ds := new(mock.Store)
	pool := redistest.SetupRedis(t, t.Name(), false, false, false)

	svc, ctx := newTestServiceWithConfig(t, ds, config.TestConfig(), nil, nil, &TestServerOpts{
		Pool: pool,
	})

	// Mock app config with SSO server URL
	appConfig := &fleet.AppConfig{
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://fleet.example.com",
		},
		SSOSettings: &fleet.SSOSettings{
			EnableSSO:    true,
			SSOServerURL: "https://admin.fleet.example.com",
			SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID: "fleet",
				IDPName:  "TestIDP",
				Metadata: `<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" entityID="test-idp">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://idp.example.com/sso"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`,
			},
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return appConfig, nil
	}

	// Test that ACS URL uses SSO URL
	sessionID, _, idpURL, err := svc.InitiateSSO(ctx, "/dashboard")
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)
	require.NotEmpty(t, idpURL)

	// The ACS URL should use the SSO server URL
	// We can't directly test the ACS URL in the SAML request here since it's embedded in the XML,
	// but the integration test verifies this works correctly
}
