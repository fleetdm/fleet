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
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConfigProfileGenerator stands in for fleetdm.com, answering each route with the status and body set on it.
type fakeConfigProfileGenerator struct {
	srv *httptest.Server

	mu         sync.Mutex
	status     int
	body       string
	delay      time.Duration
	lastPath   string
	lastBody   map[string]any
	calls      atomic.Int32
	canceledCh chan struct{}
}

func newFakeConfigProfileGenerator(t *testing.T) *fakeConfigProfileGenerator {
	f := &fakeConfigProfileGenerator{status: http.StatusOK, canceledCh: make(chan struct{}, 16)}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.lastPath, f.lastBody = r.URL.Path, body
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

func newAutofillTestService(t *testing.T, tier string, mutateConfig func(*fleet.AppConfig)) (fleet.Service, context.Context, *mock.Store) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		cfg := &fleet.AppConfig{}
		cfg.MDM.EnabledAndConfigured = true
		cfg.MDM.WindowsEnabledAndConfigured = true
		if mutateConfig != nil {
			mutateConfig(cfg)
		}
		return cfg, nil
	}
	svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tier}})
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

	t.Run("request carries the platform and Fleet mode", func(t *testing.T) {
		gen.respond(http.StatusOK, `{"name": "Disable AirDrop (macOS)", "description": "Sets allowAirDrop to false."}`)
		name, description, err := svc.AutofillConfigProfileNameDescription(ctx, "  Turn off AirDrop  ", "apple")
		require.NoError(t, err)
		assert.Equal(t, "Disable AirDrop (macOS)", name)
		assert.Equal(t, "Sets allowAirDrop to false.", description)

		gen.mu.Lock()
		defer gen.mu.Unlock()
		assert.Equal(t, autofillConfigProfileNameDescriptionPath, gen.lastPath)
		assert.Equal(t, map[string]any{"naturalLanguageInstructions": "Turn off AirDrop", "platform": "apple", "forFleet": true}, gen.lastBody)
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
