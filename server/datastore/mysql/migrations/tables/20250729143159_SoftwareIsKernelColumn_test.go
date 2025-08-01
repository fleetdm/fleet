package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250729143159(t *testing.T) {
	db := applyUpToPrev(t)

	kernelID1 := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("linux-image-6.11.0-9-generic", "deb_packages", "")`)
	kernelID2 := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ("linux-image-5.12.2-9-generic", "deb_packages", "")`)
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
			name:           "linux kernel 1",
			titleID:        kernelID1,
			shouldBeKernel: true,
		},
		{
			name:           "linux kernel 2",
			titleID:        kernelID2,
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
