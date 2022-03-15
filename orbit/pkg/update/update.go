// Package update contains the types and functions used by the update system.
package update

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/fatih/color"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/rs/zerolog/log"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
)

const (
	binDir     = "bin"
	stagingDir = "staging"

	defaultURL      = "https://tuf.fleetctl.com"
	defaultRootKeys = `[{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"6d71d3beac3b830be929f2b10d513448d49ec6bb62a680176b89ffdfca180eb4"}}]`
)

// Updater is responsible for managing update state.
//
// Updater supports updating plain executables and
// .tar.gz compressed executables.
type Updater struct {
	opt    Options
	client *client.Client
}

// Options are the options that can be provided when creating an Updater.
type Options struct {
	// RootDirectory is the root directory from which other directories should be referenced.
	RootDirectory string
	// ServerURL is the URL of the update server.
	ServerURL string
	// InsecureTransport skips TLS certificate verification in the transport if
	// set to true. Best to leave this on, but due to the file signing any
	// tampering by a MitM should be detectable.
	InsecureTransport bool
	// RootKeys is the JSON encoded root keys to use to bootstrap trust.
	RootKeys string
	// LocalStore is the local metadata store.
	LocalStore client.LocalStore
	// Targets holds the targets the Updater keeps track of.
	Targets Targets
}

// Targets is a map of target name and its tracking information.
type Targets map[string]TargetInfo

// TargetInfo holds all the information to track target updates.
type TargetInfo struct {
	// Platform is the target's platform string.
	Platform string
	// Channel is the target's update channel.
	Channel string
	// TargetFile is the name of the target file in the repository.
	TargetFile string
	// ExtractedExecSubPath is the path to the executable in case the
	// target is a compressed file.
	ExtractedExecSubPath []string
}

// New creates a new updater given the provided options. All the necessary
// directories are initialized.
func New(opt Options) (*Updater, error) {
	if opt.LocalStore == nil {
		return nil, errors.New("opt.LocalStore must be non-nil")
	}

	httpClient := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: opt.InsecureTransport,
	}))

	remoteStore, err := client.HTTPRemoteStore(opt.ServerURL, nil, httpClient)
	if err != nil {
		return nil, fmt.Errorf("init remote store: %w", err)
	}

	tufClient := client.NewClient(opt.LocalStore, remoteStore)
	var rootKeys []*data.PublicKey
	if err := json.Unmarshal([]byte(opt.RootKeys), &rootKeys); err != nil {
		return nil, fmt.Errorf("unmarshal root keys: %w", err)
	}

	meta, err := opt.LocalStore.GetMeta()
	if err != nil || meta["root.json"] == nil {
		var rootKeys []*data.PublicKey
		if err := json.Unmarshal([]byte(opt.RootKeys), &rootKeys); err != nil {
			return nil, fmt.Errorf("unmarshal root keys: %w", err)
		}
		if err := tufClient.Init(rootKeys, 1); err != nil {
			return nil, fmt.Errorf("init tuf client: %w", err)
		}
	}

	updater := &Updater{
		opt:    opt,
		client: tufClient,
	}

	if err := updater.initializeDirectories(); err != nil {
		return nil, err
	}

	return updater, nil
}

// NewDisabled creates a new disabled Updater. A disabled updater
// won't reach out for a remote repository.
//
// A disabled updater is useful to use local paths the way an
// enabled Updater would (to locate executables on environments
// where updates and/or network access are disabled).
func NewDisabled(opt Options) *Updater {
	return &Updater{
		opt: opt,
	}
}

// UpdateMetadata downloads and verifies remote repository metadata.
func (u *Updater) UpdateMetadata() error {
	if _, err := u.client.Update(); err != nil {
		// An error is returned if we are already up-to-date. We can ignore that
		// error.
		if !client.IsLatestSnapshot(ctxerr.Cause(err)) {
			return fmt.Errorf("update metadata: %w", err)
		}
	}
	return nil
}

