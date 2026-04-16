package service

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
)

// GetMostIgnoredPolicies returns policies ranked by the number of hosts
// currently failing them, descending. Used by the "most-ignored policies"
// dashboard card. Empty-bitmap policies (zero failing hosts) are included at
// the bottom of the ranking — we want to show tracked policies too.
//
// limit <= 0 returns all policies; otherwise the top N.
func (s *Service) GetMostIgnoredPolicies(ctx context.Context, limit int) ([]chart.MostIgnoredPolicy, error) {
	if err := s.authz.Authorize(ctx, &chart.Host{}, platform_authz.ActionRead); err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetPolicyFailingSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	policies, err := s.store.GetPoliciesMetadata(ctx)
	if err != nil {
		return nil, err
	}
	teams, err := s.store.GetTeamsMetadata(ctx)
	if err != nil {
		return nil, err
	}

	// Index policy metadata and team names for O(1) lookup while walking the
	// snapshot. teamNameByID maps team_id → display name; missing IDs fall back
	// to "" and the frontend renders "Global" / "Fleet #N" accordingly.
	metaByID := make(map[uint]chart.PolicyMeta, len(policies))
	for _, p := range policies {
		metaByID[p.ID] = p
	}
	teamNameByID := make(map[uint]string, len(teams))
	for _, t := range teams {
		teamNameByID[t.ID] = t.Name
	}

	results := make([]chart.MostIgnoredPolicy, 0, len(snapshot))
	for _, snap := range snapshot {
		meta, ok := metaByID[snap.PolicyID]
		if !ok {
			// Policy was deleted but SCD hasn't been cleaned up yet. Skip.
			continue
		}
		var teamName string
		if meta.TeamID != nil {
			teamName = teamNameByID[*meta.TeamID]
		}
		results = append(results, chart.MostIgnoredPolicy{
			PolicyID:         snap.PolicyID,
			Name:             meta.Name,
			TeamID:           meta.TeamID,
			TeamName:         teamName,
			FailingHostCount: chart.BlobPopcount(snap.HostBitmap),
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].FailingHostCount > results[j].FailingHostCount
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// GetComplianceLeaderboard returns one row per team ranked by the percentage of
// hosts that are NOT failing any policy, descending. The "No team" bucket
// (hosts with NULL team_id) is included as a row with a nil TeamID.
//
// Computation per team:
//   - team_host_bitmap = bitmap of host IDs currently in the team
//   - union_failing    = OR across every policy's failing bitmap in the snapshot
//   - team_failing     = AND(union_failing, team_host_bitmap)
//   - hosts_failing_any = popcount(team_failing)
//   - fully_compliant_pct = 1 - hosts_failing_any / team_host_count
func (s *Service) GetComplianceLeaderboard(ctx context.Context) ([]chart.TeamCompliance, error) {
	if err := s.authz.Authorize(ctx, &chart.Host{}, platform_authz.ActionRead); err != nil {
		return nil, err
	}

	snapshot, err := s.store.GetPolicyFailingSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	teams, err := s.store.GetTeamsMetadata(ctx)
	if err != nil {
		return nil, err
	}
	hostAssignments, err := s.store.GetHostTeamAssignments(ctx)
	if err != nil {
		return nil, err
	}
	policies, err := s.store.GetPoliciesMetadata(ctx)
	if err != nil {
		return nil, err
	}

	// Union of failing hosts across every tracked policy — AND against each
	// team's host bitmap to get per-team "failing at least one policy."
	var unionFailing []byte
	for _, snap := range snapshot {
		unionFailing = chart.BlobOR(unionFailing, snap.HostBitmap)
	}

	// Group host IDs by team, including a synthetic nil-team bucket.
	byTeam := make(map[uint][]uint)
	var noTeamHosts []uint
	for _, ht := range hostAssignments {
		if ht.TeamID == nil {
			noTeamHosts = append(noTeamHosts, ht.HostID)
		} else {
			byTeam[*ht.TeamID] = append(byTeam[*ht.TeamID], ht.HostID)
		}
	}

	// Compute per-team policies-tracked count. Global policies (TeamID nil)
	// apply to every team; team-scoped policies only to their own team.
	globalPolicyCount := 0
	teamPolicyCount := make(map[uint]int)
	for _, p := range policies {
		if p.TeamID == nil {
			globalPolicyCount++
		} else {
			teamPolicyCount[*p.TeamID]++
		}
	}

	results := make([]chart.TeamCompliance, 0, len(teams)+1)
	for _, team := range teams {
		id := team.ID
		tracked := globalPolicyCount + teamPolicyCount[team.ID]
		row := computeTeamCompliance(&id, team.Name, byTeam[team.ID], snapshot, unionFailing, tracked)
		results = append(results, row)
	}
	if len(noTeamHosts) > 0 {
		// "No team" hosts only see global policies.
		results = append(results,
			computeTeamCompliance(nil, "No team", noTeamHosts, snapshot, unionFailing, globalPolicyCount))
	}

	// Sort: highest compliance first, then by name for stable ordering.
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].FullyCompliantPct != results[j].FullyCompliantPct {
			return results[i].FullyCompliantPct > results[j].FullyCompliantPct
		}
		return results[i].Name < results[j].Name
	})
	return results, nil
}

