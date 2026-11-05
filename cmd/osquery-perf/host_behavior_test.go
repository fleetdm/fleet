package main

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/osquery-perf/softwaredb"
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

	first := stableSoftwareIndices(id, count, pool, churn, start)
	require.Len(t, first, count)
	require.Len(t, slices.Compact(slices.Sorted(slices.Values(first))), count, "indices must be distinct")
	for _, i := range first {
		require.Less(t, i, uint32(pool))
	}

	// The same host at the same moment, e.g. after a restart, reports the same inventory.
	require.Equal(t, first, stableSoftwareIndices(id, count, pool, churn, start))
	assert.Greater(t, changedShare(first, stableSoftwareIndices(hostSeed{seed: 4, index: 3}, count, pool, churn, start)), 0.9)

	// An hourly refresh changes almost nothing, and each day changes about churn of the inventory.
	assert.Less(t, changedShare(first, stableSoftwareIndices(id, count, pool, churn, start.Add(time.Hour))), 0.005)
	var total float64
	prev := first
	const days = 30
	for d := 1; d <= days; d++ {
		cur := stableSoftwareIndices(id, count, pool, churn, start.Add(time.Duration(d)*24*time.Hour))
		require.Len(t, cur, count)
		total += changedShare(prev, cur)
		prev = cur
	}
	assert.InDelta(t, churn, total/days, 0.005)

	frozen := stableSoftwareIndices(id, count, pool, 0, start)
	assert.Equal(t, frozen, stableSoftwareIndices(id, count, pool, 0, start.Add(90*24*time.Hour)))

	assert.Len(t, stableSoftwareIndices(id, count, 100, churn, start), 100)
	assert.Empty(t, stableSoftwareIndices(id, count, 0, churn, start))
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
