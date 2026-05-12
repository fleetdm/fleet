package main

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRNG returns a deterministic Rand for reproducible tests.
func newTestRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0xa5a5a5a5))
}

// fleet returns a synthetic host-ID slice of size n.
func fleet(n int) []uint {
	out := make([]uint, n)
	for i := range out {
		out[i] = uint(i + 1)
	}
	return out
}

func TestPlanCVECatalog_RowCountsPerProfile(t *testing.T) {
	const (
		cveCount = 1000
		days     = 30
		hosts    = 200
	)
	rng := newTestRNG(1)
	plans := planCVECatalog(rng, cveCount, days, fleet(hosts))
	require.Len(t, plans, cveCount)

	var stable, single, active int
	for _, p := range plans {
		switch p.profile {
		case profileStable:
			stable++
			assert.Len(t, p.events, 1, "stable CVE %s has wrong event count", p.entityID)
		case profileSingleFlip:
			single++
			assert.Len(t, p.events, 2, "single_flip CVE %s has wrong event count", p.entityID)
		case profileActive:
			active++
			// activeRowsMin..activeRowsMax total rows = events
			assert.GreaterOrEqual(t, len(p.events), activeRowsMin)
			assert.LessOrEqual(t, len(p.events), activeRowsMax)
		default:
			t.Fatalf("unexpected profile %q on %s", p.profile, p.entityID)
		}
	}

	// Weights are 20/40/40. Assert within ±5 percentage points (loose enough
	// to avoid flakes; tight enough to catch a flipped cutoff).
	assertWeight := func(name string, count, total int, want float64) {
		t.Helper()
		got := float64(count) / float64(total)
		assert.InDelta(t, want, got, 0.05, "weight for %s: got %.3f want %.3f", name, got, want)
	}
	assertWeight("stable", stable, cveCount, 0.20)
	assertWeight("single_flip", single, cveCount, 0.40)
	assertWeight("active", active, cveCount, 0.40)
}

func TestPlanCVECatalog_OutputMagnitude(t *testing.T) {
	const (
		cveCount = 1000
		days     = 30
		hosts    = 200
	)
	rng := newTestRNG(2)
	plans := planCVECatalog(rng, cveCount, days, fleet(hosts))
	spikeDays := pickSpikeDays(rng, days)
	injectSpikes(rng, plans, spikeDays, fleet(hosts))
	rows := plansToRows(plans, days)

	// Design target: 6-9k rows for 1k CVEs over 30 days. Allow some slack on
	// either side because spike count varies per seed.
	assert.GreaterOrEqual(t, len(rows), 5000, "row count below expected band")
	assert.LessOrEqual(t, len(rows), 11000, "row count above expected band")
}

func TestPickSpikeDays_WeeklyCadence(t *testing.T) {
	rng := newTestRNG(3)
	days := 30
	spikes := pickSpikeDays(rng, days)
	weeks := days / 7

	// Allow some tolerance: 2-3 per week × ~4 weeks = 8-12, but the random
	// retry loop in pickSpikeDays can dedupe a few. Use a reasonable band.
	assert.GreaterOrEqual(t, len(spikes), weeks*spikesPerWeekMin-1)
	assert.LessOrEqual(t, len(spikes), weeks*spikesPerWeekMax+1)

	// All distinct.
	seen := make(map[int]struct{})
	for _, d := range spikes {
		_, dup := seen[d]
		assert.False(t, dup, "duplicate spike day %d", d)
		seen[d] = struct{}{}
		assert.GreaterOrEqual(t, d, 1)
		assert.LessOrEqual(t, d, days-1)
	}
}

func TestInjectSpikes_PromotesEligible(t *testing.T) {
	rng := newTestRNG(4)
	hosts := fleet(200)
	plans := planCVECatalog(rng, 500, 30, hosts)

	// Count pre-spike profile distribution.
	preActive := 0
	for _, p := range plans {
		if p.profile == profileActive {
			preActive++
		}
	}

	spikeDays := pickSpikeDays(rng, 30)
	require.NotEmpty(t, spikeDays)
	injectSpikes(rng, plans, spikeDays, hosts)

	// Active CVEs are never promoted.
	for _, p := range plans {
		if p.profile == profileActive {
			assert.False(t, p.spiked, "active CVE %s was promoted", p.entityID)
		}
	}

	// Total promoted CVEs > 0. (Sanity that injection ran.)
	promoted := 0
	for _, p := range plans {
		if p.spiked {
			promoted++
		}
	}
	assert.Positive(t, promoted, "no CVEs were promoted")
}

