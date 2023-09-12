package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230908141250(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := `
          INSERT INTO cve_meta
            (cve)
          VALUES
            (?)
	`
	cveVal := "CVE-2010-3262"
	execNoErr(t, db, insertStmt, cveVal)

	applyNext(t, db)

	// retrieve the stored value
	var cveMeta struct {
		CVE         string
		Description *string
	}
	err := db.Get(&cveMeta, "SELECT * FROM cve_meta WHERE cve = ?", cveVal)
	require.NoError(t, err)
	require.Equal(t, cveVal, cveMeta.CVE)
	require.Nil(t, cveMeta.Description)

	insertStmt = `
          INSERT INTO cve_meta
            (cve, description)
          VALUES
            (?, ?)
	`
	cveVal = "CVE-2010-3263"
	descVal := "Cross-site scripting (XSS) vulnerability in setup/frames/index.inc.php in the setup script in phpMyAdmin 3.x before 3.3.7 allows remote attackers to inject arbitrary web script or HTML via a server name."
	execNoErr(t, db, insertStmt, cveVal, descVal)
	err = db.Get(&cveMeta, "SELECT * FROM cve_meta WHERE cve = ?", cveVal)
	require.NoError(t, err)
	require.Equal(t, cveVal, cveMeta.CVE)
	require.Equal(t, &descVal, cveMeta.Description)
}
