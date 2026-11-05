package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConfigProfileGenerator stands in for fleetdm.com, answering registration with the status and body set for
// it, and each generation route with the status and body set on it.
type fakeConfigProfileGenerator struct {
	srv *httptest.Server

	mu                sync.Mutex
	status            int
	body              string
	delay             time.Duration
	lastPath          string
	lastBody          map[string]any
	lastAuthorization string
	calls             atomic.Int32
	canceledCh        chan struct{}

	registerStatus   int
	registerBody     string
	registerDelay    time.Duration
	registerGate     chan struct{}
	lastRegistration map[string]any
	registrations    atomic.Int32
}

func newFakeConfigProfileGenerator(t *testing.T) *fakeConfigProfileGenerator {
	f := &fakeConfigProfileGenerator{
		status:         http.StatusOK,
		canceledCh:     make(chan struct{}, 16),
		registerStatus: http.StatusOK,
		registerBody:   `{"fleetServerSecret": "registered-secret"}`,
	}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path == autofillConfigProfileRegisterPath {
			f.registrations.Add(1)
			f.mu.Lock()
			f.lastRegistration = body
			status, respBody, delay, gate := f.registerStatus, f.registerBody, f.registerDelay, f.registerGate
			f.mu.Unlock()
			time.Sleep(delay)
			if gate != nil {
				<-gate
			}
			w.WriteHeader(status)
			_, _ = w.Write([]byte(respBody))
			return
		}
		f.mu.Lock()
		f.lastPath, f.lastBody, f.lastAuthorization = r.URL.Path, body, r.Header.Get("Authorization")
		status, respBody, delay := f.status, f.body, f.delay
		f.mu.Unlock()

		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-r.Context().Done():
				f.canceledCh <- struct{}{}
				return
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	}))
	t.Cleanup(f.srv.Close)
	dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, f.srv.URL, t)
	return f
}

func (f *fakeConfigProfileGenerator) respond(status int, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.status, f.body = status, body
}

func (f *fakeConfigProfileGenerator) setDelay(t *testing.T, delay time.Duration) {
	f.mu.Lock()
	f.delay = delay
	f.mu.Unlock()
	t.Cleanup(func() {
		f.mu.Lock()
		f.delay = 0
		f.mu.Unlock()
		// Drain cancellations so the next subtest only sees its own.
		for len(f.canceledCh) > 0 {
			<-f.canceledCh
		}
	})
}

// newAutofillTestService returns a service with an unexpired license of the given tier and a license key.  On
// Fleet Free it has already registered with fleetdm.com, so requests go straight to the generator.
func newAutofillTestService(t *testing.T, tier string, mutateConfig func(*fleet.AppConfig)) (fleet.Service, context.Context, *mock.Store) {
	return newAutofillTestServiceWithLicense(t, &fleet.LicenseInfo{Tier: tier, Expiration: time.Now().Add(time.Hour)}, "test-license-key", mutateConfig)
}

func newAutofillTestServiceWithLicense(t *testing.T, lic *fleet.LicenseInfo, licenseKey string, mutateConfig func(*fleet.AppConfig)) (fleet.Service, context.Context, *mock.Store) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		cfg := &fleet.AppConfig{}
		cfg.ServerSettings.ServerURL = "https://fleet.example.com"
		cfg.MDM.EnabledAndConfigured = true
		cfg.MDM.WindowsEnabledAndConfigured = true
		if mutateConfig != nil {
			mutateConfig(cfg)
		}
		return cfg, nil
	}
	ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
		return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
			fleet.MDMAssetProfileGeneratorFleetServerSecret: {Name: fleet.MDMAssetProfileGeneratorFleetServerSecret, Value: []byte("stored-secret")},
		}, nil
	}
	fleetConfig := config.TestConfig()
	fleetConfig.License.Key = licenseKey
	fleetConfig.Server.PrivateKey = "test-private-key-not-used-by-mock"
	svc, ctx := newTestServiceWithConfig(t, ds, fleetConfig, nil, nil, &TestServerOpts{License: lic})
	return svc, test.UserContext(ctx, test.UserAdmin), ds
}

