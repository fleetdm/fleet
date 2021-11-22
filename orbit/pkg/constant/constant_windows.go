//+build windows

package constant

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const (
	PlatformName = "windows"
	// DefaultExecutableMode is the default file mode to apply to created
	// executable files. For Windows this doesn't do anything besides setting
	// read-only. See https://golang.org/pkg/os/#Chmod.
	DefaultExecutableMode = 0o700
)

var (
	// These identifiers can be found in
	// https://docs.microsoft.com/en-us/troubleshoot/windows-server/identity/security-identifiers-in-windows
	// and are used in the same fashion as in osquery. See
	// https://github.com/osquery/osquery/blob/d2be385d71f401c85872f00d479df8f499164c5a/tools/deployment/chocolatey/tools/osquery_utils.ps1.
	SystemSID = mustSID("S-1-5-18")
	AdminSID  = mustSID("S-1-5-32-544")
	UserSID   = mustSID("S-1-5-32-545")
)

func mustSID(identifier string) *windows.SID {
	sid, err := windows.StringToSid(identifier)
	if err != nil {
		panic(fmt.Errorf("create sid: %w", err))
	}
	return sid
}
