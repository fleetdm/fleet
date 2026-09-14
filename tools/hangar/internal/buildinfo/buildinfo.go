// Package buildinfo carries the identity of the running Hangar build.
//
// Hangar is handed around as an ad-hoc-signed `.app` and rebuilt often, so
// "which build is this?" has to be answerable from three places that don't
// share any state: the bundle's Info.plist (Finder, and Fleet's own software
// inventory), the Settings tab (whoever is running it), and hangar.log
// (whoever is reading a crash report afterwards). All three are fed from here.
//
// The release string comes from `build/config.yml`, which is the single place
// to bump it; the commit and date are stamped by the build task. See the Logs
// and Versioning sections of the README.
package buildinfo

// Stamped at build time with -ldflags -X (see build/darwin/Taskfile.yml). The
// defaults below are what an unstamped build reports — a bare `go build`,
// `go run`, or a test binary — so an unstamped build is never mistaken for a
// released one.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// Info identifies one build. It crosses to the frontend as-is.
type Info struct {
	// Version is the release string from build/config.yml, or "dev".
	Version string `json:"version"`
	// Commit is the short git SHA, suffixed "-dirty" when the working tree
	// had changes — which is the normal case for a build someone made to test
	// a branch, and worth knowing when they report what it did.
	Commit string `json:"commit"`
	// Date is when the binary was built (RFC 3339, UTC), or "unknown".
	Date string `json:"date"`
}

// Current returns the identity of this build.
func Current() Info {
	return Info{Version: version, Commit: commit, Date: date}
}

// Summary is the one-line form: "1.1.0 (a1b2c3d, built 2026-08-14T09:12:03Z)".
func (i Info) Summary() string {
	s := i.Version
	if i.Commit != "" {
		s += " (" + i.Commit
		if i.Date != "" && i.Date != "unknown" {
			s += ", built " + i.Date
		}
		s += ")"
	}
	return s
}
