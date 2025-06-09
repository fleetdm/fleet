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
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mock"
	nanodep_mock "github.com/fleetdm/fleet/v4/server/mock/nanodep"
	scep_mock "github.com/fleetdm/fleet/v4/server/mock/scep"
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
			false,
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
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, isNew bool, teamID *uint) (bool, error) {
		assert.False(t, isNew)
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
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, isNew bool, teamID *uint) (bool, error) {
		assert.False(t, isNew)
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
	ds.IsEnrollSecretAvailableFunc = func(ctx context.Context, secret string, isNew bool, teamID *uint) (bool, error) {
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
			false,
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

			expectedURL := fleet.DefaultTransparencyURL
			expectedSecureframeURL := fleet.SecureframeTransparencyURL
			if tt.expectedURL != "" {
				expectedURL = tt.expectedURL
				expectedSecureframeURL = tt.expectedURL
			}

			transparencyURL, err := svc.GetTransparencyURL(ctx)
			require.NoError(t, err)
			require.Equal(t, expectedURL, transparencyURL)

			cfg := config.TestConfig()
			cfg.Partnerships.EnableSecureframe = true
			svc, ctx = newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tt.licenseTier}})
			ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
			transparencyURL, err = svc.GetTransparencyURL(ctx)
			require.NoError(t, err)
			require.Equal(t, expectedSecureframeURL, transparencyURL)
		})
	}
}