// repoPath returns the path of the target in the remote repository.
func (u *Updater) repoPath(target string) (string, error) {
	t, ok := u.opt.Targets[target]
	if !ok {
		return "", fmt.Errorf("unknown target: %s", target)
	}
	return path.Join(target, t.Platform, t.Channel, t.TargetFile), nil
}

// ExecutableLocalPath returns the configured executable local path of a target.
func (u *Updater) ExecutableLocalPath(target string) (string, error) {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return "", err
	}
	return localTarget.execPath, nil
}

// localTarget holds local paths of a target.
//
// E.g., for a osqueryd target:
//
//	localTarget{
//		info: TargetInfo{
//			Platform:             "macos-app",
//			Channel:              "stable",
//			TargetFile:           "osqueryd.app.tar.gz",
//			ExtractedExecSubPath: []string{"osquery.app", "Contents", "MacOS", "osqueryd"},
//		},
//		path: "/local/path/to/osqueryd.app.tar.gz",
//		dirPath: "/local/path/to/osqueryd.app",
//		execPath: "/local/path/to/osqueryd.app/Contents/MacOS/osqueryd",
//	}
type localTarget struct {
	info     TargetInfo
	path     string
	dirPath  string // empty for non-tar.gz targets.
	execPath string
}

// localPath returns the info and local path of a target.
func (u *Updater) localTarget(target string) (*localTarget, error) {
	t, ok := u.opt.Targets[target]
	if !ok {
		return nil, fmt.Errorf("unknown target: %s", target)
	}
	lt := &localTarget{
		info: t,
		path: filepath.Join(
			u.opt.RootDirectory, binDir, target, t.Platform, t.Channel, t.TargetFile,
		),
	}
	lt.execPath = lt.path
	if strings.HasSuffix(lt.path, ".tar.gz") {
		lt.execPath = filepath.Join(append([]string{filepath.Dir(lt.path)}, t.ExtractedExecSubPath...)...)
		lt.dirPath = filepath.Join(filepath.Dir(lt.path), lt.info.ExtractedExecSubPath[0])
	}
	return lt, nil
}

// Lookup looks up the provided target in the local target metadata. This should
// be called after UpdateMetadata.
func (u *Updater) Lookup(target string) (*data.TargetFileMeta, error) {
	repoPath, err := u.repoPath(target)
	if err != nil {
		return nil, err
	}
	t, err := u.client.Target(repoPath)
	if err != nil {
		return nil, fmt.Errorf("lookup %s: %w", target, err)
	}
	return &t, nil
}

// Targets gets all of the known targets
func (u *Updater) Targets() (data.TargetFiles, error) {
	targets, err := u.client.Targets()
	if err != nil {
		return nil, fmt.Errorf("get targets: %w", err)
	}

	return targets, nil
}

// Get returns the local path to the specified target. The target is downloaded
// if it does not yet exist locally or the hash does not match.
func (u *Updater) Get(target string) (string, error) {
	if target == "" {
		return "", errors.New("target is required")
	}

	localTarget, err := u.localTarget(target)
	if err != nil {
		return "", fmt.Errorf("failed to load local path for target %s: %w", target, err)
	}
	repoPath, err := u.repoPath(target)
	if err != nil {
		return "", fmt.Errorf("failed to load repository path for target %s: %w", target, err)
	}

	switch stat, err := os.Stat(localTarget.path); {
	case err == nil:
		if !stat.Mode().IsRegular() {
			return "", fmt.Errorf("expected %s to be regular file", localTarget.path)
		}
		meta, err := u.Lookup(target)
		if err != nil {
			return "", err
		}
		if err := checkFileHash(meta, localTarget.path); err != nil {
			log.Debug().Str("info", err.Error()).Msg("change detected")
			if err := u.download(target, repoPath, localTarget.path); err != nil {
				return "", fmt.Errorf("download %q: %w", repoPath, err)
			}
			if strings.HasSuffix(localTarget.path, ".tar.gz") {
				if err := os.RemoveAll(localTarget.dirPath); err != nil {
					return "", fmt.Errorf("failed to remove old extracted dir: %q: %w", localTarget.dirPath, err)
				}
			}
		} else {
			log.Debug().Str("path", localTarget.path).Str("target", target).Msg("found expected target locally")
		}
	case errors.Is(err, os.ErrNotExist):
		log.Debug().Err(err).Msg("stat file")
		if err := u.download(target, repoPath, localTarget.path); err != nil {
			return "", fmt.Errorf("download %q: %w", repoPath, err)
		}
	default:
		return "", fmt.Errorf("stat %q: %w", localTarget.path, err)
	}

	if strings.HasSuffix(localTarget.path, ".tar.gz") {
		switch s, err := os.Stat(localTarget.execPath); {
		case err == nil:
			if s.IsDir() {
				return "", fmt.Errorf("expected executable %q: %w", localTarget.execPath, err)
			}
		case errors.Is(err, os.ErrNotExist):
			if err := extractTarGz(localTarget.path); err != nil {
				return "", fmt.Errorf("extract %q: %w", localTarget.path, err)
			}
			s, err := os.Stat(localTarget.execPath)
			if err != nil {
				return "", fmt.Errorf("stat %q: %w", localTarget.execPath, err)
			}
			if s.IsDir() {
				return "", fmt.Errorf("expected executable %q: %w", localTarget.execPath, err)
			}
		default:
			return "", fmt.Errorf("stat %q: %w", localTarget.execPath, err)
		}
	}

	return localTarget.execPath, nil
}