func requireAutofillError(t *testing.T, err error, field string, reasonContains string, clientError bool) {
	t.Helper()
	var autofillErr *fleet.AutofillConfigProfileError
	require.ErrorAs(t, err, &autofillErr)
	assert.Equal(t, field, autofillErr.Field)              // nolint:nilaway // require.ErrorAs fails the test when it's nil
	assert.Contains(t, autofillErr.Reason, reasonContains) // nolint:nilaway // require.ErrorAs fails the test when it's nil
	assert.Equal(t, clientError, autofillErr != nil && autofillErr.IsClientError())
}

func TestAutofillConfigProfileBaseURL(t *testing.T) {
	// Earlier tests' overrides leave dev mode on for the rest of the package.
	dev_mode.ClearAllOverrides()

	// Without dev mode the variable is ignored, so a stray value can't redirect a production server.
	t.Setenv(autofillConfigProfileBaseURLEnvVar, "http://localhost:2024")
	assert.Equal(t, "https://fleetdm.com", autofillConfigProfileBaseURL())

	dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, "http://localhost:2024/", t)
	assert.Equal(t, "http://localhost:2024", autofillConfigProfileBaseURL())
}

func TestAutofillConfigProfileRequestValidation(t *testing.T) {
	gen := newFakeConfigProfileGenerator(t)

	cases := []struct {
		name         string
		mutateConfig func(*fleet.AppConfig)
		instructions string
		platform     string
		check        func(t *testing.T, err error)
	}{
		{
			name:         "AI features disabled",
			mutateConfig: func(c *fleet.AppConfig) { c.ServerSettings.AIFeaturesDisabled = true },
			instructions: "Turn off AirDrop", platform: "apple",
			check: func(t *testing.T, err error) { requireBadRequest(t, err, "AI features are disabled") },
		},
		{
			name:         "android platform",
			instructions: "Turn off the camera", platform: "android",
			check: func(t *testing.T, err error) { requireBadRequest(t, err, "'platform'") },
		},
		{
			name:         "empty instructions",
			instructions: "   ", platform: "apple",
			check: func(t *testing.T, err error) { requireBadRequest(t, err, "cannot be empty") },
		},
		{
			name:         "instructions over the limit",
			instructions: strings.Repeat("é", fleet.MaxAutofillConfigProfileInstructionsLength+1), platform: "apple",
			check: func(t *testing.T, err error) { requireBadRequest(t, err, "2,000 characters") },
		},
		{
			name:         "Apple MDM off",
			mutateConfig: func(c *fleet.AppConfig) { c.MDM.EnabledAndConfigured = false },
			instructions: "Turn off AirDrop", platform: "apple",
			check: func(t *testing.T, err error) {
				requireAutofillError(t, err, "base", fleet.AppleMDMNotConfiguredMessage, true)
			},
		},
		{
			name:         "Windows MDM off",
			mutateConfig: func(c *fleet.AppConfig) { c.MDM.WindowsEnabledAndConfigured = false },
			instructions: "Require a password", platform: "windows",
			check: func(t *testing.T, err error) {
				requireAutofillError(t, err, "base", fleet.WindowsMDMNotConfiguredMessage, true)
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, ctx, _ := newAutofillTestService(t, fleet.TierPremium, c.mutateConfig)
			before := gen.calls.Load()

			_, err := svc.AutofillConfigProfileContent(ctx, c.instructions, c.platform, 0)
			c.check(t, err)
			_, _, err = svc.AutofillConfigProfileNameDescription(ctx, c.instructions, c.platform)
			c.check(t, err)

			assert.Equal(t, before, gen.calls.Load(), "fleetdm.com must not be called")
		})
	}

	t.Run("instructions at the limit are sent", func(t *testing.T) {
		svc, ctx, _ := newAutofillTestService(t, fleet.TierPremium, nil)
		gen.respond(http.StatusOK, `{"name": "Name", "description": "Description"}`)
		_, _, err := svc.AutofillConfigProfileNameDescription(ctx, strings.Repeat("é", fleet.MaxAutofillConfigProfileInstructionsLength), "apple")
		require.NoError(t, err)
	})
}