// TestTransparencyURLDowngradeLicense tests scenarios where a transparency url value has previously
// been stored (for example, if a licensee downgraded without manually resetting the transparency url)
func TestTransparencyURLDowngradeLicense(t *testing.T) {
	ds := new(mock.Store)

	admin := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}

	cfg := config.TestConfig()
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: "free"}})
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

	// delivered URL should be the default one
	transparencyUrl, err := svc.GetTransparencyURL(ctx)
	require.NoError(t, err)
	require.Equal(t, fleet.DefaultTransparencyURL, transparencyUrl)

	// delivered URL should be the Secureframe one if we have that config value set
	cfg.Partnerships.EnableSecureframe = true
	svc, ctx = newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: "free"}})
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: admin})
	transparencyUrl, err = svc.GetTransparencyURL(ctx)
	require.NoError(t, err)
	require.Equal(t, fleet.SecureframeTransparencyURL, transparencyUrl)

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
					ManualAgentInstall:          optjson.Bool{Set: true},
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
					ManualAgentInstall:          optjson.Bool{Set: true},
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
					ManualAgentInstall:          optjson.Bool{Set: true},
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
					ManualAgentInstall:          optjson.Bool{Set: true},
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
					ManualAgentInstall:          optjson.Bool{Set: true},
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
			expectedError: "invalid URI for request",
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
					ManualAgentInstall:          optjson.Bool{Set: true},
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
		name                  string
		expectedEnabled       bool
		newEnabled            bool
		initialEnabled        bool
		licenseTier           string
		allowDisableTelemetry bool
		initialURL            string
		newURL                string
		expectedURL           string
		shouldFailModify      bool
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
		{
			name:                  "fleet premium with allow disable telemetry",
			expectedEnabled:       false,
			initialEnabled:        true,
			newEnabled:            false,
			licenseTier:           fleet.TierPremium,
			allowDisableTelemetry: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tt.licenseTier, AllowDisableTelemetry: tt.allowDisableTelemetry}})
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
	t.Parallel()
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

	fleetConfig := config.TestConfig()
	scepConfig := &scep_mock.SCEPConfigService{}
	scepConfig.ValidateSCEPURLFunc = func(_ context.Context, _ string) error { return nil }
	scepConfig.ValidateNDESSCEPAdminURLFunc = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration) error { return nil }
	svc, ctx = newTestServiceWithConfig(t, ds, fleetConfig, nil, nil, &TestServerOpts{
		License:           &fleet.LicenseInfo{Tier: fleet.TierPremium},
		SCEPConfigService: scepConfig,
	})
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
	assert.True(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.True(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
	assert.True(t, ds.SaveAppConfigFuncInvoked)
	ds.SaveAppConfigFuncInvoked = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation not done if there is no change
	appConfig = ac
	scepConfig.ValidateSCEPURLFuncInvoked = false
	scepConfig.ValidateNDESSCEPAdminURLFuncInvoked = false
	jsonPayload = fmt.Sprintf(jsonPayloadBase, " "+scepURL, adminURL+" ", " "+username+" ", fleet.MaskedPassword)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	require.NoError(t, err, jsonPayload)
	checkSCEPProxy()
	assert.False(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.False(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation not done if there is no change, part 2
	scepConfig.ValidateSCEPURLFuncInvoked = false
	scepConfig.ValidateNDESSCEPAdminURLFuncInvoked = false
	ac, err = svc.ModifyAppConfig(ctx, []byte(`{"integrations":{}}`), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.False(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.False(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
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
	assert.True(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.False(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
	appConfig = ac
	scepConfig.ValidateSCEPURLFuncInvoked = false
	scepConfig.ValidateNDESSCEPAdminURLFuncInvoked = false
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation done for SCEP admin URL
	adminURL = "https://new.com/mscep_admin/"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, fleet.MaskedPassword)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	require.NoError(t, err)
	checkSCEPProxy()
	assert.False(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.True(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
	assert.True(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Validation fails
	scepConfig.ValidateSCEPURLFuncInvoked = false
	scepConfig.ValidateNDESSCEPAdminURLFuncInvoked = false
	scepConfig.ValidateSCEPURLFunc = func(_ context.Context, _ string) error {
		return errors.New("**invalid** 1")
	}
	scepConfig.ValidateNDESSCEPAdminURLFunc = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration) error {
		return errors.New("**invalid** 2")
	}
	scepURL = "https://new2.com/mscep/mscep.dll"
	jsonPayload = fmt.Sprintf(jsonPayloadBase, scepURL, adminURL, username, password)
	ac, err = svc.ModifyAppConfig(ctx, []byte(jsonPayload), fleet.ApplySpecOptions{})
	assert.ErrorContains(t, err, "**invalid**")
	assert.True(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.True(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
	assert.False(t, ds.NewActivityFuncInvoked)
	ds.NewActivityFuncInvoked = false

	// Reset validation
	scepConfig.ValidateSCEPURLFuncInvoked = false
	scepConfig.ValidateNDESSCEPAdminURLFuncInvoked = false
	scepConfig.ValidateSCEPURLFunc = func(_ context.Context, _ string) error { return nil }
	scepConfig.ValidateNDESSCEPAdminURLFunc = func(_ context.Context, _ fleet.NDESSCEPProxyIntegration) error { return nil }

	// Config cleared with explicit null
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
	assert.False(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.False(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
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
	assert.False(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.False(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
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
	assert.False(t, scepConfig.ValidateSCEPURLFuncInvoked)
	assert.False(t, scepConfig.ValidateNDESSCEPAdminURLFuncInvoked)
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

func TestAppConfigCAs(t *testing.T) {
	t.Parallel()

	pathRegex := regexp.MustCompile(`^/mpki/api/v2/profile/([a-zA-Z0-9_-]+)$`)
	mockDigiCertServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		matches := pathRegex.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		profileID := matches[1]

		resp := map[string]string{
			"id":     profileID,
			"name":   "Test CA",
			"status": "Active",
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		require.NoError(t, err)
	}))
	defer mockDigiCertServer.Close()

	setUpDigiCert := func() configCASuite {
		mt := configCASuite{
			ctx:          license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium}),
			invalid:      &fleet.InvalidArgumentError{},
			newAppConfig: getAppConfigWithDigiCertIntegration(mockDigiCertServer.URL, "WIFI"),
			oldAppConfig: &fleet.AppConfig{},
			appConfig:    &fleet.AppConfig{},
			svc:          &Service{logger: log.NewLogfmtLogger(os.Stdout)},
		}
		mt.svc.config.Server.PrivateKey = "exists"
		mt.svc.digiCertService = digicert.NewService()
		addMockDatastoreForCA(t, mt)
		return mt
	}
	setUpCustomSCEP := func() configCASuite {
		mt := configCASuite{
			ctx:          license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium}),
			invalid:      &fleet.InvalidArgumentError{},
			newAppConfig: getAppConfigWithSCEPIntegration("https://example.com", "SCEP_WIFI"),
			oldAppConfig: &fleet.AppConfig{},
			appConfig:    &fleet.AppConfig{},
			svc:          &Service{logger: log.NewLogfmtLogger(os.Stdout)},
		}
		mt.svc.config.Server.PrivateKey = "exists"
		scepConfig := &scep_mock.SCEPConfigService{}
		scepConfig.ValidateSCEPURLFunc = func(_ context.Context, _ string) error { return nil }
		mt.svc.scepConfigService = scepConfig
		addMockDatastoreForCA(t, mt)
		return mt
	}

	t.Run("free license", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.ctx = license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierFree})
		mt.newAppConfig = &fleet.AppConfig{}
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.ndes)
		assert.Empty(t, status.digicert)
		assert.Empty(t, status.customSCEPProxy)

		mt.invalid = &fleet.InvalidArgumentError{}
		mt.newAppConfig = &fleet.AppConfig{}
		mt.newAppConfig.Integrations.DigiCert.Set = true
		mt.newAppConfig.Integrations.DigiCert.Valid = true
		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "digicert", ErrMissingLicense.Error())

		mt.invalid = &fleet.InvalidArgumentError{}
		mt.newAppConfig = &fleet.AppConfig{}
		mt.newAppConfig.Integrations.CustomSCEPProxy.Set = true
		mt.newAppConfig.Integrations.CustomSCEPProxy.Valid = true
		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "custom_scep_proxy", ErrMissingLicense.Error())
	})

	t.Run("digicert keep old value", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.ctx = license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
		mt.oldAppConfig = mt.newAppConfig
		mt.appConfig = mt.oldAppConfig.Copy()
		mt.newAppConfig = &fleet.AppConfig{}
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.ndes)
		assert.Empty(t, status.digicert)
		assert.Empty(t, status.customSCEPProxy)
		assert.Len(t, mt.appConfig.Integrations.DigiCert.Value, 1)
	})

	t.Run("custom_scep keep old value", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.ctx = license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
		mt.oldAppConfig = mt.newAppConfig
		mt.appConfig = mt.oldAppConfig.Copy()
		mt.newAppConfig = &fleet.AppConfig{}
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.ndes)
		assert.Empty(t, status.digicert)
		assert.Empty(t, status.customSCEPProxy)
		assert.Len(t, mt.appConfig.Integrations.CustomSCEPProxy.Value, 1)
	})

	t.Run("missing server private key", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.svc.config.Server.PrivateKey = ""
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert", "private key")

		mt = setUpCustomSCEP()
		mt.svc.config.Server.PrivateKey = ""
		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy", "private key")
	})

	t.Run("invalid integration name", func(t *testing.T) {
		testCases := []struct {
			testName      string
			name          string
			errorContains []string
		}{
			{
				testName:      "empty",
				name:          "",
				errorContains: []string{"CA name cannot be empty"},
			},
			{
				testName:      "NDES",
				name:          "NDES",
				errorContains: []string{"CA name cannot be NDES"},
			},
			{
				testName:      "too long",
				name:          strings.Repeat("a", 256),
				errorContains: []string{"CA name cannot be longer than"},
			},
			{
				testName:      "invalid characters",
				name:          "a/b",
				errorContains: []string{"Only letters, numbers and underscores allowed"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.testName, func(t *testing.T) {
				baseErrorContains := tc.errorContains
				mt := setUpDigiCert()
				mt.newAppConfig = getAppConfigWithDigiCertIntegration(mockDigiCertServer.URL, tc.name)
				status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
				require.NoError(t, err)
				errorContains := baseErrorContains
				errorContains = append(errorContains, "integrations.digicert.name")
				checkExpectedCAValidationError(t, mt.invalid, status, errorContains...)

				mt = setUpCustomSCEP()
				mt.newAppConfig = getAppConfigWithSCEPIntegration("https://example.com", tc.name)
				status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
				require.NoError(t, err)
				errorContains = baseErrorContains
				errorContains = append(errorContains, "integrations.custom_scep_proxy.name")
				checkExpectedCAValidationError(t, mt.invalid, status, errorContains...)
			})
		}
	})

	t.Run("invalid digicert URL", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].URL = ""
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.url",
			"empty url")

		mt = setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].URL = "nonhttp://bad.com"
		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.url",
			"URL must be https or http")
	})

	t.Run("invalid custom_scep URL", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].URL = ""
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.url",
			"empty url")

		mt = setUpCustomSCEP()
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].URL = "nonhttp://bad.com"
		status, err = mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.url",
			"URL must be https or http")
	})

	t.Run("duplicate digicert integration name", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value = append(mt.newAppConfig.Integrations.DigiCert.Value,
			mt.newAppConfig.Integrations.DigiCert.Value[0])
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.name",
			"name is already used by another certificate authority")
	})

	t.Run("duplicate custom_scep integration name", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value = append(mt.newAppConfig.Integrations.CustomSCEPProxy.Value,
			mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0])
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.name",
			"name is already used by another certificate authority")
	})

	t.Run("same digicert and custom_scep integration name", func(t *testing.T) {
		mtSCEP := setUpCustomSCEP()
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.CustomSCEPProxy = mtSCEP.newAppConfig.Integrations.CustomSCEPProxy
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Name = mt.newAppConfig.Integrations.DigiCert.Value[0].Name
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.name",
			"name is already used by another certificate authority")
	})

	t.Run("digicert more than 1 user principal name", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames = append(mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames,
			"another")
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_user_principal_names",
			"one certificate user principal name")
	})

	t.Run("digicert empty user principal name", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames = []string{" "}
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_user_principal_names",
			"user principal name cannot be empty")
	})

	t.Run("digicert Fleet vars in user principal name", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames[0] = "$FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP + " ${FLEET_VAR_" + fleet.FleetVarHostHardwareSerial + "}"
		_, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)

		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateUserPrincipalNames[0] = "$FLEET_VAR_BOZO"
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_user_principal_names",
			"FLEET_VAR_BOZO is not allowed")
	})

	t.Run("digicert Fleet vars in common name", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateCommonName = "${FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP + "}${FLEET_VAR_" + fleet.FleetVarHostHardwareSerial + "}"
		_, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)

		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateCommonName = "$FLEET_VAR_BOZO"
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_common_name",
			"FLEET_VAR_BOZO is not allowed")
	})

	t.Run("digicert Fleet vars in seat id", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateSeatID = "$FLEET_VAR_" + fleet.FleetVarHostEndUserEmailIDP + " $FLEET_VAR_" + fleet.FleetVarHostHardwareSerial
		_, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)

		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateSeatID = "$FLEET_VAR_BOZO"
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_seat_id",
			"FLEET_VAR_BOZO is not allowed")
	})

	t.Run("digicert API token not set", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].APIToken = fleet.MaskedPassword
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.api_token", "DigiCert API token must be set")
	})

	t.Run("custom_scep challenge not set", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Challenge = fleet.MaskedPassword
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.challenge", "Custom SCEP challenge must be set")
	})

	t.Run("digicert common name not set", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateCommonName = "\n\t"
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_common_name", "Common Name (CN) cannot be empty")
	})

	t.Run("digicert seat id not set", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert.Value[0].CertificateSeatID = "\t\n"
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.certificate_seat_id", "Seat ID cannot be empty")
	})

	t.Run("digicert happy path -- add one", func(t *testing.T) {
		mt := setUpDigiCert()
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.customSCEPProxy)
		require.Len(t, status.digicert, 1)
		assert.Equal(t, caStatusAdded, status.digicert[mt.newAppConfig.Integrations.DigiCert.Value[0].Name])
		require.Len(t, mt.appConfig.Integrations.DigiCert.Value, 1)
		assert.True(t, mt.newAppConfig.Integrations.DigiCert.Value[0].Equals(&mt.appConfig.Integrations.DigiCert.Value[0]))
	})

	t.Run("digicert happy path -- delete one", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.oldAppConfig = mt.newAppConfig
		mt.appConfig = mt.oldAppConfig.Copy()
		mt.newAppConfig = &fleet.AppConfig{
			Integrations: fleet.Integrations{
				DigiCert: optjson.Slice[fleet.DigiCertIntegration]{
					Set:   true,
					Valid: true,
				},
			},
		}
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.customSCEPProxy)
		require.Len(t, status.digicert, 1)
		assert.Equal(t, caStatusDeleted, status.digicert[mt.oldAppConfig.Integrations.DigiCert.Value[0].Name])
		assert.False(t, mt.appConfig.Integrations.DigiCert.Valid)
	})

	t.Run("digicert API token not set on modify", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.oldAppConfig.Integrations.DigiCert.Value = append(mt.oldAppConfig.Integrations.DigiCert.Value,
			mt.newAppConfig.Integrations.DigiCert.Value[0])
		mt.appConfig = mt.oldAppConfig.Copy()
		mt.newAppConfig.Integrations.DigiCert.Value[0].URL = "https://new.com"
		mt.newAppConfig.Integrations.DigiCert.Value[0].APIToken = ""
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.digicert.api_token", "DigiCert API token must be set when modifying")
	})

	t.Run("digicert happy path -- add one, delete one, modify one", func(t *testing.T) {
		mt := setUpDigiCert()
		mt.newAppConfig.Integrations.DigiCert = optjson.Slice[fleet.DigiCertIntegration]{
			Set:   true,
			Valid: true,
			Value: []fleet.DigiCertIntegration{
				{
					Name:                          "add",
					URL:                           mockDigiCertServer.URL,
					APIToken:                      "api_token",
					ProfileID:                     "profile_id",
					CertificateCommonName:         "common_name",
					CertificateUserPrincipalNames: []string{"user_principal_name"},
					CertificateSeatID:             "seat_id",
				},
				{
					Name:                          "modify",
					URL:                           mockDigiCertServer.URL,
					APIToken:                      "api_token",
					ProfileID:                     "profile_id",
					CertificateCommonName:         "common_name",
					CertificateUserPrincipalNames: nil,
					CertificateSeatID:             "seat_id",
				},
				{
					Name:                          "same",
					URL:                           mockDigiCertServer.URL,
					APIToken:                      "api_token",
					ProfileID:                     "profile_id",
					CertificateCommonName:         "other_cn",
					CertificateUserPrincipalNames: nil,
					CertificateSeatID:             "seat_id",
				},
			},
		}
		mt.oldAppConfig.Integrations.DigiCert = optjson.Slice[fleet.DigiCertIntegration]{
			Set:   true,
			Valid: true,
			Value: []fleet.DigiCertIntegration{
				{
					Name:                          "delete",
					URL:                           mockDigiCertServer.URL,
					APIToken:                      "api_token",
					ProfileID:                     "profile_id",
					CertificateCommonName:         "common_name",
					CertificateUserPrincipalNames: []string{"user_principal_name"},
					CertificateSeatID:             "seat_id",
				},
				{
					Name:                          "modify",
					URL:                           mockDigiCertServer.URL,
					APIToken:                      "api_token",
					ProfileID:                     "profile_id",
					CertificateCommonName:         "common_name",
					CertificateUserPrincipalNames: []string{"user_principal_name"},
					CertificateSeatID:             "seat_id",
				},
				{
					Name:                          "same",
					URL:                           mockDigiCertServer.URL,
					APIToken:                      "api_token",
					ProfileID:                     "profile_id",
					CertificateCommonName:         "other_cn",
					CertificateUserPrincipalNames: nil,
					CertificateSeatID:             "seat_id",
				},
			},
		}
		mt.appConfig = mt.oldAppConfig.Copy()
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.customSCEPProxy)
		require.Len(t, status.digicert, 3)
		assert.Equal(t, caStatusAdded, status.digicert["add"])
		assert.Equal(t, caStatusEdited, status.digicert["modify"])
		assert.Equal(t, caStatusDeleted, status.digicert["delete"])
		require.Len(t, mt.appConfig.Integrations.DigiCert.Value, 3)
	})

	t.Run("custom_scep happy path -- add one", func(t *testing.T) {
		mt := setUpCustomSCEP()
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.digicert)
		require.Len(t, status.customSCEPProxy, 1)
		assert.Equal(t, caStatusAdded, status.customSCEPProxy[mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Name])
		require.Len(t, mt.appConfig.Integrations.CustomSCEPProxy.Value, 1)
		assert.True(t, mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Equals(&mt.appConfig.Integrations.CustomSCEPProxy.Value[0]))
	})

	t.Run("custom_scep happy path -- delete one", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.oldAppConfig = mt.newAppConfig
		mt.appConfig = mt.oldAppConfig.Copy()
		mt.newAppConfig = &fleet.AppConfig{
			Integrations: fleet.Integrations{
				CustomSCEPProxy: optjson.Slice[fleet.CustomSCEPProxyIntegration]{
					Set:   true,
					Valid: true,
				},
			},
		}
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.digicert)
		require.Len(t, status.customSCEPProxy, 1)
		assert.Equal(t, caStatusDeleted, status.customSCEPProxy[mt.oldAppConfig.Integrations.CustomSCEPProxy.Value[0].Name])
		assert.False(t, mt.appConfig.Integrations.CustomSCEPProxy.Valid)
	})

	t.Run("custom_scep API token not set on modify", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.oldAppConfig.Integrations.CustomSCEPProxy.Value = append(mt.oldAppConfig.Integrations.CustomSCEPProxy.Value,
			mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0])
		mt.appConfig = mt.oldAppConfig.Copy()
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].URL = "https://new.com"
		mt.newAppConfig.Integrations.CustomSCEPProxy.Value[0].Challenge = ""
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		checkExpectedCAValidationError(t, mt.invalid, status, "integrations.custom_scep_proxy.challenge",
			"Custom SCEP challenge must be set when modifying")
	})

	t.Run("custom_scep happy path -- add one, delete one, modify one", func(t *testing.T) {
		mt := setUpCustomSCEP()
		mt.newAppConfig.Integrations.CustomSCEPProxy = optjson.Slice[fleet.CustomSCEPProxyIntegration]{
			Set:   true,
			Valid: true,
			Value: []fleet.CustomSCEPProxyIntegration{
				{
					Name:      "add",
					URL:       "https://example.com",
					Challenge: "challenge",
				},
				{
					Name:      "modify",
					URL:       "https://example.com",
					Challenge: "challenge",
				},
				{
					Name:      "SCEP_WIFI", // same
					URL:       "https://example.com",
					Challenge: "challenge",
				},
			},
		}
		mt.oldAppConfig.Integrations.CustomSCEPProxy = optjson.Slice[fleet.CustomSCEPProxyIntegration]{
			Set:   true,
			Valid: true,
			Value: []fleet.CustomSCEPProxyIntegration{
				{
					Name:      "delete",
					URL:       "https://example.com",
					Challenge: "challenge",
				},
				{
					Name:      "modify",
					URL:       "https://modify.com",
					Challenge: "challenge",
				},
				{
					Name:      "SCEP_WIFI", // same
					URL:       "https://example.com",
					Challenge: fleet.MaskedPassword,
				},
			},
		}
		mt.appConfig = mt.oldAppConfig.Copy()
		status, err := mt.svc.processAppConfigCAs(mt.ctx, mt.newAppConfig, mt.oldAppConfig, mt.appConfig, mt.invalid)
		require.NoError(t, err)
		assert.Empty(t, mt.invalid.Errors)
		assert.Empty(t, status.digicert)
		require.Len(t, status.customSCEPProxy, 3)
		assert.Equal(t, caStatusAdded, status.customSCEPProxy["add"])
		assert.Equal(t, caStatusEdited, status.customSCEPProxy["modify"])
		assert.Equal(t, caStatusDeleted, status.customSCEPProxy["delete"])
		require.Len(t, mt.appConfig.Integrations.CustomSCEPProxy.Value, 3)
	})
}

