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

// DefaultOptions are the default options to use when creating an update
// client.
var DefaultOptions = defaultOptions

// Updater is responsible for managing update state.
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

type Targets map[string]TargetInfo

type TargetInfo struct {
	Platform          string
	Channel           string
	TargetFile        string
	ExtractedExecFile string
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

func NewDisabled(opt Options) *Updater {
	return &Updater{
		opt: opt,
	}
}

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

func (u *Updater) RepoPath(target string) (string, error) {
	t, ok := u.opt.Targets[target]
	if !ok {
		return "", fmt.Errorf("unknown target: %s", target)
	}
	return path.Join(target, t.Platform, t.Channel, t.TargetFile), nil
}

func (u *Updater) ExecutableLocalPath(target string) (string, error) {
	localPath, err := u.LocalPath(target)
	if err != nil {
		return "", err
	}

	if strings.HasSuffix(localPath, ".tar.gz") {
		dirPath := strings.TrimSuffix(localPath, ".tar.gz")
		s, err := os.Stat(dirPath)
		if err != nil {
			return "", fmt.Errorf("stat %q: %w", dirPath, err)
		}
		if !s.IsDir() {
			return "", fmt.Errorf("expected directory %q: %w", dirPath, err)
		}
		t := u.opt.Targets[target] // this was accessed before.
		switch t.Platform {
		case "macos-app":
			localPath = appMacOSPath(dirPath, t.ExtractedExecFile)
		default:
			return "", fmt.Errorf("unsupported platform: %s", t.Platform)
		}
	}

	s, err := os.Stat(localPath)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", localPath, err)
	}
	if s.IsDir() {
		return "", fmt.Errorf("expected file %q: %w", localPath, err)
	}
	return localPath, nil
}

func (u *Updater) LocalPath(target string) (string, error) {
	t, ok := u.opt.Targets[target]
	if !ok {
		return "", fmt.Errorf("unknown target: %s", target)
	}
	return filepath.Join(u.opt.RootDirectory, binDir, target, t.Platform, t.Channel, t.TargetFile), nil
}

// Lookup looks up the provided target in the local target metadata. This should
// be called after UpdateMetadata.
func (u *Updater) Lookup(target string) (*data.TargetFileMeta, error) {
	repoPath, err := u.RepoPath(target)
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

func appMacOSPath(appDirPath, execFile string) string {
	return filepath.Join(appDirPath, "Contents", "MacOS", execFile)
}

// Get returns the local path to the specified target. The target is downloaded
// if it does not yet exist locally or the hash does not match.
func (u *Updater) Get(target string) (string, error) {
	if target == "" {
		return "", errors.New("target is required")
	}

	localPath, err := u.LocalPath(target)
	if err != nil {
		return "", fmt.Errorf("failed to load local path for target %s: %w", target, err)
	}
	repoPath, err := u.RepoPath(target)
	if err != nil {
		return "", fmt.Errorf("failed to load repository path for target %s: %w", target, err)
	}

	switch stat, err := os.Stat(localPath); {
	case err == nil:
		if !stat.Mode().IsRegular() {
			return "", fmt.Errorf("expected %s to be regular file", localPath)
		}
		meta, err := u.Lookup(target)
		if err != nil {
			return "", err
		}
		if err := checkFileHash(meta, localPath); err != nil {
			log.Debug().Str("info", err.Error()).Msg("change detected")
			if err := u.download(target, repoPath, localPath); err != nil {
				return "", fmt.Errorf("download %q: %w", repoPath, err)
			}
		} else {
			log.Debug().Str("path", localPath).Str("target", target).Msg("found expected target locally")
		}
	case errors.Is(err, os.ErrNotExist):
		log.Debug().Err(err).Msg("stat file")
		if err := u.download(target, repoPath, localPath); err != nil {
			return "", fmt.Errorf("download %q: %w", repoPath, err)
		}
	default:
		return "", fmt.Errorf("stat %q: %w", localPath, err)
	}

	if strings.HasSuffix(localPath, ".tar.gz") {
		dirPath := strings.TrimSuffix(localPath, ".tar.gz")
		switch s, err := os.Stat(dirPath); {
		case err == nil:
			if !s.IsDir() {
				return "", fmt.Errorf("expected directory %q: %w", dirPath, err)
			}
		case errors.Is(err, os.ErrNotExist):
			if err := extractTarGz(localPath); err != nil {
				return "", fmt.Errorf("extract %q: %w", localPath, err)
			}
		default:
			return "", fmt.Errorf("stat %q: %w", dirPath, err)
		}
		t := u.opt.Targets[target] // this was accessed before.
		switch t.Platform {
		case "macos-app":
			return appMacOSPath(dirPath, t.ExtractedExecFile), nil
		default:
			return "", fmt.Errorf("unsupported platform: %s", t.Platform)
		}
	}

	return localPath, nil
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

	if constant.PlatformName == "windows" {
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

func (u *Updater) checkExec(target, path string) error {
	if strings.HasSuffix(path, ".tar.gz") {
		if err := extractTarGz(path); err != nil {
			return fmt.Errorf("extract %q: %w", path, err)
		}
		t, ok := u.opt.Targets[target]
		if !ok {
			return fmt.Errorf("unknown target: %s", target)
		}
		extractedDir := strings.TrimPrefix(path, ".tar.gz")
		switch t.Platform {
		case "macos-app":
			path = appMacOSPath(extractedDir, t.ExtractedExecFile)
		default:
			return fmt.Errorf("unsupported platform: %s", t.Platform)
		}
	}

	// Note that this would fail for any binary that returns nonzero for --help.
	out, err := exec.Command(path, "--help").CombinedOutput()
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
