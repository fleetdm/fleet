package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfigAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	// start a TLS server and use its URL as the server URL in the app config,
	// required by the CertificateChain service call.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ServerURL: srv.URL,
			},
		}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
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
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			false,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			false,
		},
		{
			"user",
			&fleet.User{ID: 777},
			true,
			false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, err := svc.AppConfig(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyAppConfig(ctx, []byte(`{}`))
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.Version(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.CertificateChain(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

func TestEnrollSecretAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, tid *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	ds.GetEnrollSecretsFunc = func(ctx context.Context, tid *uint) ([]*fleet.EnrollSecret, error) {
		return nil, nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool
		shouldFailRead  bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
		},
		{
			"user",
			&fleet.User{ID: 777},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			err := svc.ApplyEnrollSecretSpec(ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "ABC"}}})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetEnrollSecretSpec(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

func TestCertificateChain(t *testing.T) {
	server, teardown := setupCertificateChain(t)
	defer teardown()

	certFile := "testdata/server.pem"
	cert, err := tls.LoadX509KeyPair(certFile, "testdata/server.key")
	require.Nil(t, err)
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	server.StartTLS()

	u, err := url.Parse(server.URL)
	require.Nil(t, err)

	conn, err := connectTLS(context.Background(), u)
	require.Nil(t, err)

	have, want := len(conn.ConnectionState().PeerCertificates), len(cert.Certificate)
	require.Equal(t, have, want)

	original, _ := ioutil.ReadFile(certFile)
	returned, err := chain(context.Background(), conn.ConnectionState(), "")
	require.Nil(t, err)
	require.Equal(t, returned, original)
}

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(dump)
	})
}

func setupCertificateChain(t *testing.T) (server *httptest.Server, teardown func()) {
	server = httptest.NewUnstartedServer(echoHandler())
	return server, server.Close
}

func TestSSONotPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	var p fleet.AppConfig
	validateSSOSettings(p, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())
}

func TestNeedFieldsPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: fleet.SSOSettings{
			EnableSSO:   true,
			EntityID:    "fleet",
			IssuerURI:   "http://issuer.idp.com",
			MetadataURL: "http://isser.metadata.com",
			IDPName:     "onelogin",
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())
}

func TestShortIDPName(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: fleet.SSOSettings{
			EnableSSO:   true,
			EntityID:    "fleet",
			IssuerURI:   "http://issuer.idp.com",
			MetadataURL: "http://isser.metadata.com",
			// A customer once found the Fleet server erroring when they used "SSO" for their IdP name.
			IDPName: "SSO",
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid)
	assert.False(t, invalid.HasErrors())
}

func TestMissingMetadata(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: fleet.SSOSettings{
			EnableSSO: true,
			EntityID:  "fleet",
			IssuerURI: "http://issuer.idp.com",
			IDPName:   "onelogin",
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid)
	require.True(t, invalid.HasErrors())
	assert.Contains(t, invalid.Error(), "metadata")
	assert.Contains(t, invalid.Error(), "either metadata or metadata_url must be defined")
}

func TestAppConfigSecretsObfuscated(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	// start a TLS server and use its URL as the server URL in the app config,
	// required by the CertificateChain service call.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			SMTPSettings: fleet.SMTPSettings{SMTPPassword: "smtppassword"},
			Integrations: fleet.Integrations{
				Jira: []*fleet.JiraIntegration{
					{APIToken: "jiratoken"},
				},
				Zendesk: []*fleet.ZendeskIntegration{
					{APIToken: "zendesktoken"},
				},
			},
		}, nil
	}

	testCases := []struct {
		name string
		user *fleet.User
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
		},
		{
			"user",
			&fleet.User{ID: 777},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			ac, err := svc.AppConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, ac.SMTPSettings.SMTPPassword, fleet.MaskedPassword)
			require.Equal(t, ac.Integrations.Jira[0].APIToken, fleet.MaskedPassword)
			require.Equal(t, ac.Integrations.Zendesk[0].APIToken, fleet.MaskedPassword)
		})
	}
}

// TestModifyAppConfigSMTPConfigured tests that disabling SMTP
// should set the SMTPConfigured field to false.
func TestModifyAppConfigSMTPConfigured(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds, nil, nil)

	// SMTP is initially enabled and configured.
	dsAppConfig := &fleet.AppConfig{
		SMTPSettings: fleet.SMTPSettings{
			SMTPEnabled:    true,
			SMTPConfigured: true,
		},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return dsAppConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		*dsAppConfig = *conf
		return nil
	}

	// Disable SMTP.
	newAppConfig := fleet.AppConfig{
		SMTPSettings: fleet.SMTPSettings{
			SMTPEnabled:    false,
			SMTPConfigured: true,
		},
	}
	b, err := json.Marshal(newAppConfig)
	require.NoError(t, err)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: admin})
	updatedAppConfig, err := svc.ModifyAppConfig(ctx, b)
	require.NoError(t, err)

	// After disabling SMTP, the app config should be "not configured".
	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, updatedAppConfig.SMTPSettings.SMTPConfigured)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPConfigured)
}

// TestTransparencyURL tests that Fleet Premium licensees can use custom transparency urls and Fleet
// Free licensees are restricted to the default transparency url.
func TestTransparencyURL(t *testing.T) {
	ds := new(mock.Store)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: admin})

	checkLicenseErr := func(t *testing.T, shouldFail bool, err error) {
		if shouldFail {
			require.Error(t, err)
			require.ErrorContains(t, err, "missing or invalid license")
		} else {
			require.NoError(t, err)
		}
	}
	testCases := []struct {
		name             string
		licenseTier      string
		initialURL       string
		newURL           string
		expectedURL      string
		shouldFailModify bool
	}{
		{
			name:             "customURL",
			licenseTier:      "free",
			initialURL:       "",
			newURL:           "customURL",
			expectedURL:      "",
			shouldFailModify: true,
		},
		{
			name:             "customURL",
			licenseTier:      "premium",
			initialURL:       "",
			newURL:           "customURL",
			expectedURL:      "customURL",
			shouldFailModify: false,
		},
		{
			name:             "emptyURL",
			licenseTier:      "free",
			initialURL:       "",
			newURL:           "",
			expectedURL:      "",
			shouldFailModify: false,
		},
		{
			name:             "emptyURL",
			licenseTier:      "premium",
			initialURL:       "customURL",
			newURL:           "",
			expectedURL:      "",
			shouldFailModify: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tt.licenseTier}})

			dsAppConfig := &fleet.AppConfig{FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: tt.initialURL}}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
				*dsAppConfig = *conf
				return nil
			}

			ac, err := svc.AppConfig(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.initialURL, ac.FleetDesktop.TransparencyURL)

			raw, err := json.Marshal(fleet.AppConfig{FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: tt.newURL}})
			require.NoError(t, err)
			modified, err := svc.ModifyAppConfig(ctx, raw)
			checkLicenseErr(t, tt.shouldFailModify, err)

			if modified != nil {
				require.Equal(t, tt.expectedURL, modified.FleetDesktop.TransparencyURL)
				ac, err = svc.AppConfig(ctx)
				require.NoError(t, err)
				require.Equal(t, tt.expectedURL, ac.FleetDesktop.TransparencyURL)
			}
		})
	}
}
