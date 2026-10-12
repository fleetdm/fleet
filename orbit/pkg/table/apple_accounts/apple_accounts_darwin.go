//go:build darwin

package apple_accounts

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/micromdm/plist"
	"golang.org/x/sys/unix"
)

// noFollowFlags makes opening a file fail if any component of its path is a
// symlink. The path is already resolved, so this only matters if a directory
// in it is swapped for a symlink after resolveInHome checked it.
const noFollowFlags = unix.O_NOFOLLOW_ANY

// localUsers lists the accounts whose home directories should be searched.
//
// fleetd runs as root, so every user's home directory has to be enumerated.
// The local directory node is authoritative for UIDs and for home directories
// outside /Users. Home directories in /Users that it does not know about, such
// as those of network accounts without a mobile account, are added with the
// directory owner's UID and GID and the directory name as username.
func localUsers(ctx context.Context) ([]localUser, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Without the directory, users whose homes are outside /Users would be
	// silently left out, so a failure fails the query.
	out, err := exec.CommandContext(ctx, "/usr/bin/dscl", "-plist", ".", "-readall", "/Users",
		"RecordName", "UniqueID", "PrimaryGroupID", "NFSHomeDirectory").Output()
	if err != nil {
		return nil, fmt.Errorf("dscl: %w", err)
	}
	candidates, err := parseDsclRecords(out)
	if err != nil {
		return nil, err
	}
	// Directory records go first, by name, so they win over the /Users scan
	// and the choice between two records sharing a home is stable.
	slices.SortFunc(candidates, func(a, b localUser) int { return strings.Compare(a.Username, b.Username) })

	entries, _ := os.ReadDir("/Users")
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() && !strings.HasPrefix(name, ".") && name != "Shared" {
			candidates = append(candidates, localUser{Username: name, HomeDir: filepath.Join("/Users", name)})
		}
	}

	// Home directories are compared by device and inode (os.SameFile) rather
	// than by path: APFS is usually case-insensitive, and a home can be reached
	// through a symlink, so two different paths may be the same directory.
	var seen []os.FileInfo
	var users []localUser
	for _, u := range candidates {
		info, err := os.Stat(u.HomeDir)
		if err != nil || !info.IsDir() {
			continue
		}
		if slices.ContainsFunc(seen, func(s os.FileInfo) bool { return os.SameFile(s, info) }) {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			continue
		}
		seen = append(seen, info)
		if u.UID == "" {
			u.UID = strconv.FormatUint(uint64(stat.Uid), 10)
		}
		if u.GID == "" {
			u.GID = strconv.FormatUint(uint64(stat.Gid), 10)
		}
		users = append(users, u)
	}
	// Stable ordering keeps query results comparable between runs.
	slices.SortFunc(users, func(a, b localUser) int { return strings.Compare(a.HomeDir, b.HomeDir) })
	return users, nil
}

// parseDsclRecords parses `dscl -plist . -readall /Users` output into the
// login accounts it lists. The first RecordName is the short name; any others
// are aliases. A record without a PrimaryGroupID gets its home directory's
// group in localUsers.
func parseDsclRecords(out []byte) ([]localUser, error) {
	var records []struct {
		Names []string `plist:"dsAttrTypeStandard:RecordName"`
		UIDs  []string `plist:"dsAttrTypeStandard:UniqueID"`
		GIDs  []string `plist:"dsAttrTypeStandard:PrimaryGroupID"`
		Homes []string `plist:"dsAttrTypeStandard:NFSHomeDirectory"`
	}
	if err := plist.Unmarshal(out, &records); err != nil {
		return nil, fmt.Errorf("parsing dscl output: %w", err)
	}
	var users []localUser
	for _, r := range records {
		if len(r.Names) == 0 || len(r.UIDs) == 0 || len(r.Homes) == 0 {
			continue
		}
		uid, ok := normalizeID(r.UIDs[0])
		if !ok || !isLoginAccount(r.Names[0]) || !filepath.IsAbs(r.Homes[0]) {
			continue
		}
		var gid string
		if len(r.GIDs) > 0 {
			gid, _ = normalizeID(r.GIDs[0])
		}
		users = append(users, localUser{UID: uid, GID: gid, Username: r.Names[0], HomeDir: filepath.Clean(r.Homes[0])})
	}
	return users, nil
}

// normalizeID formats a directory UniqueID or PrimaryGroupID the way osquery's
// users table reports it, so the two join on uid: dscl prints IDs of 2^31 and
// above as negative 32-bit values, which osquery reports unsigned.
func normalizeID(s string) (string, bool) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < math.MinInt32 || n > math.MaxUint32 {
		return "", false
	}
	if n < 0 {
		n += 1 << 32
	}
	return strconv.FormatInt(n, 10), true
}

// isLoginAccount reports whether a local directory record can belong to a
// person. Service accounts are prefixed with an underscore, and root, daemon,
// and nobody share or lack real home directories.
func isLoginAccount(name string) bool {
	switch name {
	case "root", "daemon", "nobody":
		return false
	}
	return !strings.HasPrefix(name, "_")
}
