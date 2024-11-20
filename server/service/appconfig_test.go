package service

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppConfigAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// start a TLS server and use its URL as the server URL in the app config,
	// required by the CertificateChain service call.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgName: "Test",
			},
			ServerSettings: fleet.ServerSettings{
				ServerURL: srv.URL,
			},
		}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		return nil
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
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
			false,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
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
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
			false,
		},
		{
			"team gitops",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
			true,
		},
		{
			"user without roles",
			&fleet.User{ID: 777},
			true,
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			_, err := svc.AppConfigObfuscated(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)

			_, err = svc.ModifyAppConfig(ctx, []byte(`{}`), fleet.ApplySpecOptions{})
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.CertificateChain(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

// TestVersion tests that all users can access the version endpoint.
func TestVersion(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

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
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
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
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
		},
		{
			"team gitops",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
		},
		{
			"user without roles",
			&fleet.User{ID: 777},
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})
			_, err := svc.Version(ctx)
			require.NoError(t, err)
		})
	}
}

func TestEnrollSecretAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

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
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.ApplyEnrollSecretSpec(
				ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "ABC"}}}, fleet.ApplySpecOptions{},
			)
			checkAuthErr(t, tt.shouldFailWrite, err)

			_, err = svc.GetEnrollSecretSpec(ctx)
			checkAuthErr(t, tt.shouldFailRead, err)
		})
	}
}

func TestApplyEnrollSecretWithGlobalEnrollConfig(t *testing.T) {
	ds := new(mock.Store)

	cfg := config.TestConfig()
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil)
	ctx = test.UserContext(ctx, test.UserAdmin)

	// Dry run
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		assert.False(t, new)
		assert.Nil(t, teamID)
		return true, nil
	}
	err := svc.ApplyEnrollSecretSpec(
		ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "ABC"}}}, fleet.ApplySpecOptions{DryRun: true},
	)
	assert.True(t, ds.IsEnrollSecretAvailableFuncInvoked)
	assert.NoError(t, err)

	// Dry run fails
	ds.IsEnrollSecretAvailableFuncInvoked = false
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		assert.False(t, new)
		assert.Nil(t, teamID)
		return false, nil
	}
	err = svc.ApplyEnrollSecretSpec(
		ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "ABC"}}}, fleet.ApplySpecOptions{DryRun: true},
	)
	assert.True(t, ds.IsEnrollSecretAvailableFuncInvoked)
	assert.ErrorContains(t, err, "secret is already being used")

	// Dry run with error
	ds.IsEnrollSecretAvailableFuncInvoked = false
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, new bool, teamID *uint) (bool, error) {
		return false, assert.AnError
	}
	err = svc.ApplyEnrollSecretSpec(
		ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "ABC"}}}, fleet.ApplySpecOptions{DryRun: true},
	)
	assert.True(t, ds.IsEnrollSecretAvailableFuncInvoked)
	assert.Equal(t, assert.AnError, err)

	ds.IsEnrollSecretAvailableFunc = nil
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		return nil
	}
	err = svc.ApplyEnrollSecretSpec(
		ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "ABC"}}}, fleet.ApplySpecOptions{},
	)
	require.True(t, ds.ApplyEnrollSecretsFuncInvoked)
	require.NoError(t, err)

	// try to change the enroll secret with the config set
	ds.ApplyEnrollSecretsFuncInvoked = false
	cfg.Packaging.GlobalEnrollSecret = "xyz"
	svc, ctx = newTestServiceWithConfig(t, ds, cfg, nil, nil)
	ctx = test.UserContext(ctx, test.UserAdmin)
	err = svc.ApplyEnrollSecretSpec(ctx, &fleet.EnrollSecretSpec{Secrets: []*fleet.EnrollSecret{{Secret: "DEF"}}}, fleet.ApplySpecOptions{})
	require.Error(t, err)
	require.False(t, ds.ApplyEnrollSecretsFuncInvoked)
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

	original, _ := os.ReadFile(certFile)
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
		w.Write(dump) //nolint:errcheck
	})
}

func setupCertificateChain(t *testing.T) (server *httptest.Server, teardown func()) {
	server = httptest.NewUnstartedServer(echoHandler())
	return server, server.Close
}

func TestSSONotPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	var p fleet.AppConfig
	validateSSOSettings(p, &fleet.AppConfig{}, invalid, &fleet.LicenseInfo{})
	assert.False(t, invalid.HasErrors())
}

func TestNeedFieldsPresent(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: &fleet.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:    "fleet",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				IDPName:     "onelogin",
			},
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid, &fleet.LicenseInfo{})
	assert.False(t, invalid.HasErrors())
}

