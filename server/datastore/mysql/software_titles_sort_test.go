package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSoftwareTitlesSortByDisplayName(t *testing.T) {
	t.Parallel()

	t.Run("order key mapping uses display name", func(t *testing.T) {
		t.Parallel()
		// The "name" order key should use COALESCE to prefer display_name over st.name.
		orderExpr, ok := softwareTitlesAllowedOrderKeys["name"]
		assert.True(t, ok)
		assert.Contains(t, orderExpr, "COALESCE")
		assert.Contains(t, orderExpr, "stdn.display_name")
		assert.Contains(t, orderExpr, "st.name")
	})

	t.Run("secondary sort uses display name", func(t *testing.T) {
		t.Parallel()
		// When primary sort is NOT "name", the secondary sort should use COALESCE.
		stmt := "SELECT * FROM t ORDER BY hosts_count DESC"
		result := spliceSecondaryOrderBySoftwareTitlesSQL(stmt, fleet.ListOptions{
			OrderKey:       "hosts_count",
			OrderDirection: fleet.OrderDescending,
		})
		assert.Contains(t, result, "COALESCE(stdn.display_name, st.name) ASC")
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
		assert.NotContains(t, result, "COALESCE(stdn.display_name, st.name) ASC")
	})

	t.Run("SQL template includes display_names join", func(t *testing.T) {
		t.Parallel()
		sql, _, err := selectSoftwareTitlesSQL(fleet.SoftwareTitleListOptions{
			TeamID: nil,
		})
		require.NoError(t, err)
		assert.Contains(t, sql, "software_title_display_names stdn")
	})
}