// requireAutofillAuthz checks only the authorization outcome: an allowed request reaches fleetdm.com, which this
// test's fake answers with an error.
func requireAutofillAuthz(t *testing.T, allowed bool, err error) {
	t.Helper()
	require.Error(t, err)
	var authzErr *authz.Forbidden
	if allowed {
		assert.NotErrorAs(t, err, &authzErr, "expected the request to be authorized")
		var autofillErr *fleet.AutofillConfigProfileError
		assert.ErrorAs(t, err, &autofillErr)
		return
	}
	assert.ErrorAs(t, err, &authzErr)
}

func requireBadRequest(t *testing.T, err error, contains string) {
	t.Helper()
	var badReq *fleet.BadRequestError
	require.ErrorAs(t, err, &badReq)
	assert.Contains(t, badReq.Message, contains) // nolint:nilaway // require.ErrorAs fails the test when it's nil
}

func TestAutofillConfigProfileAuthorization(t *testing.T) {
	gen := newFakeConfigProfileGenerator(t)
	// Every authorized request gets as far as fleetdm.com and fails there, so the checks don't need datastore mocks.
	gen.respond(http.StatusInternalServerError, `{}`)

	svc, ctx, _ := newAutofillTestService(t, fleet.TierPremium, nil)

	cases := []struct {
		name                string
		user                *fleet.User
		contentAllowed      bool
		nameDescriptAllowed bool
	}{
		{"global admin", test.UserAdmin, true, true},
		{"global maintainer", test.UserMaintainer, true, true},
		{"global gitops", test.UserGitOps, true, true},
		{"global technician", test.UserTechnician, false, false},
		{"global observer", test.UserObserver, false, false},
		{"global observer+", test.UserObserverPlus, false, false},
		// Unassigned (fleet_id 0) is global, so team roles can't generate content for it.
		{"team maintainer", test.UserTeamMaintainerTeam1, false, true},
		{"team admin", test.UserTeamAdminTeam1, false, true},
		{"team gitops", test.UserTeamGitOpsTeam1, false, true},
		{"team observer", test.UserTeamObserverTeam1, false, false},
		{"team observer and admin of another team", test.UserTeamObserverTeam1TeamAdminTeam2, false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			userCtx := test.UserContext(ctx, c.user)

			_, err := svc.AutofillConfigProfileContent(userCtx, "Turn off AirDrop", "apple", 0)
			requireAutofillAuthz(t, c.contentAllowed, err)

			_, _, err = svc.AutofillConfigProfileNameDescription(userCtx, "Turn off AirDrop", "apple")
			requireAutofillAuthz(t, c.nameDescriptAllowed, err)
		})
	}
}