func TestShortIDPName(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: &fleet.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:    "fleet",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				// A customer once found the Fleet server erroring when they used "SSO" for their IdP name.
				IDPName: "SSO",
			},
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid, &fleet.LicenseInfo{})
	assert.False(t, invalid.HasErrors())
}

func TestMissingMetadata(t *testing.T) {
	invalid := &fleet.InvalidArgumentError{}
	config := fleet.AppConfig{
		SSOSettings: &fleet.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:  "fleet",
				IssuerURI: "http://issuer.idp.com",
				IDPName:   "onelogin",
			},
		},
	}
	validateSSOSettings(config, &fleet.AppConfig{}, invalid, &fleet.LicenseInfo{})
	require.True(t, invalid.HasErrors())
	assert.Contains(t, invalid.Error(), "metadata")
	assert.Contains(t, invalid.Error(), "either metadata or metadata_url must be defined")
}

func TestSSOValidationValidatesSchemaInMetadataURL(t *testing.T) {
	var schemas []string
	schemas = append(schemas, getURISchemas()...)
	schemas = append(schemas, "asdfaklsdfjalksdfja")

	for _, scheme := range schemas {
		actual := &fleet.InvalidArgumentError{}
		sut := fleet.AppConfig{
			SSOSettings: &fleet.SSOSettings{
				EnableSSO: true,
				SSOProviderSettings: fleet.SSOProviderSettings{
					EntityID:    "fleet",
					IDPName:     "onelogin",
					MetadataURL: fmt.Sprintf("%s://somehost", scheme),
				},
			},
		}

		validateSSOSettings(sut, &fleet.AppConfig{}, actual, &fleet.LicenseInfo{})

		require.Equal(t, scheme == "http" || scheme == "https", !actual.HasErrors())
		require.Equal(t, scheme == "http" || scheme == "https", !strings.Contains(actual.Error(), "metadata_url"))
		require.Equal(t, scheme == "http" || scheme == "https", !strings.Contains(actual.Error(), "must be either https or http"))
	}
}

func TestJITProvisioning(t *testing.T) {
	config := fleet.AppConfig{
		SSOSettings: &fleet.SSOSettings{
			EnableSSO:             true,
			EnableJITProvisioning: true,
			SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:    "fleet",
				IssuerURI:   "http://issuer.idp.com",
				IDPName:     "onelogin",
				MetadataURL: "http://isser.metadata.com",
			},
		},
	}

	t.Run("doesn't allow to enable JIT provisioning without a premium license", func(t *testing.T) {
		invalid := &fleet.InvalidArgumentError{}
		validateSSOSettings(config, &fleet.AppConfig{}, invalid, &fleet.LicenseInfo{})
		require.True(t, invalid.HasErrors())
		assert.Contains(t, invalid.Error(), "enable_jit_provisioning")
		assert.Contains(t, invalid.Error(), "missing or invalid license")
	})

	t.Run("allows JIT provisioning to be enabled with a premium license", func(t *testing.T) {
		invalid := &fleet.InvalidArgumentError{}
		validateSSOSettings(config, &fleet.AppConfig{}, invalid, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		require.False(t, invalid.HasErrors())
	})

	t.Run("doesn't care if JIT provisioning is set to false on free licenses", func(t *testing.T) {
		invalid := &fleet.InvalidArgumentError{}
		oldConfig := &fleet.AppConfig{
			SSOSettings: &fleet.SSOSettings{
				EnableJITProvisioning: false,
			},
		}
		config.SSOSettings.EnableJITProvisioning = false
		validateSSOSettings(config, oldConfig, invalid, &fleet.LicenseInfo{})
		require.False(t, invalid.HasErrors())
	})
}

func TestAppConfigSecretsObfuscated(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// start a TLS server and use its URL as the server URL in the app config,
	// required by the CertificateChain service call.
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			SMTPSettings: &fleet.SMTPSettings{
				SMTPPassword: "smtppassword",
			},
			Integrations: fleet.Integrations{
				Jira: []*fleet.JiraIntegration{
					{APIToken: "jiratoken"},
				},
				Zendesk: []*fleet.ZendeskIntegration{
					{APIToken: "zendesktoken"},
				},
				GoogleCalendar: []*fleet.GoogleCalendarIntegration{
					{ApiKey: map[string]string{fleet.GoogleCalendarPrivateKey: "google-calendar-private-key"}},
				},
			},
		}, nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			false,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			false,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			false,
		},
		{
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			false,
		},
		{
			"team gitops",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"user without roles",
			&fleet.User{ID: 777},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			ac, err := svc.AppConfigObfuscated(ctx)
			if tt.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, ac.SMTPSettings.SMTPPassword, fleet.MaskedPassword)
				require.Equal(t, ac.Integrations.Jira[0].APIToken, fleet.MaskedPassword)
				require.Equal(t, ac.Integrations.Zendesk[0].APIToken, fleet.MaskedPassword)
				// Google Calendar private key is not obfuscated
				require.Equal(t, ac.Integrations.GoogleCalendar[0].ApiKey[fleet.GoogleCalendarPrivateKey], "google-calendar-private-key")
			}
		})
	}
}

