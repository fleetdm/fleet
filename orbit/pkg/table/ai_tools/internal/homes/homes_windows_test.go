//go:build windows

package homes

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

func TestAllNeverAttributesToANonUserSID(t *testing.T) {
	for _, h := range All() {
		if h.UID == "" {
			// Ownership could not be established. Both columns stay empty, which
			// is the documented outcome — checked below.
			if h.Username != "" {
				t.Errorf("home %q has username %q with no uid", h.Dir, h.Username)
			}
			continue
		}
		if _, res := resolveUserAccount(h.UID); res == resolvedNonUser {
			t.Errorf("home %q attributed to %q, which is not a user account", h.Dir, h.UID)
		}
	}
}

func TestAllAttributesTheCurrentUsersHome(t *testing.T) {
	cur, err := user.Current()
	if err != nil {
		t.Fatalf("user.Current: %v", err)
	}
	if _, res := resolveUserAccount(cur.Uid); res != resolvedUser {
		t.Skipf("running as %s, which does not resolve as a user account; no profile to attribute", cur.Username)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir: %v", err)
	}

	var got *Home
	for _, h := range All() {
		if strings.EqualFold(h.Dir, home) {
			got = &h
			break
		}
	}
	if got == nil {
		t.Fatalf("All() did not include the current user's home %q", home)
	}
	if got.UID != cur.Uid {
		t.Errorf("uid = %q, want the current account's SID %q", got.UID, cur.Uid)
	}
	// os/user formats Username as DOMAIN\account; the table reports the bare
	// account name, which is what osquery's own users table carries.
	if want := cur.Username[strings.LastIndex(cur.Username, `\`)+1:]; got.Username != want {
		t.Errorf("username = %q, want %q", got.Username, want)
	}
}

// ownerOf reads dir's security-descriptor OWNER, failing the test when it
// can't be read.
func ownerOf(t *testing.T, dir string) *windows.SID {
	t.Helper()
	owner, ok := securityDescriptorOwner(dir)
	if !ok {
		t.Fatalf("could not read the owner of %q", dir)
	}
	return owner
}

func TestStatOwnerNeverReportsANonUserOwner(t *testing.T) {
	dir := t.TempDir()
	admins, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		t.Fatalf("CreateWellKnownSid: %v", err)
	}
	if err := windows.SetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION,
		admins, nil, nil, nil); err != nil {
		t.Skipf("cannot make the directory Administrators-owned (needs an elevated token): %v", err)
	}
	if uid, username, ok := statOwner(dir, nil); ok {
		t.Errorf("statOwner = %q/%q for an Administrators-owned directory, want no owner", uid, username)
	}
}

func TestDiskConsistent(t *testing.T) {
	dir := t.TempDir()
	owner := ownerOf(t, dir)
	_, ownerIsUser := userAccountForSID(owner)

	if diskConsistent(sidLocalUser, filepath.Join(dir, "missing")) {
		t.Error("a path with no directory behind it was accepted")
	}
	if !diskConsistent(owner.String(), dir) {
		t.Errorf("a claim matching the on-disk owner %s was rejected", owner)
	}
	// sidLocalUser is a fabricated SID that exists on no host, so it never
	// matches the real owner: the claim must be rejected exactly when the owner
	// is itself a user account (which then contradicts the claim).
	if got := diskConsistent(sidLocalUser, dir); got != !ownerIsUser {
		t.Errorf("diskConsistent with a mismatched claim = %v; directory owner %s (user account: %v)",
			got, owner, ownerIsUser)
	}
}

func TestDiskConsistentRejectsReparsePoints(t *testing.T) {
	target := t.TempDir()
	junction := filepath.Join(t.TempDir(), "junction")
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Skipf("mklink /J: %v (%s)", err, out)
	}
	if diskConsistent(ownerOf(t, target).String(), junction) {
		t.Error("a junction was accepted as a profile directory")
	}
}

// The SIDs below are the shapes that really show up under ProfileList. Only the
// first two name a person's account; the rest are what makes the raw key
// unusable as-is.
const (
	sidLocalUser  = "S-1-5-21-4207797923-693487475-3254343130-1001"
	sidEntraUser  = "S-1-12-1-1234567890-1234567890-1234567890-1234567890"
	sidDeletedAcc = "S-1-5-21-4207797923-693487475-3254343130-1002"
	sidDomainUsrs = "S-1-5-21-4207797923-693487475-3254343130-513"
	sidSystem     = "S-1-5-18"
	sidLocalSvc   = "S-1-5-19"
)

// accountLookup stands in for LookupAccountSid so tests can construct accounts
// the runner doesn't have: SIDs in users resolve as user accounts, SIDs listed
// as unresolvable fail outright (unreachable domain controller, deleted
// account), and every other SID resolves as a group or well-known account,
// which is how the SidTypeUser gate sees them on a live host.
func accountLookup(users map[string]string, unresolvable ...string) func(string) (string, accountResolution) {
	return func(sid string) (string, accountResolution) {
		if name, ok := users[sid]; ok {
			return name, resolvedUser
		}
		if slices.Contains(unresolvable, sid) {
			return "", unresolved
		}
		return "", resolvedNonUser
	}
}

// diskAlwaysConsistent stands in for the on-disk verification, for tests
// exercising the other gates.
func diskAlwaysConsistent(string, string) bool { return true }

// checkProfileHomes runs profileHomes over entries and reports any deviation
// from exactly the homes wanted (nil to require rejection).
func checkProfileHomes(t *testing.T, entries []profileEntry, lookup func(string) (string, accountResolution), verified func(sid, dir string) bool, want []Home) {
	t.Helper()
	if got := profileHomes(entries, lookup, verified); !slices.Equal(got, want) {
		t.Errorf("profileHomes(%+v) = %+v, want %+v", entries, got, want)
	}
}

func TestProfileHomes(t *testing.T) {
	lookup := accountLookup(map[string]string{
		sidLocalUser: "xpkoa",
		sidEntraUser: "bob",
	}, sidDeletedAcc)

	entries := []profileEntry{
		// Service profiles. Every host has these, and none belongs to a person.
		{SID: sidSystem, Path: `C:\Windows\system32\config\systemprofile`},
		{SID: sidLocalSvc, Path: `C:\Windows\ServiceProfiles\LocalService`},

		{SID: sidLocalUser, Path: `C:\Users\xpkoa`},

		// A domain group SID: same S-1-5-21-<authority> shape as a user, so only
		// the account-type gate rejects it.
		{SID: sidDomainUsrs, Path: `C:\Users\domainusers`},

		// BUILTIN\Administrators
		{SID: "S-1-5-32-544", Path: `C:\Users\admins`},

		// Entra ID account whose profile was redirected off the system drive —
		// the directory listing of C:\Users never sees this one.
		{SID: sidEntraUser, Path: `D:\Profiles\bob`},

		// Leftover entry for an account the host can no longer resolve: the SID
		// still identifies the profile's owner, so it is kept, unnamed.
		{SID: sidDeletedAcc, Path: `C:\Users\departed`},

		// Present but with no ProfileImagePath value.
		{SID: sidLocalUser, Path: ""},
	}

	checkProfileHomes(t, entries, lookup, diskAlwaysConsistent, []Home{
		{UID: sidLocalUser, Username: "xpkoa", Dir: `C:\Users\xpkoa`},
		{UID: sidEntraUser, Username: "bob", Dir: `D:\Profiles\bob`},
		{UID: sidDeletedAcc, Username: "", Dir: `C:\Users\departed`},
	})
}

func TestProfileHomesRejectsNonLocalPaths(t *testing.T) {
	lookup := accountLookup(map[string]string{sidLocalUser: "xpkoa"})

	for _, path := range []string{
		``,
		`C:`,
		`C:\`,
		`Users\xpkoa`,
		`\Users\xpkoa`,
		`\\fileserver\profiles\xpkoa`,
		`C:/Users/xpkoa`,
		`%SystemDrive%\Users\xpkoa`,
	} {
		checkProfileHomes(t, []profileEntry{{SID: sidLocalUser, Path: path}}, lookup, diskAlwaysConsistent, nil)
	}
}

func TestProfileHomesRejectsNonUserSIDs(t *testing.T) {
	var looked []string
	lookup := func(sid string) (string, accountResolution) {
		looked = append(looked, sid)
		return "resolved", resolvedUser
	}

	for _, sid := range []string{
		"S-1-5-18", // LocalSystem — observed resolving as a user on a real host
		"S-1-5-19",
		"S-1-5-20",
		"S-1-5-32-544", // BUILTIN\Administrators
		"S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464",
		"S-1-5-82-3006700770-424185619-1745488364-794895919-4004696415",
		"S-1-5-83-1-2-3-4-5",
		"s-1-5-82-3006700770-424185619-1745488364-794895919-4004696415", // case must not matter
		"S-1-5-90-0-1",
		"S-1-5-96-0-0",
	} {
		checkProfileHomes(t, []profileEntry{{SID: sid, Path: `C:\Users\svcprofile`}}, lookup, diskAlwaysConsistent, nil)
	}
	if len(looked) != 0 {
		t.Errorf("profileHomes() resolved non-user SIDs %v; the by-value gate must reject them before any lookup", looked)
	}
}

func TestProfileHomesRequiresOnDiskConsistency(t *testing.T) {
	lookup := accountLookup(map[string]string{
		sidLocalUser: "xpkoa",
		sidEntraUser: "bob",
	}, sidDeletedAcc)

	var checked []profileEntry
	verify := func(sid, dir string) bool {
		checked = append(checked, profileEntry{SID: sid, Path: dir})
		return sid == sidLocalUser
	}

	entries := []profileEntry{
		{SID: sidLocalUser, Path: `C:\Users\xpkoa`},
		{SID: sidEntraUser, Path: `D:\Profiles\bob`}, // fails verification: planted or replaced
		{SID: sidSystem, Path: `C:\Windows\system32\config\systemprofile`},
		// The consistency check is not waived for a SID the lookup couldn't
		// resolve: it too must survive verification, and here it doesn't.
		{SID: sidDeletedAcc, Path: `C:\Users\departed`},
	}

	checkProfileHomes(t, entries, lookup, verify,
		[]Home{{UID: sidLocalUser, Username: "xpkoa", Dir: `C:\Users\xpkoa`}})

	// Every entry that survived the lookup gate was checked against the disk,
	// with the SID and path the registry paired; the SYSTEM entry never got
	// that far.
	wantChecked := []profileEntry{
		{SID: sidLocalUser, Path: `C:\Users\xpkoa`},
		{SID: sidEntraUser, Path: `D:\Profiles\bob`},
		{SID: sidDeletedAcc, Path: `C:\Users\departed`},
	}
	if !slices.Equal(checked, wantChecked) {
		t.Errorf("on-disk verification saw %+v, want %+v", checked, wantChecked)
	}
}
