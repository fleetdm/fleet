package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260922124827(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	for _, tc := range []struct{ table, index string }{
		{"windows_mdm_responses", "idx_windows_mdm_responses_created_at"},
		{"windows_mdm_commands", "idx_windows_mdm_commands_created_at"},
		{"host_mdm_actions", "idx_host_mdm_actions_wipe_ref"},
	} {
		require.True(t, indexExists(db, tc.table, tc.index), "%s.%s", tc.table, tc.index)
	}
}
