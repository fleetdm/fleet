package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		err := ds.SetTraceSamplerSettings(ctx, &fleet.TraceSamplerSettings{
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
		err := ds.SetTraceSamplerSettings(ctx, &fleet.TraceSamplerSettings{
			HighVolumeRatio: 1.5,
			StandardRatio:   0.02,
			ForceFull:       false,
		})
		require.Error(t, err)
	})
}
