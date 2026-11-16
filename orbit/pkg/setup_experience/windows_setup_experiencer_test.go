package setupexperience

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

type noopWindowsInitiator struct{ initiated bool }

func (n *noopWindowsInitiator) InitiateSetupExperience() (fleet.SetupExperienceInitResult, error) {
	n.initiated = true
	return fleet.SetupExperienceInitResult{Enabled: true}, nil
}

func (n *noopWindowsInitiator) GetSetupExperienceStatus(bool) (*fleet.SetupExperienceStatusPayload, error) {
	return &fleet.SetupExperienceStatusPayload{}, nil
}

func TestNewWindowsSetupExperiencerLoadsState(t *testing.T) {
	t.Run("fresh host", func(t *testing.T) {
		s := NewWindowsSetupExperiencer(&noopWindowsInitiator{}, t.TempDir(), nil, nil)
		require.False(t, s.finished)
		require.Empty(t, s.pinPendingUser)
	})

	t.Run("in progress resumes without initiating again", func(t *testing.T) {
		dir := t.TempDir()
		started := time.Now().Add(-time.Hour)
		require.NoError(t, WriteSetupExperienceStatusFile(dir, &SetupExperienceInfo{TimeInitiated: started, Enabled: true}))
		client := &noopWindowsInitiator{}
		opened := 0
		s := NewWindowsSetupExperiencer(client, dir, func() error { opened++; return nil }, nil)
		s.interactiveSession = func() (string, uint32, bool, error) { return "Fleet-Provision", 1, true, nil }
		require.False(t, s.finished)
		require.True(t, s.enabled)
		require.True(t, s.enabledAt.Equal(started))

		require.NoError(t, s.Run(&fleet.OrbitConfig{}))
		require.False(t, client.initiated)
		require.Equal(t, 1, opened, "setup page reopens after a restart")
		require.NoError(t, s.Run(&fleet.OrbitConfig{}))
		require.Equal(t, 1, opened)
	})

	t.Run("finished does not rerun", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, WriteSetupExperienceStatusFile(dir, &SetupExperienceInfo{
			TimeInitiated: time.Now(), Enabled: true, TimeFinished: new(time.Now()),
		}))
		client := &noopWindowsInitiator{}
		s := NewWindowsSetupExperiencer(client, dir, nil, nil)
		require.True(t, s.finished)
		require.NoError(t, s.Run(&fleet.OrbitConfig{}))
		require.False(t, client.initiated)
	})

	t.Run("pending PIN enrollment survives restart", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, pinEnrollmentPendingFilename), []byte("luke\n"), 0o600))
		s := NewWindowsSetupExperiencer(&noopWindowsInitiator{}, dir, nil, nil)
		require.Equal(t, "luke", s.pinPendingUser)

		// No interactive session for that user, so the prompt stays pending.
		require.NoError(t, s.Run(&fleet.OrbitConfig{}))
		require.Equal(t, "luke", s.pinPendingUser)
		require.FileExists(t, filepath.Join(dir, pinEnrollmentPendingFilename))
	})
}

type scriptedStatusInitiator struct {
	noopWindowsInitiator
	statuses []fleet.SetupExperienceStatusResultStatus
	calls    int
}

func (c *scriptedStatusInitiator) GetSetupExperienceStatus(bool) (*fleet.SetupExperienceStatusPayload, error) {
	st := c.statuses[min(c.calls, len(c.statuses)-1)]
	c.calls++
	return &fleet.SetupExperienceStatusPayload{
		Software: []*fleet.SetupExperienceStatusResult{{Name: "Chrome", Status: st}},
	}, nil
}

func TestWatchSoftwareOpensPasswordPage(t *testing.T) {
	client := &scriptedStatusInitiator{statuses: []fleet.SetupExperienceStatusResultStatus{
		fleet.SetupExperienceStatusRunning,
		fleet.SetupExperienceStatusRunning,
		fleet.SetupExperienceStatusSuccess,
	}}
	var opened string
	s := NewWindowsSetupExperiencer(client, t.TempDir(), nil, func(u string) error {
		opened = u
		return nil
	})
	s.interactiveSession = func() (string, uint32, bool, error) { return ProvisioningAccountName, 1, true, nil }
	s.statusPollInterval = time.Millisecond
	s.cachedIdP = &fleet.OrbitEndUserIdPAccount{Email: "luke@example.com"}

	s.watchSoftware()

	require.Equal(t, 3, client.calls)
	require.NotNil(t, s.passwordPage)
	t.Cleanup(func() { _ = s.passwordPage.Close() })
	require.True(t, strings.HasPrefix(opened, "http://127.0.0.1:"), opened)
	require.Equal(t, s.passwordPage.URL(), opened)
}

