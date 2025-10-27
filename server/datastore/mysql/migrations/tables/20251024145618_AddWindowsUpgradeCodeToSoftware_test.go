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

	ms := fleet.SoftwareTitle{
		Name:             "iTerm.app",
		Source:           "apps",
		BundleIdentifier: ptr.String("com.googlecode.iterm2"),
		UpgradeCode:      nil,
	}

	ws := fleet.SoftwareTitle{
		Name:        "Notepad",
		Source:      "programs",
		UpgradeCode: ptr.String("{1BF42825-7B65-4CA9-AFFF-B7B5E1CE27B4}"),
	}

	// Add Mac and Windows software, no upgrade codes yet. The unique_identifier should be the bundle_identifier for the
	// macOS software and the name for the Windows software.

	ms.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, ?, ?)`, ms.Name, ms.Source, ms.BundleIdentifier))
	ws.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES (?, ?)`, ws.Name, ws.Source))

	// // //
	// Apply current migration.
	applyNext(t, db)
	// // //

	// Check default values are set as expected
	var winUC *string
	err := db.Get(&winUC, `SELECT upgrade_code FROM software_titles WHERE id = ?`, ws.ID)
	require.NoError(t, err)
	require.Equal(t, "", *winUC)

	var macUC *string
	err = db.Get(&macUC, `SELECT upgrade_code FROM software_titles WHERE id = ?`, ms.ID)
	require.NoError(t, err)
	require.Nil(t, macUC)

	// Delete the existing Windows software, then add it back now including an upgrade_code
	// software_titles
	execNoErr(t, db, `DELETE FROM software_titles WHERE id = ?`, ws.ID)

	ws.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, upgrade_code) VALUES ("Notepad", "programs", ?)`, ws.UpgradeCode))

	cases := []struct {
		name                string
		titleID             uint
		source              string
		expectedBundleID    *string
		expectedUpgradeCode *string
		expectedUniqueID    string
	}{
		{
			name:                "macOS software title",
			titleID:             ms.ID,
			source:              ms.Source,
			expectedBundleID:    ms.BundleIdentifier,
			expectedUpgradeCode: ms.UpgradeCode,       // nil
			expectedUniqueID:    *ms.BundleIdentifier, // expect COALESCE to choose populated bundle id
		},
		{
			name:                "windows software title",
			titleID:             ws.ID,
			source:              ws.Source,
			expectedBundleID:    nil,
			expectedUpgradeCode: ws.UpgradeCode,
			expectedUniqueID:    *ws.UpgradeCode, // expect COALESCE to choose populated upgrade code
		},
	}

	for _, tC := range cases {
		t.Run(tC.name, func(t *testing.T) {
			var title fleet.SoftwareTitle
			err := db.Get(&title, `SELECT id, source, bundle_identifier, upgrade_code FROM software_titles WHERE id = ?`, tC.titleID)
			require.NoError(t, err)
			if tC.expectedUpgradeCode == nil {
				// mac sw
				require.Nil(t, title.UpgradeCode)
				require.NotNil(t, title.BundleIdentifier)
				assert.Equal(t, *tC.expectedBundleID, *title.BundleIdentifier)
			} else {
				// windows sw
				require.Nil(t, title.BundleIdentifier)
				require.NotNil(t, title.UpgradeCode)
				assert.Equal(t, *tC.expectedUpgradeCode, *title.UpgradeCode)
			}

			var gotUniqueID string
			err = db.Get(&gotUniqueID, "SELECT unique_identifier FROM software_titles WHERE id = ?", tC.titleID)
			require.NoError(t, err)

			assert.Equal(t, tC.expectedUniqueID, gotUniqueID)
		})
	}
}