func TestInjectSpikes_PromoteCountInBand(t *testing.T) {
	rng := newTestRNG(5)
	hosts := fleet(200)
	plans := planCVECatalog(rng, 2000, 30, hosts)

	// Pick a single spike day so we can isolate per-spike promotion count.
	spikeDays := []int{15}

	// Snapshot eligibility before injection.
	eligible := 0
	for _, p := range plans {
		if p.profile == profileActive {
			continue
		}
		hasAt := false
		for _, e := range p.events {
			if e.dayOffset == 15 {
				hasAt = true
				break
			}
		}
		if !hasAt {
			eligible++
		}
	}

	injectSpikes(rng, plans, spikeDays, hosts)

	promoted := 0
	for _, p := range plans {
		if p.spiked {
			promoted++
		}
	}
	frac := float64(promoted) / float64(eligible)
	assert.GreaterOrEqual(t, frac, spikePromoteFracMin-0.01, "promote fraction below band")
	assert.LessOrEqual(t, frac, spikePromoteFracMax+0.01, "promote fraction above band")
}

func TestEvolveSet_EmptyForcesAdd(t *testing.T) {
	// With delta=1 from an empty start, the only legal move is "add",
	// so the result must be size 1. (Multi-step calls can drift up and
	// back down, so testing the single-step boundary is the contract.)
	rng := newTestRNG(6)
	hosts := fleet(100)
	out := evolveSet(rng, []uint{}, hosts, 1)
	assert.Len(t, out, 1, "empty input + delta=1 should add exactly 1 host")
}

func TestEvolveSet_FullForcesRemove(t *testing.T) {
	rng := newTestRNG(7)
	hosts := fleet(20)
	full := append([]uint(nil), hosts...)
	out := evolveSet(rng, full, hosts, 1)
	assert.Len(t, out, len(hosts)-1, "full input + delta=1 should remove exactly 1 host")
}

func TestEvolveSet_NarrowBandDoesNotDriftToEmpty(t *testing.T) {
	// Many iterations on a narrow-band CVE should not collapse the set to
	// empty: the empty-boundary force-add saves it whenever it does briefly
	// hit zero. Assert that after many evolutions, the set is non-empty.
	rng := newTestRNG(11)
	hosts := fleet(1000)
	current := []uint{42, 99} // narrow band starting set
	for range 100 {
		current = evolveSet(rng, current, hosts, 2)
	}
	assert.NotEmpty(t, current, "narrow-band CVE drifted to empty across many flips")
}

func TestEvolveSet_DoesNotMutateInput(t *testing.T) {
	rng := newTestRNG(8)
	hosts := fleet(100)
	current := []uint{1, 2, 3, 4, 5}
	snapshot := append([]uint(nil), current...)
	_ = evolveSet(rng, current, hosts, 3)
	assert.Equal(t, snapshot, current, "evolveSet mutated its input slice")
}

func TestEvolveSet_DeltaSize(t *testing.T) {
	rng := newTestRNG(9)
	hosts := fleet(100)
	current := []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	out := evolveSet(rng, current, hosts, 4)
	// Net size change is at most ±deltaSize; could be smaller if adds and
	// removes cancel out. Use absolute bound check.
	diff := len(out) - len(current)
	assert.LessOrEqual(t, diff, 4)
	assert.GreaterOrEqual(t, diff, -4)
}

func TestPlansToRows_StableSingleRow(t *testing.T) {
	plans := []*cvePlan{
		{
			entityID: "CVE-1",
			profile:  profileStable,
			events:   []scdEvent{{dayOffset: 0, hostSet: []uint{1, 2}}},
		},
	}
	rows := plansToRows(plans, 30)
	require.Len(t, rows, 1)
	assert.Equal(t, 0, rows[0].fromDay)
	assert.Equal(t, 30, rows[0].toDay)
}

func TestPlansToRows_MultiEventRowsClose(t *testing.T) {
	plans := []*cvePlan{
		{
			entityID: "CVE-1",
			profile:  profileSingleFlip,
			events: []scdEvent{
				{dayOffset: 0, hostSet: []uint{1, 2}},
				{dayOffset: 12, hostSet: []uint{1, 2, 3}},
			},
		},
	}
	rows := plansToRows(plans, 30)
	require.Len(t, rows, 2)
	assert.Equal(t, 0, rows[0].fromDay)
	assert.Equal(t, 12, rows[0].toDay)
	assert.Equal(t, 12, rows[1].fromDay)
	assert.Equal(t, 30, rows[1].toDay)
}
