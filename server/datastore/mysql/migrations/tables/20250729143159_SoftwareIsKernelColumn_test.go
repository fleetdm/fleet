package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250729143159(t *testing.T) {
	db := applyUpToPrev(t)

	// Name as reported for Ubuntu
	kernelID1 := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("linux-image-6.11.0-9-generic", "deb_packages", "")`)
	// Name as reported for Debian
	kernelID2 := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("linux-image-6.1.0-37-cloud-arm64", "deb_packages", "")`)
	amazonKernelID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("kernel", "rpm_packages", "")`)
	rhelKernelID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("kernel-core", "rpm_packages", "")`)
	otherLinuxAppID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("vim", "deb_packages", "")`)
	otherAppMacOSID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("Calculator", "apps", "")`)
	otherAppWindowsID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("Notepad", "programs", "")`)

	// Apply current migration.
	applyNext(t, db)

	tests := []struct {
		name           string
		titleID        int64
		shouldBeKernel bool
	}{
		{
			name:           "ubuntu kernel",
			titleID:        kernelID1,
			shouldBeKernel: true,
		},
		{
			name:           "debian kernel",
			titleID:        kernelID2,
			shouldBeKernel: true,
		},
		{
			name:           "amazon linuxkernel",
			titleID:        amazonKernelID,
			shouldBeKernel: true,
		},
		{
			name:           "rhel kernel",
			titleID:        rhelKernelID,
			shouldBeKernel: true,
		},
		{
			name:           "other linux title",
			titleID:        otherLinuxAppID,
			shouldBeKernel: false,
		},
		{
			name:           "other title macOS",
			titleID:        otherAppMacOSID,
			shouldBeKernel: false,
		},
		{
			name:           "other title Windows",
			titleID:        otherAppWindowsID,
			shouldBeKernel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var isKernel bool
			err := db.Get(&isKernel, `SELECT is_kernel FROM software_titles WHERE id = ?`, tt.titleID)
			require.NoError(t, err)
			require.Equal(t, tt.shouldBeKernel, isKernel)
		})
	}

}
