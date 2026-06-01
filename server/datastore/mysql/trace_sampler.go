package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const traceSamplerSettingsID = 1

func (ds *Datastore) GetTraceSamplerSettings(ctx context.Context) (*fleet.TraceSamplerSettings, error) {
	const stmt = `
		SELECT high_volume_ratio, standard_ratio, force_full, updated_at
		FROM trace_sampler_settings
		WHERE id = ?
	`
	// Read from the primary, not a replica. Operators who PATCH /debug/trace_sampler
	// expect their change to take effect on every replica's next 60s tick;
	// reading the row from a read replica adds a stale-read window equal to
	// the replication lag. The query is one indexed PK lookup per replica
	// per minute (~0.17 QPS at 10 replicas) — negligible primary load.
	var settings fleet.TraceSamplerSettings
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &settings, stmt, traceSamplerSettingsID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("TraceSamplerSettings"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get trace_sampler_settings")
	}
	return &settings, nil
}

func (ds *Datastore) SetTraceSamplerSettings(ctx context.Context, settings *fleet.TraceSamplerSettings) error {
	const stmt = `
		UPDATE trace_sampler_settings
		SET high_volume_ratio = ?, standard_ratio = ?, force_full = ?
		WHERE id = ?
	`
	res, err := ds.writer(ctx).ExecContext(ctx, stmt,
		settings.HighVolumeRatio,
		settings.StandardRatio,
		settings.ForceFull,
		traceSamplerSettingsID,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set trace_sampler_settings")
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set trace_sampler_settings: rows affected")
	}
	// The singleton row is seeded by the migration; a missing row means the
	// invariant is broken. Surface it loudly rather than silently no-op.
	if rows != 1 {
		return ctxerr.Wrap(ctx, fmt.Errorf("set trace_sampler_settings: expected 1 row updated, got %d", rows))
	}
	return nil
}