func TestAutofillConfigProfileUpstreamFailures(t *testing.T) {
	gen := newFakeConfigProfileGenerator(t)
	svc, ctx, _ := newAutofillTestService(t, fleet.TierPremium, nil)

	t.Run("request carries the platform and the license key", func(t *testing.T) {
		gen.respond(http.StatusOK, `{"name": "Disable AirDrop (macOS)", "description": "Sets allowAirDrop to false."}`)
		name, description, err := svc.AutofillConfigProfileNameDescription(ctx, "  Turn off AirDrop  ", "apple")
		require.NoError(t, err)
		assert.Equal(t, "Disable AirDrop (macOS)", name)
		assert.Equal(t, "Sets allowAirDrop to false.", description)

		gen.mu.Lock()
		defer gen.mu.Unlock()
		assert.Equal(t, autofillConfigProfileNameDescriptionPath, gen.lastPath)
		assert.Equal(t, map[string]any{"naturalLanguageInstructions": "Turn off AirDrop", "platform": "apple", "fleetLicenseKey": "test-license-key"}, gen.lastBody)
	})

	failures := []struct {
		name   string
		status int
		body   string
		reason string
	}{
		{"server error", http.StatusInternalServerError, `{}`, "unavailable"},
		{"bad request that isn't a decline", http.StatusBadRequest, `{"error": "badRequest"}`, "unavailable"},
		{"malformed body", http.StatusOK, `{not json`, "couldn't read"},
		{"missing fields", http.StatusOK, `{}`, "couldn't read"},
	}
	for _, f := range failures {
		t.Run(f.name, func(t *testing.T) {
			gen.respond(f.status, f.body)
			_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
			requireAutofillError(t, err, "base", f.reason, false)
			_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
			requireAutofillError(t, err, "base", f.reason, false)
		})
	}

	t.Run("unreachable", func(t *testing.T) {
		closed := httptest.NewServer(http.NotFoundHandler())
		closed.Close()
		// Restored rather than cleared: clearing would send the remaining subtests to the real fleetdm.com.
		dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, closed.URL)
		t.Cleanup(func() { dev_mode.SetOverride(autofillConfigProfileBaseURLEnvVar, gen.srv.URL) })
		_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
		requireAutofillError(t, err, "base", "couldn't be reached", false)
	})

	t.Run("timeout", func(t *testing.T) {
		oldTimeout := autofillConfigProfileTimeout
		autofillConfigProfileTimeout = 50 * time.Millisecond
		t.Cleanup(func() { autofillConfigProfileTimeout = oldTimeout })
		gen.setDelay(t, time.Second)

		_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
		requireAutofillError(t, err, "base", "timed out", false)
		_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
		requireAutofillError(t, err, "base", "timed out", false)
	})

	t.Run("client disconnect cancels the upstream call", func(t *testing.T) {
		gen.setDelay(t, 5*time.Second)

		reqCtx, cancel := context.WithCancel(ctx)
		time.AfterFunc(50*time.Millisecond, cancel)
		_, err := svc.AutofillConfigProfileContent(reqCtx, "Turn off AirDrop", "apple", 0)
		require.ErrorIs(t, err, context.Canceled)
		var autofillErr *fleet.AutofillConfigProfileError
		assert.NotErrorAs(t, err, &autofillErr, "a disconnect isn't an AI service failure")

		select {
		case <-gen.canceledCh:
		case <-time.After(2 * time.Second):
			t.Fatal("the upstream request was not canceled")
		}
	})

	t.Run("decline", func(t *testing.T) {
		gen.respond(http.StatusBadRequest, `{"error": "couldNotGenerateProfile", "reason": "Platform SSO isn't supported yet."}`)
		_, err := svc.AutofillConfigProfileContent(ctx, "Set up Platform SSO", "apple", 0)
		requireAutofillError(t, err, "natural_language_instructions", "Platform SSO isn't supported yet.", true)

		gen.respond(http.StatusBadRequest, `{"error": "couldNotGenerateProfile"}`)
		_, err = svc.AutofillConfigProfileContent(ctx, "Set up Platform SSO", "apple", 0)
		requireAutofillError(t, err, "natural_language_instructions", "couldn't generate", true)
	})

	t.Run("additional profiles", func(t *testing.T) {
		gen.respond(http.StatusOK, `{"profileType": "csp", "profile": "<Replace/>", "additionalProfiles": [{"filename": "chrome.xml", "profile": "<Add/>"}]}`)
		_, err := svc.AutofillConfigProfileContent(ctx, "Set the Chrome homepage", "windows", 0)
		requireAutofillError(t, err, "natural_language_instructions", "more than one file", true)
	})

	t.Run("profile type that doesn't match the platform", func(t *testing.T) {
		gen.respond(http.StatusOK, `{"profileType": "csp", "profile": "<Replace/>"}`)
		_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
		requireAutofillError(t, err, "base", "couldn't read", false)
	})
}

