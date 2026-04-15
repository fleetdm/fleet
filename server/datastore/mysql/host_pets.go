package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// GetHostPet returns the pet associated with the given host. Returns a
// not-found error when the host has not adopted a pet yet.
func (ds *Datastore) GetHostPet(ctx context.Context, hostID uint) (*fleet.HostPet, error) {
	const stmt = `
		SELECT
			id, host_id, name, species,
			health, happiness, hunger, cleanliness,
			last_interacted_at, created_at, updated_at
		FROM host_pets
		WHERE host_id = ?`

	var pet fleet.HostPet
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &pet, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("HostPet").WithID(hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host pet")
	}
	return &pet, nil
}

// CreateHostPet adopts a new pet for the given host. Returns an already-exists
// error if the host already has a pet. Stats start at sensible defaults.
func (ds *Datastore) CreateHostPet(ctx context.Context, hostID uint, name, species string) (*fleet.HostPet, error) {
	const insertStmt = `
		INSERT INTO host_pets (host_id, name, species)
		VALUES (?, ?, ?)`

	res, err := ds.writer(ctx).ExecContext(ctx, insertStmt, hostID, name, species)
	if err != nil {
		if IsDuplicate(err) {
			return nil, ctxerr.Wrap(ctx, alreadyExists("HostPet", hostID))
		}
		return nil, ctxerr.Wrap(ctx, err, "create host pet")
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host pet last insert id")
	}

	pet, err := ds.GetHostPet(ctx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load newly-created host pet")
	}
	// Belt-and-braces: make sure the returned pet is the one we just wrote.
	if pet.ID != uint(id) { //nolint:gosec // safe narrowing from AUTO_INCREMENT id
		return nil, ctxerr.New(ctx, "created pet id mismatch")
	}
	return pet, nil
}

// SaveHostPet persists the pet's stats and last_interacted_at.
func (ds *Datastore) SaveHostPet(ctx context.Context, pet *fleet.HostPet) error {
	const stmt = `
		UPDATE host_pets
		SET
			name = ?,
			species = ?,
			health = ?,
			happiness = ?,
			hunger = ?,
			cleanliness = ?,
			last_interacted_at = ?
		WHERE id = ?`

	res, err := ds.writer(ctx).ExecContext(ctx, stmt,
		pet.Name,
		pet.Species,
		pet.Health,
		pet.Happiness,
		pet.Hunger,
		pet.Cleanliness,
		pet.LastInteractedAt,
		pet.ID,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save host pet")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "save host pet rows affected")
	}
	if n == 0 {
		return ctxerr.Wrap(ctx, notFound("HostPet").WithID(pet.ID))
	}
	return nil
}

// ApplyHostPetHappinessDelta clamps and bumps the pet's happiness atomically
// in MySQL. No-op (returns nil) when the host has no pet — used by event-driven
// signals (e.g. self-service install success) that don't know whether the user
// has adopted.
func (ds *Datastore) ApplyHostPetHappinessDelta(ctx context.Context, hostID uint, delta int) error {
	// LEAST/GREATEST does the clamp inline so we don't need a read-then-write
	// round trip and there's no race between concurrent installs.
	const stmt = `
		UPDATE host_pets
		SET happiness = LEAST(?, GREATEST(?, CAST(happiness AS SIGNED) + ?))
		WHERE host_id = ?`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt,
		fleet.HostPetStatCeiling,
		fleet.HostPetStatFloor,
		delta,
		hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "apply host pet happiness delta")
	}
	return nil
}

// CountOpenHostVulnsBySeverity returns the count of open critical (CVSS >= 9)
// and high (7 <= CVSS < 9) CVEs affecting software installed on the host.
// Uninstalled software is not considered. Vulns missing CVSS scores are
// ignored — they can't be bucketed.
func (ds *Datastore) CountOpenHostVulnsBySeverity(ctx context.Context, hostID uint) (critical, high uint, err error) {
	const stmt = `
		SELECT
			COALESCE(SUM(CASE WHEN cm.cvss_score >= 9.0 THEN 1 ELSE 0 END), 0) AS critical_count,
			COALESCE(SUM(CASE WHEN cm.cvss_score >= 7.0 AND cm.cvss_score < 9.0 THEN 1 ELSE 0 END), 0) AS high_count
		FROM host_software hs
		JOIN software_cve sc ON sc.software_id = hs.software_id
		JOIN cve_meta cm     ON cm.cve = sc.cve
		WHERE hs.host_id = ?`

	var row struct {
		CriticalCount uint `db:"critical_count"`
		HighCount     uint `db:"high_count"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, hostID); err != nil {
		// No matching rows still produces a single SUM=0 row; an actual error here is
		// a real failure.
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, nil
		}
		return 0, 0, ctxerr.Wrap(ctx, err, "count open host vulns by severity")
	}
	return row.CriticalCount, row.HighCount, nil
}

//----------------------------------------------------------------------------//
// Demo overrides                                                             //
//----------------------------------------------------------------------------//

// GetHostPetDemoOverrides returns the override row for the host, or nil if
// none has been set. Returning nil-and-no-error here (rather than a NotFound)
// keeps callers branch-free: the demo overlay is "if non-nil, apply".
func (ds *Datastore) GetHostPetDemoOverrides(ctx context.Context, hostID uint) (*fleet.HostPetDemoOverrides, error) {
	const stmt = `
		SELECT
			host_id, seen_time_override, time_offset_hours,
			extra_failing_policies, extra_critical_vulns, extra_high_vulns,
			created_at, updated_at
		FROM host_pet_demo_overrides
		WHERE host_id = ?`

	var o fleet.HostPetDemoOverrides
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &o, stmt, hostID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get host pet demo overrides")
	}
	return &o, nil
}

// UpsertHostPetDemoOverrides creates or updates the override row. The caller
// owns the merge: pass the values you want stored. (Demo endpoints with PATCH
// semantics should read first, mutate, then call this.)
func (ds *Datastore) UpsertHostPetDemoOverrides(ctx context.Context, o *fleet.HostPetDemoOverrides) error {
	const stmt = `
		INSERT INTO host_pet_demo_overrides
			(host_id, seen_time_override, time_offset_hours,
			 extra_failing_policies, extra_critical_vulns, extra_high_vulns)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			seen_time_override     = VALUES(seen_time_override),
			time_offset_hours      = VALUES(time_offset_hours),
			extra_failing_policies = VALUES(extra_failing_policies),
			extra_critical_vulns   = VALUES(extra_critical_vulns),
			extra_high_vulns       = VALUES(extra_high_vulns)`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt,
		o.HostID,
		o.SeenTimeOverride,
		o.TimeOffsetHours,
		o.ExtraFailingPolicies,
		o.ExtraCriticalVulns,
		o.ExtraHighVulns,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "upsert host pet demo overrides")
	}
	return nil
}

// DeleteHostPetDemoOverrides removes the override row for a host. No-op when
// there's nothing to delete.
func (ds *Datastore) DeleteHostPetDemoOverrides(ctx context.Context, hostID uint) error {
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`DELETE FROM host_pet_demo_overrides WHERE host_id = ?`, hostID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host pet demo overrides")
	}
	return nil
}
