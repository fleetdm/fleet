package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// critical is the filter the dashboard sends by default.
func critical() CVEFilter { return CVEFilter{CVSSMin: new(9.0), CVSSMax: new(10.0)} }

func TestCVEFilterAggregateEntityIDShape(t *testing.T) {
	id := critical().AggregateEntityID()

	require.True(t, strings.HasPrefix(id, AggregateEntityPrefix), "must carry the reserved prefix")
	require.False(t, strings.HasPrefix(id, "CVE-"), "must never look like a real CVE ID")
	require.LessOrEqual(t, len(id), 100, "must fit host_scd_data.entity_id")
	require.Equal(t, id, critical().AggregateEntityID(), "must be stable across calls")
}

// Canonicalization: filters that resolve to the same CVE set must share an ID,
// otherwise the collector writes duplicate aggregates for one logical series.
func TestCVEFilterAggregateEntityIDCanonicalizesEquivalentFilters(t *testing.T) {
	all := []string{CVECategoryOS, CVECategoryBrowsers, CVECategoryOffice, CVECategoryAdobe}

	cases := []struct {
		name string
		a, b CVEFilter
	}{
		{
			name: "category order",
			a:    CVEFilter{Categories: []string{CVECategoryOS, CVECategoryAdobe}},
			b:    CVEFilter{Categories: []string{CVECategoryAdobe, CVECategoryOS}},
		},
		{
			name: "duplicate categories",
			a:    CVEFilter{Categories: []string{CVECategoryOS, CVECategoryOS}},
			b:    CVEFilter{Categories: []string{CVECategoryOS}},
		},
		{
			// Every category selected narrows nothing, which is what an empty
			// list already means to the resolver.
			name: "every category equals no category filter",
			a:    CVEFilter{Categories: all},
			b:    CVEFilter{},
		},
		{
			name: "nil versus empty category slice",
			a:    CVEFilter{Categories: []string{}},
			b:    CVEFilter{Categories: nil},
		},
		{
			name: "excluded CVE order",
			a:    CVEFilter{ExcludeCVEs: []string{"CVE-2026-2", "CVE-2026-1"}},
			b:    CVEFilter{ExcludeCVEs: []string{"CVE-2026-1", "CVE-2026-2"}},
		},
		{
			name: "duplicate excluded CVEs",
			a:    CVEFilter{ExcludeCVEs: []string{"CVE-2026-1", "CVE-2026-1"}},
			b:    CVEFilter{ExcludeCVEs: []string{"CVE-2026-1"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.a.AggregateEntityID(), tc.b.AggregateEntityID())
		})
	}
}

// Every field must participate in the key: two filters that resolve to
// different CVE sets sharing an ID would serve one filter's chart for the other.
func TestCVEFilterAggregateEntityIDSeparatesDistinctFilters(t *testing.T) {
	cases := []struct {
		name string
		a, b CVEFilter
	}{
		{"cvss min", CVEFilter{CVSSMin: new(9.0)}, CVEFilter{CVSSMin: new(7.0)}},
		{"cvss max", CVEFilter{CVSSMax: new(10.0)}, CVEFilter{CVSSMax: new(8.9)}},
		{"epss min", CVEFilter{EPSSMin: new(0.3)}, CVEFilter{EPSSMin: new(0.4)}},
		{"epss max", CVEFilter{EPSSMax: new(0.3)}, CVEFilter{EPSSMax: new(0.4)}},
		{"known exploit", CVEFilter{KnownExploit: true}, CVEFilter{}},
		{"categories", CVEFilter{Categories: []string{CVECategoryOS}}, CVEFilter{Categories: []string{CVECategoryAdobe}}},
		{"excluded CVEs", CVEFilter{ExcludeCVEs: []string{"CVE-2026-1"}}, CVEFilter{}},
		// A nil bound drops the predicate; a zero bound keeps it and so still
		// excludes unscored CVEs. The two resolve to different sets.
		{"nil versus zero cvss min", CVEFilter{CVSSMin: nil}, CVEFilter{CVSSMin: new(0.0)}},
		{"nil versus zero epss min", CVEFilter{EPSSMin: nil}, CVEFilter{EPSSMin: new(0.0)}},
		// Bounds must not be interchangeable across fields.
		{"min versus max", CVEFilter{CVSSMin: new(5.0)}, CVEFilter{CVSSMax: new(5.0)}},
		{"cvss versus epss", CVEFilter{CVSSMin: new(0.5)}, CVEFilter{EPSSMin: new(0.5)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEqual(t, tc.a.AggregateEntityID(), tc.b.AggregateEntityID())
		})
	}
}

// A category the resolver does not know narrows nothing, so it must not change
// the key or produce an aggregate that differs from the same filter without it.
func TestCVEFilterAggregateEntityIDIgnoresUnknownCategories(t *testing.T) {
	withUnknown := CVEFilter{Categories: []string{CVECategoryOS, "not-a-category"}}
	require.Equal(t,
		CVEFilter{Categories: []string{CVECategoryOS}}.AggregateEntityID(),
		withUnknown.AggregateEntityID())
}