type configCASuite struct {
	ctx          context.Context
	svc          *Service
	appConfig    *fleet.AppConfig
	newAppConfig *fleet.AppConfig
	oldAppConfig *fleet.AppConfig
	invalid      *fleet.InvalidArgumentError
}

func addMockDatastoreForCA(t *testing.T, s configCASuite) {
	mockDS := &mock.Store{}
	s.svc.ds = mockDS
	mockDS.GetAllCAConfigAssetsByTypeFunc = func(ctx context.Context, assetType fleet.CAConfigAssetType) (map[string]fleet.CAConfigAsset, error) {
		switch assetType {
		case fleet.CAConfigDigiCert:
			return map[string]fleet.CAConfigAsset{
				"WIFI": {
					Name:  "WIFI",
					Value: []byte("api_token"),
					Type:  fleet.CAConfigDigiCert,
				},
			}, nil
		case fleet.CAConfigCustomSCEPProxy:
			return map[string]fleet.CAConfigAsset{
				"SCEP_WIFI": {
					Name:  "SCEP_WIFI",
					Value: []byte("challenge"),
					Type:  fleet.CAConfigCustomSCEPProxy,
				},
			}, nil
		default:
			t.Fatalf("unexpected asset type: %s", assetType)
		}
		return nil, nil
	}
}

