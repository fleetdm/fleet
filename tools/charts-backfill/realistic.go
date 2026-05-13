package main

import (
	"database/sql"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
)

// Churn profile names.
const (
	profileStable     = "stable"
	profileSingleFlip = "single_flip"
	profileActive     = "active"
)

// Host-count band names.
const (
	bandNarrow = "narrow"
	bandMedium = "medium"
	bandBroad  = "broad"
)

// Profile assignment weights (cumulative).
//
//	stable      [0, 0.20)
//	single_flip [0.20, 0.60)
//	active      [0.60, 1.00)
const (
	profileStableCutoff     = 0.20
	profileSingleFlipCutoff = 0.60
)

// Host-count band weights (cumulative).
const (
	bandNarrowCutoff = 0.70
	bandMediumCutoff = 0.95
)

// Per-flip membership delta size (uniform in [1, maxDeltaSize]).
const maxDeltaSize = 3

// Active CVE flip-count range: total rows in [activeRowsMin, activeRowsMax].
// Number of flips = totalRows - 1.
const (
	activeRowsMin = 3
	activeRowsMax = 14
)

// Spike injection: 2-3 spike days per 7-day window.
const (
	spikesPerWeekMin = 2
	spikesPerWeekMax = 3
)

// Spike injection: promote 25-35% of eligible candidates per spike day.
const (
	spikePromoteFracMin = 0.25
	spikePromoteFracMax = 0.35
)

// cvePlan describes one CVE's planned evolution across the window.
type cvePlan struct {
	entityID string
	profile  string
	band     string
	events   []scdEvent
	spiked   bool
}

// scdEvent is a state-change point: from this dayOffset until the next event
// (or window end), the CVE's affected-host set is hostSet.
type scdEvent struct {
	dayOffset int
	hostSet   []uint
}

// scdRow is the closed-form row that gets written to host_scd_data.
type scdRow struct {
	entityID string
	hostSet  []uint
	fromDay  int // window-relative day offset, inclusive
	toDay    int // window-relative day offset, exclusive
}

// pickProfile returns a churn profile weighted per the constants above.
func pickProfile(rng *rand.Rand) string {
	r := rng.Float64()
	switch {
	case r < profileStableCutoff:
		return profileStable
	case r < profileSingleFlipCutoff:
		return profileSingleFlip
	default:
		return profileActive
	}
}

// pickHostBand returns a host-count band weighted per the constants above.
func pickHostBand(rng *rand.Rand) string {
	r := rng.Float64()
	switch {
	case r < bandNarrowCutoff:
		return bandNarrow
	case r < bandMediumCutoff:
		return bandMedium
	default:
		return bandBroad
	}
}

// bandFraction returns a random fraction-of-fleet within the band's range.
func bandFraction(rng *rand.Rand, band string) float64 {
	switch band {
	case bandNarrow:
		return 0.001 + rng.Float64()*(0.05-0.001)
	case bandMedium:
		return 0.05 + rng.Float64()*(0.25-0.05)
	case bandBroad:
		return 0.25 + rng.Float64()*(1.00-0.25)
	default:
		return 0.05
	}
}

// initialHostSet samples a host subset sized by fraction of the fleet.
func initialHostSet(rng *rand.Rand, hostIDs []uint, fraction float64) []uint {
	count := int(float64(len(hostIDs)) * fraction)
	count = max(count, 1)
	count = min(count, len(hostIDs))
	perm := rng.Perm(len(hostIDs))
	out := make([]uint, count)
	for i, idx := range perm[:count] {
		out[i] = hostIDs[idx]
	}
	return out
}

// evolveSet applies deltaSize membership changes to current. Each unit is
// either an add or remove. Boundary rules: empty → force add; full → force
// remove. Returned slice is a fresh allocation; current is not mutated.
func evolveSet(rng *rand.Rand, current []uint, allHosts []uint, deltaSize int) []uint {
	out := append([]uint(nil), current...)
	inSet := make(map[uint]bool, len(out))
	for _, h := range out {
		inSet[h] = true
	}
	for range deltaSize {
		var add bool
		switch {
		case len(out) == 0:
			add = true
		case len(out) >= len(allHosts):
			add = false
		default:
			add = rng.IntN(2) == 0
		}
		if add {
			// Try up to 50 random picks for a not-yet-present host.
			for range 50 {
				h := allHosts[rng.IntN(len(allHosts))]
				if !inSet[h] {
					out = append(out, h)
					inSet[h] = true
					break
				}
			}
		} else {
			idx := rng.IntN(len(out))
			removed := out[idx]
			out[idx] = out[len(out)-1]
			out = out[:len(out)-1]
			delete(inSet, removed)
		}
	}
	// Per-iteration boundary checks can still leave the set empty if the
	// last action removed the only remaining host. host_bitmap is NOT NULL,
	// and HostIDsToBlob returns nil for empty input, so force a single add
	// to guarantee a non-empty result.
	if len(out) == 0 {
		for range 50 {
			h := allHosts[rng.IntN(len(allHosts))]
			if !inSet[h] {
				out = append(out, h)
				inSet[h] = true
				break
			}
		}
	}
	return out
}

