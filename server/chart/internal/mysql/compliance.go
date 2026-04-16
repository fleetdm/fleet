package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// GetPolicyFailingSnapshot returns the currently-open per-policy failing-host
// bitmaps. One row per policy that has an open SCD row in the policy_failing
// dataset. Fully-passing policies are included as empty bitmaps, which the
// callers treat as "tracked but nothing failing today."
func (ds *Datastore) GetPolicyFailingSnapshot(ctx context.Context) ([]chart.PolicyFailingSnapshot, error) {
	type row struct {
		EntityID   string `db:"entity_id"`
		HostBitmap []byte `db:"host_bitmap"`
	}
	var rows []row
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
		`SELECT entity_id, host_bitmap
		 FROM host_scd_data
		 WHERE dataset = 'policy_failing' AND valid_to = ?`,
		scdOpenSentinel); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policy_failing snapshot")
	}

	out := make([]chart.PolicyFailingSnapshot, 0, len(rows))
	for _, r := range rows {
		pid, err := strconv.ParseUint(r.EntityID, 10, 64)
		if err != nil {
			// Skip malformed entity IDs rather than fail the whole query — they
			// shouldn't exist, but one stray row shouldn't break the leaderboard.
			ds.logger.WarnContext(ctx, "skipping malformed policy_failing entity_id",
				"entity_id", r.EntityID)
			continue
		}
		out = append(out, chart.PolicyFailingSnapshot{
			PolicyID:   uint(pid),
			HostBitmap: r.HostBitmap,
		})
	}
	return out, nil
}

// GetPoliciesMetadata returns id/name/team_id for every policy. Used to hydrate
// rows from the policy_failing snapshot with human-readable names.
func (ds *Datastore) GetPoliciesMetadata(ctx context.Context) ([]chart.PolicyMeta, error) {
	type row struct {
		ID     uint           `db:"id"`
		Name   string         `db:"name"`
		TeamID sql.NullInt64  `db:"team_id"`
	}
	var rows []row
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
		`SELECT id, name, team_id FROM policies`); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies metadata")
	}
	out := make([]chart.PolicyMeta, 0, len(rows))
	for _, r := range rows {
		meta := chart.PolicyMeta{ID: r.ID, Name: r.Name}
		if r.TeamID.Valid {
			t := uint(r.TeamID.Int64)
			meta.TeamID = &t
		}
		out = append(out, meta)
	}
	return out, nil
}

// GetTeamsMetadata returns id/name for every team. Leaderboard adds a synthetic
// "No team" row in the service layer for hosts with NULL team_id.
func (ds *Datastore) GetTeamsMetadata(ctx context.Context) ([]chart.TeamMeta, error) {
	var out []chart.TeamMeta
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &out,
		`SELECT id, name FROM teams ORDER BY id`); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get teams metadata")
	}
	return out, nil
}

// GetTopNonCompliantHosts returns the N hosts with the largest number of
// currently-failing policies. For each open policy_failing bitmap we walk set
// bits to increment a per-host counter, then sort descending and join host
// and team metadata for the top N.
//
// A limit <= 0 or > 500 is clamped; the source of truth is the in-memory
// counter, so extremely large limits would hold a lot of rows in memory.
func (ds *Datastore) GetTopNonCompliantHosts(ctx context.Context, limit int) ([]chart.HostFailingSummary, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 500 {
		limit = 500
	}

	snapshot, err := ds.GetPolicyFailingSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	// Count, per host, how many policies it's currently failing. We iterate
	// set bits of each bitmap directly (bit N = host ID N) rather than
	// converting to a host-ID slice first — saves allocations when a bitmap
	// has many set bits.
	failureCount := make(map[uint]int)
	for _, snap := range snapshot {
		for byteIdx, b := range snap.HostBitmap {
			if b == 0 {
				continue
			}
			for bit := 0; bit < 8; bit++ {
				if b&(1<<bit) != 0 {
					failureCount[uint(byteIdx*8+bit)]++
				}
			}
		}
	}

	// Sort host IDs by failing count descending, break ties on host_id asc so
	// the output is deterministic across runs.
	type pair struct {
		hostID uint
		count  int
	}
	pairs := make([]pair, 0, len(failureCount))
	for id, count := range failureCount {
		pairs = append(pairs, pair{id, count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].hostID < pairs[j].hostID
	})
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	if len(pairs) == 0 {
		return nil, nil
	}

	// Fetch hostname, computer_name, team_id, and team name for the top N.
	hostIDs := make([]uint, len(pairs))
	for i, p := range pairs {
		hostIDs[i] = p.hostID
	}
	type hostRow struct {
		ID           uint           `db:"id"`
		Hostname     string         `db:"hostname"`
		ComputerName string         `db:"computer_name"`
		TeamID       sql.NullInt64  `db:"team_id"`
		TeamName     sql.NullString `db:"team_name"`
	}
	query, args, err := sqlx.In(`
		SELECT h.id, h.hostname, h.computer_name, h.team_id, t.name AS team_name
		FROM hosts h
		LEFT JOIN teams t ON t.id = h.team_id
		WHERE h.id IN (?)`, hostIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand top non-compliant hosts query")
	}
	query = ds.rebind(query)

	var rows []hostRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetch host metadata for non-compliant ranking")
	}
	hostByID := make(map[uint]hostRow, len(rows))
	for _, r := range rows {
		hostByID[r.ID] = r
	}

	results := make([]chart.HostFailingSummary, 0, len(pairs))
	for _, p := range pairs {
		row, ok := hostByID[p.hostID]
		if !ok {
			// Bitmap references a host ID that isn't in the hosts table —
			// host was deleted or metadata sync is partial. Surface it with
			// a placeholder hostname so ops sees the ID rather than silently
			// dropping it from the ranking.
			results = append(results, chart.HostFailingSummary{
				HostID:             p.hostID,
				Hostname:           fmt.Sprintf("host-%d", p.hostID),
				FailingPolicyCount: p.count,
			})
			continue
		}
		summary := chart.HostFailingSummary{
			HostID:             row.ID,
			Hostname:           row.Hostname,
			ComputerName:       row.ComputerName,
			FailingPolicyCount: p.count,
		}
		if row.TeamID.Valid {
			tid := uint(row.TeamID.Int64)
			summary.TeamID = &tid
		}
		if row.TeamName.Valid {
			summary.TeamName = row.TeamName.String
		}
		results = append(results, summary)
	}
	return results, nil
}

