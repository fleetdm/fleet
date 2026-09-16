//go:build windows

package fsutil

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// maxACEs bounds the DACL walk, so an implausibly long ACL can't turn one row of
// a table query into an unbounded loop of syscalls. No real file DACL is close.
const maxACEs = 4096

// statPerm reads path's DACL and reports whether it grants read or write to a
// well-known world SID — the Windows analogue of the POSIX other/group bits read
// on macOS and Linux.
//
// Every failure path reports an unknown posture rather than a clean one, so a
// file we couldn't inspect is never mistaken for a file we cleared. Reading the
// DACL needs an open handle, which POSIX mode bits don't, so a file held
// exclusively by another process reports unknown where Unix would still answer.
func statPerm(path string) Perm {
	// Go through OpenRegular to inherit its guards: this runs as SYSTEM over
	// user-writable paths, so it must refuse reparse points and non-regular files
	// rather than be redirected onto another object. GetNamedSecurityInfo would
	// skip the open, but it resolves reparse points, which is exactly what those
	// guards exist to prevent. O_RDONLY maps to GENERIC_READ, which includes the
	// READ_CONTROL right GetSecurityInfo needs.
	f, err := OpenRegular(path)
	if err != nil {
		return Perm{}
	}
	defer func() { _ = f.Close() }()

	sd, err := windows.GetSecurityInfo(windows.Handle(f.Fd()), windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return Perm{}
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return Perm{}
	}
	if dacl == nil {
		// A NULL DACL is not an absent one: it grants full access to everyone.
		return Perm{WorldReadable: true, WorldWritable: true, Known: true}
	}

	count := int(dacl.AceCount)
	if count > maxACEs {
		// Truncating would drop the tail of the list, which under canonical
		// ordering is where the allow ACEs live — a silent false negative. Report
		// a DACL this size as a posture we did not establish.
		return Perm{}
	}
	aces := make([]aceEntry, 0, count)
	for i := range count {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			return Perm{}
		}
		// Only the two basic ACE types carry a plain SID at SidStart; object and
		// callback ACE types lay out their trailing data differently, so reading a
		// SID from them would be reading the wrong bytes. Neither type appears on
		// an ordinary file DACL, and skipping one only loses a signal.
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE && ace.Header.AceType != windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		// SidStart is the first DWORD of the variable-length SID that the ACE
		// struct is only the fixed-size header of, so the SID is addressed
		// through it rather than copied out.
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart)) //nolint:gosec // G103: reading the variable-length SID that trails the fixed ACE header is the documented Win32 layout
		aces = append(aces, aceEntry{
			SID:         sid.String(),
			Allow:       ace.Header.AceType == windows.ACCESS_ALLOWED_ACE_TYPE,
			InheritOnly: ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0,
			Mask:        uint32(ace.Mask),
		})
	}
	return worldPermFromACEs(aces)
}
