package main

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/softwaredb"
	"github.com/fleetdm/fleet/v4/pkg/mdm/apnsmock"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHostUUIDs(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = uuid.NewString()
	}
	return out
}

func TestStableFraction(t *testing.T) {
	for _, u := range testHostUUIDs(100) {
		v := stableFraction(u, "k")
		require.GreaterOrEqual(t, v, 0.0)
		require.Less(t, v, 1.0)
		require.Equal(t, v, stableFraction(u, "k"))
	}
	u := uuid.NewString()
	assert.NotEqual(t, stableFraction(u, "a"), stableFraction(u, "b"))
}

func TestInstallPermafailCohort(t *testing.T) {
	hosts := testHostUUIDs(20000)
	for _, prob := range []float64{0, 0.05, 0.5, 1} {
		n := 0
		for _, u := range hosts {
			if inInstallPermafailCohort(u, prob) {
				n++
			}
		}
		assert.InDelta(t, prob, float64(n)/float64(len(hosts)), 0.02, "prob %v", prob)
	}

	for _, u := range hosts[:100] {
		require.False(t, inInstallPermafailCohort(u, 0))
		require.True(t, inInstallPermafailCohort(u, 1))
		require.Equal(t, inInstallPermafailCohort(u, 0.3), inInstallPermafailCohort(u, 0.3))
	}
}

func TestRunPolicySticky(t *testing.T) {
	const policies = 20
	hosts := testHostUUIDs(2000)
	passes := 0
	for _, u := range hosts {
		a := &agent{UUID: u, policyPassProb: 0.3, stickyPolicies: true}
		for p := range policies {
			name := fmt.Sprintf("fleet_policy_query_%d", p)
			first := len(a.runPolicy(name, "SELECT 1 FROM os_version"))
			for range 5 {
				require.Equal(t, first, len(a.runPolicy(name, "SELECT 1 FROM os_version")))
			}
			passes += first
		}
	}
	assert.InDelta(t, 0.3, float64(passes)/float64(len(hosts)*policies), 0.02)

	a := &agent{UUID: uuid.NewString(), policyPassProb: 0, stickyPolicies: true}
	assert.Len(t, a.runPolicy("fleet_policy_query_1", "select 1;"), 1)
	a.policyPassProb = 1
	assert.Empty(t, a.runPolicy("fleet_policy_query_1", "SELECT 0"))
}

func TestRunPolicyNotSticky(t *testing.T) {
	a := &agent{UUID: uuid.NewString(), policyPassProb: 0.5}
	seen := map[int]bool{}
	for range 200 {
		seen[len(a.runPolicy("fleet_policy_query_1", "SELECT 1 FROM os_version"))] = true
	}
	assert.Len(t, seen, 2, "without sticky results a host re-rolls on every report")
}

func TestHostSeed(t *testing.T) {
	a, b := hostSeed{seed: 42, index: 7}, hostSeed{seed: 42, index: 7}
	require.Equal(t, a.uuid("host-uuid"), b.uuid("host-uuid"))
	require.Equal(t, a.str("serial", serialCharset, 12), b.str("serial", serialCharset, 12))
	require.Equal(t, a.fraction("mdm"), b.fraction("mdm"))

	_, err := uuid.Parse(a.uuid("host-uuid"))
	require.NoError(t, err)
	assert.NotEqual(t, a.uuid("host-uuid"), a.uuid("device-auth-token"))

	serial := a.str("serial", serialCharset, 12)
	require.Len(t, serial, 12)
	for _, c := range serial {
		require.Contains(t, serialCharset, string(c))
	}

	// Neighbouring hosts, and the same host index in another container, are different hosts.
	seen := map[string]bool{}
	for seed := int64(1); seed <= 20; seed++ {
		for index := 1; index <= 500; index++ {
			u := hostSeed{seed: seed, index: index}.uuid("host-uuid")
			require.False(t, seen[u], "duplicate UUID for seed %d index %d", seed, index)
			seen[u] = true
		}
	}
}

