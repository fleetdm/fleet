package mysql

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// hostVPPAppKey identifies the host_vpp_software_installs requests that share one set of host_vpp_app_installs rows.
type hostVPPAppKey struct {
	HostID   uint   `db:"host_id"`
	AdamID   string `db:"adam_id"`
	Platform string `db:"platform"`
}

// hostVPPAppInstallState is a host_vpp_app_installs row. Each pointer is the request id one status read selects within
// the version bucket.
type hostVPPAppInstallState struct {
	HostID                      uint       `db:"host_id"`
	AdamID                      string     `db:"adam_id"`
	Platform                    string     `db:"platform"`
	VersionBucket               uint       `db:"version_bucket"`
	VPPAppTeamID                *uint      `db:"vpp_app_team_id"`
	InstalledInstallID          *uint      `db:"installed_install_id"`
	FailedInstallID             *uint      `db:"failed_install_id"`
	UnfinishedInstallID         *uint      `db:"unfinished_install_id"`
	VisibleInstallID            *uint      `db:"visible_install_id"`
	LatestInstallID             *uint      `db:"latest_install_id"`
	UnfailedInstallID           *uint      `db:"unfailed_install_id"`
	VisibleCreatedAt            *time.Time `db:"visible_created_at"`
	VisibleCommandUUID          *string    `db:"visible_command_uuid"`
	VisibleVerificationAt       *time.Time `db:"visible_verification_at"`
	VisibleVerificationFailedAt *time.Time `db:"visible_verification_failed_at"`
	LatestCreatedAt             *time.Time `db:"latest_created_at"`
	LatestCommandUUID           *string    `db:"latest_command_uuid"`
	LatestVerificationAt        *time.Time `db:"latest_verification_at"`
	LatestVerificationFailedAt  *time.Time `db:"latest_verification_failed_at"`
}

// sortedVPPAppKeys returns the keys deduplicated and in lock order.
func sortedVPPAppKeys(keys []hostVPPAppKey) []hostVPPAppKey {
	keys = slices.Clone(keys)
	slices.SortFunc(keys, func(a, b hostVPPAppKey) int {
		return cmp.Or(cmp.Compare(a.HostID, b.HostID), cmp.Compare(a.AdamID, b.AdamID), cmp.Compare(a.Platform, b.Platform))
	})
	return slices.Compact(keys)
}

// lockVPPAppInstallKeys locks each key's state rows so writers serialize without scanning request history.
// For keys without state rows, the gap lock can deadlock concurrent first writers, which callers retry.
func lockVPPAppInstallKeys(ctx context.Context, tx sqlx.ExtContext, keys []hostVPPAppKey) error {
	for _, k := range sortedVPPAppKeys(keys) {
		var versionBuckets []uint
		if err := sqlx.SelectContext(ctx, tx, &versionBuckets,
			`SELECT version_bucket FROM host_vpp_app_installs WHERE host_id = ? AND adam_id = ? AND platform = ? FOR UPDATE`,
			k.HostID, k.AdamID, k.Platform); err != nil {
			return ctxerr.Wrap(ctx, err, "lock vpp app install key")
		}
	}
	return nil
}

// refreshVPPAppInstallState recomputes every host_vpp_app_installs row of the given keys from their requests and
// deletes rows whose version bucket has no requests left. Callers hold the keys' locks from lockVPPAppInstallKeys.
func refreshVPPAppInstallState(ctx context.Context, tx sqlx.ExtContext, keys []hostVPPAppKey) error {
	for _, k := range sortedVPPAppKeys(keys) {
		if err := refreshVPPAppInstallStateKey(ctx, tx, k); err != nil {
			return err
		}
	}
	return nil
}

