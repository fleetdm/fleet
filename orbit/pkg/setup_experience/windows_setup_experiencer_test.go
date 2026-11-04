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

	t.Run("in progress is not finished", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, WriteSetupExperienceStatusFile(dir, &SetupExperienceInfo{TimeInitiated: time.Now(), Enabled: true}))
		s := NewWindowsSetupExperiencer(&noopWindowsInitiator{}, dir, nil, nil)
		require.False(t, s.finished)
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
	s.statusPollInterval = time.Millisecond
	s.cachedIdP = &fleet.OrbitEndUserIdPAccount{Email: "luke@example.com"}

	s.watchSoftware()

	require.Equal(t, 3, client.calls)
	require.NotNil(t, s.passwordPage)
	t.Cleanup(func() { _ = s.passwordPage.Close() })
	require.True(t, strings.HasPrefix(opened, "http://127.0.0.1:"), opened)
	require.Equal(t, s.passwordPage.URL(), opened)
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
