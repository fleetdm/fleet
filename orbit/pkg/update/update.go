// package update contains the types and functions used by the update system.
package update

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"

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
	// Platform is the target of the platform to update for. In the default
	// options this is the current platform.
	Platform string
	// OrbitChannel is the update channel to use for Orbit.
	OrbitChannel string
	// OsquerydChannel is the update channel to use for osquery (osqueryd).
	OsquerydChannel string
}

// New creates a new updater given the provided options. All the necessary
// directories are initialized.
func New(opt Options) (*Updater, error) {
	if opt.LocalStore == nil {
		return nil, errors.New("opt.LocalStore must be non-nil")
	}

	if opt.Platform == "" {
		opt.Platform = constant.PlatformName
	}

	httpClient := fleethttp.NewClient(fleethttp.WithTLSClientConfig(&tls.Config{
		InsecureSkipVerify: opt.InsecureTransport,
	}))

	remoteStore, err := client.HTTPRemoteStore(opt.ServerURL, nil, httpClient)
	if err != nil {
		return nil, fmt.Errorf("init remote store: %w", err)
	}

	tufClient := client.NewClient(opt.LocalStore, remoteStore)
	var rootKeys []*data.Key
	if err := json.Unmarshal([]byte(opt.RootKeys), &rootKeys); err != nil {
		return nil, fmt.Errorf("unmarshal root keys: %w", err)
	}

	meta, err := opt.LocalStore.GetMeta()
	if err != nil || meta["root.json"] == nil {
		var rootKeys []*data.Key
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

func (u *Updater) RepoPath(target, channel string) string {
	return path.Join(target, u.opt.Platform, channel, target+constant.ExecutableExtension(u.opt.Platform))
}

func (u *Updater) LocalPath(target, channel string) string {
	return u.pathFromRoot(filepath.Join(binDir, target, u.opt.Platform, channel, target+constant.ExecutableExtension(u.opt.Platform)))
}

// Lookup looks up the provided target in the local target metadata. This should
// be called after UpdateMetadata.
func (u *Updater) Lookup(target, channel string) (*data.TargetFileMeta, error) {
	t, err := u.client.Target(u.RepoPath(target, channel))
	if err != nil {
		return nil, fmt.Errorf("lookup %s@%s: %w", target, channel, err)
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
func (u *Updater) Get(target, channel string) (string, error) {
	if target == "" {
		return "", errors.New("target is required")
	}
	if channel == "" {
		return "", errors.New("channel is required")
	}

	localPath := u.LocalPath(target, channel)
	repoPath := u.RepoPath(target, channel)
	stat, err := os.Stat(localPath)
	if err != nil {
		log.Debug().Err(err).Msg("stat file")
		return localPath, u.Download(repoPath, localPath)
	}
	if !stat.Mode().IsRegular() {
		return "", fmt.Errorf("expected %s to be regular file", localPath)
	}

	meta, err := u.Lookup(target, channel)
	if err != nil {
		return "", err
	}

	if err := CheckFileHash(meta, localPath); err != nil {
		log.Debug().Str("info", err.Error()).Msg("change detected")
		return localPath, u.Download(repoPath, localPath)
	}

	log.Debug().Str("path", localPath).Str("target", target).Str("channel", channel).Msg("found expected target locally")

	return localPath, nil
}

// Download downloads the target to the provided path. The file is deleted and
// an error is returned if the hash does not match.
func (u *Updater) Download(repoPath, localPath string) error {
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

	// Attempt to exec the new binary only if the platform matches. This will
	// always fail if the binary doesn't match the platform, so there's not
	// really anything we can check.
	if u.opt.Platform == constant.PlatformName {
		// Note that this would fail for any binary that returns nonzero for --help.
		out, err := exec.Command(tmp.Name(), "--help").CombinedOutput()
		if err != nil {
			return fmt.Errorf("exec new version: %s: %w", string(out), err)
		}
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

func (u *Updater) pathFromRoot(parts ...string) string {
	return filepath.Join(append([]string{u.opt.RootDirectory}, parts...)...)
}

func (u *Updater) initializeDirectories() error {
	for _, dir := range []string{
		u.pathFromRoot(binDir),
	} {
		err := secure.MkdirAll(dir, constant.DefaultDirMode)
		if err != nil {
			return fmt.Errorf("initialize directories: %w", err)
		}
	}

	return nil
}