func refreshVPPAppInstallStateKey(ctx context.Context, tx sqlx.ExtContext, k hostVPPAppKey) error {
	// The locking read sees the latest committed requests rather than the transaction's snapshot. Ordering by the
	// status reads' own comparator makes the first eligible request per bucket the winner of the ordered pointers,
	// with MySQL's NULL created_at ordering.
	var reqs []struct {
		ID                   uint       `db:"id"`
		VPPAppTeamID         *uint      `db:"vpp_app_team_id"`
		Canceled             bool       `db:"canceled"`
		Removed              bool       `db:"removed"`
		CreatedAt            *time.Time `db:"created_at"`
		CommandUUID          string     `db:"command_uuid"`
		VerificationAt       *time.Time `db:"verification_at"`
		VerificationFailedAt *time.Time `db:"verification_failed_at"`
	}
	if err := sqlx.SelectContext(ctx, tx, &reqs, `
SELECT
	id, vpp_app_team_id, canceled, removed, created_at, command_uuid,
	verification_at, verification_failed_at
FROM host_vpp_software_installs
WHERE host_id = ? AND adam_id = ? AND platform = ?
ORDER BY created_at DESC, id DESC
FOR SHARE`, k.HostID, k.AdamID, k.Platform); err != nil {
		return ctxerr.Wrap(ctx, err, "select vpp app install requests for state")
	}

	maxID := func(cur *uint, id uint) *uint {
		if cur == nil || id > *cur {
			return new(id)
		}
		return cur
	}
	firstID := func(cur *uint, id uint) *uint {
		if cur == nil {
			return new(id)
		}
		return cur
	}

	var states []*hostVPPAppInstallState
	byBucket := make(map[uint]*hostVPPAppInstallState)
	for _, r := range reqs {
		var bucket uint
		if r.VPPAppTeamID != nil {
			bucket = *r.VPPAppTeamID
		}
		s := byBucket[bucket]
		if s == nil {
			s = &hostVPPAppInstallState{
				HostID:        k.HostID,
				AdamID:        k.AdamID,
				Platform:      k.Platform,
				VersionBucket: bucket,
				VPPAppTeamID:  r.VPPAppTeamID,
			}
			byBucket[bucket] = s
			states = append(states, s)
		}
		if r.Canceled {
			continue
		}
		if s.LatestInstallID == nil {
			s.LatestInstallID = new(r.ID)
			s.LatestCreatedAt = r.CreatedAt
			s.LatestCommandUUID = new(r.CommandUUID)
			s.LatestVerificationAt = r.VerificationAt
			s.LatestVerificationFailedAt = r.VerificationFailedAt
		}
		if r.VerificationFailedAt == nil {
			s.UnfailedInstallID = firstID(s.UnfailedInstallID, r.ID)
		}
		if r.Removed {
			continue
		}
		if s.VisibleInstallID == nil {
			s.VisibleInstallID = new(r.ID)
			s.VisibleCreatedAt = r.CreatedAt
			s.VisibleCommandUUID = new(r.CommandUUID)
			s.VisibleVerificationAt = r.VerificationAt
			s.VisibleVerificationFailedAt = r.VerificationFailedAt
		}
		switch {
		case r.VerificationAt != nil:
			s.InstalledInstallID = maxID(s.InstalledInstallID, r.ID)
		case r.VerificationFailedAt != nil:
			s.FailedInstallID = maxID(s.FailedInstallID, r.ID)
		default:
			s.UnfinishedInstallID = maxID(s.UnfinishedInstallID, r.ID)
		}
	}

	if len(states) == 0 {
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM host_vpp_app_installs WHERE host_id = ? AND adam_id = ? AND platform = ?`,
			k.HostID, k.AdamID, k.Platform); err != nil {
			return ctxerr.Wrap(ctx, err, "delete vpp app install state")
		}
		return nil
	}

	buckets := make([]uint, 0, len(states))
	args := make([]any, 0, len(states)*19)
	for _, s := range states {
		buckets = append(buckets, s.VersionBucket)
		args = append(args, s.HostID, s.AdamID, s.Platform, s.VersionBucket, s.VPPAppTeamID,
			s.InstalledInstallID, s.FailedInstallID, s.UnfinishedInstallID,
			s.VisibleInstallID, s.LatestInstallID, s.UnfailedInstallID,
			s.VisibleCreatedAt, s.VisibleCommandUUID, s.VisibleVerificationAt, s.VisibleVerificationFailedAt,
			s.LatestCreatedAt, s.LatestCommandUUID, s.LatestVerificationAt, s.LatestVerificationFailedAt)
	}
	values := strings.TrimSuffix(strings.Repeat("(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),", len(states)), ",")
	//nolint:gosec // values is placeholders only
	if _, err := tx.ExecContext(ctx, `
INSERT INTO host_vpp_app_installs (
	host_id, adam_id, platform, version_bucket, vpp_app_team_id,
	installed_install_id, failed_install_id, unfinished_install_id,
	visible_install_id, latest_install_id, unfailed_install_id,
	visible_created_at, visible_command_uuid, visible_verification_at, visible_verification_failed_at,
	latest_created_at, latest_command_uuid, latest_verification_at, latest_verification_failed_at
) VALUES `+values+`
ON DUPLICATE KEY UPDATE
	vpp_app_team_id = VALUES(vpp_app_team_id),
	installed_install_id = VALUES(installed_install_id),
	failed_install_id = VALUES(failed_install_id),
	unfinished_install_id = VALUES(unfinished_install_id),
	visible_install_id = VALUES(visible_install_id),
	latest_install_id = VALUES(latest_install_id),
	unfailed_install_id = VALUES(unfailed_install_id),
	visible_created_at = VALUES(visible_created_at),
	visible_command_uuid = VALUES(visible_command_uuid),
	visible_verification_at = VALUES(visible_verification_at),
	visible_verification_failed_at = VALUES(visible_verification_failed_at),
	latest_created_at = VALUES(latest_created_at),
	latest_command_uuid = VALUES(latest_command_uuid),
	latest_verification_at = VALUES(latest_verification_at),
	latest_verification_failed_at = VALUES(latest_verification_failed_at)`, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert vpp app install state")
	}

	stmt, args, err := sqlx.In(
		`DELETE FROM host_vpp_app_installs WHERE host_id = ? AND adam_id = ? AND platform = ? AND version_bucket NOT IN (?)`,
		k.HostID, k.AdamID, k.Platform, buckets)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build delete stale vpp app install state")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "delete stale vpp app install state")
	}
	return nil
}

type hostVPPAppInstallRequest struct {
	ID uint `db:"id"`
	hostVPPAppKey
}

var errVPPAppInstallKeysChanged = errors.New("vpp app install target key changed after discovery")

// Rediscovery runs outside the transaction so waiting for key locks doesn't pin its snapshot.
func (ds *Datastore) withVPPAppInstallKeyRetry(ctx context.Context, discover func() ([]hostVPPAppKey, error), fn func(sqlx.ExtContext, []hostVPPAppKey) error) error {
	var err error
	for range 3 {
		keys, discoveryErr := discover()
		if discoveryErr != nil {
			return discoveryErr
		}
		err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			return fn(tx, keys)
		})
		if !errors.Is(err, errVPPAppInstallKeysChanged) {
			return err
		}
	}
	return ctxerr.Wrap(ctx, err, "vpp app install key discovery retry limit reached")
}

// discoverVPPAppInstallKeys runs on the primary before an owned transaction, so discovery
// doesn't establish its snapshot before waiting for the key locks.
func discoverVPPAppInstallKeys(ctx context.Context, q sqlx.QueryerContext, query string, args ...any) ([]hostVPPAppKey, error) {
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build vpp app install key discovery")
	}
	var requests []hostVPPAppInstallRequest
	if err := sqlx.SelectContext(ctx, q, &requests, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "discover vpp app install keys")
	}
	var keys []hostVPPAppKey
	for _, r := range requests {
		keys = append(keys, r.hostVPPAppKey)
	}
	return sortedVPPAppKeys(keys), nil
}

// lockVPPAppInstallRequests locks previously discovered keys before validating the current targets.
// A changed target set must retry discovery in a new transaction without widening these locks.
func lockVPPAppInstallRequests(ctx context.Context, tx sqlx.ExtContext, keys []hostVPPAppKey, query string, args ...any) ([]uint, []hostVPPAppKey, error) {
	if err := lockVPPAppInstallKeys(ctx, tx, keys); err != nil {
		return nil, nil, err
	}
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build vpp app install request selection")
	}
	var requests []hostVPPAppInstallRequest
	if err := sqlx.SelectContext(ctx, tx, &requests, query+` FOR SHARE`, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "validate vpp app install request keys")
	}
	lockedKeys := make(map[hostVPPAppKey]struct{}, len(keys))
	for _, key := range keys {
		lockedKeys[key] = struct{}{}
	}

	var ids []uint
	for _, r := range requests {
		if _, ok := lockedKeys[r.hostVPPAppKey]; !ok {
			return nil, nil, ctxerr.Wrap(ctx, errVPPAppInstallKeysChanged, "validate vpp app install request keys")
		}
		ids = append(ids, r.ID)
	}
	return ids, keys, nil
}