func TestAutofillConfigProfileFleetFree(t *testing.T) {
	gen := newFakeConfigProfileGenerator(t)
	svc, ctx, ds := newAutofillTestService(t, fleet.TierFree, nil)

	t.Run("fleet_id above 0 is rejected before calling fleetdm.com", func(t *testing.T) {
		before := gen.calls.Load()
		_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 1)
		require.ErrorIs(t, err, fleet.ErrMissingLicense)
		assert.Equal(t, before, gen.calls.Load())
	})

	t.Run("an OS updates declaration gets the license message", func(t *testing.T) {
		ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
			return document, nil, nil
		}
		ds.ValidateReferencedCustomHostVitalsFunc = func(ctx context.Context, documents []string) error { return nil }
		declaration, err := json.Marshal(map[string]any{
			"Type":       "com.apple.configuration.softwareupdate.enforcement.specific",
			"Identifier": "com.example.os-update",
			"Payload":    map[string]any{"TargetOSVersion": "26.1", "TargetLocalDateTime": "2026-12-15T18:00:00"},
		})
		require.NoError(t, err)
		body, err := json.Marshal(map[string]any{"profileType": "ddm", "profile": string(declaration), "deliveryNotes": ""})
		require.NoError(t, err)
		gen.respond(http.StatusOK, string(body))

		_, err = svc.AutofillConfigProfileContent(ctx, "Update Macs to macOS 26.1 by December 15", "apple", 0)
		requireAutofillError(t, err, "natural_language_instructions", "Fleet Premium", true)
	})
}

func TestAutofillConfigProfileUsageStatistics(t *testing.T) {
	gen := newFakeConfigProfileGenerator(t)
	svc, ctx, ds := newAutofillTestService(t, fleet.TierPremium, nil)
	ds.ExpandEmbeddedSecretsAndUpdatedAtFunc = func(ctx context.Context, document string) (string, *time.Time, error) {
		return document, nil, nil
	}
	ds.ValidateReferencedCustomHostVitalsFunc = func(ctx context.Context, documents []string) error { return nil }

	var counted []string
	var countErr error
	ds.IncrementAIGeneratedProfileCountFunc = func(ctx context.Context, platform string) error {
		counted = append(counted, platform)
		return countErr
	}
	declaration := `{"profileType": "ddm", "profile": "{\"Type\": \"com.apple.configuration.passcode.settings\", \"Identifier\": \"com.example.passcode\", \"Payload\": {\"RequirePasscode\": true}}"}`

	gen.respond(http.StatusOK, declaration)
	_, err := svc.AutofillConfigProfileContent(ctx, "Require a passcode", "apple", 0)
	require.NoError(t, err)
	assert.Equal(t, []string{"apple"}, counted)

	// A decline, a failed check, and the name and description endpoint don't count.
	gen.respond(http.StatusBadRequest, `{"error": "couldNotGenerateProfile", "reason": "No."}`)
	_, err = svc.AutofillConfigProfileContent(ctx, "Require a passcode", "apple", 0)
	require.Error(t, err)
	gen.respond(http.StatusOK, `{"profileType": "ddm", "profile": "{\"Type\": \"com.apple.configuration.package\", \"Identifier\": \"com.example.package\", \"Payload\": {}}"}`)
	_, err = svc.AutofillConfigProfileContent(ctx, "Install a package", "apple", 0)
	require.Error(t, err)
	gen.respond(http.StatusOK, `{"name": "Name", "description": "Description"}`)
	_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Require a passcode", "apple")
	require.NoError(t, err)
	assert.Equal(t, []string{"apple"}, counted)

	// A counting failure doesn't cost the admin their profile.
	countErr = errors.New("database is down")
	gen.respond(http.StatusOK, declaration)
	profile, err := svc.AutofillConfigProfileContent(ctx, "Require a passcode", "apple", 0)
	require.NoError(t, err)
	assert.Equal(t, fleet.AutofillConfigProfileTypeDeclaration, profile.Type)
	assert.Equal(t, []string{"apple", "apple"}, counted)
}

