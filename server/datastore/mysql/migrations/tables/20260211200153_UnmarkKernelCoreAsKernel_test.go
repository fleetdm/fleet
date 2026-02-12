package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260211200153(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert software titles with is_kernel = 1 for various kernel packages

	// kernel-core with rpm_packages source - should be unmarked as kernel
	kernelCoreTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, is_kernel)
		VALUES ('kernel-core', 'rpm_packages', 1)
	`)

	// kernel with rpm_packages source - should remain as kernel
	kernelTitleID := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, is_kernel)
		VALUES ('kernel', 'rpm_packages', 1)
	`)

	// Insert kernel_host_counts for each software title
	_, err := db.Exec(`
		INSERT INTO kernel_host_counts (software_title_id, software_id, os_version_id, hosts_count, team_id)
		VALUES (?, 1, 100, 5, 0)
	`, kernelCoreTitleID)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO kernel_host_counts (software_title_id, software_id, os_version_id, hosts_count, team_id)
		VALUES (?, 2, 100, 3, 0)
	`, kernelTitleID)
	require.NoError(t, err)

	// Verify initial state
	var initialCount int
	err = db.Get(&initialCount, `SELECT COUNT(*) FROM kernel_host_counts`)
	require.NoError(t, err)
	require.Equal(t, 2, initialCount, "Should have 3 kernel_host_counts entries before migration")

	// Apply migration
	applyNext(t, db)

	// Verify software_titles is_kernel values after migration
	t.Run("kernel-core with rpm_packages is unmarked", func(t *testing.T) {
		var isKernel int
		err := db.Get(&isKernel, `SELECT is_kernel FROM software_titles WHERE id = ?`, kernelCoreTitleID)
		require.NoError(t, err)
		require.Equal(t, 0, isKernel, "kernel-core with rpm_packages should have is_kernel = 0")
	})

	t.Run("kernel with rpm_packages remains marked", func(t *testing.T) {
		var isKernel int
		err := db.Get(&isKernel, `SELECT is_kernel FROM software_titles WHERE id = ?`, kernelTitleID)
		require.NoError(t, err)
		require.Equal(t, 1, isKernel, "kernel with rpm_packages should still have is_kernel = 1")
	})

	// Verify kernel_host_counts cleanup
	t.Run("kernel_host_counts for kernel-core rpm_packages deleted", func(t *testing.T) {
		var count int
		err := db.Get(&count, `
			SELECT COUNT(*) FROM kernel_host_counts khc
			JOIN software_titles st ON st.id = khc.software_title_id
			WHERE st.name = 'kernel-core' AND st.source = 'rpm_packages'
		`)
		require.NoError(t, err)
		require.Equal(t, 0, count, "kernel_host_counts for kernel-core/rpm_packages should be deleted")
	})

	t.Run("kernel_host_counts for kernel rpm_packages preserved", func(t *testing.T) {
		var count int
		err := db.Get(&count, `
			SELECT COUNT(*) FROM kernel_host_counts khc
			JOIN software_titles st ON st.id = khc.software_title_id
			WHERE st.name = 'kernel' AND st.source = 'rpm_packages'
		`)
		require.NoError(t, err)
		require.Equal(t, 1, count, "kernel_host_counts for kernel/rpm_packages should be preserved")
	})

	t.Run("total kernel_host_counts reduced by one", func(t *testing.T) {
		var finalCount int
		err := db.Get(&finalCount, `SELECT COUNT(*) FROM kernel_host_counts`)
		require.NoError(t, err)
		require.Equal(t, 1, finalCount, "Should have 2 kernel_host_counts entries after migration (one deleted)")
	})
}
