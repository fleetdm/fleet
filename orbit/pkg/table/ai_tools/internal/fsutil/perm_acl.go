package fsutil

// This file holds the Windows DACL decision logic, deliberately split from the
// Win32 calls in perm_windows.go so it can be tested on any platform. Only
// perm_windows.go uses it; the tests exercise it everywhere.

// Well-known SIDs whose membership is effectively "any user who can log in to
// this machine". A grant to one of them is the Windows analogue of the POSIX
// group/other permission bits that drive Perm on macOS and Linux.
//
// BUILTIN\Users (S-1-5-32-545) is deliberately excluded: several standard
// locations (ProgramData, for one) grant it create-only rights by default, so
// treating it as "world" would report a risk flag on ordinary installs.
const (
	sidEveryone           = "S-1-1-0"
	sidAuthenticatedUsers = "S-1-5-11"
)

func isWorldSID(sid string) bool {
	return sid == sidEveryone || sid == sidAuthenticatedUsers
}

// Windows file access-mask bits (winnt.h). Declared here rather than taken from
// golang.org/x/sys/windows so this file stays buildable on every platform.
const (
	fileReadData   = 0x00000001 // FILE_READ_DATA
	fileWriteData  = 0x00000002 // FILE_WRITE_DATA
	fileAppendData = 0x00000004 // FILE_APPEND_DATA
	stdDelete      = 0x00010000 // DELETE
	stdWriteDAC    = 0x00040000 // WRITE_DAC
	stdWriteOwner  = 0x00080000 // WRITE_OWNER
	genericAll     = 0x10000000 // GENERIC_ALL
	genericWrite   = 0x40000000 // GENERIC_WRITE
	genericRead    = 0x80000000 // GENERIC_READ

	fileAllAccess   = 0x001F01FF // FILE_ALL_ACCESS — what `icacls /grant <x>:F` writes
	fileGenericRead = 0x00120089 // FILE_GENERIC_READ
)

// writeMask is every bit that lets the holder change the file's contents, or
// escalate to being able to. DELETE allows replacing the file wholesale, and
// WRITE_DAC/WRITE_OWNER allow granting yourself the rest — all three are as good
// as write for an attacker editing an agent instruction file.
const writeMask = fileWriteData | fileAppendData | stdDelete | stdWriteDAC | stdWriteOwner | genericWrite | genericAll

const readMask = fileReadData | genericRead | genericAll

// aceEntry is one DACL entry reduced to the fields the world-permission decision
// needs.
type aceEntry struct {
	SID         string
	Allow       bool   // ACCESS_ALLOWED_ACE_TYPE; false means ACCESS_DENIED_ACE_TYPE
	InheritOnly bool   // INHERIT_ONLY_ACE — applies to children, not this object
	Mask        uint32 // ACCESS_MASK
}

// worldPermFromACEs evaluates a DACL for read/write access granted to a world
// SID. ACEs are processed in order and the first one touching a given right
// decides it, which is how the Windows access check behaves given the canonical
// DACL ordering that puts deny ACEs ahead of allow ACEs.
func worldPermFromACEs(aces []aceEntry) Perm {
	var read, write, readDecided, writeDecided bool
	for _, a := range aces {
		if a.InheritOnly || !isWorldSID(a.SID) {
			continue
		}
		if !readDecided && a.Mask&readMask != 0 {
			read, readDecided = a.Allow, true
		}
		if !writeDecided && a.Mask&writeMask != 0 {
			write, writeDecided = a.Allow, true
		}
	}
	return Perm{WorldReadable: read, WorldWritable: write, Known: true}
}
