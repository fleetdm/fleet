//go:build windows

package homes

import (
	"os"
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// statOwner reads the owning account of dir from its security descriptor and
// returns the owner SID as a string together with its account name, for a home
// that ProfileList does not record (see platformHomes below, which is where
// ownership normally comes from). The SID is reported only if it names a user
// account.
func statOwner(dir string, _ os.FileInfo) (uid, username string, ok bool) {
	sid, ok := securityDescriptorOwner(dir)
	if !ok {
		return "", "", false
	}
	account, isUser := userAccountForSID(sid)
	if !isUser {
		return "", "", false
	}
	return sid.String(), account, true
}

// securityDescriptorOwner reads the OWNER field of dir's security descriptor.
func securityDescriptorOwner(dir string) (*windows.SID, bool) {
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION)
	if err != nil {
		return nil, false
	}
	sid, _, err := sd.Owner()
	if err != nil || sid == nil {
		return nil, false
	}
	return sid, true
}

// diskConsistent reports whether the directory on disk is consistent with
// ProfileList's claim that it is profileSID's profile. The registry entry is an
// assertion about a path, not about the directory now occupying it, so before
// it is trusted the directory must:
//
//   - be a real directory reached without a reparse point in the final
//     component — a junction or symlink here would attribute whatever tree it
//     points at (creating either needs no privilege) to profileSID, and
//   - not be OWNED by a different user account. Ownership is kernel-recorded
//     and settable only to SIDs in the setter's own token, so a directory an
//     unprivileged user planted at a stale or redirected entry's path is owned
//     by its creator and contradicts the claim. A group or service owner
//     (BUILTIN\Administrators on real admin profiles — the normal case) says
//     nothing about which user the profile belongs to and contradicts nothing.
//
// An owner that cannot be read counts as inconsistent: this check exists to
// veto forgery, and unverifiable is not verified.
func diskConsistent(profileSID, dir string) bool {
	claim, err := windows.StringToSid(profileSID)
	if err != nil {
		return false
	}
	fi, err := os.Lstat(dir)
	if err != nil || fi.Mode()&(os.ModeSymlink|os.ModeIrregular) != 0 || !fi.IsDir() {
		return false
	}
	owner, ok := securityDescriptorOwner(dir)
	if !ok {
		return false
	}
	if owner.Equals(claim) {
		return true
	}
	_, ownerIsUser := userAccountForSID(owner)
	return !ownerIsUser
}

// resolveUserAccount resolves a SID string to its account name, distinguishing
// a SID that positively names a non-user (dropped by profileHomes) from one
// the host simply cannot resolve right now (kept, unnamed). A malformed SID —
// ProfileList grows "<SID>.bak" subkeys, which are duplicates of real entries —
// counts as a non-user.
func resolveUserAccount(sid string) (string, accountResolution) {
	if isNonUserSID(sid) {
		return "", resolvedNonUser
	}
	s, err := windows.StringToSid(sid)
	if err != nil {
		return "", resolvedNonUser
	}
	account, _, accType, err := s.LookupAccount("")
	if err != nil {
		return "", unresolved
	}
	if accType != windows.SidTypeUser {
		return "", resolvedNonUser
	}
	return account, resolvedUser
}

// userAccountForSID returns the bare account name for sid, reporting false
// unless the account is a user — established by SID value first (see
// nonUserSIDs) and by the reported account type second.
//
// The name is deliberately bare: os/user formats Username as DOMAIN\account,
// whereas osquery's own users table carries the account name alone, and these
// columns exist to be lined up with that table.
func userAccountForSID(sid *windows.SID) (string, bool) {
	if isNonUserSID(sid.String()) {
		return "", false
	}
	account, _, accType, err := sid.LookupAccount("")
	if err != nil || accType != windows.SidTypeUser {
		return "", false
	}
	return account, true
}

// Windows records which user a profile directory belongs to in
// HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList: one subkey per
// account, named by SID, holding the profile's path in ProfileImagePath.
const profileListKey = `SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList`

// maxProfileListEntries bounds how many ProfileList records are read. Every
// entry can cost an account lookup, and on a domain-joined host with an
// unreachable domain controller each unresolvable SID is a network timeout —
// long-lived terminal servers accumulate stale entries in the hundreds. The
// bound is far above the number of live profiles a healthy host carries.
const maxProfileListEntries = 512

// profileEntry is one raw ProfileList record: the account SID naming the subkey
// and the profile directory that subkey points at.
type profileEntry struct {
	SID  string
	Path string
}

// accountResolution is what a SID lookup established about an account.
type accountResolution int

const (
	// resolvedUser: the SID names a user account, whose name came back with it.
	resolvedUser accountResolution = iota
	// resolvedNonUser: the SID names something that is not a user — a group, a
	// well-known account like SYSTEM, a service identity — or is malformed.
	resolvedNonUser
	// unresolved: the lookup failed outright, saying nothing about the account:
	// an unreachable domain controller, an Entra ID SID with no line to
	// resolve it, an account since deleted.
	unresolved
)

// nonUserSIDs and nonUserSIDPrefixes name the machine and service accounts that
// never belong to a person.
var (
	nonUserSIDs = []string{
		"S-1-5-18", // LocalSystem
		"S-1-5-19", // LocalService
		"S-1-5-20", // NetworkService
	}
	nonUserSIDPrefixes = []string{
		"S-1-5-32-", // BUILTIN aliases: Administrators, Users, ...
		"S-1-5-80-", // NT SERVICE virtual accounts
		"S-1-5-82-", // IIS application pool identities
		"S-1-5-83-", // NT VIRTUAL MACHINE accounts
		"S-1-5-90-", // Window Manager (DWM-*)
		"S-1-5-96-", // Font Driver Host (UMFD-*)
	}
)

