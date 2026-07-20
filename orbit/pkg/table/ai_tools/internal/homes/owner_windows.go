//go:build windows

package homes

import (
	"os"

	"golang.org/x/sys/windows"
)

// statOwnerUID reads the owning account of dir from its security descriptor and
// returns the owner SID as a string. os/user.LookupId resolves a SID to an
// account on Windows, so owner() maps it to a username the same way it maps a
// numeric uid on Unix — and, crucially, without trusting the directory name, so
// a folder named after another account can't forge the attribution.
//
// The FileInfo is unused: Win32 file attributes don't carry ownership, so the
// security descriptor must be queried by path.
func statOwnerUID(dir string, _ os.FileInfo) (string, bool) {
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return "", false
	}
	sid, _, err := sd.Owner()
	if err != nil || sid == nil {
		return "", false
	}
	return sid.String(), true
}