// TestModifyAppConfigSMTPConfigured tests that disabling SMTP
// should set the SMTPConfigured field to false.
func TestModifyAppConfigSMTPConfigured(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// SMTP is initially enabled and configured.
	dsAppConfig := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://example.org",
		},
		SMTPSettings: &fleet.SMTPSettings{
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

	ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
		return nil
	}

	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{}, nil
	}

	// Disable SMTP.
	newAppConfig := fleet.AppConfig{
		SMTPSettings: &fleet.SMTPSettings{
			SMTPEnabled:    false,
			SMTPConfigured: true,
		},
	}
	b, err := json.Marshal(newAppConfig.SMTPSettings) // marshaling appconfig sets all fields, resetting e.g. OrgName to empty
	require.NoError(t, err)
	b = []byte(`{"smtp_settings":` + string(b) + `}`)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	updatedAppConfig, err := svc.ModifyAppConfig(ctx, b, fleet.ApplySpecOptions{})
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
			licenseTier:      fleet.TierPremium,
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
			licenseTier:      fleet.TierPremium,
			initialURL:       "customURL",
			newURL:           "",
			expectedURL:      "",
			shouldFailModify: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tt.licenseTier}})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

			dsAppConfig := &fleet.AppConfig{
				OrgInfo: fleet.OrgInfo{
					OrgName: "Test",
				},
				ServerSettings: fleet.ServerSettings{
					ServerURL: "https://example.org",
				},
				FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: tt.initialURL},
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
				*dsAppConfig = *conf
				return nil
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

			ac, err := svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.initialURL, ac.FleetDesktop.TransparencyURL)

			raw, err := json.Marshal(fleet.FleetDesktopSettings{TransparencyURL: tt.newURL})
			require.NoError(t, err)
			raw = []byte(`{"fleet_desktop":` + string(raw) + `}`)
			modified, err := svc.ModifyAppConfig(ctx, raw, fleet.ApplySpecOptions{})
			checkLicenseErr(t, tt.shouldFailModify, err)

			if modified != nil {
				require.Equal(t, tt.expectedURL, modified.FleetDesktop.TransparencyURL)
				ac, err = svc.AppConfigObfuscated(ctx)
				require.NoError(t, err)
				require.Equal(t, tt.expectedURL, ac.FleetDesktop.TransparencyURL)
			}
		})
	}
}

// TestTransparencyURLDowngradeLicense tests scenarios where a transparency url value has previously
// been stored (for example, if a licensee downgraded without manually resetting the transparency url)
func TestTransparencyURLDowngradeLicense(t *testing.T) {
	ds := new(mock.Store)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}

	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: "free"}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	dsAppConfig := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://example.org",
		},
		FleetDesktop: fleet.FleetDesktopSettings{TransparencyURL: "https://example.com/transparency"},
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return dsAppConfig, nil
	}

	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		*dsAppConfig = *conf
		return nil
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

	ac, err := svc.AppConfigObfuscated(ctx)
	require.NoError(t, err)
	require.Equal(t, "https://example.com/transparency", ac.FleetDesktop.TransparencyURL)

	// setting transparency url fails
	raw, err := json.Marshal(fleet.FleetDesktopSettings{TransparencyURL: "https://f1337.com/transparency"})
	require.NoError(t, err)
	raw = []byte(`{"fleet_desktop":` + string(raw) + `}`)
	_, err = svc.ModifyAppConfig(ctx, raw, fleet.ApplySpecOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "missing or invalid license")

	// setting unrelated config value does not fail and resets transparency url to ""
	raw, err = json.Marshal(fleet.OrgInfo{OrgName: "f1337"})
	require.NoError(t, err)
	raw = []byte(`{"org_info":` + string(raw) + `}`)
	modified, err := svc.ModifyAppConfig(ctx, raw, fleet.ApplySpecOptions{})
	require.NoError(t, err)
	require.NotNil(t, modified)
	require.Equal(t, "", modified.FleetDesktop.TransparencyURL)
	ac, err = svc.AppConfigObfuscated(ctx)
	require.NoError(t, err)
	require.Equal(t, "f1337", ac.OrgInfo.OrgName)
	require.Equal(t, "", ac.FleetDesktop.TransparencyURL)
}