func TestWatchSoftwareWaitsForProvisioningSession(t *testing.T) {
	client := &scriptedStatusInitiator{statuses: []fleet.SetupExperienceStatusResultStatus{fleet.SetupExperienceStatusSuccess}}
	s := NewWindowsSetupExperiencer(client, t.TempDir(), nil, func(string) error {
		t.Fatal("password page opened outside the provisioning session")
		return nil
	})
	s.interactiveSession = func() (string, uint32, bool, error) { return "someone", 2, true, nil }
	s.statusPollInterval = time.Millisecond
	s.cachedIdP = &fleet.OrbitEndUserIdPAccount{Email: "luke@example.com"}
	s.watchingSoftware = true

	s.watchSoftware()

	require.Nil(t, s.passwordPage)
	require.False(t, s.watchingSoftware, "Run takes over again")
}

func TestWindowsSetupDone(t *testing.T) {
	sw := func(st fleet.SetupExperienceStatusResultStatus) *fleet.SetupExperienceStatusResult {
		return &fleet.SetupExperienceStatusResult{Name: "app", Status: st}
	}
	s := NewWindowsSetupExperiencer(&noopWindowsInitiator{}, t.TempDir(), nil, nil)

	s.enabledAt = time.Now()
	require.False(t, s.setupDone(&fleet.SetupExperienceStatusPayload{}), "empty status right after initiation is still loading")
	s.enabledAt = time.Now().Add(-emptyStatusGrace)
	require.True(t, s.setupDone(&fleet.SetupExperienceStatusPayload{}), "empty status after the grace period means nothing to install")

	require.False(t, s.setupDone(&fleet.SetupExperienceStatusPayload{Software: []*fleet.SetupExperienceStatusResult{sw(fleet.SetupExperienceStatusSuccess), sw(fleet.SetupExperienceStatusRunning)}}))
	require.False(t, s.setupDone(&fleet.SetupExperienceStatusPayload{Script: sw(fleet.SetupExperienceStatusPending)}))
	require.True(t, s.setupDone(&fleet.SetupExperienceStatusPayload{
		Software: []*fleet.SetupExperienceStatusResult{sw(fleet.SetupExperienceStatusSuccess), sw(fleet.SetupExperienceStatusFailure)},
		Script:   sw(fleet.SetupExperienceStatusSuccess),
	}))
	require.False(t, s.setupDone(&fleet.SetupExperienceStatusPayload{
		Software:           []*fleet.SetupExperienceStatusResult{sw(fleet.SetupExperienceStatusSuccess), sw(fleet.SetupExperienceStatusFailure)},
		RequireAllSoftware: true,
	}), "a failed install holds setup when all software is required")
	require.False(t, s.setupDone(&fleet.SetupExperienceStatusPayload{
		Software:           []*fleet.SetupExperienceStatusResult{sw(fleet.SetupExperienceStatusCancelled)},
		RequireAllSoftware: true,
	}))
}

func TestUniqueUsername(t *testing.T) {
	taken := map[string]struct{}{"administrator": {}, "luke": {}, "luke2": {}, "abcdefghijklmnopqrst": {}}
	exists := func(n string) bool { _, ok := taken[n]; return ok }
	require.Equal(t, "jdoe", uniqueUsername("jdoe", exists))
	require.Equal(t, "luke3", uniqueUsername("luke", exists))
	require.Equal(t, "administrator2", uniqueUsername("administrator", exists))
	require.Equal(t, "abcdefghijklmnopqrs2", uniqueUsername("abcdefghijklmnopqrst", exists), "suffix stays within 20 characters")
	require.Empty(t, uniqueUsername("", exists))
	require.Empty(t, uniqueUsername("x", func(string) bool { return true }))
}

func TestWindowsUsernameFromIdP(t *testing.T) {
	cases := []struct {
		name string
		idp  fleet.OrbitEndUserIdPAccount
		want string
	}{
		{"username preferred", fleet.OrbitEndUserIdPAccount{Username: "jdoe", Email: "john@example.com"}, "jdoe"},
		{"email fallback", fleet.OrbitEndUserIdPAccount{Email: "john@example.com"}, "john"},
		{"plus tag dropped", fleet.OrbitEndUserIdPAccount{Email: "luke+windows@example.com"}, "luke"},
		{"username that is an email", fleet.OrbitEndUserIdPAccount{Username: "jane.doe@example.com"}, "jane.doe"},
		{"reserved characters stripped", fleet.OrbitEndUserIdPAccount{Username: `a"b/c\d[e]f:g;h|i=j,k*l?m<n>o p`}, "abcdefghijklmnop"},
		{"truncated to 20", fleet.OrbitEndUserIdPAccount{Username: "abcdefghijklmnopqrstuvwxyz"}, "abcdefghijklmnopqrst"},
		{"truncated on a rune boundary", fleet.OrbitEndUserIdPAccount{Username: "ééééééééééééééééééééé"}, "éééééééééééééééééééé"},
		{"surrogate pairs count as two", fleet.OrbitEndUserIdPAccount{Username: "abcdefghijklmnopqrs😀"}, "abcdefghijklmnopqrs"},
		{"nothing usable", fleet.OrbitEndUserIdPAccount{}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, windowsUsernameFromIdP(&c.idp))
		})
	}
}
