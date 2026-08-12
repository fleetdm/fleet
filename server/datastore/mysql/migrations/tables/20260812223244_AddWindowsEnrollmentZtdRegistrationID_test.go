package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260812223244(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	var count int
	require.NoError(t, db.Get(&count, `
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = DATABASE() AND table_name = 'mdm_windows_enrollments' AND column_name = 'ztd_registration_id'`))
	require.Equal(t, 1, count)
}
