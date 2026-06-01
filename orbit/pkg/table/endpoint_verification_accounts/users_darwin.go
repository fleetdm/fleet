//go:build darwin

package endpoint_verification_accounts

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// listLocalUsersDarwin enumerates `/Users/*` directories owned by an
// interactive uid (>= 500) and returns each as a userHome. Using the
// filesystem directly avoids parsing `dscl` output or reading
// `/etc/passwd` (which is incomplete on macOS).
func listLocalUsersDarwin() ([]userHome, error) {
	entries, err := os.ReadDir("/Users")
	if err != nil {
		return nil, err
	}
	users := make([]userHome, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip the placeholder Shared directory and any dot-files.
		if name == "Shared" || len(name) == 0 || name[0] == '.' {
			continue
		}
		home := filepath.Join("/Users", name)
		fi, err := os.Stat(home)
		if err != nil {
			continue
		}
		stat, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			continue
		}
		if stat.Uid < 500 {
			// macOS system / service accounts.
			continue
		}
		users = append(users, userHome{
			uid:      strconv.FormatUint(uint64(stat.Uid), 10),
			username: name,
			homeDir:  home,
		})
	}
	return users, nil
}