// GetPolicyFailingByTeamTrend returns per-day, per-team counts of hosts failing
// ≥1 policy over the given date range. A single pass over the SCD rows composes
// bitmaps for every team, avoiding N*days separate queries.
//
// The map key in each returned point is the stringified team_id; hosts with no
// team are bucketed under the sentinel key returned by NoTeamBucketKey.
func (ds *Datastore) GetPolicyFailingByTeamTrend(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]chart.TeamTrendPoint, error) {
	startDay := startDate.UTC().Truncate(24 * time.Hour)
	endDay := endDate.UTC().Truncate(24 * time.Hour)

	// Fetch all policy_failing rows overlapping the range. Same overlap semantics
	// as GetSCDData: valid_from <= endDay AND valid_to > startDay.
	type scdRow struct {
		EntityID   string `db:"entity_id"`
		HostBitmap []byte `db:"host_bitmap"`
		ValidFrom  string `db:"valid_from"`
		ValidTo    string `db:"valid_to"`
	}
	var rows []scdRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, `
		SELECT entity_id, host_bitmap,
			DATE_FORMAT(valid_from, '%Y-%m-%d') AS valid_from,
			DATE_FORMAT(valid_to,   '%Y-%m-%d') AS valid_to
		FROM host_scd_data
		WHERE dataset = 'policy_failing'
			AND valid_from <= ?
			AND valid_to   >  ?`,
		endDay.Format(scdDateFormat),
		startDay.Format(scdDateFormat)); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policy_failing rows for team trend")
	}

	// Group host IDs by team to build per-team bitmap masks.
	assignments, err := ds.GetHostTeamAssignments(ctx)
	if err != nil {
		return nil, err
	}
	teamHostIDs := make(map[string][]uint)
	for _, ht := range assignments {
		key := chart.NoTeamBucketKey
		if ht.TeamID != nil {
			key = strconv.FormatUint(uint64(*ht.TeamID), 10)
		}
		teamHostIDs[key] = append(teamHostIDs[key], ht.HostID)
	}
	teamMasks := make(map[string][]byte, len(teamHostIDs))
	for teamKey, ids := range teamHostIDs {
		teamMasks[teamKey] = chart.HostIDsToBlob(ids)
	}

	// Walk each day in the range. For each day: OR all in-range bitmaps, then
	// AND with each team's mask and popcount. This captures "hosts failing ≥1
	// policy, scoped to this team on this day."
	var results []chart.TeamTrendPoint
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		dayStr := d.Format(scdDateFormat)

		var unionFailing []byte
		for _, r := range rows {
			if r.ValidFrom > dayStr || r.ValidTo <= dayStr {
				continue
			}
			unionFailing = chart.BlobOR(unionFailing, r.HostBitmap)
		}

		counts := make(map[string]int, len(teamMasks))
		for teamKey, mask := range teamMasks {
			counts[teamKey] = chart.BlobPopcount(chart.BlobAND(unionFailing, mask))
		}
		results = append(results, chart.TeamTrendPoint{
			Timestamp: d,
			Counts:    counts,
		})
	}
	return results, nil
}

// GetHostTeamAssignments returns (host_id, team_id) for every host. Callers
// group by team_id to build per-team host-ID bitmaps. NULL team_id becomes the
// synthetic "No team" bucket.
func (ds *Datastore) GetHostTeamAssignments(ctx context.Context) ([]chart.HostTeam, error) {
	type row struct {
		HostID uint          `db:"id"`
		TeamID sql.NullInt64 `db:"team_id"`
	}
	var rows []row
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
		`SELECT id, team_id FROM hosts`); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host team assignments")
	}
	out := make([]chart.HostTeam, 0, len(rows))
	for _, r := range rows {
		ht := chart.HostTeam{HostID: r.HostID}
		if r.TeamID.Valid {
			t := uint(r.TeamID.Int64)
			ht.TeamID = &t
		}
		out = append(out, ht)
	}
	return out, nil
}
