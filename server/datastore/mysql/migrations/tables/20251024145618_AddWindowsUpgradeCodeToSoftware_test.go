package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20251024145618(t *testing.T) {
	db := applyUpToPrev(t)

	// Add Mac and Windows software. The unique_identifier should be the bundle_identifier for the macOS software and the name for the Windows software.
	stIDMac := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, bundle_identifier) VALUES ("iTerm.app", "apps", "com.googlecode.iterm2")`)
	stIDWindows := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ("Notepad", "programs")`)

	// TODO - add to `software` table

	// Apply current migration.
	applyNext(t, db)

	// Delete the existing Windows software, then add it back now including an upgrade_code
	// software_titles
	execNoErr(t, db, `DELETE FROM software_titles WHERE id = ?`, stIDWindows)

	uC := "{1BF42825-7B65-4CA9-AFFF-B7B5E1CE27B4}"
	stIDWindows = execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, upgrade_code) VALUES ("Notepad", "programs", ?)`, uC)
	// TODO software

	cases := []struct {
		name        string
		titleID     int64
		source      string
		expectedBID *string
		expectedUC  *string
	}{
		{
			name:        "macOS software title",
			titleID:     stIDMac,
			source:      "apps",
			expectedBID: ptr.String("com.googlecode.iterm2"),
			expectedUC:  nil,
		},
		{
			name:        "windows software title",
			titleID:     stIDWindows,
			source:      "programs",
			expectedBID: nil,
			expectedUC:  ptr.String(uC),
		},
	}

	for _, tC := range cases {
		t.Run(tC.name, func(t *testing.T) {
			var title fleet.SoftwareTitle
			err := db.Get(&title, `SELECT id, source, bundle_identifier, upgrade_code FROM software_titles WHERE id = ?`, tC.titleID)
			require.NoError(t, err)
			if tC.expectedUC == nil {
				// mac sw
				require.Nil(t, title.UpgradeCode)
				require.NotNil(t, title.BundleIdentifier)
				assert.Equal(t, *tC.expectedBID, *title.BundleIdentifier)
			} else {
				// windows sw
				require.Nil(t, title.BundleIdentifier)
				require.NotNil(t, title.UpgradeCode)
				assert.Equal(t, *tC.expectedUC, *title.UpgradeCode)
			}

			// TODO - test unique_identifier?
		})
	}
}
