package externalrefs

import (
	"fmt"
	"regexp"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

var sublimeTextBuildRegex = regexp.MustCompile(`sublime_text_build_(\d{4})_`)

// SublimeTextFileVersion sets the version to the file version of sublime_text.exe.
//
// Sublime Text's installer registers no DisplayVersion, so the patch policy
// compares osquery's file.file_version instead. The exe encodes build 4215 as
// "4.2.1.5", while winget reports "4.0.0.421500". The build is read from the
// installer URL (sublime_text_build_4215_x64_setup.exe).
func SublimeTextFileVersion(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	m := sublimeTextBuildRegex.FindStringSubmatch(app.InstallerURL)
	if m == nil {
		return app, fmt.Errorf("no 4-digit Sublime Text build number in installer URL '%s'", app.InstallerURL)
	}

	app.Version = strings.Join(strings.Split(m[1], ""), ".")

	return app, nil
}