func TestMDMAppleConfig(t *testing.T) {
	ds := new(mock.Store)
	depStorage := new(nanodep_mock.Storage)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}

	depSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
		case "/profile":
			_, _ = w.Write([]byte(`{"profile_uuid": "xyz"}`))
		}
	}))
	t.Cleanup(depSrv.Close)

	const licenseErr = "missing or invalid license"
	const notFoundErr = "not found"
	testCases := []struct {
		name          string
		licenseTier   string
		oldMDM        fleet.MDM
		newMDM        fleet.MDM
		expectedMDM   fleet.MDM
		expectedError string
		findTeam      bool
	}{
		{
			name:        "nochange",
			licenseTier: "free",
			expectedMDM: fleet.MDM{
				AppleBusinessManager: optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{Set: true, Value: []fleet.MDMAppleABMAssignmentInfo{}},
				MacOSSetup: fleet.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []fleet.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
				},
			},
		}, {
			name:          "newDefaultTeamNoLicense",
			licenseTier:   "free",
			newMDM:        fleet.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedError: licenseErr,
		}, {
			name:          "notFoundNew",
			licenseTier:   "premium",
			newMDM:        fleet.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedError: notFoundErr,
		}, {
			name:          "notFoundEdit",
			licenseTier:   "premium",
			oldMDM:        fleet.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			newMDM:        fleet.MDM{DeprecatedAppleBMDefaultTeam: "bar"},
			expectedError: notFoundErr,
		}, {
			name:        "foundNew",
			licenseTier: "premium",
			findTeam:    true,
			newMDM:      fleet.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedMDM: fleet.MDM{
				AppleBusinessManager:         optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{Set: true, Value: []fleet.MDMAppleABMAssignmentInfo{}},
				DeprecatedAppleBMDefaultTeam: "foobar",
				MacOSSetup: fleet.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []fleet.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
				},
			},
		}, {
			name:        "foundEdit",
			licenseTier: "premium",
			findTeam:    true,
			oldMDM:      fleet.MDM{DeprecatedAppleBMDefaultTeam: "bar"},
			newMDM:      fleet.MDM{DeprecatedAppleBMDefaultTeam: "foobar"},
			expectedMDM: fleet.MDM{
				AppleBusinessManager:         optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{Set: true, Value: []fleet.MDMAppleABMAssignmentInfo{}},
				DeprecatedAppleBMDefaultTeam: "foobar",
				MacOSSetup: fleet.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []fleet.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
				},
			},
		}, {
			name:          "ssoFree",
			licenseTier:   "free",
			findTeam:      true,
			newMDM:        fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{EntityID: "foo"}}},
			expectedError: licenseErr,
		}, {
			name:        "ssoFreeNoChanges",
			licenseTier: "free",
			findTeam:    true,
			newMDM:      fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{EntityID: "foo"}}},
			oldMDM:      fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{EntityID: "foo"}}},
			expectedMDM: fleet.MDM{
				AppleBusinessManager:  optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{Set: true, Value: []fleet.MDMAppleABMAssignmentInfo{}},
				EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{EntityID: "foo"}},
				MacOSSetup: fleet.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []fleet.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
				},
			},
		}, {
			name:        "ssoAllFields",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:    "fleet",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				IDPName:     "onelogin",
			}}},
			expectedMDM: fleet.MDM{
				AppleBusinessManager: optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{Set: true, Value: []fleet.MDMAppleABMAssignmentInfo{}},
				EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{
					EntityID:    "fleet",
					IssuerURI:   "http://issuer.idp.com",
					MetadataURL: "http://isser.metadata.com",
					IDPName:     "onelogin",
				}},
				MacOSSetup: fleet.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []fleet.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
				},
			},
		}, {
			name:        "ssoShortEntityID",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:    "f",
				IssuerURI:   "http://issuer.idp.com",
				MetadataURL: "http://isser.metadata.com",
				IDPName:     "onelogin",
			}}},
			expectedError: "validation failed: entity_id must be 5 or more characters",
		}, {
			name:        "ssoMissingMetadata",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:  "fleet",
				IssuerURI: "http://issuer.idp.com",
				IDPName:   "onelogin",
			}}},
			expectedError: "either metadata or metadata_url must be defined",
		}, {
			name:        "ssoMultiMetadata",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:    "fleet",
				IssuerURI:   "http://issuer.idp.com",
				Metadata:    "not-empty",
				MetadataURL: "not-empty",
				IDPName:     "onelogin",
			}}},
			expectedError: "metadata both metadata and metadata_url are defined, only one is allowed",
		}, {
			name:        "ssoIdPName",
			licenseTier: "premium",
			findTeam:    true,
			newMDM: fleet.MDM{EndUserAuthentication: fleet.MDMEndUserAuthentication{SSOProviderSettings: fleet.SSOProviderSettings{
				EntityID:  "fleet",
				IssuerURI: "http://issuer.idp.com",
				Metadata:  "not-empty",
			}}},
			expectedError: "idp_name required",
		}, {
			name:        "disableDiskEncryption",
			licenseTier: "premium",
			newMDM: fleet.MDM{
				EnableDiskEncryption: optjson.SetBool(false),
			},
			expectedMDM: fleet.MDM{
				AppleBusinessManager: optjson.Slice[fleet.MDMAppleABMAssignmentInfo]{Set: true, Value: []fleet.MDMAppleABMAssignmentInfo{}},
				EnableDiskEncryption: optjson.Bool{Set: true, Valid: true, Value: false},
				MacOSSetup: fleet.MacOSSetup{
					BootstrapPackage:            optjson.String{Set: true},
					MacOSSetupAssistant:         optjson.String{Set: true},
					EnableReleaseDeviceManually: optjson.SetBool(false),
					Software:                    optjson.Slice[*fleet.MacOSSetupSoftware]{Set: true, Value: []*fleet.MacOSSetupSoftware{}},
					Script:                      optjson.String{Set: true},
				},
				MacOSUpdates:            fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IOSUpdates:              fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				IPadOSUpdates:           fleet.AppleOSUpdateSettings{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}},
				VolumePurchasingProgram: optjson.Slice[fleet.MDMAppleVolumePurchasingProgramInfo]{Set: true, Value: []fleet.MDMAppleVolumePurchasingProgramInfo{}},
				WindowsUpdates:          fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}},
				WindowsSettings: fleet.WindowsSettings{
					CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
				},
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tt.licenseTier}, DEPStorage: depStorage})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

			dsAppConfig := &fleet.AppConfig{
				OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
				ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"},
				MDM:            tt.oldMDM,
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
				*dsAppConfig = *conf
				return nil
			}
			ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
				if tt.findTeam {
					return &fleet.Team{}, nil
				}
				return nil, sql.ErrNoRows
			}
			ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
				return &fleet.MDMAppleEnrollmentProfile{}, nil
			}
			ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ fleet.MDMAppleEnrollmentType) (*fleet.MDMAppleEnrollmentProfile, error) {
				raw := json.RawMessage("{}")
				return &fleet.MDMAppleEnrollmentProfile{DEPProfile: &raw}, nil
			}
			ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
				return job, nil
			}
			ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
				return []*fleet.ABMToken{{ID: 1}}, nil
			}
			ds.SaveABMTokenFunc = func(ctx context.Context, token *fleet.ABMToken) error {
				return nil
			}
			depStorage.RetrieveConfigFunc = func(p0 context.Context, p1 string) (*nanodep_client.Config, error) {
				return &nanodep_client.Config{BaseURL: depSrv.URL}, nil
			}
			depStorage.RetrieveAuthTokensFunc = func(ctx context.Context, name string) (*nanodep_client.OAuth1Tokens, error) {
				return &nanodep_client.OAuth1Tokens{}, nil
			}
			depStorage.StoreAssignerProfileFunc = func(ctx context.Context, name string, profileUUID string) error {
				return nil
			}
			ds.SaveABMTokenFunc = func(ctx context.Context, tok *fleet.ABMToken) error {
				return nil
			}
			ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
				return []*fleet.VPPTokenDB{}, nil
			}
			ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
				return []*fleet.ABMToken{{OrganizationName: t.Name()}}, nil
			}

			ac, err := svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.oldMDM, ac.MDM)

			raw, err := json.Marshal(tt.newMDM)
			require.NoError(t, err)
			raw = []byte(`{"mdm":` + string(raw) + `}`)
			modified, err := svc.ModifyAppConfig(ctx, raw, fleet.ApplySpecOptions{})
			if tt.expectedError != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedMDM, modified.MDM)
			ac, err = svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.expectedMDM, ac.MDM)
		})
	}
}