// pickFlipDays returns `count` distinct day offsets in [1, days-1], sorted.
// If count exceeds the available range, returns the full range.
func pickFlipDays(rng *rand.Rand, days, count int) []int {
	maxPicks := days - 1
	if maxPicks < 1 {
		return nil
	}
	if count >= maxPicks {
		out := make([]int, maxPicks)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}
	pool := make([]int, maxPicks)
	for i := range pool {
		pool[i] = i + 1
	}
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	out := pool[:count]
	sort.Ints(out)
	return out
}

// planCVECatalog generates one CVE plan per entityID evolving over `days`
// days, before spike injection.
func planCVECatalog(rng *rand.Rand, entityIDs []string, days int, hostIDs []uint) []*cvePlan {
	plans := make([]*cvePlan, 0, len(entityIDs))
	for _, id := range entityIDs {
		profile := pickProfile(rng)
		band := pickHostBand(rng)
		initial := initialHostSet(rng, hostIDs, bandFraction(rng, band))

		plan := &cvePlan{
			entityID: id,
			profile:  profile,
			band:     band,
			events:   []scdEvent{{dayOffset: 0, hostSet: initial}},
		}

		switch profile {
		case profileSingleFlip:
			if days > 1 {
				flipDay := 1 + rng.IntN(days-1)
				delta := 1 + rng.IntN(maxDeltaSize)
				evolved := evolveSet(rng, initial, hostIDs, delta)
				plan.events = append(plan.events, scdEvent{dayOffset: flipDay, hostSet: evolved})
			}
		case profileActive:
			totalRows := activeRowsMin + rng.IntN(activeRowsMax-activeRowsMin+1)
			flipCount := totalRows - 1
			flipDays := pickFlipDays(rng, days, flipCount)
			current := initial
			for _, fd := range flipDays {
				delta := 1 + rng.IntN(maxDeltaSize)
				current = evolveSet(rng, current, hostIDs, delta)
				plan.events = append(plan.events, scdEvent{dayOffset: fd, hostSet: current})
			}
		}
		plans = append(plans, plan)
	}
	return plans
}

// pickSpikeDays returns the day offsets for spike events: 2-3 per 7-day week
// of the window, distinct, sorted, in [1, days-1].
func pickSpikeDays(rng *rand.Rand, days int) []int {
	if days < 2 {
		return nil
	}
	weeks := max(days/7, 1)
	used := make(map[int]bool)
	for range weeks {
		n := spikesPerWeekMin + rng.IntN(spikesPerWeekMax-spikesPerWeekMin+1)
		for range n {
			for range 20 {
				d := 1 + rng.IntN(days-1)
				if !used[d] {
					used[d] = true
					break
				}
			}
		}
	}
	out := make([]int, 0, len(used))
	for d := range used {
		out = append(out, d)
	}
	sort.Ints(out)
	return out
}

// injectSpikes promotes eligible stable/single_flip CVEs by inserting a new
// flip event at each spike day. Active CVEs are excluded. Each CVE can be
// promoted by at most one spike (tracked via cvePlan.spiked).
func injectSpikes(rng *rand.Rand, plans []*cvePlan, spikeDays []int, hostIDs []uint) {
	for _, sd := range spikeDays {
		candidates := make([]*cvePlan, 0, len(plans))
		for _, p := range plans {
			if p.spiked {
				continue
			}
			if p.profile == profileActive {
				continue
			}
			// Skip if this CVE already has an event landing exactly on sd.
			hasAtSd := false
			for _, e := range p.events {
				if e.dayOffset == sd {
					hasAtSd = true
					break
				}
			}
			if hasAtSd {
				continue
			}
			candidates = append(candidates, p)
		}
		if len(candidates) == 0 {
			continue
		}
		pct := spikePromoteFracMin + rng.Float64()*(spikePromoteFracMax-spikePromoteFracMin)
		promoteCount := int(float64(len(candidates)) * pct)
		if promoteCount == 0 {
			promoteCount = 1
		}
		rng.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		for _, p := range candidates[:promoteCount] {
			// Every plan has a day-0 seed event, and spike days are ≥ 1, so
			// the seed always precedes sd. Initialize from the seed and let
			// the loop advance to the latest pre-sd event.
			prevSet := p.events[0].hostSet
			insertIdx := len(p.events)
			for i, e := range p.events {
				if e.dayOffset < sd {
					prevSet = e.hostSet
				} else {
					insertIdx = i
					break
				}
			}
			delta := 1 + rng.IntN(maxDeltaSize)
			evolved := evolveSet(rng, prevSet, hostIDs, delta)
			p.events = append(p.events, scdEvent{})
			copy(p.events[insertIdx+1:], p.events[insertIdx:])
			p.events[insertIdx] = scdEvent{dayOffset: sd, hostSet: evolved}
			p.spiked = true
		}
	}
}

// trackedCVESoftwareMatcher mirrors the curated list in
// server/chart/internal/mysql/charts.go. Kept here to avoid a heavy import.
type trackedCVESoftwareMatcher struct {
	Pattern string
	Sources []string // empty means any source
}

