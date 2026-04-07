package mysql

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestListAPIEndpoints(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	t.Run("no filter returns all", func(t *testing.T) {
		opts := fleet.ListOptions{}
		got, meta, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 5, count)
		require.Len(t, got, 5)
		require.False(t, meta.HasNextResults)
		require.False(t, meta.HasPreviousResults)
		require.NoError(t, err)
	})

	t.Run("filter by display_name case-insensitive", func(t *testing.T) {
		opts := fleet.ListOptions{MatchQuery: "LIST"}
		got, _, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 3, count)
		require.Len(t, got, 3)
		require.NoError(t, err)
		for _, e := range got {
			require.Contains(t, strings.ToLower(e.DisplayName), "list")
		}
	})

	t.Run("filter by normalized_path case-insensitive", func(t *testing.T) {
		opts := fleet.ListOptions{MatchQuery: ":PLACEHOLDER_1"}
		got, _, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 2, count)
		require.Len(t, got, 2)
		require.NoError(t, err)
	})

	t.Run("filter matches either display_name or normalized_path", func(t *testing.T) {
		// "software" appears in both the DisplayName of two entries and the NormalizedPath of two entries
		opts := fleet.ListOptions{MatchQuery: "software"}
		got, _, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 2, count)
		require.Len(t, got, 2)
		require.NoError(t, err)
	})

	t.Run("filter with no matches returns empty", func(t *testing.T) {
		opts := fleet.ListOptions{MatchQuery: "zzznomatch"}
		got, _, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 0, count)
		require.Empty(t, got)
		require.NoError(t, err)
	})

	t.Run("pagination first page", func(t *testing.T) {
		opts := fleet.ListOptions{Page: 0, PerPage: 2}
		got, meta, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 5, count)
		require.Len(t, got, 2)
		require.True(t, meta.HasNextResults)
		require.False(t, meta.HasPreviousResults)
		require.NoError(t, err)
	})

	t.Run("pagination middle page", func(t *testing.T) {
		opts := fleet.ListOptions{Page: 1, PerPage: 2}
		got, meta, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 5, count)
		require.Len(t, got, 2)
		require.True(t, meta.HasNextResults)
		require.True(t, meta.HasPreviousResults)
		require.NoError(t, err)
	})

	t.Run("pagination last page", func(t *testing.T) {
		opts := fleet.ListOptions{Page: 2, PerPage: 2}
		got, meta, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 5, count)
		require.Len(t, got, 1)
		require.False(t, meta.HasNextResults)
		require.True(t, meta.HasPreviousResults)
		require.NoError(t, err)
	})

	t.Run("pagination beyond last page returns empty", func(t *testing.T) {
		opts := fleet.ListOptions{Page: 99, PerPage: 2}
		got, meta, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 5, count)
		require.Empty(t, got)
		require.False(t, meta.HasNextResults)
		require.True(t, meta.HasPreviousResults)
		require.NoError(t, err)
	})

	t.Run("pagination with filter", func(t *testing.T) {
		opts := fleet.ListOptions{MatchQuery: "list", Page: 0, PerPage: 2}
		got, meta, count, err := ds.ListAPIEndpoints(ctx, opts)
		require.Equal(t, 3, count)
		require.Len(t, got, 2)
		require.True(t, meta.HasNextResults)
		require.False(t, meta.HasPreviousResults)
		require.NoError(t, err)
	})
}
