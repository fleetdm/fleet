package mysql

import (
	"context"
	"fmt"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	fleetptr "github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

// TestCombinedIncludeExcludeLabels validates that the combined include+exclude
// label scoping works correctly end-to-end through the database layer.
// This test requires MYSQL_TEST=1 and a running MySQL instance (port 3307).
func TestCombinedIncludeExcludeLabels(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()
	ctx := context.Background()

	// --- Setup: Create labels ---
	inclLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "engineering",
		Query:               "SELECT 1",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	exclLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "g-mdm",
		Query:               "SELECT 1",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	otherLabel, err := ds.NewLabel(ctx, &fleet.Label{
		Name:                "finance",
		Query:               "SELECT 1",
		LabelType:           fleet.LabelTypeRegular,
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	// --- Setup: Create hosts ---
	hostInIncludeOnly, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   fleetptr.String("host-include-only"),
		NodeKey:         fleetptr.String("node-include-only"),
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	hostInBoth, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   fleetptr.String("host-in-both"),
		NodeKey:         fleetptr.String("node-in-both"),
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	hostInExcludeOnly, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   fleetptr.String("host-exclude-only"),
		NodeKey:         fleetptr.String("node-exclude-only"),
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	hostInNeither, err := ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID:   fleetptr.String("host-in-neither"),
		NodeKey:         fleetptr.String("node-in-neither"),
		DetailUpdatedAt: ds.clock.Now(),
		LabelUpdatedAt:  ds.clock.Now(),
		SeenTime:        ds.clock.Now(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// --- Setup: Create label memberships ---
	boolTrue := true
	boolFalse := false

	// hostInIncludeOnly -> in "engineering" only
	err = ds.RecordLabelQueryExecutions(ctx, hostInIncludeOnly, map[uint]*bool{
		inclLabel.ID: &boolTrue,
		exclLabel.ID: &boolFalse,
	}, ds.clock.Now(), false)
	require.NoError(t, err)

	// hostInBoth -> in "engineering" AND "g-mdm"
	err = ds.RecordLabelQueryExecutions(ctx, hostInBoth, map[uint]*bool{
		inclLabel.ID: &boolTrue,
		exclLabel.ID: &boolTrue,
	}, ds.clock.Now(), false)
	require.NoError(t, err)

	// hostInExcludeOnly -> in "g-mdm" only
	err = ds.RecordLabelQueryExecutions(ctx, hostInExcludeOnly, map[uint]*bool{
		inclLabel.ID: &boolFalse,
		exclLabel.ID: &boolTrue,
	}, ds.clock.Now(), false)
	require.NoError(t, err)

	// hostInNeither -> in neither
	err = ds.RecordLabelQueryExecutions(ctx, hostInNeither, map[uint]*bool{
		inclLabel.ID: &boolFalse,
		exclLabel.ID: &boolFalse,
	}, ds.clock.Now(), false)
	require.NoError(t, err)

	// --- Test 1: Create a policy with COMBINED include_any + exclude_any ---
	policy, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:             "combined-label-policy",
		Query:            "SELECT 1 FROM osquery_info",
		LabelsIncludeAny: []string{inclLabel.Name},
		LabelsExcludeAny: []string{exclLabel.Name},
	})
	require.NoError(t, err)
	require.NotNil(t, policy)

	policyIDStr := fmt.Sprintf("%d", policy.ID)

	// --- Verify: policy_labels table has BOTH include and exclude rows ---
	type policyLabelRow struct {
		PolicyID uint `db:"policy_id"`
		LabelID  uint `db:"label_id"`
		Exclude  bool `db:"exclude"`
	}
	var pLabels []policyLabelRow
	err = ds.writer(ctx).SelectContext(ctx, &pLabels,
		"SELECT policy_id, label_id, exclude FROM policy_labels WHERE policy_id = ? ORDER BY label_id", policy.ID)
	require.NoError(t, err)
	require.Len(t, pLabels, 2, "expected 2 policy_labels rows (1 include + 1 exclude)")

	// Find include and exclude rows
	var foundInclude, foundExclude bool
	for _, pl := range pLabels {
		if pl.LabelID == inclLabel.ID && !pl.Exclude {
			foundInclude = true
		}
		if pl.LabelID == exclLabel.ID && pl.Exclude {
			foundExclude = true
		}
	}
	require.True(t, foundInclude, "expected include label row with exclude=false")
	require.True(t, foundExclude, "expected exclude label row with exclude=true")

	// --- Test 2: PolicyQueriesForHost returns correct results ---
	t.Run("host in include only -> policy applies", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInIncludeOnly)
		require.NoError(t, err)
		_, applies := queries[policyIDStr]
		require.True(t, applies, "policy should apply to host in include label only")
	})

	t.Run("host in both include and exclude -> policy does NOT apply", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInBoth)
		require.NoError(t, err)
		_, applies := queries[policyIDStr]
		require.False(t, applies, "policy should NOT apply to host in both include and exclude labels")
	})

	t.Run("host in exclude only -> policy does NOT apply", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInExcludeOnly)
		require.NoError(t, err)
		_, applies := queries[policyIDStr]
		require.False(t, applies, "policy should NOT apply to host in exclude label only")
	})

	t.Run("host in neither -> policy does NOT apply", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInNeither)
		require.NoError(t, err)
		_, applies := queries[policyIDStr]
		require.False(t, applies, "policy should NOT apply to host in neither label")
	})

	// --- Test 3: Backward compat - include_any only ---
	policyIncludeOnly, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:             "include-only-policy",
		Query:            "SELECT 1 FROM osquery_info WHERE 1=1",
		LabelsIncludeAny: []string{inclLabel.Name},
	})
	require.NoError(t, err)
	inclOnlyIDStr := fmt.Sprintf("%d", policyIncludeOnly.ID)

	t.Run("include-only: host with label -> applies", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInIncludeOnly)
		require.NoError(t, err)
		_, applies := queries[inclOnlyIDStr]
		require.True(t, applies)
	})

	t.Run("include-only: host without label -> does not apply", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInNeither)
		require.NoError(t, err)
		_, applies := queries[inclOnlyIDStr]
		require.False(t, applies)
	})

	// --- Test 4: Backward compat - exclude_any only ---
	policyExcludeOnly, err := ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{
		Name:             "exclude-only-policy",
		Query:            "SELECT 1 FROM osquery_info WHERE 2=2",
		LabelsExcludeAny: []string{exclLabel.Name},
	})
	require.NoError(t, err)
	exclOnlyIDStr := fmt.Sprintf("%d", policyExcludeOnly.ID)

	t.Run("exclude-only: host with exclude label -> does not apply", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInExcludeOnly)
		require.NoError(t, err)
		_, applies := queries[exclOnlyIDStr]
		require.False(t, applies)
	})

	t.Run("exclude-only: host without exclude label -> applies", func(t *testing.T) {
		queries, err := ds.PolicyQueriesForHost(ctx, hostInIncludeOnly)
		require.NoError(t, err)
		_, applies := queries[exclOnlyIDStr]
		require.True(t, applies)
	})

	_ = otherLabel // suppress unused
}
