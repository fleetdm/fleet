// package build provides build metadata through variables set at build time
// (with -ldflags="-X ...")
package build

var (
	// Version is the commit tag version number for release builds, or a
	// generated version for untagged builds.
	Version string
	// Commit is the commit SHA.
	Commit string
	// Date is the date of the build.
	Date string
)
