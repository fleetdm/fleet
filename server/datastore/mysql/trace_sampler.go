package mysql

import (
	"context"
	"database/sql"
	"errors"

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
	var settings fleet.TraceSamplerSettings
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &settings, stmt, traceSamplerSettingsID); err != nil {
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
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt,
		settings.HighVolumeRatio,
		settings.StandardRatio,
		settings.ForceFull,
		traceSamplerSettingsID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "set trace_sampler_settings")
	}
	return nil
}
