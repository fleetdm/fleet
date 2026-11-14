//go:build windows

package fsutil

import "golang.org/x/sys/windows"

// The access-mask constants in perm_acl.go are hand-declared so that file builds
// on every platform, which leaves nine literal hex values one typo away from
// being silently wrong: TestWorldPermFromACEs uses those same constants on both
// sides of its assertions, so a bad value is self-consistent and invisible
// there, and the icacls test would only notice a mask that was broken outright.
//
// Pin them to x/sys's definitions here, where those are in scope. Each line is
// zero while the two agree; any difference makes it a negative constant, which
// does not fit in uint, so the package fails to compile with "constant -N
// overflows uint" pointing at the offending line.
//
// fileAllAccess is absent below because x/sys declares no FILE_ALL_ACCESS.
// TestStatPermWindowsACL covers it against a DACL that icacls really wrote.
const (
	_ uint = -(fileReadData ^ windows.FILE_READ_DATA)
	_ uint = -(fileWriteData ^ windows.FILE_WRITE_DATA)
	_ uint = -(fileAppendData ^ windows.FILE_APPEND_DATA)
	_ uint = -(stdDelete ^ windows.DELETE)
	_ uint = -(stdWriteDAC ^ windows.WRITE_DAC)
	_ uint = -(stdWriteOwner ^ windows.WRITE_OWNER)
	_ uint = -(genericAll ^ windows.GENERIC_ALL)
	_ uint = -(genericExecute ^ windows.GENERIC_EXECUTE)
	_ uint = -(genericWrite ^ windows.GENERIC_WRITE)
	_ uint = -(genericRead ^ windows.GENERIC_READ)

	// The generic mapping mapGenericRights applies.
	_ uint = -(fileGenericRead ^ windows.FILE_GENERIC_READ)
	_ uint = -(fileGenericWrite ^ windows.FILE_GENERIC_WRITE)
	_ uint = -(fileGenericExecute ^ windows.FILE_GENERIC_EXECUTE)
)
