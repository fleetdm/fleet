package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// OPENFRAME(mysql-multitenancy): helpers for shared-database row-level tenant isolation.
// Each per-tenant Fleet process is pinned to one team via
// FLEET_OPENFRAME_TEAM_ID (fleet.OpenframeTeamID); these helpers scope by-id datastore
// operations to that team so one tenant cannot read/mutate another tenant's rows in the
// shared database.

// filterHostIDsByTeam returns the subset of hostIDs that belong to teamID. It is used to
// fence destructive/by-id host operations to the calling process's pinned team. The order
// of the result is not significant.
func filterHostIDsByTeam(ctx context.Context, q sqlx.QueryerContext, hostIDs []uint, teamID uint) ([]uint, error) {
	if len(hostIDs) == 0 {
		return hostIDs, nil
	}
	stmt, args, err := sqlx.In(`SELECT id FROM hosts WHERE id IN (?) AND team_id = ?`, hostIDs, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building team-scoped host id filter")
	}
	var owned []uint
	if err := sqlx.SelectContext(ctx, q, &owned, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "filtering host ids by team")
	}
	return owned, nil
}

// filterPolicyIDsByTeam returns the subset of policy ids that belong to teamID. Used to fence
// by-id policy deletes to the calling process's pinned team on a shared DB.
func filterPolicyIDsByTeam(ctx context.Context, q sqlx.QueryerContext, ids []uint, teamID uint) ([]uint, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	stmt, args, err := sqlx.In(`SELECT id FROM policies WHERE id IN (?) AND team_id = ?`, ids, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building team-scoped policy id filter")
	}
	var owned []uint
	if err := sqlx.SelectContext(ctx, q, &owned, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "filtering policy ids by team")
	}
	return owned, nil
}

// filterQueryIDsByTeam returns the subset of query ids that belong to teamID. Used to fence by-id
// query deletes to the calling process's pinned team on a shared DB.
func filterQueryIDsByTeam(ctx context.Context, q sqlx.QueryerContext, ids []uint, teamID uint) ([]uint, error) {
	if len(ids) == 0 {
		return ids, nil
	}
	stmt, args, err := sqlx.In(`SELECT id FROM queries WHERE id IN (?) AND team_id = ?`, ids, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building team-scoped query id filter")
	}
	var owned []uint
	if err := sqlx.SelectContext(ctx, q, &owned, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "filtering query ids by team")
	}
	return owned, nil
}

// openframeForeignTeam reports whether teamID is NOT this process's pinned team (only when pinned).
// Used to reject explicit-team (URL fleet_id) control-plane requests for another tenant on a shared
// DB — the team param is caller-supplied and is not itself a tenant boundary. Returns false when
// unpinned (no scope) so upstream behavior is unchanged.
func openframeForeignTeam(ctx context.Context, teamID uint) bool {
	pinned, ok := fleet.OpenframeTeamID(ctx)
	return ok && pinned != teamID
}

// openframeTeamScope replaces the caller's team scope of an inventory read (software titles,
// vulnerabilities) with this process's pinned team, so counts, versions and CVEs come only from
// the tenant's own hosts. Unpinned it returns teamID unchanged.
func openframeTeamScope(ctx context.Context, teamID *uint) *uint {
	if pinned, ok := fleet.OpenframeTeamID(ctx); ok {
		return &pinned
	}
	return teamID
}

// VulnerabilityHostCountsUpdatedAt returns when vulnerability host counts were last recalculated for
// the whole instance, or the zero time if they never were. It is instance metadata, not tenant data,
// so it is deliberately not fenced.
func (ds *Datastore) VulnerabilityHostCountsUpdatedAt(ctx context.Context) (time.Time, error) {
	var updatedAt sql.NullTime
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &updatedAt, `SELECT MAX(updated_at) FROM vulnerability_host_counts`); err != nil {
		return time.Time{}, ctxerr.Wrap(ctx, err, "reading vulnerability host counts updated_at")
	}
	return updatedAt.Time, nil
}

// openframeScopePolicyHosts fences a host-assignment operation to this process's pinned team: it
// verifies the parent policy belongs to the team (NotFound otherwise) and returns the subset of
// hostIDs in the team. When unpinned it returns hostIDs unchanged. Pass nil hostIDs to use it as a
// parent-ownership guard only (e.g. for list endpoints).
func (ds *Datastore) openframeScopePolicyHosts(ctx context.Context, policyID uint, hostIDs []uint) ([]uint, error) {
	teamID, ok := fleet.OpenframeTeamID(ctx)
	if !ok {
		return hostIDs, nil
	}
	// Verify on the primary: these fences guard writes (Add/Remove/Replace host assignments), a
	// read-after-write where a lagging replica would spuriously 404 a just-created policy.
	var x int
	err := sqlx.GetContext(ctx, ds.writer(ctx), &x, `SELECT 1 FROM policies WHERE id = ? AND team_id = ?`, policyID, teamID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("Policy").WithID(policyID))
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "verify policy team for host assignment")
	}
	return filterHostIDsByTeam(ctx, ds.writer(ctx), hostIDs, teamID)
}

// openframeScopeQueryHosts is the query analog of openframeScopePolicyHosts.
func (ds *Datastore) openframeScopeQueryHosts(ctx context.Context, queryID uint, hostIDs []uint) ([]uint, error) {
	teamID, ok := fleet.OpenframeTeamID(ctx)
	if !ok {
		return hostIDs, nil
	}
	// Verify on the primary (read-after-write; see openframeScopePolicyHosts).
	var x int
	err := sqlx.GetContext(ctx, ds.writer(ctx), &x, `SELECT 1 FROM queries WHERE id = ? AND team_id = ?`, queryID, teamID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return nil, ctxerr.Wrap(ctx, notFound("Query").WithID(queryID))
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "verify query team for host assignment")
	}
	return filterHostIDsByTeam(ctx, ds.writer(ctx), hostIDs, teamID)
}

