//go:build darwin

package apple_accounts

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDsclRecords(t *testing.T) {
	record := func(names, uid, gid, home string) string {
		r := "<dict><key>dsAttrTypeStandard:RecordName</key><array>" + names + "</array>"
		for key, value := range map[string]string{"UniqueID": uid, "PrimaryGroupID": gid, "NFSHomeDirectory": home} {
			if value != "" {
				r += "<key>dsAttrTypeStandard:" + key + "</key><array><string>" + value + "</string></array>"
			}
		}
		return r + "</dict>"
	}
	out := `<?xml version="1.0" encoding="UTF-8"?><plist version="1.0"><array>` +
		record("<string>_amavisd</string><string>amavisd</string>", "83", "83", "/var/virusmails") +
		record("<string>nobody</string>", "-2", "-2", "/var/empty") +
		record("<string>alice</string><string>Alice Example</string>", "501", "20", "/Users/alice/") +
		record("<string>bob</string>", "502", "", "/Volumes/Data/Home Directories/bob") +
		record("<string>carol</string>", "-1294967296", "-1294967296", "/Users/carol") +
		record("<string>nohome</string>", "503", "20", "") +
		record("<string>relative</string>", "504", "20", "Users/relative") +
		record("<string>baduid</string>", "x", "20", "/Users/baduid") +
		`</array></plist>`

	users, err := parseDsclRecords([]byte(out))
	require.NoError(t, err)
	assert.Equal(t, []localUser{
		{UID: "501", GID: "20", Username: "alice", HomeDir: "/Users/alice"},
		{UID: "502", Username: "bob", HomeDir: "/Volumes/Data/Home Directories/bob"},
		{UID: "3000000000", GID: "3000000000", Username: "carol", HomeDir: "/Users/carol"},
	}, users)

	_, err = parseDsclRecords([]byte("not a plist"))
	assert.Error(t, err)
}

func TestNormalizeID(t *testing.T) {
	for in, want := range map[string]string{
		"501": "501", "-2": "4294967294", "4294967295": "4294967295", "-2147483648": "2147483648",
	} {
		got, ok := normalizeID(in)
		assert.True(t, ok, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "x", "4294967296", "-2147483649"} {
		_, ok := normalizeID(in)
		assert.False(t, ok, in)
	}
}

// O_NOFOLLOW_ANY refuses a symlink anywhere in the path, which is what an
// attacker would swap in for a directory after resolveInHome checked it.
func TestCopyOwnedFileRejectsLinkedDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dir")
	require.NoError(t, os.Mkdir(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "db"), []byte("x"), 0o600))
	require.NoError(t, os.Symlink(dir, filepath.Join(root, "link")))

	_, err := copyOwnedFile(filepath.Join(root, "link", "db"), filepath.Join(root, "copy"), uint32(os.Getuid())) //nolint:gosec // uid fits
	require.ErrorIs(t, err, syscall.ELOOP)
}

// localUsers runs dscl, so this checks the parsing against the host's real
// directory output: the account running the tests must be listed once, with
// its own UID and home directory.
func TestLocalUsersIncludesCurrentUser(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root's home is not a login account's")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	homeInfo, err := os.Stat(home)
	if err != nil {
		t.Skip(err)
	}

	users, err := localUsers(t.Context())
	require.NoError(t, err)
	var matches []localUser
	for _, u := range users {
		if info, err := os.Stat(u.HomeDir); err == nil && os.SameFile(info, homeInfo) {
			matches = append(matches, u)
		}
	}
	require.Len(t, matches, 1, "users with home %s", home)
	assert.Equal(t, strconv.Itoa(os.Getuid()), matches[0].UID)
	assert.Equal(t, strconv.Itoa(os.Getgid()), matches[0].GID)
}

func TestIsLoginAccount(t *testing.T) {
	for name, want := range map[string]bool{
		"alice": true, "root": false, "daemon": false, "nobody": false, "_mbsetupuser": false,
	} {
		assert.Equal(t, want, isLoginAccount(name), name)
	}
}
