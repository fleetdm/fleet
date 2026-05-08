package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareTitlesSortByDisplayName(t *testing.T) {
	t.Parallel()

	t.Run("order key mapping uses NULLIF for empty display names", func(t *testing.T) {
		t.Parallel()
		// The "name" order key should use COALESCE(NULLIF(...)) to treat empty
		// display names as NULL, falling back to st.name.
		orderExpr, ok := softwareTitlesAllowedOrderKeys["name"]
		assert.True(t, ok)
		assert.Contains(t, orderExpr, "NULLIF(stdn.display_name, '')")
		assert.Contains(t, orderExpr, "st.name")
	})

	t.Run("secondary sort uses NULLIF for empty display names", func(t *testing.T) {
		t.Parallel()
		// When primary sort is NOT "name", the secondary sort should use COALESCE(NULLIF(...)).
		stmt := "SELECT * FROM t ORDER BY hosts_count DESC"
		result := spliceSecondaryOrderBySoftwareTitlesSQL(stmt, fleet.ListOptions{
			OrderKey:       "hosts_count",
			OrderDirection: fleet.OrderDescending,
		})
		assert.Contains(t, result, "COALESCE(NULLIF(stdn.display_name, ''), st.name) ASC")
	})

	t.Run("primary name sort does not add redundant secondary name sort", func(t *testing.T) {
		t.Parallel()
		// When primary sort IS "name", the secondary sort should be hosts_count, not another name sort.
		stmt := "SELECT * FROM t ORDER BY name ASC"
		result := spliceSecondaryOrderBySoftwareTitlesSQL(stmt, fleet.ListOptions{
			OrderKey:       "name",
			OrderDirection: fleet.OrderAscending,
		})
		assert.Contains(t, result, "hosts_count DESC")
		assert.NotContains(t, result, "COALESCE(NULLIF(stdn.display_name, ''), st.name) ASC")
	})

	t.Run("SQL template includes display_names join", func(t *testing.T) {
		t.Parallel()
		sql, _, err := selectSoftwareTitlesSQL(fleet.SoftwareTitleListOptions{
			TeamID: nil,
		})
		require.NoError(t, err)
		assert.Contains(t, sql, "software_title_display_names stdn")
	})

	t.Run("empty display name falls back to st.name in sort expression", func(t *testing.T) {
		t.Parallel()
		// Verify that NULLIF is used so empty strings are treated as NULL,
		// causing COALESCE to fall back to st.name for sorting.
		orderExpr := softwareTitlesAllowedOrderKeys["name"]
		// The expression should be: COALESCE(NULLIF(stdn.display_name, ''), st.name)
		// NULLIF returns NULL when display_name = '', so COALESCE picks st.name.
		assert.Equal(t, "COALESCE(NULLIF(stdn.display_name, ''), st.name)", orderExpr)
	})
}