// GetTopNonCompliantHosts returns hosts ranked by number of currently-failing
// policies, descending. Used by the "most non-compliant hosts" dashboard card.
//
// limit <= 0 defaults to 10 (per backend clamp).
func (s *Service) GetTopNonCompliantHosts(ctx context.Context, limit int) ([]chart.HostFailingSummary, error) {
	if err := s.authz.Authorize(ctx, &chart.Host{}, platform_authz.ActionRead); err != nil {
		return nil, err
	}
	return s.store.GetTopNonCompliantHosts(ctx, limit)
}

// getPolicyFailingChartData produces the multi-series stacked-bar data for the
// unified /charts/policy_failing endpoint. Returns DataPoints keyed by team_id
// (matching SeriesMeta.Key), SeriesMeta with per-team stats, and totalHosts.
func (s *Service) getPolicyFailingChartData(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]chart.DataPoint, []chart.SeriesMeta, int, error) {
	trend, err := s.store.GetPolicyFailingByTeamTrend(ctx, startDate, endDate)
	if err != nil {
		return nil, nil, 0, err
	}

	// Reuse leaderboard for per-team metadata. Auth already checked by caller.
	teams, err := s.GetComplianceLeaderboard(ctx)
	if err != nil {
		return nil, nil, 0, err
	}

	// Build series metadata from the leaderboard rows.
	series := make([]chart.SeriesMeta, 0, len(teams))
	totalHosts := 0
	for _, t := range teams {
		key := chart.NoTeamBucketKey
		if t.TeamID != nil {
			key = strconv.FormatUint(uint64(*t.TeamID), 10)
		}
		series = append(series, chart.SeriesMeta{
			Key:   key,
			Label: t.Name,
			Stats: map[string]any{
				"host_count":          t.HostCount,
				"hosts_failing_any":   t.HostsFailingAny,
				"fully_compliant_pct": t.FullyCompliantPct,
				"policies_tracked":    t.PoliciesTracked,
				"policies_failing":    t.PoliciesFailing,
			},
		})
		totalHosts += t.HostCount
	}

	// Convert TeamTrendPoints → unified DataPoints.
	data := make([]chart.DataPoint, 0, len(trend))
	for _, tp := range trend {
		data = append(data, chart.DataPoint{
			Timestamp: tp.Timestamp,
			Values:    tp.Counts,
		})
	}

	return data, series, totalHosts, nil
}

// computeTeamCompliance builds one TeamCompliance row. Extracted so the same
// composition logic handles both real teams and the synthetic "No team" bucket.
func computeTeamCompliance(
	teamID *uint,
	name string,
	hostIDs []uint,
	snapshot []chart.PolicyFailingSnapshot,
	unionFailing []byte,
	policiesTracked int,
) chart.TeamCompliance {
	row := chart.TeamCompliance{
		TeamID:          teamID,
		Name:            name,
		HostCount:       len(hostIDs),
		PoliciesTracked: policiesTracked,
	}
	if len(hostIDs) == 0 {
		// No hosts — treat as 100% to keep the row from misleadingly ranking
		// at 0%. Consumers can filter on HostCount == 0 if they want to hide.
		row.FullyCompliantPct = 1.0
		return row
	}

	teamMask := chart.HostIDsToBlob(hostIDs)
	teamFailing := chart.BlobAND(unionFailing, teamMask)
	row.HostsFailingAny = chart.BlobPopcount(teamFailing)

	// PoliciesFailing counts the distinct policies where at least one host in
	// this team is failing. This is the "4" in "4/5 policies failing."
	for _, snap := range snapshot {
		if chart.BlobPopcount(chart.BlobAND(snap.HostBitmap, teamMask)) > 0 {
			row.PoliciesFailing++
		}
	}

	row.FullyCompliantPct = 1.0 - float64(row.HostsFailingAny)/float64(row.HostCount)
	return row
}