func TestAutofillConfigProfileCredentials(t *testing.T) {
	gen := newFakeConfigProfileGenerator(t)
	nameAndDescription := `{"name": "Disable AirDrop (macOS)", "description": "Sets allowAirDrop to false."}`

	// A Fleet Free service that hasn't registered, storing the secret it gets in memory.  Each lookup of the
	// secret is sent on lookups, if set.
	var lookups chan struct{}
	newUnregisteredService := func(t *testing.T, lic *fleet.LicenseInfo, licenseKey string) (fleet.Service, context.Context, *mock.Store, func() string) {
		svc, ctx, ds := newAutofillTestServiceWithLicense(t, lic, licenseKey, nil)
		var mu sync.Mutex
		var stored string
		ds.GetAllMDMConfigAssetsByNameFunc = func(ctx context.Context, names []fleet.MDMAssetName, _ sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
			if lookups != nil {
				lookups <- struct{}{}
			}
			mu.Lock()
			defer mu.Unlock()
			if stored == "" {
				return nil, newNotFoundError()
			}
			return map[fleet.MDMAssetName]fleet.MDMConfigAsset{
				fleet.MDMAssetProfileGeneratorFleetServerSecret: {Name: fleet.MDMAssetProfileGeneratorFleetServerSecret, Value: []byte(stored)},
			}, nil
		}
		ds.InsertOrReplaceMDMConfigAssetFunc = func(ctx context.Context, asset fleet.MDMConfigAsset) error {
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, fleet.MDMAssetProfileGeneratorFleetServerSecret, asset.Name)
			stored = string(asset.Value)
			return nil
		}
		return svc, ctx, ds, func() string {
			mu.Lock()
			defer mu.Unlock()
			return stored
		}
	}
	resetCounts := func() {
		gen.calls.Store(0)
		gen.registrations.Store(0)
	}
	setRegistration := func(t *testing.T, status int, body string, delay time.Duration) {
		gen.mu.Lock()
		gen.registerStatus, gen.registerBody, gen.registerDelay = status, body, delay
		gen.mu.Unlock()
		t.Cleanup(func() {
			gen.mu.Lock()
			gen.registerStatus, gen.registerBody, gen.registerDelay = http.StatusOK, `{"fleetServerSecret": "registered-secret"}`, 0
			gen.mu.Unlock()
		})
	}
	expiredPremium := func() *fleet.LicenseInfo {
		return &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(-time.Hour)}
	}

	t.Run("an unexpired Premium license sends its key, never registers, and needs no private key", func(t *testing.T) {
		resetCounts()
		testSetEmptyPrivateKey = true
		t.Cleanup(func() { testSetEmptyPrivateKey = false })
		svc, ctx, ds := newAutofillTestService(t, fleet.TierPremium, nil)
		gen.respond(http.StatusOK, nameAndDescription)

		_, _, err := svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
		require.NoError(t, err)
		gen.mu.Lock()
		assert.Equal(t, "test-license-key", gen.lastBody["fleetLicenseKey"])
		assert.Empty(t, gen.lastAuthorization)
		gen.mu.Unlock()
		assert.Zero(t, gen.registrations.Load())
		assert.False(t, ds.GetAllMDMConfigAssetsByNameFuncInvoked)
	})

	for _, c := range []struct {
		name       string
		lic        *fleet.LicenseInfo
		licenseKey string
	}{
		{"Fleet Free", &fleet.LicenseInfo{Tier: fleet.TierFree}, ""},
		{"an expired license", expiredPremium(), "test-license-key"},
		{"Premium with no license key configured", &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(time.Hour)}, ""},
	} {
		t.Run(c.name+" registers on first use, then reuses the secret", func(t *testing.T) {
			resetCounts()
			svc, ctx, _, storedSecret := newUnregisteredService(t, c.lic, c.licenseKey)
			gen.respond(http.StatusOK, nameAndDescription)

			_, _, err := svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
			require.NoError(t, err)
			assert.EqualValues(t, 1, gen.registrations.Load())
			assert.Equal(t, "registered-secret", storedSecret())
			gen.mu.Lock()
			assert.Equal(t, map[string]any{"fleetServerUrl": "https://fleet.example.com"}, gen.lastRegistration)
			assert.Equal(t, "Bearer registered-secret", gen.lastAuthorization)
			assert.NotContains(t, gen.lastBody, "fleetLicenseKey")
			gen.mu.Unlock()

			_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
			require.NoError(t, err)
			assert.EqualValues(t, 1, gen.registrations.Load())
		})
	}

	t.Run("concurrent first requests share one registration", func(t *testing.T) {
		resetCounts()
		svc, ctx, _, _ := newUnregisteredService(t, &fleet.LicenseInfo{Tier: fleet.TierFree}, "")
		gen.respond(http.StatusOK, nameAndDescription)
		const requests = 5
		// fleetdm.com answers the registration only once every request has found no secret, and the registration
		// has checked again, so they all overlap the first registration.
		lookups = make(chan struct{}, 2*requests)
		gate := make(chan struct{})
		gen.mu.Lock()
		gen.registerGate = gate
		gen.mu.Unlock()
		t.Cleanup(func() {
			lookups = nil
			gen.mu.Lock()
			gen.registerGate = nil
			gen.mu.Unlock()
		})
		go func() {
			for range requests + 1 {
				<-lookups
			}
			close(gate)
		}()

		var wg sync.WaitGroup
		errs := make([]error, requests)
		for i := range errs {
			wg.Go(func() {
				_, _, errs[i] = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
			})
		}
		wg.Wait()
		for _, err := range errs {
			require.NoError(t, err)
		}
		assert.EqualValues(t, 1, gen.registrations.Load())
	})

	t.Run("a rejected secret is reported as unavailable and doesn't register again", func(t *testing.T) {
		resetCounts()
		svc, ctx, _ := newAutofillTestService(t, fleet.TierFree, nil)
		gen.respond(http.StatusUnauthorized, `Unauthorized`)

		_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
		requireAutofillError(t, err, "base", "The AI service is unavailable.", false)
		_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
		requireAutofillError(t, err, "base", "The AI service is unavailable.", false)
		assert.Zero(t, gen.registrations.Load())
		gen.mu.Lock()
		assert.Equal(t, "Bearer stored-secret", gen.lastAuthorization)
		gen.mu.Unlock()
	})

	t.Run("without a private key, Fleet Free is a 422 with no outbound call", func(t *testing.T) {
		testSetEmptyPrivateKey = true
		t.Cleanup(func() { testSetEmptyPrivateKey = false })
		for _, lic := range []*fleet.LicenseInfo{{Tier: fleet.TierFree}, expiredPremium()} {
			resetCounts()
			svc, ctx, _, _ := newUnregisteredService(t, lic, "test-license-key")
			_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
			requireAutofillError(t, err, "base", fleet.AutofillConfigProfileMissingPrivateKeyReason, true)
			assert.Equal(t, fleet.AutofillConfigProfileContentErrorMessage, autofillErrorOf(t, err).Message)
			_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
			requireAutofillError(t, err, "base", fleet.AutofillConfigProfileMissingPrivateKeyReason, true)
			assert.Equal(t, fleet.AutofillConfigProfileNameDescriptionErrorMessage, autofillErrorOf(t, err).Message)
			assert.Zero(t, gen.calls.Load())
		}
	})

	failedRegistrations := []struct {
		name   string
		status int
		body   string
		reason string
	}{
		{"rejected", http.StatusBadRequest, `Bad Request`, "The AI service is unavailable."},
		{"server error", http.StatusInternalServerError, `{}`, "The AI service is unavailable."},
		{"no secret in the response", http.StatusOK, `{}`, "couldn't read"},
		{"malformed response", http.StatusOK, `{not json`, "couldn't read"},
	}
	for _, f := range failedRegistrations {
		t.Run("a registration that fails is reported as an AI service failure: "+f.name, func(t *testing.T) {
			resetCounts()
			svc, ctx, _, storedSecret := newUnregisteredService(t, &fleet.LicenseInfo{Tier: fleet.TierFree}, "")
			setRegistration(t, f.status, f.body, 0)

			_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
			requireAutofillError(t, err, "base", f.reason, false)
			assert.Equal(t, fleet.AutofillConfigProfileContentErrorMessage, autofillErrorOf(t, err).Message)
			assert.Empty(t, storedSecret())
			assert.EqualValues(t, 1, gen.calls.Load(), "nothing is generated without a secret")
		})
	}

	t.Run("fleetdm.com's limit is a 429 with Retry-After on both endpoints", func(t *testing.T) {
		svc, ctx, _ := newAutofillTestService(t, fleet.TierFree, nil)
		resetsAt := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
		gen.respond(http.StatusTooManyRequests, `{"error": "generationLimitReached", "resetsAt": "`+resetsAt.Format(time.RFC3339)+`"}`)

		_, err := svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
		requireAutofillError(t, err, "base", fleet.AutofillConfigProfileLimitReachedReason, true)
		limitErr := autofillErrorOf(t, err)
		assert.Equal(t, http.StatusTooManyRequests, limitErr.StatusCode())
		assert.Equal(t, fleet.AutofillConfigProfileContentErrorMessage, limitErr.Message)
		assert.InDelta(t, 48*60*60, limitErr.RetryAfterSeconds, 5)

		_, _, err = svc.AutofillConfigProfileNameDescription(ctx, "Turn off AirDrop", "apple")
		requireAutofillError(t, err, "base", fleet.AutofillConfigProfileLimitReachedReason, true)
		limitErr = autofillErrorOf(t, err)
		assert.Equal(t, http.StatusTooManyRequests, limitErr.StatusCode())
		assert.Equal(t, fleet.AutofillConfigProfileNameDescriptionErrorMessage, limitErr.Message)

		// A 429 that isn't the limit, such as a rate limit in front of fleetdm.com, is an outage like any other.
		gen.respond(http.StatusTooManyRequests, `Too Many Requests`)
		_, err = svc.AutofillConfigProfileContent(ctx, "Turn off AirDrop", "apple", 0)
		requireAutofillError(t, err, "base", "The AI service is unavailable.", false)
	})
}