func TestDiskEncryptionSetting(t *testing.T) {
	ds := new(mock.Store)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	t.Run("enableDiskEncryptionWithNoPrivateKey", func(t *testing.T) {
		testConfig = config.TestConfig()
		testConfig.Server.PrivateKey = ""
		svc, ctx := newTestServiceWithConfig(t, ds, testConfig, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})
		ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

		dsAppConfig := &fleet.AppConfig{
			OrgInfo:        fleet.OrgInfo{OrgName: "Test"},
			ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"},
			MDM:            fleet.MDM{},
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return dsAppConfig, nil
		}

		ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			*dsAppConfig = *conf
			return nil
		}
		ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
			return nil, sql.ErrNoRows
		}
		ds.NewMDMAppleEnrollmentProfileFunc = func(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
			return &fleet.MDMAppleEnrollmentProfile{}, nil
		}
		ds.GetMDMAppleEnrollmentProfileByTypeFunc = func(ctx context.Context, typ fleet.MDMAppleEnrollmentType) (*fleet.MDMAppleEnrollmentProfile, error) {
			raw := json.RawMessage("{}")
			return &fleet.MDMAppleEnrollmentProfile{DEPProfile: &raw}, nil
		}
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}

		ac, err := svc.AppConfigObfuscated(ctx)
		require.NoError(t, err)
		require.Equal(t, dsAppConfig.MDM, ac.MDM)

		raw, err := json.Marshal(fleet.MDM{
			EnableDiskEncryption: optjson.SetBool(true),
		})
		require.NoError(t, err)
		raw = []byte(`{"mdm":` + string(raw) + `}`)
		_, err = svc.ModifyAppConfig(ctx, raw, fleet.ApplySpecOptions{})
		require.Error(t, err)
		require.ErrorContains(t, err, "Missing required private key")
	})
}