func writeDevWarningBanner(w io.Writer) {
	warningColor := color.New(color.FgWhite, color.Bold, color.BgRed)
	warningColor.Fprintf(w, "WARNING: You are attempting to override orbit with a dev build.\nPress Enter to continue, or Control-c to exit.")
	// We need to disable color and print a new line to make it look somewhat neat, otherwise colors continue to the
	// next line
	warningColor.DisableColor()
	warningColor.Fprintln(w)
	bufio.NewScanner(os.Stdin).Scan()
}

// CopyDevBuilds uses a development build for the given target+channel.
//
// This is just for development, must not be used in production.
func (u *Updater) CopyDevBuild(target, devBuildPath string) {
	writeDevWarningBanner(os.Stderr)

	localPath, err := u.ExecutableLocalPath(target)
	if err != nil {
		panic(err)
	}
	if err := secure.MkdirAll(filepath.Dir(localPath), constant.DefaultDirMode); err != nil {
		panic(err)
	}
	dst, err := secure.OpenFile(localPath, os.O_CREATE|os.O_WRONLY, constant.DefaultExecutableMode)
	if err != nil {
		panic(err)
	}
	defer dst.Close()

	src, err := secure.OpenFile(devBuildPath, os.O_RDONLY, constant.DefaultExecutableMode)
	if err != nil {
		panic(err)
	}
	defer src.Close()

	if _, err := src.Stat(); err != nil {
		panic(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		panic(err)
	}
}

// download downloads the target to the provided path. The file is deleted and
// an error is returned if the hash does not match.
func (u *Updater) download(target, repoPath, localPath string) error {
	staging := filepath.Join(u.opt.RootDirectory, stagingDir)

	if err := secure.MkdirAll(staging, constant.DefaultDirMode); err != nil {
		return fmt.Errorf("initialize download dir: %w", err)
	}

	// Additional chmod only necessary on Windows, effectively a no-op on other
	// platforms.
	if err := platform.ChmodExecutableDirectory(staging); err != nil {
		return err
	}

	tmp, err := secure.OpenFile(
		filepath.Join(staging, filepath.Base(localPath)),
		os.O_CREATE|os.O_WRONLY,
		constant.DefaultExecutableMode,
	)
	if err != nil {
		return fmt.Errorf("open temp file for download: %w", err)
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()
	if err := platform.ChmodExecutable(tmp.Name()); err != nil {
		return fmt.Errorf("chmod download: %w", err)
	}

	if err := secure.MkdirAll(filepath.Dir(localPath), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("initialize download dir: %w", err)
	}

	// Additional chmod only necessary on Windows, effectively a no-op on other
	// platforms.
	if err := platform.ChmodExecutableDirectory(filepath.Dir(localPath)); err != nil {
		return err
	}

	// The go-tuf client handles checking of max size and hash.
	if err := u.client.Download(repoPath, &fileDestination{tmp}); err != nil {
		return fmt.Errorf("download target %s: %w", repoPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close tmp file: %w", err)
	}

	if err := u.checkExec(target, tmp.Name()); err != nil {
		return fmt.Errorf("exec check failed %q: %w", tmp.Name(), err)
	}

	if runtime.GOOS == "windows" {
		// Remove old file first
		if err := os.Rename(localPath, localPath+".old"); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("rename old: %w", err)
		}
	}

	if err := os.Rename(tmp.Name(), localPath); err != nil {
		return fmt.Errorf("move download: %w", err)
	}

	return nil
}

func goosFromPlatform(platform string) (string, error) {
	switch platform {
	case "macos", "macos-app":
		return "darwin", nil
	case "windows", "linux":
		return platform, nil
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}
}

// checkExec checks/verifies a downloaded executable target by executing it.
func (u *Updater) checkExec(target, tmpPath string) error {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return err
	}
	platformGOOS, err := goosFromPlatform(localTarget.info.Platform)
	if err != nil {
		return err
	}
	if platformGOOS != runtime.GOOS {
		// Nothing to do, we can't check the executable if running cross-platform.
		// This generally happens when generating a package from a different platform
		// than the target package (e.g. generating an MSI package from macOS).
		return nil
	}

	if strings.HasSuffix(tmpPath, ".tar.gz") {
		if err := extractTarGz(tmpPath); err != nil {
			return fmt.Errorf("extract %q: %w", tmpPath, err)
		}
		tmpDirPath := filepath.Join(filepath.Dir(tmpPath), localTarget.info.ExtractedExecSubPath[0])
		defer os.RemoveAll(tmpDirPath)
		tmpPath = filepath.Join(append([]string{filepath.Dir(tmpPath)}, localTarget.info.ExtractedExecSubPath...)...)
	}

	// Note that this would fail for any binary that returns nonzero for --help.
	out, err := exec.Command(tmpPath, "--help").CombinedOutput()
	if err != nil {
		return fmt.Errorf("exec new version: %s: %w", string(out), err)
	}
	return nil
}

// extractTagGz extracts the contents of the provided tar.gz file.
func extractTarGz(path string) error {
	tarGzFile, err := secure.OpenFile(path, os.O_RDONLY, 0o755)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer tarGzFile.Close()

	gzipReader, err := gzip.NewReader(tarGzFile)
	if err != nil {
		return fmt.Errorf("gzip reader %q: %w", path, err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		switch {
		case err == nil:
			// OK
		case errors.Is(err, io.EOF):
			return nil
		default:
			return fmt.Errorf("tar reader %q: %w", path, err)
		}

		// Prevent zip-slip attack.
		if strings.Contains(header.Name, "..") {
			return fmt.Errorf("invalid path in tar.gz: %q", header.Name)
		}

		targetPath := filepath.Join(filepath.Dir(path), header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := secure.MkdirAll(targetPath, constant.DefaultDirMode); err != nil {
				return fmt.Errorf("mkdir %q: %w", header.Name, err)
			}
		case tar.TypeReg:
			err := func() error {
				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY, header.FileInfo().Mode())
				if err != nil {
					return fmt.Errorf("failed to create %q: %w", header.Name, err)
				}
				defer outFile.Close()

				if _, err := io.Copy(outFile, tarReader); err != nil {
					return fmt.Errorf("failed to copy %q: %w", header.Name, err)
				}
				return nil
			}()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown flag type %q: %d", header.Name, header.Typeflag)
		}
	}
}

func (u *Updater) initializeDirectories() error {
	for _, dir := range []string{
		filepath.Join(u.opt.RootDirectory, binDir),
	} {
		err := secure.MkdirAll(dir, constant.DefaultDirMode)
		if err != nil {
			return fmt.Errorf("initialize directories: %w", err)
		}
	}

	return nil
}