func TestAutofillLimitErrorRetryAfter(t *testing.T) {
	ctx := t.Context()
	now := time.Date(2026, time.November, 30, 23, 0, 0, 0, time.UTC)
	retryAfter := func(body string) int {
		err := autofillLimitError(ctx, fleet.AutofillConfigProfileContentErrorMessage, http.StatusTooManyRequests, []byte(body), now)
		require.Error(t, err)
		return autofillErrorOf(t, err).RetryAfterSeconds
	}
	assert.Equal(t, 3600, retryAfter(`{"error": "generationLimitReached", "resetsAt": "2026-12-01T00:00:00.000Z"}`))
	// A fraction of a second still waits a whole one.
	assert.Equal(t, 3601, retryAfter(`{"error": "generationLimitReached", "resetsAt": "2026-12-01T00:00:00.500Z"}`))
	// With no usable date, the 1st of next month (UTC).
	assert.Equal(t, 3600, retryAfter(`{"error": "generationLimitReached"}`))
	assert.Equal(t, 3600, retryAfter(`{"error": "generationLimitReached", "resetsAt": "2026-11-01T00:00:00.000Z"}`))

	require.NoError(t, autofillLimitError(ctx, "", http.StatusTooManyRequests, []byte(`{"error": "somethingElse"}`), now))
	require.NoError(t, autofillLimitError(ctx, "", http.StatusOK, []byte(`{"error": "generationLimitReached"}`), now))
}

func autofillErrorOf(t *testing.T, err error) *fleet.AutofillConfigProfileError {
	t.Helper()
	var autofillErr *fleet.AutofillConfigProfileError
	require.ErrorAs(t, err, &autofillErr)
	return autofillErr
}