func TestCachedStringStableForSeed(t *testing.T) {
	id := hostSeed{seed: 3, index: 11}
	first := (&agent{id: id, strings: map[string]string{}}).CachedString("hostname")
	again := (&agent{id: id, strings: map[string]string{}}).CachedString("hostname")
	require.Equal(t, first, again)
	other := (&agent{id: hostSeed{seed: 3, index: 12}, strings: map[string]string{}}).CachedString("hostname")
	assert.NotEqual(t, first, other)

	ios := (&mdmAgent{id: id, strings: map[string]string{}}).CachedString("hostname")
	require.Equal(t, ios, (&mdmAgent{id: id, strings: map[string]string{}}).CachedString("hostname"))
}

func TestSelectKernelsStableForSeed(t *testing.T) {
	kernels := make([]map[string]string, 10)
	for i := range kernels {
		kernels[i] = map[string]string{"version": fmt.Sprint(i)}
	}
	for index := 1; index <= 50; index++ {
		id := hostSeed{seed: 9, index: index}
		require.Equal(t, selectKernels(id.rng("kernels"), kernels), selectKernels(id.rng("kernels"), kernels))
	}
}

func TestScriptExitCode(t *testing.T) {
	const n = 20000
	for _, prob := range []float64{0, 0.2, 1} {
		a := &agent{scriptExecFailProb: prob}
		fails := 0
		for range n {
			fails += a.scriptExitCode()
		}
		assert.InDelta(t, prob, float64(fails)/n, 0.02, "prob %v", prob)
	}
}

// changedShare is the share of before's items missing from after.
func changedShare[T comparable](before, after []T) float64 {
	in := make(map[T]struct{}, len(after))
	for _, v := range after {
		in[v] = struct{}{}
	}
	gone := 0
	for _, v := range before {
		if _, ok := in[v]; !ok {
			gone++
		}
	}
	return float64(gone) / float64(len(before))
}

func TestStableSoftwareIndices(t *testing.T) {
	const pool, count, churn = 50000, 2000, 0.03
	start := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	id := hostSeed{seed: 4, index: 2}

	first := stableSoftwareIndices(id, "software-slots", count, pool, churn, start)
	require.Len(t, first, count)
	require.Len(t, slices.Compact(slices.Sorted(slices.Values(first))), count, "indices must be distinct")
	for _, i := range first {
		require.Less(t, i, uint32(pool))
	}

	// The same host at the same moment, e.g. after a restart, reports the same inventory.
	require.Equal(t, first, stableSoftwareIndices(id, "software-slots", count, pool, churn, start))
	assert.Greater(t, changedShare(first, stableSoftwareIndices(hostSeed{seed: 4, index: 3}, "software-slots", count, pool, churn, start)), 0.9)

	// An hourly refresh changes almost nothing, and each day changes about churn of the inventory.
	assert.Less(t, changedShare(first, stableSoftwareIndices(id, "software-slots", count, pool, churn, start.Add(time.Hour))), 0.005)
	var total float64
	prev := first
	const days = 30
	for d := 1; d <= days; d++ {
		cur := stableSoftwareIndices(id, "software-slots", count, pool, churn, start.Add(time.Duration(d)*24*time.Hour))
		require.Len(t, cur, count)
		total += changedShare(prev, cur)
		prev = cur
	}
	assert.InDelta(t, churn, total/days, 0.005)

	frozen := stableSoftwareIndices(id, "software-slots", count, pool, 0, start)
	assert.Equal(t, frozen, stableSoftwareIndices(id, "software-slots", count, pool, 0, start.Add(90*24*time.Hour)))

	assert.Len(t, stableSoftwareIndices(id, "software-slots", count, 100, churn, start), 100)
	assert.Empty(t, stableSoftwareIndices(id, "software-slots", count, 0, churn, start))
}

