package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
	"github.com/tj/assert"
)

func TestUp_20251015103700(t *testing.T) {
	db := applyUpToPrev(t)

	// Add some non-Android software. The unique_identifier should be the bundle_identifier for the macOS software and the name for the Windows software.
	stIDMac := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, bundle_identifier) VALUES ("iTerm.app", "apps", "com.googlecode.iterm2")`)
	stIDWindows := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ("Notepad", "programs")`)

	// Apply current migration.
	applyNext(t, db)

	// Now that the application_id column exists, we can add some Android software.
	stIDAndroid := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, application_id) VALUES ("YouTube", "android_apps", "com.google.youtube")`)

	cases := []struct {
		name                     string
		titleID                  int64
		expectedBundleID         *string
		expectedApplicationID    *string
		expectedUniqueIdentifier string
	}{
		{
			name:                     "macOS software title",
			titleID:                  stIDMac,
			expectedBundleID:         ptr.String("com.googlecode.iterm2"),
			expectedUniqueIdentifier: "com.googlecode.iterm2",
		},
		{
			name:                     "android software title",
			titleID:                  stIDAndroid,
			expectedApplicationID:    ptr.String("com.google.youtube"),
			expectedUniqueIdentifier: "com.google.youtube",
		},

		{
			name:                     "windows software title",
			titleID:                  stIDWindows,
			expectedUniqueIdentifier: "Notepad",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var title fleet.SoftwareTitle
			err := db.Get(&title, "SELECT id, name, source, extension_for, application_id, bundle_identifier FROM software_titles WHERE id = ?", tt.titleID)
			require.NoError(t, err)
			switch {
			case tt.expectedBundleID == nil:
				require.Nil(t, title.BundleIdentifier)

			case tt.expectedBundleID != nil:
				require.NotNil(t, tt.expectedBundleID)
				assert.Equal(t, *tt.expectedBundleID, *title.BundleIdentifier)

			case tt.expectedApplicationID == nil:
				require.Nil(t, title.ApplicationID)

			case tt.expectedApplicationID != nil:
				require.NotNil(t, title.ApplicationID)
				assert.Equal(t, tt.expectedApplicationID, title.ApplicationID)

			}

			var gotUniqueID string
			err = db.Get(&gotUniqueID, "SELECT unique_identifier FROM software_titles WHERE id = ?", tt.titleID)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedUniqueIdentifier, gotUniqueID)
		})
	}

}
