package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestPreProcessUninstallScript(t *testing.T) {
	var input = `
blah$PACKAGE_IDS
pkgids=$PACKAGE_ID
they are $PACKAGE_ID, right?
here $PACKAGE_ID reside $MY_SECRET
more ${PACKAGE_ID}
${PACKAGE_ID}`

	payload := fleet.UploadSoftwareInstallerPayload{
		Extension:       "exe",
		UninstallScript: input,
		PackageIDs:      []string{"com.foo"},
	}

	preProcessUninstallScript(&payload)
	expected := `
blah$PACKAGE_IDS
pkgids="com.foo"
they are "com.foo", right?
here "com.foo" reside $MY_SECRET
more "com.foo"
"com.foo"`
	assert.Equal(t, expected, payload.UninstallScript)

	payload = fleet.UploadSoftwareInstallerPayload{
		Extension:       "pkg",
		UninstallScript: input,
		PackageIDs:      []string{"com.foo", "com.bar"},
	}
	preProcessUninstallScript(&payload)
	expected = `
blah$PACKAGE_IDS
pkgids=(
  "com.foo"
  "com.bar"
)
they are (
  "com.foo"
  "com.bar"
), right?
here (
  "com.foo"
  "com.bar"
) reside $MY_SECRET
more (
  "com.foo"
  "com.bar"
)
(
  "com.foo"
  "com.bar"
)`
	assert.Equal(t, expected, payload.UninstallScript)

}