func TestModifyAppConfigSMTPSSOAgentOptions(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	// SMTP and SSO are initially set.
	agentOptions := json.RawMessage(`
{
  "config": {
      "options": {
        "distributed_interval": 10
      }
  },
  "overrides": {
    "platforms": {
      "darwin": {
        "options": {
          "distributed_interval": 5
        }
      }
    }
  }
}`)
	dsAppConfig := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://example.org",
		},
		SMTPSettings: &fleet.SMTPSettings{
			SMTPEnabled:       true,
			SMTPConfigured:    true,
			SMTPSenderAddress: "foobar@example.com",
		},
		SSOSettings: &fleet.SSOSettings{
			EnableSSO: true,
			SSOProviderSettings: fleet.SSOProviderSettings{
				MetadataURL: "foobar.example.com/metadata",
			},
		},
		AgentOptions: &agentOptions,
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return dsAppConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		*dsAppConfig = *conf
		return nil
	}
	ds.NewActivityFunc = func(
		ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
	) error {
		return nil
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

	// Not sending smtp_settings, sso_settings or agent_settings will do nothing.
	b := []byte(`{}`)
	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	updatedAppConfig, err := svc.ModifyAppConfig(ctx, b, fleet.ApplySpecOptions{})
	require.NoError(t, err)

	require.True(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.True(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.True(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.True(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, agentOptions, *updatedAppConfig.AgentOptions)
	require.Equal(t, agentOptions, *dsAppConfig.AgentOptions)

	// Not sending sso_settings or agent settings will not change them, and
	// sending SMTP settings will change them.
	b = []byte(`{"smtp_settings": {"enable_smtp": false}}`)
	updatedAppConfig, err = svc.ModifyAppConfig(ctx, b, fleet.ApplySpecOptions{})
	require.NoError(t, err)

	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.True(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.True(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, agentOptions, *updatedAppConfig.AgentOptions)
	require.Equal(t, agentOptions, *dsAppConfig.AgentOptions)

	// Not sending smtp_settings or agent settings will not change them, and
	// sending SSO settings will change them.
	b = []byte(`{"sso_settings": {"enable_sso": false}}`)
	updatedAppConfig, err = svc.ModifyAppConfig(ctx, b, fleet.ApplySpecOptions{})
	require.NoError(t, err)

	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.False(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, agentOptions, *updatedAppConfig.AgentOptions)
	require.Equal(t, agentOptions, *dsAppConfig.AgentOptions)

	// Not sending smtp_settings or sso_settings will not change them, and
	// sending agent options will change them.
	newAgentOptions := json.RawMessage(`{
  "config": {
      "options": {
        "distributed_interval": 100
      }
  },
  "overrides": {
    "platforms": {
      "darwin": {
        "options": {
          "distributed_interval": 2
        }
      }
    }
  }
}`)
	b = []byte(`{"agent_options": ` + string(newAgentOptions) + `}`)
	updatedAppConfig, err = svc.ModifyAppConfig(ctx, b, fleet.ApplySpecOptions{})
	require.NoError(t, err)

	require.False(t, updatedAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, dsAppConfig.SMTPSettings.SMTPEnabled)
	require.False(t, updatedAppConfig.SSOSettings.EnableSSO)
	require.False(t, dsAppConfig.SSOSettings.EnableSSO)
	require.Equal(t, newAgentOptions, *dsAppConfig.AgentOptions)
	require.Equal(t, newAgentOptions, *dsAppConfig.AgentOptions)
}

// TestModifyEnableAnalytics tests that a premium customer cannot set ServerSettings.EnableAnalytics to be false.
// Free customers should be able to set the value to false, however.
func TestModifyEnableAnalytics(t *testing.T) {
	ds := new(mock.Store)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}

	testCases := []struct {
		name             string
		expectedEnabled  bool
		newEnabled       bool
		initialEnabled   bool
		licenseTier      string
		initialURL       string
		newURL           string
		expectedURL      string
		shouldFailModify bool
	}{
		{
			name:            "fleet free",
			expectedEnabled: false,
			initialEnabled:  true,
			newEnabled:      false,
			licenseTier:     fleet.TierFree,
		},
		{
			name:            "fleet premium",
			expectedEnabled: true,
			initialEnabled:  true,
			newEnabled:      false,
			licenseTier:     fleet.TierPremium,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tt.licenseTier}})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

			dsAppConfig := &fleet.AppConfig{
				OrgInfo: fleet.OrgInfo{
					OrgName: "Test",
				},
				ServerSettings: fleet.ServerSettings{
					EnableAnalytics: true,
					ServerURL:       "https://localhost:8080",
				},
			}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return dsAppConfig, nil
			}

			ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
				*dsAppConfig = *conf
				return nil
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

			ac, err := svc.AppConfigObfuscated(ctx)
			require.NoError(t, err)
			require.Equal(t, tt.initialEnabled, ac.ServerSettings.EnableAnalytics)

			raw, err := json.Marshal(fleet.ServerSettings{EnableAnalytics: tt.newEnabled, ServerURL: "https://localhost:8080"})
			require.NoError(t, err)
			raw = []byte(`{"server_settings":` + string(raw) + `}`)
			modified, err := svc.ModifyAppConfig(ctx, raw, fleet.ApplySpecOptions{})
			require.NoError(t, err)

			if modified != nil {
				require.Equal(t, tt.expectedEnabled, modified.ServerSettings.EnableAnalytics)
				ac, err = svc.AppConfigObfuscated(ctx)
				require.NoError(t, err)
				require.Equal(t, tt.expectedEnabled, ac.ServerSettings.EnableAnalytics)
			}
		})
	}
}

