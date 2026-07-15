package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/jmoiron/sqlx"
)

const traceSamplerSettingsID = 1

func (ds *Datastore) GetTraceSamplerSettings(ctx context.Context) (*tracing.Settings, error) {
	const stmt = `
		SELECT high_volume_ratio, standard_ratio, force_full, updated_at
		FROM trace_sampler_settings
		WHERE id = ?
	`
	var settings tracing.Settings
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &settings, stmt, traceSamplerSettingsID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("TraceSamplerSettings"))
		}
		return nil, ctxerr.Wrap(ctx, err, "get trace_sampler_settings")
	}
	return &settings, nil
}

func (ds *Datastore) SetTraceSamplerSettings(ctx context.Context, settings *tracing.Settings) error {
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
	// The singleton row is seeded by the migration. A missing row means the invariant is broken.
	if rows != 1 {
		return ctxerr.Wrap(ctx, fmt.Errorf("set trace_sampler_settings: expected 1 row updated, got %d", rows))
	}
	return nil
}
