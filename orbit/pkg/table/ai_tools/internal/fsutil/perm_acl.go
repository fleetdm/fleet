package fsutil

// This file holds the Windows DACL decision logic, deliberately split from the
// Win32 calls in perm_windows.go so it can be tested on any platform. Only
// perm_windows.go uses it; the tests exercise it everywhere.

// Well-known SIDs whose membership is effectively "any user who can log in to
// this machine". A grant to one of them is the Windows analogue of the POSIX
// group/other permission bits that drive Perm on macOS and Linux.
//
// Only these two universal SIDs count. Local groups — BUILTIN\Users
// (S-1-5-32-545) above all — are excluded, so this is narrower than the POSIX
// side, which counts the group bit as well: a grant to a group whose membership
// varies per machine isn't the same claim as "anyone who can log in". Widening
// it is a product decision about what the risk flag means, not an
// implementation detail.
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
	genericExecute = 0x20000000 // GENERIC_EXECUTE
	genericWrite   = 0x40000000 // GENERIC_WRITE
	genericRead    = 0x80000000 // GENERIC_READ

	// The file object's GENERIC_MAPPING: what each generic right stands for on a
	// file. Note FILE_GENERIC_WRITE carries none of DELETE, WRITE_DAC or
	// WRITE_OWNER.
	fileAllAccess      = 0x001F01FF // FILE_ALL_ACCESS — GENERIC_ALL, `icacls /grant <x>:F`
	fileGenericRead    = 0x00120089 // FILE_GENERIC_READ — GENERIC_READ, `icacls /grant <x>:R`
	fileGenericWrite   = 0x00120116 // FILE_GENERIC_WRITE — GENERIC_WRITE, `icacls /grant <x>:W`
	fileGenericExecute = 0x001200A0 // FILE_GENERIC_EXECUTE — GENERIC_EXECUTE
)

// writeMask is every bit that lets the holder change the file's contents, or
// escalate to being able to. DELETE allows replacing the file wholesale, and
// WRITE_DAC/WRITE_OWNER allow granting yourself the rest — all three are as good
// as write for an attacker editing an agent instruction file.
//
// Both masks name specific rights only: mapGenericRights has already translated
// the generic aliases away by the time a mask is evaluated.
const writeMask = fileWriteData | fileAppendData | stdDelete | stdWriteDAC | stdWriteOwner

const readMask = fileReadData

// mapGenericRights rewrites an ACE mask's generic bits into the specific file
// rights they stand for, and clears them — the same translation MapGenericMask
// performs, and what the object manager is supposed to have done before a
// descriptor reaches an object. It has to be done here because generic bits do
// reach real file DACLs verbatim (icacls renders them GR/GW/GE/GA, and SDDL
// strings write them as-is), and precedence has to be decided in one vocabulary:
// otherwise a deny naming GENERIC_ALL and an allow naming FILE_ALL_ACCESS look
// like disjoint sets of rights and neither settles the other.
func mapGenericRights(m uint32) uint32 {
	if m&genericRead != 0 {
		m |= fileGenericRead
	}
	if m&genericWrite != 0 {
		m |= fileGenericWrite
	}
	if m&genericExecute != 0 {
		m |= fileGenericExecute
	}
	if m&genericAll != 0 {
		m |= fileAllAccess
	}
	return m &^ (genericRead | genericWrite | genericExecute | genericAll)
}

// aceEntry is one DACL entry reduced to the fields the world-permission decision
// needs.
type aceEntry struct {
	SID         string
	Allow       bool   // ACCESS_ALLOWED_ACE_TYPE; false means ACCESS_DENIED_ACE_TYPE
	InheritOnly bool   // INHERIT_ONLY_ACE — applies to children, not this object
	Mask        uint32 // ACCESS_MASK
}

// worldPermFromACEs evaluates a DACL for read/write access granted to a world
// SID, mirroring the Windows access check: ACEs are walked in order and each
// individual right is settled by the first ACE that mentions it, so an earlier
// deny beats a later allow (the canonical DACL ordering) but only for the bits
// it actually names.
//
// Resolving per bit rather than per ACE matters. A DACL of "deny Everyone the
// data-write rights, allow Everyone:(F)" — icacls /deny <sid>:(WD,AD) /grant
// <sid>:(F) — still leaves Everyone holding DELETE, WRITE_DAC and WRITE_OWNER,
// so the file is fully tamperable. Letting the deny settle write wholesale would
// report it as safe, which is a two-command way to hide a tampered file.
func worldPermFromACEs(aces []aceEntry) Perm {
	var allowed, decided uint32
	for _, a := range aces {
		if a.InheritOnly || !isWorldSID(a.SID) {
			continue
		}
		fresh := mapGenericRights(a.Mask) & ^decided // rights no earlier ACE has settled
		if fresh == 0 {
			continue
		}
		if a.Allow {
			allowed |= fresh
		}
		decided |= fresh
	}
	return Perm{
		WorldReadable: allowed&readMask != 0,
		WorldWritable: allowed&writeMask != 0,
		Known:         true,
	}
}
