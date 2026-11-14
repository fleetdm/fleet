package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestGoogleMarkRetryDue(t *testing.T) {
	stored := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	failed := func(lastFailure time.Duration) fleet.GoogleConditionalAccessDeviceUser {
		return fleet.GoogleConditionalAccessDeviceUser{
			ErrorMessage: new("Google returned HTTP 500"), CreatedAt: stored, UpdatedAt: stored.Add(lastFailure),
		}
	}
	cases := []struct {
		name string
		du   fleet.GoogleConditionalAccessDeviceUser
		at   time.Duration // since the last change
		due  bool
	}{
		{"just stored", fleet.GoogleConditionalAccessDeviceUser{CreatedAt: stored, UpdatedAt: stored}, 30 * time.Second, false},
		{"stored a minute ago", fleet.GoogleConditionalAccessDeviceUser{CreatedAt: stored, UpdatedAt: stored}, time.Minute, true},
		{"failed at once, waits a minute", failed(0), 30 * time.Second, false},
		{"failed at once, after a minute", failed(0), time.Minute, true},
		{"failed after 10m, waits 10m", failed(10 * time.Minute), 9 * time.Minute, false},
		{"failed after 10m, after 10m", failed(10 * time.Minute), 10 * time.Minute, true},
		{"failing for a day, waits an hour", failed(24 * time.Hour), 59 * time.Minute, false},
		{"failing for a day, after an hour", failed(24 * time.Hour), time.Hour, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.due, googleMarkRetryDue(c.du, c.du.UpdatedAt.Add(c.at)))
		})
	}
}