// EnsureOpenframeTeamID resolves a Flamingo tenant UUID to its Fleet team id, creating the team if
// it does not yet exist (idempotent and safe under concurrent process startup via the unique
// `openframe_tenant_uuid` index). It is the bridge between the platform's UUID tenant identity and
// Fleet's integer team_id: the process is pinned by UUID (FLEET_OPENFRAME_TENANT_UUID) and resolves
// to the int here at startup.
//
// A newly created team is seeded with one random team-scoped enroll secret (same default as the EE
// team-creation service), because agent enrollment is the tenant's entry point: without a secret a
// fresh tenant could never enroll a host (the pinned GET /spec/enroll_secret would return an empty
// set). It is also seeded with its per-tenant app config row (app_config_json, id = team id) built
// via ApplyDefaultsForNewInstalls — a tenant minted here is a new install from its own perspective,
// and this row is the shared-mode equivalent of the config that POST /setup persists on a dedicated
// Fleet. Without it, pinned config reads fall back to ApplyDefaults (upgrade semantics), which
// leaves software inventory — and therefore vulnerabilities — permanently off for the tenant.
// All seeds happen only on the create path, in the same transaction as the team INSERT — an
// existing team's secrets and config (operator-applied or backfilled) are never touched.
func (ds *Datastore) EnsureOpenframeTeamID(ctx context.Context, tenantUUID string) (uint, error) {
	// Read on the primary: this runs at startup and must see a row another replica just created.
	selectID := func() (uint, bool, error) {
		var id uint
		err := sqlx.GetContext(ctx, ds.writer(ctx), &id,
			`SELECT id FROM teams WHERE openframe_tenant_uuid = ?`, tenantUUID)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return 0, false, nil
		case err != nil:
			return 0, false, ctxerr.Wrap(ctx, err, "selecting openframe team by tenant uuid")
		default:
			return id, true, nil
		}
	}

	if id, ok, err := selectID(); err != nil || ok {
		return id, err
	}

	// Not found: create a minimal team plus its default enroll secret in one transaction, so a
	// failure between the two cannot leave a permanently secret-less team (the create path would
	// never run again for this UUID). Name is derived from the UUID so it satisfies the unique
	// team-name constraint without colliding with operator-named teams.
	var id uint
	err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO teams (name, openframe_tenant_uuid, config) VALUES (?, ?, '{"mdm": {}}')`,
			"openframe-"+tenantUUID, tenantUUID)
		if err != nil {
			return err // likely a concurrent create (unique tenant_uuid / name); re-selected below
		}
		lastID, err := res.LastInsertId()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "reading created openframe team id")
		}
		id = uint(lastID) //nolint:gosec // team ids are small positive ints
		secret, err := server.GenerateRandomText(fleet.EnrollSecretDefaultLength)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating openframe team enroll secret")
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO enroll_secrets (secret, team_id) VALUES (?, ?)`, secret, id); err != nil {
			return ctxerr.Wrap(ctx, err, "seeding openframe team enroll secret")
		}
		appConfig := fleet.OpenframeDefaultAppConfig()
		configBytes, err := json.Marshal(appConfig)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "marshaling openframe team app config")
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO app_config_json (id, json_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE json_value = json_value`,
			id, configBytes); err != nil {
			return ctxerr.Wrap(ctx, err, "seeding openframe team app config")
		}
		return nil
	})
	if err != nil {
		// Likely a concurrent create by another replica (unique tenant_uuid / name): re-select.
		if id, ok, selErr := selectID(); selErr == nil && ok {
			return id, nil
		}
		return 0, ctxerr.Wrap(ctx, err, "creating openframe team for tenant uuid")
	}
	return id, nil
}

// openframeMigrationLockName is the cluster-wide MySQL named lock serializing schema
// migrations on a shared database. Fleet's goose pipeline takes no advisory lock, so with N
// clusters' migration jobs (and replicas) pointed at one shared MySQL, concurrent
// `fleet prepare db` runs would race on DDL (the idempotency guards are check-then-ALTER, not
// atomic).
const openframeMigrationLockName = "openframe_fleet_migrations"

// AcquireOpenframeMigrationLock takes the named MySQL lock guarding schema migrations, waiting
// up to timeout for a concurrent holder to finish. It returns a release func that must be
// called (deferred) when migrations complete. The lock is session-scoped: it is held on a
// dedicated connection and is automatically released by MySQL if the process dies, so a
// crashed migration job cannot wedge the lock.
func (ds *Datastore) AcquireOpenframeMigrationLock(ctx context.Context, timeout time.Duration) (release func(), err error) {
	conn, err := ds.writer(ctx).Conn(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting a connection for the openframe migration lock")
	}
	var got sql.NullInt64
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", openframeMigrationLockName, int(timeout.Seconds())).Scan(&got); err != nil {
		_ = conn.Close()
		return nil, ctxerr.Wrap(ctx, err, "acquiring the openframe migration lock")
	}
	if !got.Valid || got.Int64 != 1 {
		_ = conn.Close()
		return nil, ctxerr.Errorf(ctx, "timed out after %s waiting for the %q MySQL lock — is another migration run in progress?", timeout, openframeMigrationLockName)
	}
	return func() {
		// Best-effort explicit release; closing the connection releases the lock regardless.
		_, _ = conn.ExecContext(context.Background(), "SELECT RELEASE_LOCK(?)", openframeMigrationLockName)
		_ = conn.Close()
	}, nil
}
