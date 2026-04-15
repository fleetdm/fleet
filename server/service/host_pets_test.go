package service

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestHungerFromMetrics(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		seenAgo  time.Duration
		zeroSeen bool
		want     uint8
	}{
		{name: "never seen", zeroSeen: true, want: fleet.HostPetTargetHungerBaseline},
		{name: "fresh check-in (30 min)", seenAgo: 30 * time.Minute, want: fleet.HostPetTargetHungerFresh},
		{name: "boundary just under fresh", seenAgo: 59 * time.Minute, want: fleet.HostPetTargetHungerFresh},
		{name: "stale (3h)", seenAgo: 3 * time.Hour, want: fleet.HostPetTargetHungerStale},
		{name: "very stale (12h)", seenAgo: 12 * time.Hour, want: fleet.HostPetTargetHungerVeryStale},
		{name: "starving (3 days)", seenAgo: 72 * time.Hour, want: fleet.HostPetTargetHungerStarving},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := fleet.HostPetMetrics{}
			if !tc.zeroSeen {
				m.SeenTime = now.Add(-tc.seenAgo)
			}
			got := hungerFromMetrics(m, now)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCleanlinessFromMetrics(t *testing.T) {
	cases := []struct {
		name           string
		failingPolicy  uint
		wantAtLeastMin bool // for floor cases
		wantExact      uint8
	}{
		{name: "no failing policies", failingPolicy: 0, wantExact: fleet.HostPetTargetCleanlinessBaseline},
		{name: "1 failing policy", failingPolicy: 1, wantExact: fleet.HostPetTargetCleanlinessBaseline - fleet.HostPetCleanlinessPerFailingPolicy},
		{name: "3 failing policies", failingPolicy: 3, wantExact: fleet.HostPetTargetCleanlinessBaseline - 3*fleet.HostPetCleanlinessPerFailingPolicy},
		{name: "many failing policies clamps to floor", failingPolicy: 50, wantAtLeastMin: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cleanlinessFromMetrics(fleet.HostPetMetrics{FailingPolicyCount: tc.failingPolicy})
			if tc.wantAtLeastMin {
				assert.Equal(t, fleet.HostPetStatFloor, got)
			} else {
				assert.Equal(t, tc.wantExact, got)
			}
		})
	}
}