func checkExpectedCAValidationError(t *testing.T, invalid *fleet.InvalidArgumentError, status appConfigCAStatus, contains ...string) {
	assert.Len(t, invalid.Errors, 1)
	for _, expected := range contains {
		assert.Contains(t, invalid.Error(), expected)
	}
	assert.Empty(t, status.ndes)
	assert.Empty(t, status.digicert)
	assert.Empty(t, status.customSCEPProxy)
}

func getAppConfigWithDigiCertIntegration(url string, name string) *fleet.AppConfig {
	newAppConfig := &fleet.AppConfig{
		Integrations: fleet.Integrations{
			DigiCert: optjson.Slice[fleet.DigiCertIntegration]{
				Set:   true,
				Valid: true,
				Value: []fleet.DigiCertIntegration{getDigiCertIntegration(url, name)},
			},
		},
	}
	return newAppConfig
}

func getDigiCertIntegration(url string, name string) fleet.DigiCertIntegration {
	digiCertCA := fleet.DigiCertIntegration{
		Name:                          name,
		URL:                           url,
		APIToken:                      "api_token",
		ProfileID:                     "profile_id",
		CertificateCommonName:         "common_name",
		CertificateUserPrincipalNames: []string{"user_principal_name"},
		CertificateSeatID:             "seat_id",
	}
	return digiCertCA
}

func getAppConfigWithSCEPIntegration(url string, name string) *fleet.AppConfig {
	newAppConfig := &fleet.AppConfig{
		Integrations: fleet.Integrations{
			CustomSCEPProxy: optjson.Slice[fleet.CustomSCEPProxyIntegration]{
				Set:   true,
				Valid: true,
				Value: []fleet.CustomSCEPProxyIntegration{getCustomSCEPIntegration(url, name)},
			},
		},
	}
	return newAppConfig
}

func getCustomSCEPIntegration(url string, name string) fleet.CustomSCEPProxyIntegration {
	challenge, _ := server.GenerateRandomText(6)
	return fleet.CustomSCEPProxyIntegration{
		Name:      name,
		URL:       url,
		Challenge: challenge,
	}
}