func TestStableSoftwareInventory(t *testing.T) {
	orig := softwareDB
	t.Cleanup(func() { softwareDB = orig })
	softwareDB = &softwaredb.DB{
		Darwin:  make([]softwaredb.DarwinSoftware, 5000),
		Windows: make([]softwaredb.WindowsSoftware, 5000),
		Ubuntu:  make([]softwaredb.UbuntuSoftware, 5000),
	}
	for i := range 5000 {
		name := fmt.Sprintf("app-%d", i)
		softwareDB.Darwin[i] = softwaredb.DarwinSoftware{Name: name, Version: "1.0", Source: "apps"}
		softwareDB.Windows[i] = softwaredb.WindowsSoftware{Name: name, Version: "1.0", Source: "programs"}
		softwareDB.Ubuntu[i] = softwaredb.UbuntuSoftware{Name: name, Version: "1.0", Source: "deb_packages"}
	}

	newHost := func(os string, index int) *agent {
		return &agent{
			os: os, id: hostSeed{seed: 5, index: index}, softwareChurnPerDay: 0.03,
			strings: map[string]string{}, softwareVersionMap: map[rune]int{}, cachedLastOpenedAt: map[string]*time.Time{},
			softwareCount: softwareEntityCount{entityCount: entityCount{common: 10, unique: 1}, vulnerable: 10},
		}
	}
	queries := map[string]string{
		"macos":   "fleet_detail_query_software_macos",
		"windows": "fleet_detail_query_software_windows",
		"ubuntu":  "fleet_detail_query_software_linux",
	}
	report := func(a *agent) []string {
		_, rows, _, _, _ := a.processQuery(queries[a.os], "", &cachedResults{})
		keys := make([]string, 0, len(rows))
		for _, r := range rows {
			keys = append(keys, r["name"]+"|"+r["version"]+"|"+r["source"])
		}
		slices.Sort(keys)
		return keys
	}

	for os := range queries {
		for index := 1; index <= 20; index++ {
			a := newHost(os, index)
			first := report(a)
			require.NotEmpty(t, first)
			for range 5 {
				require.Equal(t, first, report(a), "%s host %d changed between reports", os, index)
			}
			require.Equal(t, first, report(newHost(os, index)), "%s host %d changed after a restart", os, index)
		}
	}
}

func TestStickyPolicyFlips(t *testing.T) {
	const policies, days = 2000, 30
	start := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	names := make([]string, policies)
	for i := range names {
		names[i] = fmt.Sprintf("fleet_policy_query_%d", i)
	}
	results := func(a *agent, at time.Time) []bool {
		out := make([]bool, policies)
		for i, n := range names {
			out[i] = a.stickyPolicyRoll(n, at) < a.policyPassProb
		}
		return out
	}

	a := &agent{UUID: "host-1", policyPassProb: 0.9, policyFlipPerDay: 0.02}
	prev := results(a, start)
	var flipped, passing float64
	for d := 1; d <= days; d++ {
		cur := results(a, start.Add(time.Duration(d)*24*time.Hour))
		for i := range cur {
			if cur[i] != prev[i] {
				flipped++
			}
			if cur[i] {
				passing++
			}
		}
		prev = cur
	}
	assert.InDelta(t, 0.02, flipped/(policies*days), 0.005, "daily flip rate")
	assert.InDelta(t, 0.9, passing/(policies*days), 0.02, "pass rate")

	// A restarted host reports the same results at the same moment.
	later := start.Add(36 * time.Hour)
	require.Equal(t, results(a, later), results(&agent{UUID: "host-1", policyPassProb: 0.9, policyFlipPerDay: 0.02}, later))

	// Without flips, results never change.
	still := &agent{UUID: "host-1", policyPassProb: 0.9}
	require.Equal(t, results(still, start), results(still, start.Add(90*24*time.Hour)))
}

func TestLabelMatchProb(t *testing.T) {
	const labels = 2000
	members := func(a *agent) []bool {
		out := make([]bool, labels)
		for i := range out {
			_, rows, status, _, _ := a.processQuery(fmt.Sprintf("fleet_label_query_%d", 1000+i), "SELECT 1 FROM os_version WHERE major = 26", nil)
			require.NotNil(t, status)
			out[i] = len(rows) > 0
		}
		return out
	}
	count := func(m []bool) (n int) {
		for _, v := range m {
			if v {
				n++
			}
		}
		return n
	}

	a := &agent{UUID: "host-1", labelMatchProb: 0.1}
	first := members(a)
	assert.InDelta(t, 0.1, float64(count(first))/labels, 0.02)
	// Decided once per host and label: the same after a restart, different for another host.
	require.Equal(t, first, members(&agent{UUID: "host-1", labelMatchProb: 0.1}))
	assert.NotEqual(t, first, members(&agent{UUID: "host-2", labelMatchProb: 0.1}))

	// The "All hosts" label always matches.
	_, rows, _, _, _ := a.processQuery("fleet_label_query_6", "select 1;", nil)
	require.Len(t, rows, 1)
}

