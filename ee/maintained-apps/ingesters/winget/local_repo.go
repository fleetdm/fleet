package winget

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// wingetPkgsRepoEnv names the environment variable that points the ingester at a local
// clone of microsoft/winget-pkgs. When it is set, manifests are read with git instead of
// the GitHub API, for environments where api.github.com is rate-limited or blocked for
// that repo. A partial clone is enough and much smaller than a full one:
//
//	git clone --depth 1 --filter=tree:0 --no-checkout https://github.com/microsoft/winget-pkgs.git
//
// Manifests are read from the clone's HEAD, so fetch before ingesting to pick up new
// versions.
const wingetPkgsRepoEnv = "WINGET_PKGS_REPO"

// errManifestNotFound is returned when a manifest file doesn't exist in winget-pkgs.
var errManifestNotFound = errors.New("winget manifest not found")

// localRepo reads winget-pkgs manifests from a local git clone.
type localRepo struct {
	dir string
}

// gitCommandTimeout bounds a single git call. Reads from a partial clone fetch the
// missing objects on demand, so the first read of a path can take a while.
const gitCommandTimeout = 5 * time.Minute

func newLocalRepo(ctx context.Context, dir string) (*localRepo, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "%s must point at a local clone of microsoft/winget-pkgs", wingetPkgsRepoEnv)
	}
	r := &localRepo{dir: dir}
	if _, _, err := r.git(ctx, "rev-parse", "--git-dir"); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "%s=%q is not a git repository", wingetPkgsRepoEnv, dir)
	}
	return r, nil
}

// listDirs returns the names of the subdirectories of dirPath, which is relative to the
// repository root. A missing dirPath yields an empty list.
func (r *localRepo) listDirs(ctx context.Context, dirPath string) ([]string, error) {
	stdout, _, err := r.git(ctx, "ls-tree", "-d", "--name-only", "HEAD", dirPath+"/")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing directories in local winget-pkgs clone")
	}

	var names []string
	for line := range strings.SplitSeq(stdout, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, path.Base(line))
		}
	}
	return names, nil
}

// readFile returns the contents of filePath, which is relative to the repository root.
// errManifestNotFound is returned if the file doesn't exist.
func (r *localRepo) readFile(ctx context.Context, filePath string) (string, error) {
	stdout, stderr, err := r.git(ctx, "show", "HEAD:"+filePath)
	if err != nil {
		if strings.Contains(stderr, "does not exist") || strings.Contains(stderr, "exists on disk, but not in") {
			return "", ctxerr.Wrapf(ctx, errManifestNotFound, "reading %s from local winget-pkgs clone", filePath)
		}
		return "", ctxerr.Wrapf(ctx, err, "reading %s from local winget-pkgs clone", filePath)
	}
	return stdout, nil
}

func (r *localRepo) git(ctx context.Context, args ...string) (stdout, stderr string, err error) {
	ctx, cancel := context.WithTimeout(ctx, gitCommandTimeout)
	defer cancel()

	var outBuf, errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.dir}, args...)...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if err := cmd.Run(); err != nil {
		return "", errBuf.String(), ctxerr.Wrapf(ctx, err, "git %s: %s", strings.Join(args, " "), strings.TrimSpace(errBuf.String()))
	}
	return outBuf.String(), errBuf.String(), nil
}