func TestModifyAppConfigForNDESSCEPProxy(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierFree}})
	scepURL := "https://example.com/mscep/mscep.dll"
	adminURL := "https://example.com/mscep_admin/"
	username := "user"
	password := "password"

	appConfig := &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{
			OrgName: "Test",
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "https://localhost:8080",
		},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		if appConfig.Integrations.NDESSCEPProxy.Valid {
			appConfig.Integrations.NDESSCEPProxy.Value.Password = fleet.MaskedPassword
		}
		return appConfig, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		appConfig = conf
		return nil
	}
	ds.ListABMTokensFunc = func(ctx context.Context) ([]*fleet.ABMToken, error) {
		return []*fleet.ABMToken{{ID: 1}}, nil
	}
	ds.SaveABMTokenFunc = func(ctx context.Context, token *fleet.ABMToken) error {
		return nil
	}
	ds.ListVPPTokensFunc = func(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
		return []*fleet.VPPTokenDB{}, nil
	}

	jsonPayloadBase := `
{
	"integrations": {
		"ndes_scep_proxy": {
			"url": "%s",
			"admin_url": "%s",
			"username": "%s",
			"password": "%s"
		}
	}
}
`
	jsonPayload := fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, password)
	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})

	// SCEP proxy not configured for free users
	_, err := svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	assert.ErrorContains(t, err, ErrMissingLicense.Error())
	assert.ErrorContains(t, err, "integrations.ndes_scep_proxy")

	origValidateNDESSCEPURL := validateNDESSCEPURL
	origValidateNDESSCEPAdminURL := validateNDESSCEPAdminURL
	t.Cleanup(func() {
		validateNDESSCEPURL = origValidateNDESSCEPURL
		validateNDESSCEPAdminURL = origValidateNDESSCEPAdminURL
	})
	validateNDESSCEPURLCalled := false
	validateNDESSCEPURL = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration, _ log.Logger) error {
		validateNDESSCEPURLCalled = true
		return nil
	}
	validateNDESSCEPAdminURLCalled := false
	validateNDESSCEPAdminURL = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration) error {
		validateNDESSCEPAdminURLCalled = true
		return nil
	}

	fleetConfig := config.TestConfig()
	svc, ctx = newTestServiceWithConfig(t, ds, fleetConfig, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte,
		createdAt time.Time,
	) error {
		assert.IsType(t, fleet.ActivityAddedNDESSCEPProxy{}, activity)
		return nil
	}
	ac, err := svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy := func() {
		require.NotNil(t, ac.Integrations.NDESSCEPProxy)
		assert.Equal(t, scepURL, ac.Integrations.NDESSCEPProxy.Value.URL)
		assert.Equal(t, adminURL, ac.Integrations.NDESSCEPProxy.Value.AdminURL)
		assert.Equal(t, username, ac.Integrations.NDESSCEPProxy.Value.Username)
		assert.Equal(t, fleet.MaskedPassword, ac.Integrations.NDESSCEPProxy.Value.Password)
	}
	checkSCEPProxy()
	assert.True(t, validateNDESSCEPURLCalled)
	assert.True(t, validateNDESSCEPAdminURLCalled)
	assert.True(t, ds.SaveAppConfigFuncInvoked)
	ds.SaveAppConfigFuncInvoked = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation not done if there is no change
	appConfig = ac
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	jsonPayload = fmt.Sprintf(jsonPayloadBase, " "+scepURL, adminURL+" ", " "+username+" ", fleet.MaskedPassword)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	require.NoError(t, err, jsonPayload)
	checkSCEPProxy()
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation not done if there is no change, part 2
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	ac, err = svc.ModifyAppConfig(ctx, []byte(`{"integrations":{}}`), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation done for SCEP URL. Password is blank, which is not considered a change.
	scepURL = "https://new.com/mscep/mscep.dll"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, "")
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte,
		createdAt time.Time,
	) error {
		assert.IsType(t, fleet.ActivityEditedNDESSCEPProxy{}, activity)
		return nil
	}
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.True(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	appConfig = ac
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation done for SCEP admin URL
	adminURL = "https://new.com/mscep_admin/"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, fleet.MaskedPassword)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.False(t, validateNDESSCEPURLCalled)
	assert.True(t, validateNDESSCEPAdminURLCalled)
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation fails
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	validateNDESSCEPURL = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration, _ log.Logger) error {
		validateNDESSCEPURLCalled = true
		return errors.New("**invalid** 1")
	}
	validateNDESSCEPAdminURL = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration) error {
		validateNDESSCEPAdminURLCalled = true
		return errors.New("**invalid** 2")
	}
	scepURL = "https://new2.com/mscep/mscep.dll"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, password)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	assert.ErrorContains(t, err, "**invalid**")
	assert.True(t, validateNDESSCEPURLCalled)
	assert.True(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Reset validation
	validateNDESSCEPURLCalled = false
	validateNDESSCEPURL = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration, _ log.Logger) error {
		validateNDESSCEPURLCalled = true
		return nil
	}
	validateNDESSCEPAdminURLCalled = false
	validateNDESSCEPAdminURL = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration) error {
		validateNDESSCEPAdminURLCalled = true
		return nil
	}

	// Config cleared with explicit null
	validateNDESSCEPURLCalled = false
	validateNDESSCEPAdminURLCalled = false
	payload := `
{
	"integrations": {
		"ndes_scep_proxy": null
	}
}
`
	// First, dry run.
	appConfig.Integrations.NDESSCEPProxy.Valid = true
	ac, err = svc.ModifyAppConfig(ctx, []byte(payload), fleet.ApplySpecOptions{DryRun: true})
	require.NoError(t, err)
	assert.False(t, ac.Integrations.NDESSCEPProxy.Valid)
	// Also check what was saved.
	assert.False(t, appConfig.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.HardDeleteMDMConfigAssetFuncInvoked, "DB write should not happen in dry run")
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Second, real run.
	appConfig.Integrations.NDESSCEPProxy.Valid = true
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte,
		createdAt time.Time,
	) error {
		assert.IsType(t, fleet.ActivityDeletedNDESSCEPProxy{}, activity)
		return nil
	}
	ds.HardDeleteMDMConfigAssetFunc = func(ctx context.Context, assetName fleet.MDMAssetName) error {
		return nil
	}
	ac, err = svc.ModifyAppConfig(ctx, []byte(payload), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	assert.False(t, ac.Integrations.NDESSCEPProxy.Valid)
	// Also check what was saved.
	assert.False(t, appConfig.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.True(t, ds.HardDeleteMDMConfigAssetFuncInvoked)
	ds.HardDeleteMDMConfigAssetFuncInvoked = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Deleting again should be a no-op
	appConfig.Integrations.NDESSCEPProxy.Valid = false
	ac, err = svc.ModifyAppConfig(ctx, []byte(payload), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	assert.False(t, ac.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, appConfig.Integrations.NDESSCEPProxy.Valid)
	assert.False(t, validateNDESSCEPURLCalled)
	assert.False(t, validateNDESSCEPAdminURLCalled)
	assert.False(t, ds.HardDeleteMDMConfigAssetFuncInvoked)
	ds.HardDeleteMDMConfigAssetFuncInvoked = false
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Cannot configure NDES without private key
	fleetConfig.Server.PrivateKey = ""
	svc, ctx = newTestServiceWithConfig(t, ds, fleetConfig, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	_, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	assert.ErrorContains(t, err, "private key")
}
