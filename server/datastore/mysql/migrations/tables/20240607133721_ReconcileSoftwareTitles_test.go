package tables

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20240607133721(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert data into software_titles
	title1 := execNoErrLastID(t, db, "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)", "sw1", "src1", "")

	// Insert software
	const insertStmt = `INSERT INTO software
		(name, version, source, browser, checksum, title_id)
	VALUES
		(?, ?, ?, ?, ?, ?)`

	execNoErr(t, db, insertStmt, "sw1", "1.0", "src1", "", "1", title1)
	execNoErr(t, db, insertStmt, "sw1", "1.0.1", "src1", "", "1a", nil)
	execNoErr(t, db, insertStmt, "sw2", "2.0", "src2", "", "2", nil)
	execNoErr(t, db, insertStmt, "sw3", "3.0", "src3", "browser3", "3", nil)

	applyNext(t, db)

	var softwareTitles []fleet.SoftwareTitle
	require.NoError(t, db.SelectContext(context.Background(), &softwareTitles, `SELECT * FROM software_titles`))
	require.Len(t, softwareTitles, 3)

	var software []fleet.Software
	require.NoError(t, db.SelectContext(context.Background(), &software, `SELECT id, name, source, browser, title_id FROM software`))
	require.Len(t, software, 4)

	for _, sw := range software {
		require.NotNil(t, sw.TitleID)
		var found bool
		for _, title := range softwareTitles {
			if *sw.TitleID == title.ID {
				assert.Equal(t, sw.Name, title.Name)
				assert.Equal(t, sw.Source, title.Source)
				assert.Equal(t, sw.Browser, title.Browser)
				found = true
				break
			}
		}
		assert.True(t, found)
	}

}
