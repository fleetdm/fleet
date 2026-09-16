//go:build darwin

package homebrew_outdated

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	tbl_common "github.com/fleetdm/fleet/v4/orbit/pkg/table/common"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// brewPaths are the well-known Homebrew binary locations: Apple Silicon first,
// then Intel.
var brewPaths = []string{
	"/opt/homebrew/bin/brew",
	"/usr/local/bin/brew",
}

// brewTimeout is the total budget shared across all brew invocations in a single
// Generate (outdated + optional fallback + cask enrichment). `brew outdated` may
// perform a `git fetch` of formula/cask metadata, so it is generous.
const brewTimeout = 60 * time.Second

// Generate is called to return the results for the table at query time.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	// osquery runs as root, but brew refuses to run as root; resolve the console
	// user up front so we can both run brew as them and look for a Homebrew install
	// under their home directory.
	uid, gid, err := tbl_common.GetConsoleUidGid()
	if err != nil {
		return nil, fmt.Errorf("failed to get console user: %w", err)
	}
	var homeDir string
	if uid != 0 {
		homeDir = consoleHome(uid)
	}

	brewPath := findBrew(homeDir)
	if brewPath == "" {
		// Homebrew is not installed anywhere, return no rows rather than an
		// error so the query simply yields nothing on hosts without Homebrew.
		log.Debug().Msg("homebrew_outdated: no Homebrew installation found; returning no rows")
		return nil, nil
	}
	prefix := filepath.Dir(filepath.Dir(brewPath))

	if uid == 0 {
		// Homebrew is installed system-wide, but there is no non-root console user
		// to run it as (host at the login window or headless). brew won't run as
		// root, so return no rows.
		log.Debug().
			Str("prefix", prefix).
			Msg("homebrew_outdated: no console user available (login window or headless host); returning no rows")
		return nil, nil
	}

	// Warn if the console user doesn't own the Homebrew install: brew can still
	// read as a non-owner, but its auto-update git fetch may fail (the tap repos
	// are owned by the install user), which can leave current_version stale. Stat
	// the brew binary, not the prefix root: on Intel the prefix (/usr/local) root
	// is root-owned even for a normal install, while the binary is owned by the
	// installer on both Intel and Apple Silicon. Best-effort diagnostics only.
	if fi, statErr := os.Stat(brewPath); statErr == nil {
		if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid != uid {
			log.Warn().
				Str("brew", brewPath).
				Uint32("owner_uid", st.Uid).
				Uint32("console_uid", uid).
				Msg("homebrew_outdated: console user does not own the Homebrew installation; brew auto-update may fail, so current_version could be stale")
		}
	}

	// Build brew's environment: HOME points at the console user's home so brew
	// reads/writes caches as that user rather than root, and the prefix's bin is on
	// PATH so brew finds its own tooling (including for a non-standard per-user
	// prefix).
	env := []string{"PATH=" + prefix + "/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"}
	if homeDir != "" {
		env = append(env, "HOME="+homeDir)
	}

	// Bound the whole sequence of brew calls (pushdown outdated + optional fallback
	// + cask enrichment) with a single shared deadline so cumulative latency stays
	// capped rather than each call getting its own full timeout.
	ctx, cancel := context.WithTimeout(ctx, brewTimeout)
	defer cancel()

	run := func(args ...string) ([]byte, error) {
		return runBrew(ctx, brewPath, uid, gid, env, args...)
	}

	// Push `name = <x>` constraints down to brew so a query for specific packages
	// (e.g. a policy) doesn't trigger a full `brew outdated` scan. outdatedPackages
	// falls back to a full scan if a pushed-down name is unknown.
	pkgs, err := outdatedPackages(run, nameConstraints(queryContext))
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return []map[string]string{}, nil
	}

	// Enrich casks with app_name and auto_updates via a single `brew info` call.
	// Only cask names are passed (those are the only fields brew info supplies), so
	// when nothing outdated is a cask we skip the call entirely.
	casks := map[string]caskDetail{}
	if caskNames := uniqueCaskNames(pkgs); len(caskNames) > 0 {
		infoArgs := append([]string{"info", "--json=v2"}, caskNames...)
		if infoOut, infoErr := runBrew(ctx, brewPath, uid, gid, env, infoArgs...); infoErr == nil {
			if parsed, perr := parseCaskInfo(infoOut); perr == nil {
				casks = parsed
			}
		}
		// If `brew info` fails, we still return the core columns from `brew outdated`;
		// only the cask-specific app_name/auto_updates columns will be empty.
	}

	return buildRows(pkgs, casks, prefix), nil
}

// findBrew returns the first existing Homebrew binary path, or "" if none exist.
// The standard system prefixes are checked first; when homeDir is set, Homebrew's
// documented per-user install location (<home>/homebrew) is checked as a fallback
// so a user who installed brew without admin rights is still detected.
func findBrew(homeDir string) string {
	candidates := append([]string{}, brewPaths...)
	if homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, "homebrew", "bin", "brew"))
	}
	return firstExistingFile(candidates)
}

// consoleHome returns the home directory of the console user, or "" if it can't
// be resolved.
func consoleHome(uid uint32) string {
	u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
	if err != nil {
		return ""
	}
	return u.HomeDir
}

// runBrew executes brew with the given args as the console user and returns
// stdout. It honors the deadline on ctx (set once by Generate) so all brew calls
// in a single Generate share one budget.
func runBrew(ctx context.Context, brewPath string, uid, gid uint32, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, brewPath, args...)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}
	return cmd.Output()
}