func TestOnlineWindows(t *testing.T) {
	const hosts = 2000
	start := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	agents := make([]*agent, hosts)
	staleCount := 0
	for i := range agents {
		id := hostSeed{seed: 11, index: i + 1}
		agents[i] = &agent{id: id, stale: id.stale(0.3), onlineFraction: 0.73, onlineOffset: id.onlineOffset()}
		if agents[i].stale {
			staleCount++
		}
	}
	assert.InDelta(t, 0.3, float64(staleCount)/hosts, 0.03)

	// About 70% x 73% of hosts are online at any hour of the day.
	for h := 0; h < 24; h++ {
		at := start.Add(time.Duration(h) * time.Hour)
		online := 0
		for _, a := range agents {
			if a.online(at) {
				online++
				require.False(t, a.stale, "a stale host checked in")
			}
		}
		assert.InDelta(t, 0.7*0.73, float64(online)/hosts, 0.04, "hour %d", h)
	}

	// Each host is online for its window, the same every day.
	a := agents[0]
	for a.stale {
		a = agents[1]
	}
	minutesOnline := 0
	for m := 0; m < 24*60; m++ {
		at := start.Add(time.Duration(m) * time.Minute)
		if a.online(at) {
			minutesOnline++
		}
		require.Equal(t, a.online(at), a.online(at.Add(24*time.Hour)))
	}
	assert.InDelta(t, 0.73*24*60, float64(minutesOnline), 2)

	assert.True(t, (&agent{onlineFraction: 1}).online(start))
}

func TestAwaitMDMPush(t *testing.T) {
	tick := make(chan time.Time, 1)

	// Online: a push means check in now.
	pings := make(chan apnsmock.Ping, 1)
	pings <- apnsmock.Ping{}
	missed := false
	require.True(t, (&agent{onlineFraction: 1}).awaitMDMPush(pings, tick, &missed))

	// Offline: the push is remembered, not answered.
	offline := &agent{stale: true}
	pings <- apnsmock.Ping{}
	close(pings)
	require.False(t, offline.awaitMDMPush(pings, tick, &missed))
	require.True(t, missed)

	// Back online: the next check answers the missed push.
	tick <- time.Now()
	require.True(t, (&agent{onlineFraction: 1}).awaitMDMPush(make(chan apnsmock.Ping), tick, &missed))
	require.False(t, missed)
}

func TestLargeInventorySizes(t *testing.T) {
	for platform, shape := range largeInventory {
		const hosts = 4000
		sizes := make([]int, hosts)
		sum := 0
		for i := range sizes {
			sizes[i] = hostSeed{seed: 13, index: i + 1}.inventorySize(shape)
			sum += sizes[i]
		}
		slices.Sort(sizes)
		mu := math.Log(shape.mean) - shape.sigma*shape.sigma/2
		assert.InEpsilon(t, shape.mean, float64(sum)/hosts, 0.05, platform)
		assert.InEpsilon(t, math.Exp(mu+1.2816*shape.sigma), float64(sizes[hosts*9/10]), 0.08, platform)
		assert.LessOrEqual(t, float64(sizes[hosts-1]), shape.max, platform)
	}
}