func TestHealthFromMetrics(t *testing.T) {
	cases := []struct {
		name          string
		critical      uint
		high          uint
		mdmUnenrolled bool
		want          uint8
	}{
		{
			name: "clean host",
			want: fleet.HostPetTargetHealthBaseline,
		},
		{
			name:     "1 critical vuln",
			critical: 1,
			want:     fleet.HostPetTargetHealthBaseline - fleet.HostPetHealthPerCriticalVuln,
		},
		{
			name: "5 high vulns",
			high: 5,
			want: fleet.HostPetTargetHealthBaseline - 5*fleet.HostPetHealthPerHighVuln,
		},
		{
			name:          "mdm unenrolled penalty",
			mdmUnenrolled: true,
			want:          fleet.HostPetTargetHealthBaseline - fleet.HostPetHealthMDMUnenrolledPenalty,
		},
		{
			name:          "stack of penalties clamps to floor",
			critical:      20,
			high:          20,
			mdmUnenrolled: true,
			want:          fleet.HostPetStatFloor,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := healthFromMetrics(fleet.HostPetMetrics{
				CriticalVulnCount: tc.critical,
				HighVulnCount:     tc.high,
				MDMUnenrolled:     tc.mdmUnenrolled,
			})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestHappinessTargetFromMetrics(t *testing.T) {
	tr := true
	fa := false
	cases := []struct {
		name string
		disk *bool
		want uint8
	}{
		{name: "unknown disk encryption", disk: nil, want: fleet.HostPetTargetHappinessBaseline},
		{name: "disk encryption on", disk: &tr, want: fleet.HostPetTargetHappinessBaseline + fleet.HostPetHappinessDiskEncOnBonus},
		{name: "disk encryption off", disk: &fa, want: fleet.HostPetTargetHappinessBaseline - fleet.HostPetHappinessDiskEncOffPenalty},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := happinessTargetFromMetrics(fleet.HostPetMetrics{DiskEncryptionEnabled: tc.disk})
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDecayedHappiness(t *testing.T) {
	cases := []struct {
		name    string
		current uint8
		target  uint8
		elapsed time.Duration
		want    uint8
	}{
		{name: "current at target stays put", current: 70, target: 70, elapsed: 24 * time.Hour, want: 70},
		{name: "no elapsed time means no movement", current: 90, target: 70, elapsed: 0, want: 90},
		{name: "negative elapsed clamped to zero", current: 90, target: 70, elapsed: -time.Hour, want: 90},
		{name: "1h decays by happinessDecayPerHour above target", current: 90, target: 70, elapsed: time.Hour, want: 90 - happinessDecayPerHour},
		{name: "5h overshoots target then clamps", current: 75, target: 70, elapsed: 5 * time.Hour, want: 70},
		{name: "5h slides up toward target", current: 50, target: 70, elapsed: 5 * time.Hour, want: 50 + 5*happinessDecayPerHour},
		{name: "month gap caps at decay window", current: 100, target: 50, elapsed: 30 * 24 * time.Hour, want: 50},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := decayedHappiness(tc.current, tc.target, tc.elapsed)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestApplyHostMetricsToPet_FullPipeline(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	tr := true

	pet := &fleet.HostPet{
		Health:           50,
		Happiness:        80,
		Hunger:           50,
		Cleanliness:      50,
		LastInteractedAt: now.Add(-2 * time.Hour),
	}
	m := fleet.HostPetMetrics{
		SeenTime:              now.Add(-30 * time.Minute), // fresh
		FailingPolicyCount:    2,
		CriticalVulnCount:     0,
		HighVulnCount:         0,
		DiskEncryptionEnabled: &tr,
		MDMUnenrolled:         false,
	}

	applyHostMetricsToPet(pet, m, now)

	// Hunger should snap to "fresh" band.
	assert.Equal(t, fleet.HostPetTargetHungerFresh, pet.Hunger)
	// Cleanliness: baseline - 2*15 = 60.
	assert.Equal(t, uint8(fleet.HostPetTargetCleanlinessBaseline-2*fleet.HostPetCleanlinessPerFailingPolicy), pet.Cleanliness)
	// Health: full baseline (no vulns, mdm enrolled).
	assert.Equal(t, fleet.HostPetTargetHealthBaseline, pet.Health)
	// Happiness target: baseline + disk-on bonus = 75. Current 80, 2h elapsed,
	// decay 2/h → 76.
	wantHappiness := uint8(80 - 2*happinessDecayPerHour)
	assert.Equal(t, wantHappiness, pet.Happiness)
	// Mood is computed.
	assert.NotEmpty(t, pet.Mood)
}

func TestHostPetDemoOverrides_Apply(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	overrideSeen := now.Add(-48 * time.Hour)

	m := fleet.HostPetMetrics{
		SeenTime:           now.Add(-30 * time.Minute),
		FailingPolicyCount: 1,
		CriticalVulnCount:  0,
		HighVulnCount:      0,
	}
	o := &fleet.HostPetDemoOverrides{
		SeenTimeOverride:     &overrideSeen,
		ExtraFailingPolicies: 4,
		ExtraCriticalVulns:   2,
		ExtraHighVulns:       3,
	}
	o.Apply(&m)

	assert.Equal(t, overrideSeen, m.SeenTime, "seen time override should replace, not stack")
	assert.Equal(t, uint(5), m.FailingPolicyCount, "extras should add to the real count")
	assert.Equal(t, uint(2), m.CriticalVulnCount)
	assert.Equal(t, uint(3), m.HighVulnCount)

	// Nil overrides is a safe no-op.
	var nilOverride *fleet.HostPetDemoOverrides
	nilOverride.Apply(&m) // must not panic
}

func TestComputeMood(t *testing.T) {
	cases := []struct {
		name string
		pet  fleet.HostPet
		want fleet.HostPetMood
	}{
		{name: "low health -> sick", pet: fleet.HostPet{Health: 20, Happiness: 80, Hunger: 30, Cleanliness: 80}, want: fleet.HostPetMoodSick},
		{name: "high hunger -> hungry", pet: fleet.HostPet{Health: 80, Happiness: 80, Hunger: 90, Cleanliness: 80}, want: fleet.HostPetMoodHungry},
		{name: "low cleanliness -> dirty", pet: fleet.HostPet{Health: 80, Happiness: 80, Hunger: 30, Cleanliness: 10}, want: fleet.HostPetMoodDirty},
		{name: "low happiness -> sad", pet: fleet.HostPet{Health: 80, Happiness: 20, Hunger: 30, Cleanliness: 80}, want: fleet.HostPetMoodSad},
		{name: "all good -> happy", pet: fleet.HostPet{Health: 90, Happiness: 90, Hunger: 10, Cleanliness: 90}, want: fleet.HostPetMoodHappy},
		{name: "middle of the road -> content", pet: fleet.HostPet{Health: 70, Happiness: 60, Hunger: 40, Cleanliness: 50}, want: fleet.HostPetMoodContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pet := tc.pet
			got := computeMood(&pet)
			assert.Equal(t, tc.want, got)
		})
	}
}
