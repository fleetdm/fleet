//go:build darwin

package homebrew_outdated

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
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

// brewTimeout is the budget shared by a scan's brew calls, which may still
// download the package API data or, after a Homebrew upgrade, portable Ruby.
const brewTimeout = 60 * time.Second

// apiRefreshInterval is how old the package API data may get before a query
// re-downloads it.
const apiRefreshInterval = 24 * time.Hour

// brewQueryEnv returns env for the query calls. Homebrew's auto-update (self
// update plus a git fetch of every tap) can outlast the query budget, so it is
// skipped; only the package API data, where core formulae and casks get
// current_version, is refreshed.
func brewQueryEnv(env []string) []string {
	return append(slices.Clone(env),
		"HOMEBREW_NO_AUTO_UPDATE=1",
		"HOMEBREW_FORCE_API_AUTO_UPDATE=1",
		"HOMEBREW_API_AUTO_UPDATE_SECS="+strconv.Itoa(int(apiRefreshInterval.Seconds())),
		// Keeps configuration hints out of surfaced errors.
		"HOMEBREW_NO_ENV_HINTS=1",
	)
}

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

	// Build brew's environment: HOME points at the console user's home so brew
	// reads/writes caches as that user rather than root, and the prefix's bin is on
	// PATH so brew finds its own tooling (including for a non-standard per-user
	// prefix).
	env := []string{"PATH=" + prefix + "/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"}
	if homeDir != "" {
		env = append(env, "HOME="+homeDir)
	}

	rows, err := cache.get(ctx, brewPath+":"+strconv.FormatUint(uint64(uid), 10), time.Now(), func(scanCtx context.Context) ([]map[string]string, error) {
		return queryRows(scanCtx, brewPath, prefix, uid, gid, brewQueryEnv(env))
	})
	if err != nil {
		return nil, err
	}
	return filterRows(rows, nameConstraints(queryContext)), nil
}

// cache holds the last full scan; see scanCache.
var cache scanCache

// queryRows runs the brew calls for a full scan and builds the rows.
func queryRows(ctx context.Context, brewPath, prefix string, uid, gid uint32, env []string) ([]map[string]string, error) {
	// Both brew calls share one deadline.
	ctx, cancel := context.WithTimeout(ctx, brewTimeout)
	defer cancel()

	run := func(args ...string) ([]byte, error) {
		return runBrew(ctx, brewPath, uid, gid, env, args...)
	}

	pkgs, err := outdatedPackages(run)
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
// stdout, honoring the scan's shared deadline on ctx.
func runBrew(ctx context.Context, brewPath string, uid, gid uint32, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, brewPath, args...)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{Uid: uid, Gid: gid},
	}
	out, err := cmd.Output()
	if err != nil {
		// A call killed at the deadline surfaces as "signal: killed"; name the
		// deadline too.
		if ctxErr := ctx.Err(); ctxErr != nil && !errors.Is(err, ctxErr) {
			err = fmt.Errorf("%w: %w", ctxErr, err)
		}
		return out, describeBrewError(err)
	}
	return out, nil
}