func TestLargeInventoryReports(t *testing.T) {
	orig := softwareDB
	t.Cleanup(func() { softwareDB = orig })
	softwareDB = &softwaredb.DB{
		Darwin:    make([]softwaredb.DarwinSoftware, 8000),
		Windows:   make([]softwaredb.WindowsSoftware, 8000),
		Ubuntu:    make([]softwaredb.UbuntuSoftware, 8000),
		DarwinNPM: make([]softwaredb.DarwinSoftware, 400),
	}
	for i := range 8000 {
		name := fmt.Sprintf("app-%d", i)
		softwareDB.Darwin[i] = softwaredb.DarwinSoftware{Name: name, Version: "1.0", Source: "apps"}
		softwareDB.Windows[i] = softwaredb.WindowsSoftware{Name: name, Version: "1.0", Source: "programs"}
		softwareDB.Ubuntu[i] = softwaredb.UbuntuSoftware{Name: name, Version: "1.0", Source: "deb_packages"}
	}
	for i := range softwareDB.DarwinNPM {
		softwareDB.DarwinNPM[i] = softwaredb.DarwinSoftware{Name: fmt.Sprintf("pkg-%d", i), Version: "1.0", Source: "npm_packages"}
	}

	newHost := func(os string, index int) *agent {
		return &agent{
			os: os, id: hostSeed{seed: 5, index: index}, softwareChurnPerDay: 0.05, largeInventory: true,
			strings: map[string]string{}, softwareVersionMap: map[rune]int{}, cachedLastOpenedAt: map[string]*time.Time{},
			softwareCount: softwareEntityCount{entityCount: entityCount{common: 10, unique: 1}, vulnerable: 10},
		}
	}
	queries := map[string]string{
		"macos":   "fleet_detail_query_software_macos",
		"windows": "fleet_detail_query_software_windows",
		"ubuntu":  "fleet_detail_query_software_linux",
	}
	platforms := map[string]string{"macos": "darwin", "windows": "windows", "ubuntu": "ubuntu"}
	report := func(a *agent) []map[string]string {
		_, rows, _, _, _ := a.processQuery(queries[a.os], "", &cachedResults{})
		return rows
	}

	const hosts = 300
	for os, query := range queries {
		total, npm := 0, 0
		for index := 1; index <= hosts; index++ {
			rows := report(newHost(os, index))
			total += len(rows)
			for _, r := range rows {
				if r["source"] == "npm_packages" {
					npm++
					assert.Equal(t, "/usr/local/lib/node_modules/"+r["name"]+"/package.json", r["installed_path"])
				}
			}
			if index <= 5 {
				names := func(rows []map[string]string) []string {
					keys := make([]string, 0, len(rows))
					for _, r := range rows {
						keys = append(keys, r["name"]+"|"+r["source"])
					}
					slices.Sort(keys)
					return keys
				}
				require.Equal(t, names(rows), names(report(newHost(os, index))), "%s host %d after a restart", query, index)
			}
		}
		assert.InEpsilon(t, largeInventory[platforms[os]].mean, float64(total)/hosts, 0.1, os)
		if os == "macos" {
			assert.InEpsilon(t, 100, float64(npm)/hosts, 0.15)
		} else {
			assert.Zero(t, npm, os)
		}
	}
}

func TestStablePerHostCerts(t *testing.T) {
	start := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	newHost := func(index int, churn float64) *agent {
		return &agent{
			id: hostSeed{seed: 17, index: index}, agentIndex: index, hostCertCount: 20, certChurnPerDay: churn,
			userCount: entityCount{common: 4, unique: 4},
		}
	}
	serials := func(specs []simulatedCert) []string {
		out := make([]string, 0, len(specs))
		for _, c := range specs {
			out = append(out, c.serial+"|"+c.username)
		}
		return out
	}

	const hosts = 500
	total, changed, kept := 0, 0, 0
	for index := 1; index <= hosts; index++ {
		a := newHost(index, 0.02)
		first := a.stablePerHostCertSpecs(start)
		total += len(first)
		require.Equal(t, serials(first), serials(newHost(index, 0.02).stablePerHostCertSpecs(start)), "host %d after a restart", index)
		for _, c := range first {
			require.Less(t, c.notValidBeforeUnix, c.notValidAfterUnix)
		}
		next := a.stablePerHostCertSpecs(start.Add(24 * time.Hour))
		for i := range first[:max(0, len(first)-1)] { // the last is the cross-scope duplicate
			if first[i].serial == next[i].serial {
				kept++
			} else {
				changed++
			}
		}

		frozen := newHost(index, 0)
		require.Equal(t, serials(frozen.stablePerHostCertSpecs(start)), serials(frozen.stablePerHostCertSpecs(start.Add(90*24*time.Hour))))
	}
	assert.InDelta(t, 21, float64(total)/hosts, 1.5) // 20 + the duplicate
	assert.InDelta(t, 0.02, float64(changed)/float64(changed+kept), 0.006)
}