// platformHomes returns the profile directories Windows records against a user
// account. See the ProfileList comment above for why the registry, not the
// filesystem, is the source of truth for ownership.
func platformHomes() []Home {
	entries := readProfileList()
	for i := range entries {
		// Canonicalize to the long-name spelling: a ProfileImagePath stored in
		// 8.3 short form (migrated hosts) would otherwise evade the case-folded
		// de-duplication in All() and the same profile would be reported twice.
		entries[i].Path = longPathName(entries[i].Path)
	}
	return profileHomes(entries, resolveUserAccount, diskConsistent)
}

// profileHomes converts raw ProfileList records into homes, keeping only the
// ones that verifiably belong to a user account. lookup and verified are
// injected so tests can construct hosts the runner doesn't have (domain groups,
// unreachable domain controllers, planted directories); production wires in
// resolveUserAccount and diskConsistent. Each entry must pass, in order:
//
//   - a path shape check: absolute, on a local drive letter, with at least one
//     component under the root. This drops empty and unexpanded values, and UNC
//     paths, which would otherwise send a SYSTEM process authenticating to a
//     remote share.
//   - not a machine, service, or builtin SID (see nonUserSIDs). Checked by
//     value before lookup so no account resolution is spent on them.
//   - lookup must not resolve the SID as a non-user. A SID that resolves as a
//     group or well-known account (every host has service profiles under
//     ProfileList, and none belongs to a person) is dropped: a column
//     documented as a user ID is better left empty than filled with a group
//     SID. A SID the lookup cannot resolve at all — unreachable domain
//     controller, offline Entra ID, deleted account — is kept with an empty
//     username: it is still the profile's admin-attested owner (only accounts
//     that log on get ProfileList subkeys, never groups), and reporting it
//     keeps offline correlation against the users table working.
//   - verified reports the directory on disk is consistent with the entry — a
//     real directory (not a reparse point) whose OWNER does not name a
//     different user (see diskConsistent).
func profileHomes(entries []profileEntry, lookup func(sid string) (string, accountResolution), verified func(sid, dir string) bool) []Home {
	var out []Home
	for _, e := range entries {
		if !isLocalDriveAbs(e.Path) {
			continue
		}
		if isNonUserSID(e.SID) {
			continue
		}
		username, res := lookup(e.SID)
		if res == resolvedNonUser {
			continue
		}
		if !verified(e.SID, e.Path) {
			continue
		}
		out = append(out, Home{UID: e.SID, Username: username, Dir: e.Path})
	}
	return out
}

// isLocalDriveAbs reports whether path has the shape of an absolute path on a
// local drive letter naming something under the drive root (`X:\name...`).
// Deliberately stricter than filepath.IsAbs, which also accepts the UNC paths
// this must reject. It remains a shape check: a drive letter mapped to a
// network share still passes, and the on-disk ownership check is the backstop
// there.
func isLocalDriveAbs(path string) bool {
	if len(path) < 4 || path[1] != ':' || path[2] != '\\' {
		return false
	}
	c := path[0]
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z'
}

// isNonUserSID reports whether sid names a machine, service, or builtin
// account by SID value alone (see nonUserSIDs).
func isNonUserSID(sid string) bool {
	for _, s := range nonUserSIDs {
		if strings.EqualFold(sid, s) {
			return true
		}
	}
	for _, p := range nonUserSIDPrefixes {
		if len(sid) >= len(p) && strings.EqualFold(sid[:len(p)], p) {
			return true
		}
	}
	return false
}

// readProfileList returns raw ProfileList records, bounded by
// maxProfileListEntries. The key is readable by any account, so this works
// whether or not the daemon is elevated.
func readProfileList() []profileEntry {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, profileListKey, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return nil
	}
	defer k.Close()

	// With a positive bound, reaching the end of the key before the bound
	// reports io.EOF. Any other error can only leave a partial listing, which
	// is still worth attributing homes from, so the names are used regardless.
	sids, _ := k.ReadSubKeyNames(maxProfileListEntries)
	out := make([]profileEntry, 0, len(sids))
	for _, sid := range sids {
		sk, err := registry.OpenKey(k, sid, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		path, valType, err := sk.GetStringValue("ProfileImagePath")
		sk.Close()
		if err != nil {
			continue
		}
		if valType == registry.EXPAND_SZ {
			expanded, err := registry.ExpandString(path)
			if err != nil {
				continue
			}
			path = expanded
		}
		out = append(out, profileEntry{SID: sid, Path: path})
	}
	return out
}

// longPathName returns the long-name spelling of path, best-effort: on any
// failure (path missing, buffer trouble) the path is returned as given and the
// later gates decide its fate.
func longPathName(path string) string {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return path
	}
	buf := make([]uint16, windows.MAX_PATH)
	n, err := windows.GetLongPathName(p, &buf[0], uint32(len(buf))) //nolint:gosec // G115: len(buf) is MAX_PATH here, or the size the API itself asked for below
	if err != nil || n == 0 {
		return path
	}
	if int(n) > len(buf) {
		// n is the required buffer size, terminator included.
		buf = make([]uint16, n)
		n, err = windows.GetLongPathName(p, &buf[0], uint32(len(buf))) //nolint:gosec // G115: len(buf) == n, a uint32 the API reported
		if err != nil || n == 0 || int(n) > len(buf) {
			return path
		}
	}
	return windows.UTF16ToString(buf[:n])
}