// trackedCVESoftwareMatchers is the matcher list duplicated from
// server/chart/internal/mysql/charts.go. Keep these in sync — load tests
// generate data for the same curated set the chart reads.
var trackedCVESoftwareMatchers = []trackedCVESoftwareMatcher{
	{"Google Chrome%", nil},
	{"Firefox%", nil},
	{"Mozilla Firefox%", nil},
	{"Brave Browser%", nil},
	{"Safari%", []string{"apps"}},
	{"Opera%", nil},
	{"Microsoft Word%", nil},
	{"Microsoft Excel%", nil},
	{"Microsoft PowerPoint%", nil},
	{"Microsoft Outlook%", nil},
	{"Microsoft Office%", nil},
	{"Adobe Flash%", nil},
	{"Shockwave Flash%", nil},
	{"Adobe Acrobat%", nil},
	{"linux-image-%", []string{"deb_packages"}},
	{"linux-signed-image-%", []string{"deb_packages"}},
	{"kernel-%", []string{"rpm_packages"}},
}

// queryTrackedCVEIDs returns the deduplicated set of CVE IDs the chart
// currently considers "tracked" (matching curated software list with CVSS
// >= 9.0, plus all OS-vuln CVEs at the same threshold). Mirrors the query
// in server/chart/internal/mysql/charts.go TrackedCriticalCVEs.
func queryTrackedCVEIDs(db *sql.DB) ([]string, error) {
	const criticalCVSS = 9.0
	set := make(map[string]struct{})

	args := []any{criticalCVSS}
	clauses := make([]string, 0, len(trackedCVESoftwareMatchers))
	for _, m := range trackedCVESoftwareMatchers {
		if len(m.Sources) == 0 {
			clauses = append(clauses, "s.name LIKE ?")
			args = append(args, m.Pattern)
		} else {
			placeholders := strings.TrimRight(strings.Repeat("?,", len(m.Sources)), ",")
			clauses = append(clauses, fmt.Sprintf("(s.name LIKE ? AND s.source IN (%s))", placeholders))
			args = append(args, m.Pattern)
			for _, src := range m.Sources {
				args = append(args, src)
			}
		}
	}
	softwareQuery := `
		SELECT DISTINCT sc.cve
		FROM software_cve sc
		JOIN software s  ON s.id = sc.software_id
		JOIN cve_meta cm ON cm.cve = sc.cve
		WHERE cm.cvss_score >= ?
		  AND (` + strings.Join(clauses, " OR ") + `)`
	if err := streamCVEs(db, softwareQuery, args, set); err != nil {
		return nil, fmt.Errorf("query software-side tracked CVEs: %w", err)
	}

	const osQuery = `
		SELECT DISTINCT osv.cve
		FROM operating_system_vulnerabilities osv
		JOIN cve_meta cm ON cm.cve = osv.cve
		WHERE cm.cvss_score >= ?`
	if err := streamCVEs(db, osQuery, []any{criticalCVSS}, set); err != nil {
		return nil, fmt.Errorf("query os-side tracked CVEs: %w", err)
	}

	out := make([]string, 0, len(set))
	for cve := range set {
		out = append(out, cve)
	}
	sort.Strings(out)
	return out, nil
}

func streamCVEs(db *sql.DB, query string, args []any, out map[string]struct{}) error {
	rows, err := db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cve string
		if err := rows.Scan(&cve); err != nil {
			return err
		}
		out[cve] = struct{}{}
	}
	return rows.Err()
}

// pickCatalog selects up to `want` entity IDs for the realistic catalog.
// If real tracked CVEs are available, samples from those; otherwise falls
// back to synthetic IDs (which won't appear in the chart but allow the
// tool to run on a fresh DB without crashing).
func pickCatalog(rng *rand.Rand, realCVEs []string, want int) ([]string, bool) {
	if len(realCVEs) == 0 {
		out := make([]string, want)
		for i := range out {
			out[i] = fmt.Sprintf("CVE-2024-%05d", i+1)
		}
		return out, false
	}
	if len(realCVEs) <= want {
		// Use everything we have. Return a shuffled copy for stable iteration.
		out := append([]string(nil), realCVEs...)
		rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
		return out, true
	}
	// Sample `want` from realCVEs.
	rng.Shuffle(len(realCVEs), func(i, j int) { realCVEs[i], realCVEs[j] = realCVEs[j], realCVEs[i] })
	return append([]string(nil), realCVEs[:want]...), true
}

// plansToRows converts the per-CVE event lists into closed-form scdRow values
// suitable for inserting into host_scd_data.
func plansToRows(plans []*cvePlan, days int) []scdRow {
	rows := make([]scdRow, 0, len(plans))
	for _, p := range plans {
		for i, e := range p.events {
			toDay := days
			if i+1 < len(p.events) {
				toDay = p.events[i+1].dayOffset
			}
			rows = append(rows, scdRow{
				entityID: p.entityID,
				hostSet:  e.hostSet,
				fromDay:  e.dayOffset,
				toDay:    toDay,
			})
		}
	}
	return rows
}