func TestWindowsLastOpened(t *testing.T) {
	orig := softwareDB
	t.Cleanup(func() { softwareDB = orig })
	softwareDB = &softwaredb.DB{Windows: make([]softwaredb.WindowsSoftware, 2000)}
	for i := range softwareDB.Windows {
		softwareDB.Windows[i] = softwaredb.WindowsSoftware{Name: fmt.Sprintf("App: %d", i), Version: "1.0", Source: "programs"}
	}
	a := &agent{
		os: "windows", id: hostSeed{seed: 21, index: 1}, softwareChurnPerDay: 0.05,
		strings: map[string]string{}, softwareVersionMap: map[rune]int{}, cachedLastOpenedAt: map[string]*time.Time{},
		softwareCount: softwareEntityCount{withLastOpened: 50, lastOpenedProb: 0.025},
	}
	process := osquery_utils.SoftwareOverrideQueries["windows_last_opened_at"].SoftwareProcessResults
	report := func() map[string]string {
		cached := &cachedResults{}
		_, software, _, _, _ := a.processQuery("fleet_detail_query_software_windows", "", cached)
		_, prefetch, _, _, _ := a.processQuery("fleet_detail_query_software_windows_last_opened_at", "", cached)
		opened := map[string]string{}
		for _, r := range process(software, prefetch) {
			require.NotEmpty(t, r["installed_path"])
			if r["last_opened_at"] != "" {
				opened[r["name"]] = r["last_opened_at"]
			}
		}
		return opened
	}

	require.Len(t, report(), 50)
	// Age the launches by a day, then a day of hourly software reports reopens a share of the apps.
	for name, at := range a.cachedLastOpenedAt {
		if at != nil {
			a.cachedLastOpenedAt[name] = new(at.Add(-24 * time.Hour))
		}
	}
	first := report()
	var last map[string]string
	for range 24 {
		last = report()
	}
	reopened := 0
	for name, at := range last {
		if at != first[name] {
			reopened++
		}
	}
	assert.Len(t, last, 50)
	assert.Greater(t, reopened, 10)
	assert.Less(t, reopened, 36)
}

func TestSoftwareReleases(t *testing.T) {
	now := releaseEpoch.Add(200 * 24 * time.Hour)
	items := make([]map[string]string, 3000)
	rows := func() []map[string]string {
		out := make([]map[string]string, len(items))
		for i := range items {
			out[i] = map[string]string{"name": fmt.Sprintf("app-%d", i), "version": "1.0", "source": "apps"}
		}
		return out
	}
	newHost := func(index int) *agent {
		id := hostSeed{seed: 23, index: index}
		return &agent{id: id, releaseDays: 60, rolloutDays: 7, releaseLagKey: id.rng("release-lag").Uint64()}
	}

	// Off: versions are untouched.
	assert.Equal(t, "1.0", (&agent{}).withReleases(rows(), now)[0]["version"])

	a, b := newHost(1), newHost(2)
	today := a.withReleases(rows(), now)
	require.Equal(t, today, newHost(1).withReleases(rows(), now), "a restarted host")

	// A day later 1/60 of the items have a new release on a host.
	tomorrow := a.withReleases(rows(), now.Add(24*time.Hour))
	updated := 0
	for i := range today {
		if today[i]["version"] != tomorrow[i]["version"] {
			updated++
			assert.Greater(t, tomorrow[i]["version"], today[i]["version"])
		}
	}
	assert.InDelta(t, 1.0/60, float64(updated)/float64(len(items)), 0.006)

	// Every host is on the same releases once the rollout window has passed.
	later := now.Add(8 * 24 * time.Hour)
	same, onB := 0, b.withReleases(rows(), later)
	for i, r := range a.withReleases(rows(), later) {
		if r["version"] == onB[i]["version"] {
			same++
		}
	}
	assert.Greater(t, float64(same)/float64(len(items)), 0.85)

	// Before the epoch no item has a release yet.
	for _, r := range a.withReleases(rows(), releaseEpoch.Add(-time.Hour)) {
		require.Equal(t, "1.0", r["version"])
	}
}

func TestMacLastOpenedParses(t *testing.T) {
	a := &agent{
		os: "macos", id: hostSeed{seed: 25, index: 1}, softwareChurnPerDay: 0.05,
		strings: map[string]string{}, softwareVersionMap: map[rune]int{}, cachedLastOpenedAt: map[string]*time.Time{},
		softwareCount: softwareEntityCount{entityCount: entityCount{common: 10, unique: 1}, vulnerable: 10, withLastOpened: 50, lastOpenedProb: 0.025},
	}
	_, rows, _, _, _ := a.processQuery("fleet_detail_query_software_macos", "", &cachedResults{})
	opened := 0
	for _, r := range rows {
		at, err := fleet.ParseSoftwareLastOpenedAtRowValue(r["last_opened_at"])
		require.NoError(t, err, r["name"])
		if !at.IsZero() {
			opened++
		}
	}
	assert.Equal(t, 50, opened)
}
