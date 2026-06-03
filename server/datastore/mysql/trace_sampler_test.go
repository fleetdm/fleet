package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/platform/tracing"
	"github.com/stretchr/testify/require"
)

func TestTraceSamplerSettings(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	t.Run("seeded defaults are returned", func(t *testing.T) {
		got, err := ds.GetTraceSamplerSettings(ctx)
		require.NoError(t, err)
		require.NotNil(t, got)
		require.InDelta(t, 0.001, got.HighVolumeRatio, 1e-9)
		require.InDelta(t, 0.02, got.StandardRatio, 1e-9)
		require.False(t, got.ForceFull)
	})

	t.Run("round trip persists changes", func(t *testing.T) {
		err := ds.SetTraceSamplerSettings(ctx, &tracing.Settings{
			HighVolumeRatio: 0.005,
			StandardRatio:   0.1,
			ForceFull:       true,
		})
		require.NoError(t, err)

		got, err := ds.GetTraceSamplerSettings(ctx)
		require.NoError(t, err)
		require.InDelta(t, 0.005, got.HighVolumeRatio, 1e-9)
		require.InDelta(t, 0.1, got.StandardRatio, 1e-9)
		require.True(t, got.ForceFull)
	})

	t.Run("out of range ratio rejected by CHECK constraint", func(t *testing.T) {
		err := ds.SetTraceSamplerSettings(ctx, &tracing.Settings{
			HighVolumeRatio: 1.5,
			StandardRatio:   0.02,
			ForceFull:       false,
		})
		require.Error(t, err)
	})

	t.Run("set fails when singleton row is missing", func(t *testing.T) {
		// Locks in the RowsAffected != 1 guard in SetTraceSamplerSettings. If the seeded singleton row is missing (DB invariant
		// broken), Set must surface a loud error rather than silently no-op.
		_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM trace_sampler_settings`)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := ds.writer(ctx).ExecContext(ctx, `INSERT INTO trace_sampler_settings (id) VALUES (1)`)
			require.NoError(t, err)
		})

		err = ds.SetTraceSamplerSettings(ctx, &tracing.Settings{
			HighVolumeRatio: 0.01,
			StandardRatio:   0.05,
			ForceFull:       false,
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "expected 1 row updated")
	})
}
