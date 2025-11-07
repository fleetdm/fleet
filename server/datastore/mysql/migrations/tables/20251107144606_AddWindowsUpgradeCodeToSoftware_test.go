package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20251107144606(t *testing.T) {
	db := applyUpToPrev(t)

	ms := fleet.SoftwareTitle{
		Name:             "iTerm.app",
		Source:           "apps",
		BundleIdentifier: ptr.String("com.googlecode.iterm2"),
		UpgradeCode:      nil,
	}

	ws1 := fleet.SoftwareTitle{
		Name:        "Notepad",
		Source:      "programs",
		UpgradeCode: ptr.String("{1BF42825-7B65-4CA9-AFFF-B7B5E1CE27B4}"),
	}

	ws2 := fleet.SoftwareTitle{
		Name:        "NoteFad",
		Source:      "programs",
		UpgradeCode: ptr.String(""),
	}

	// Add Mac and Windows software, no upgrade codes yet. The unique_identifier should be the bundle_identifier for the
	// macOS software and the name for the Windows software.

	// these type conversions are safe from integer overflow since they are all sourced from database
	// auto-incremented ids, which there will only be a small amount of in the context of this test
	ms.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, bundle_identifier) VALUES (?, ?, ?)`, ms.Name, ms.Source, ms.BundleIdentifier)) //nolint:gosec // dismiss G115
	ws1.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES (?, ?)`, ws1.Name, ws1.Source))                                         //nolint:gosec // dismiss G115
	ws2.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES (?, ?)`, ws2.Name, ws2.Source))                                         //nolint:gosec // dismiss G115

	// // //
	// Apply current migration.
	applyNext(t, db)
	// // //

	// Check default values are set as expected
	var winUC *string
	err := db.Get(&winUC, `SELECT upgrade_code FROM software_titles WHERE id = ?`, ws1.ID)
	require.NoError(t, err)
	require.Equal(t, "", *winUC)

	err = db.Get(&winUC, `SELECT upgrade_code FROM software_titles WHERE id = ?`, ws2.ID)
	require.NoError(t, err)
	require.Equal(t, "", *winUC)

	var macUC *string
	err = db.Get(&macUC, `SELECT upgrade_code FROM software_titles WHERE id = ?`, ms.ID)
	require.NoError(t, err)
	require.Nil(t, macUC)

	// Delete the existing Windows software, then them back now with one empty and one non-empty upgrade_code
	execNoErr(t, db, `DELETE FROM software_titles WHERE id IN (?, ?)`, ws1.ID, ws2.ID)

	ws1.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, upgrade_code) VALUES (?, ?, ?)`, ws1.Name, ws1.Source, ws1.UpgradeCode)) //nolint:gosec // dismiss G115
	ws2.ID = uint(execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, upgrade_code) VALUES (?, ?, ?)`, ws2.Name, ws2.Source, ws2.UpgradeCode)) //nolint:gosec // dismiss G115

	cases := []struct {
		name                string
		titleID             uint
		source              string
		expectedBundleID    *string
		expectedUpgradeCode *string
		expectedUniqueID    string
	}{
		{
			name:                "macSW",
			titleID:             ms.ID,
			source:              ms.Source,
			expectedBundleID:    ms.BundleIdentifier,
			expectedUpgradeCode: ms.UpgradeCode,       // nil
			expectedUniqueID:    *ms.BundleIdentifier, // expect COALESCE to choose populated bundle id
		},
		{
			name:                "winSW with UC",
			titleID:             ws1.ID,
			source:              ws1.Source,
			expectedBundleID:    nil,
			expectedUpgradeCode: ws1.UpgradeCode,
			expectedUniqueID:    *ws1.UpgradeCode, // expect COALESCE to choose populated upgrade code
		},
		{
			name:                "winSW no UC",
			titleID:             ws2.ID,
			source:              ws2.Source,
			expectedBundleID:    nil,
			expectedUpgradeCode: ws2.UpgradeCode, // ""
			expectedUniqueID:    ws2.Name,        // expect NULLIF to nullify "" so COALESCE chooses the software name
		},
	}

	for _, tC := range cases {
		t.Run(tC.name, func(t *testing.T) {
			var title fleet.SoftwareTitle
			err := db.Get(&title, `SELECT id, source, bundle_identifier, upgrade_code FROM software_titles WHERE id = ?`, tC.titleID)
			require.NoError(t, err)
			if title.ID == ms.ID {
				// mac
				require.Nil(t, title.UpgradeCode)
				require.NotNil(t, title.BundleIdentifier)
				assert.Equal(t, *tC.expectedBundleID, *title.BundleIdentifier)
			} else {
				// windows
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
