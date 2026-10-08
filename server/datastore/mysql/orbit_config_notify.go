package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// WithOrbitConfigNotifier sets the notifier that tells connected agents to
// fetch their orbit config after a write changing it commits. Unset, no
// notifications are sent (agents rely on polling).
func (ds *Datastore) WithOrbitConfigNotifier(n fleet.AgentCheckInNotifier) {
	ds.orbitConfigNotifier = n
}

// notifyOrbitConfig tells hostIDs to fetch their orbit config once tx commits
// (immediately when tx is not a transaction), mirroring the APNs push in
// activateNextUpcomingActivity: notifying earlier would let the agent fetch
// before the change is visible, and a rolled back change notifies no one.
// Errors are logged: the agent's fallback poll still picks the change up.
func (ds *Datastore) notifyOrbitConfig(ctx context.Context, tx sqlx.ExtContext, reason string, hostIDs ...uint) {
	if ds.orbitConfigNotifier == nil || len(hostIDs) == 0 {
		return
	}

	// The hook may run after the request that made the change has ended.
	ctx = context.WithoutCancel(ctx)
	notify := func() {
		if err := ds.orbitConfigNotifier.NotifyOrbitConfigHosts(ctx, hostIDs, reason); err != nil {
			ds.logger.ErrorContext(ctx, "notify orbit config change", "reason", reason, "err", err)
		}
	}
	if wtx, ok := tx.(common_mysql.WrappedExtContext); ok {
		wtx.AddOnCommitHook(notify)
		return
	}
	notify()
}

// notifyOrbitConfigByHostUUIDs is notifyOrbitConfig for writers keyed by host
// UUID. The UUIDs are resolved within tx, and only when a notifier is set. A
// failed lookup is returned: it may mean the transaction is no longer usable.
func (ds *Datastore) notifyOrbitConfigByHostUUIDs(ctx context.Context, tx sqlx.ExtContext, reason string, hostUUIDs ...string) error {
	if ds.orbitConfigNotifier == nil || len(hostUUIDs) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(`SELECT id FROM hosts WHERE uuid IN (?)`, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build orbit config notification host UUIDs query")
	}
	return ds.notifyOrbitConfigByQuery(ctx, tx, reason, stmt, args...)
}

// notifyOrbitConfigWindowsMDMSync tells the hosts of the given
// mdm_windows_enrollments, now with pending commands, to fetch their orbit
// config: it asks fleetd to start an MDM session (WindowsMDMSyncRequest).
// Hosts whose fleetd can't are skipped.
func (ds *Datastore) notifyOrbitConfigWindowsMDMSync(ctx context.Context, tx sqlx.ExtContext, enrollmentIDs []uint) error {
	if ds.orbitConfigNotifier == nil || len(enrollmentIDs) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(`
SELECT h.id
FROM mdm_windows_enrollments mwe
JOIN hosts h ON h.uuid = mwe.host_uuid
WHERE mwe.id IN (?) AND mwe.fleetd_sync_capable = 1`, enrollmentIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build orbit config notification enrollment IDs query")
	}
	return ds.notifyOrbitConfigByQuery(ctx, tx, fleet.AgentWSReasonMDM, stmt, args...)
}

func (ds *Datastore) notifyOrbitConfigByQuery(ctx context.Context, tx sqlx.ExtContext, reason, stmt string, args ...any) error {
	var hostIDs []uint
	if err := sqlx.SelectContext(ctx, tx, &hostIDs, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "resolve hosts for orbit config notification")
	}
	ds.notifyOrbitConfig(ctx, tx, reason, hostIDs...)
	return nil
}
