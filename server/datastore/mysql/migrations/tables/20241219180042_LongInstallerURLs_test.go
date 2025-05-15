package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241219180042(t *testing.T) {
	db := applyUpToPrev(t)

	script1 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo hi', 'a')")
	script2 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo bye', 'b')")

	software := execNoErrLastID(t, db, `
INSERT INTO software_installers (
  filename,
  version,
  platform,
  install_script_content_id,
  post_install_script_content_id,
  uninstall_script_content_id,
  storage_id,
  package_ids,
  url
) VALUES (
  'fleet',
  '1.0.0',
  'windows',
  ?,
  ?,
  ?,
  'a',
  '',
  ?
)`, script1, script2, script2, "https://google.com/")

	applyNext(t, db)

	var url string
	err := db.Get(&url, "SELECT url FROM software_installers WHERE id = ?", software)
	require.NoError(t, err)
	require.Equal(t, "https://google.com/", url)

	longUrl := "https://dl.google.com/tag/s/appguid%3D%7B8A69D345-D564-463C-AFF1-A69D9E530F96%7D%26iid%3D%7B53CCDE8D-FD40-46DE-67E7-61E96CFEFCAA%7D%26lang%3Den%26browser%3D4%26usagestats%3D0%26appname%3DGoogle%2520Chrome%26needsadmin%3Dtrue%26ap%3Dx64-stable-statsdef_0%26brand%3DGCEA/dl/chrome/install/googlechromestandaloneenterprise64.msi"
	execNoErr(t, db, `UPDATE software_installers SET url = ? WHERE id = ?`, longUrl, software)

	err = db.Get(&url, "SELECT url FROM software_installers WHERE id = ?", software)
	require.NoError(t, err)
	require.Equal(t, longUrl, url)
}
