// Package update contains the types and functions used by the update system.
package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
	"github.com/theupdateframework/go-tuf/verify"
)

const (
	binDir     = "bin"
	stagingDir = "staging"
)

// Defining these as variables so that we can overwrite them during development/testing of the migration.
var (
	//
	// For users using Fleet's TUF:
	// 	- orbit 1.38.0+ we migrate TUF from https://tuf.fleetctl.com to https://updates.fleetdm.com.
	//	- orbit 1.38.0+ will start using `updates-metadata.json` instead of `tuf-metadata.json`. If it is missing
	//	(which will be the case for the first run after the auto-update) then it will generate it from the new pinned roots.
	//
	// For users using a custom TUF:
	//	- orbit 1.38.0+ will start using `updates-metadata.json` instead of `tuf-metadata.json` (if it is missing then
	// 	  it will perform a copy).
	//
	// For both Fleet's TUF and custom TUF, fleetd packages built with fleetctl 4.63.0+ will contain both files
	// `updates-metadata.json` and `tuf-metadata.json` (same content) to support downgrades to orbit 1.37.0 or lower.

	//
	// The following variables are used by `fleetctl package` and by orbit in the migration to the new TUF repository.
	//

	// TODO(lucas): Change to https://updates.fleetdm.com when it's ready.
	DefaultURL          = `https://updates-staging.fleetdm.com`
	MetadataFileName    = "updates-metadata.json"
	defaultRootMetadata = `{"signed":{"_type":"root","spec_version":"1.0","version":6,"expires":"2026-01-08T22:23:47Z","keys":{"13f738181c9c50798d66316afaccf117658481b4db7b38715c358f522dc3bc40":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"34684874493cce2ac307c0dca88c241a287180c3eec9225c5f3e29bc4aeae080"}},"1ab0b9598e8b6ea653a24112f69b3eb3a84152c6a73d8dfdf43de4f63a93d3af":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"47a38623303bbe7b4ce47948011042b9387d7ec184642c156e06459d8fb6411b"}},"216e2dae905e73df943e003c428c7d340f21280750edb9d9ce2b15beeada667a":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"5c73ff497dc14380861892706c23bba0e3272c77c7f6f9524238b3a6b808b844"}},"2a83f45d24101ba91f669dca42981f97fc8bcde7cdf29c1887bc55c485103c49":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"e0bae1fae56d87e7f30f6359fd0a8cbfed231b19f58882e155c091c3bdb13c40"}},"3bc9408c1bcd999e69ba56b39e747656c6ebdafbd1e2c3e378c53e96e4477a64":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"8899adaa7ccd5bceb6a8c5f552fe4a9e59eb67e2a364db6638e06bbcf6f6eaeb"}},"4c0a5f49dc9665f13329d8157a2184985bd183503990469d06e32ad1bd6e70ee":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"eceeea79c6a353f5c7ed3be552a6144458ecf5fe78972eba88a77a45a230c58b"}},"57227c64d19605636d0afbab41d0887455de4287c6f328c5f69650005f793de0":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"d962cdf1d3e974f6c2b3d602260c87e0647fd54372afe7c31238f26a56a75443"}},"61e70c06858064c5e33e5009734222000610013e26fb6846ee17f55ddfb22da3":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"7e42b715cd9eedd8252a6b715fcfb8ef789531782ed19027a3c2ae11a2b0243b"}},"79a257e77793cb26d5d0cc0af6b2d2c94e7e8ca8b875dc504eb10fb343753f94":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"93409c7be4e3942ecff111d36cd097cda5778cb4f53305a07f20855b08f26071"}},"868858a9723ce357e8e251617ae11f7d3ae8a348588872cb2ce4149ee70ba155":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"f91c5b1fcb4ed3a1a65b956fa9025a89f458cea9036259b8cdfa276bc04faf45"}},"91629787db6e18b226027587733b2f667a7982eed9509c2e39dfeaf4cfb1a17a":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"3e2ac750e2e0eb22f87f35ad5309932b7b081c40891d249493fa9e2cef28066b"}},"97b3353aa23d09f88323e63cdc587a368df0a8818da67b91720b2cab00e68297":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"6b92f54f51eb617069a963a41aed75b4a23fca45e4c9ca8fc6d748d9b58b0451"}},"bc711f19576de2d71d1ca41eeccf7412f12c3ee4971185cd69066f8dc51d1ce6":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"7e39afe9a0310e7645ed389f243dc7156069d6972505cfbb460f8147949343cd"}},"c1ce9675f7302d2f09514f78ec7b3bdc00758d69b659e00c1c6731a4d0836bb9":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"a97c44dc10ee979ead46beb3be22f3407238e72b68240a41f92e65051eb27cb1"}}},"roles":{"root":{"keyids":["bc711f19576de2d71d1ca41eeccf7412f12c3ee4971185cd69066f8dc51d1ce6","4c0a5f49dc9665f13329d8157a2184985bd183503990469d06e32ad1bd6e70ee","57227c64d19605636d0afbab41d0887455de4287c6f328c5f69650005f793de0"],"threshold":1},"snapshot":{"keyids":["1ab0b9598e8b6ea653a24112f69b3eb3a84152c6a73d8dfdf43de4f63a93d3af","868858a9723ce357e8e251617ae11f7d3ae8a348588872cb2ce4149ee70ba155","97b3353aa23d09f88323e63cdc587a368df0a8818da67b91720b2cab00e68297"],"threshold":1},"targets":{"keyids":["3bc9408c1bcd999e69ba56b39e747656c6ebdafbd1e2c3e378c53e96e4477a64","c1ce9675f7302d2f09514f78ec7b3bdc00758d69b659e00c1c6731a4d0836bb9","2a83f45d24101ba91f669dca42981f97fc8bcde7cdf29c1887bc55c485103c49"],"threshold":1},"timestamp":{"keyids":["13f738181c9c50798d66316afaccf117658481b4db7b38715c358f522dc3bc40","79a257e77793cb26d5d0cc0af6b2d2c94e7e8ca8b875dc504eb10fb343753f94","91629787db6e18b226027587733b2f667a7982eed9509c2e39dfeaf4cfb1a17a","61e70c06858064c5e33e5009734222000610013e26fb6846ee17f55ddfb22da3","216e2dae905e73df943e003c428c7d340f21280750edb9d9ce2b15beeada667a"],"threshold":1}},"consistent_snapshot":false},"signatures":[{"keyid":"bc711f19576de2d71d1ca41eeccf7412f12c3ee4971185cd69066f8dc51d1ce6","sig":"7a0d8eda3e6058bf10f21bdda4b876c499b4182335ab943737c2121603d0e2ec707222e92eace3d10051264988705d9c51e2159d13e234d57441e60ba1a3c10a"}]}`

	OldFleetTUFURL      = "https://tuf.fleetctl.com"
	OldMetadataFileName = "tuf-metadata.json"
)

// Updater is responsible for managing update state.
//
// Updater supports updating plain executables and
// .tar.gz compressed executables.
type Updater struct {
	opt     Options
	client  *client.Client
	retryer *retry.LimitedWithCooldown
	mu      sync.Mutex
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
	// ServerCertificatePath is the TLS certificate CA file path to use for certificate
	// verification.
	//
	// If not set, then the OS CA certificate store is used.
	ServerCertificatePath string
	// ClientCertificate is the client TLS certificate to use to authenticate
	// to the update server.
	ClientCertificate *tls.Certificate
}

// Targets is a map of target name and its tracking information.
type Targets map[string]TargetInfo

// SetTargetChannel sets the channel of a target in the map.
func (ts Targets) SetTargetChannel(target, channel string) {
	t := ts[target]
	t.Channel = channel
	ts[target] = t
}

// SetTargetInfo sets/updates the TargetInfo for the given target.
func (u *Updater) SetTargetInfo(name string, info TargetInfo) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.opt.Targets[name] = info
}

// RemoveTargetInfo removes the TargetInfo for the given target.
func (u *Updater) RemoveTargetInfo(name string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.opt.Targets, name)
}

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
	// CustomCheckExec allows for a custom method for checking a downloaded executable.
	CustomCheckExec func(execPath string) error
}

// NewUpdater creates a new updater given the provided options. All the necessary
// directories are initialized.
func NewUpdater(opt Options) (*Updater, error) {
	if opt.LocalStore == nil {
		return nil, errors.New("opt.LocalStore must be non-nil")
	}

	remoteStore, err := createTUFRemoteStore(opt, opt.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("get tls config: %w", err)
	}
	tufClient := client.NewClient(opt.LocalStore, remoteStore)

	// TODO(lucas): Related to the NOTE below.
	//
	// NewUpdater is used when packaging Orbit (`fleetctl package`) and by Orbit
	// itself. We should refactor NewUpdater to receive an optional roots JSON string
	// which would only be set when packaging Orbit. Orbit should always trust its
	// local metadata and fail if it doesn't exist. (Alternatively introduce two New*
	// methods, NewUpdaterFromRoots and NewUpdaterFromMeta)

	// GetMeta returns empty metadata map if it doesn't exist in local store.
	meta, err := opt.LocalStore.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	if meta["root.json"] == nil {
		// NOTE: This path is currently only used when (1) packaging Orbit (`fleetctl package`), or
		// (2) in the edge-case when orbit's metadata JSON local file is removed for some reason, or
		// (3) first run on TUF migration from https://tuf.fleetctl.com to https://updates.fleetdm.com.
		//
		// When edge-case (2) happens, orbit will attempt to use Fleet DM's root JSON
		// (which may be unexpected on custom TUF Orbit deployments).
		log.Info().Msg("initialize TUF from embedded root keys")
		if err := tufClient.Init([]byte(opt.RootKeys)); err != nil {
			return nil, fmt.Errorf("client init with configuration metadata: %w", err)
		}
	}

	updater := &Updater{
		opt:    opt,
		client: tufClient,
		// per product spec, retry up to three consecutive times, then
		// wait 24 hours to try again.
		retryer: retry.NewLimitedWithCooldown(3, 24*time.Hour),
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
		return fmt.Errorf("client update: %w", err)
	}
	return nil
}

func IsExpiredErr(err error) bool {
	var errDecodeFailed client.ErrDecodeFailed
	if errors.As(err, &errDecodeFailed) {
		err = errDecodeFailed.Err
	}
	var errExpired verify.ErrExpired
	return errors.As(err, &errExpired)
}

// SignaturesExpired returns true if the "root", "targets", or "snapshot" signature is expired.
func (u *Updater) SignaturesExpired() bool {
	// When the "root", "targets", or "snapshot" signature is expired
	// client.Target fails with an expiration error.
	_, err := u.Lookup(constant.OrbitTUFTargetName)
	return IsExpiredErr(err)
}

// LookupsFail returns true if lookups are failing for any of the targets.
func (u *Updater) LookupsFail() bool {
	for target := range u.opt.Targets {
		if _, err := u.Lookup(target); err != nil {
			return true
		}
	}
	return false
}

// repoPath returns the path of the target in the remote repository.
func (u *Updater) repoPath(target string) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

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
	ok, err := entryExists(localTarget.ExecPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("path %q does not exist", localTarget.ExecPath)
	}
	return localTarget.ExecPath, nil
}

// entryExists returns whether a file or directory at path exists.
func entryExists(path string) (bool, error) {
	switch _, err := os.Stat(path); {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// DirLocalPath returns the configured root directory local path of a tar.gz target.
//
// Returns empty for a non tar.gz target.
func (u *Updater) DirLocalPath(target string) (string, error) {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return "", err
	}
	ok, err := entryExists(localTarget.DirPath)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("path %q does not exist", localTarget.DirPath)
	}
	return localTarget.DirPath, nil
}

// LocalTargetPaths returns (path, execPath, dirPath) to be used by this
// package for a given TargetInfo target.
func LocalTargetPaths(rootDir string, targetName string, t TargetInfo) (path, execPath, dirPath string) {
	path = filepath.Join(
		rootDir, binDir, targetName, t.Platform, t.Channel, t.TargetFile,
	)

	execPath = path
	if strings.HasSuffix(path, ".tar.gz") {
		execPath = filepath.Join(append([]string{filepath.Dir(path)}, t.ExtractedExecSubPath...)...)
		dirPath = filepath.Join(filepath.Dir(path), t.ExtractedExecSubPath[0])
	}

	return path, execPath, dirPath
}

// LocalTarget holds local paths of a target.
//
// E.g., for a osqueryd target:
//
//	LocalTarget{
//		Info: TargetInfo{
//			Platform:             "macos-app",
//			Channel:              "stable",
//			TargetFile:           "osqueryd.app.tar.gz",
//			ExtractedExecSubPath: []string{"osquery.app", "Contents", "MacOS", "osqueryd"},
//		},
//		Path: "/local/path/to/osqueryd.app.tar.gz",
//		DirPath: "/local/path/to/osqueryd.app",
//		ExecPath: "/local/path/to/osqueryd.app/Contents/MacOS/osqueryd",
//	}
type LocalTarget struct {
	// Info holds the TUF target and package structure info.
	Info TargetInfo
	// Path holds the location of the target as downloaded from TUF.
	Path string
	// DirPath holds the path of the extracted target.
	//
	// DirPath is empty for non-tar.gz targets.
	DirPath string
	// ExecPath is the path of the executable.
	ExecPath string
}

// localTarget returns the info and local path of a target.
func (u *Updater) localTarget(target string) (*LocalTarget, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	t, ok := u.opt.Targets[target]
	if !ok {
		return nil, fmt.Errorf("unknown target: %s", target)
	}
	path, execPath, dirPath := LocalTargetPaths(u.opt.RootDirectory, target, t)
	lt := &LocalTarget{
		Info:     t,
		Path:     path,
		ExecPath: execPath,
		DirPath:  dirPath,
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

// Get downloads (if it doesn't exist) a target and returns its local information.
func (u *Updater) Get(target string) (*LocalTarget, error) {
	meta, err := u.Lookup(target)
	if err != nil {
		return nil, err
	}

	// use the specific target hash as the key for retries. This allows new
	// updates to the TUF server to be downloaded immediately without
	// having to wait the cooldown.
	key := target
	if _, metaHash, err := selectHashFunction(meta); err == nil {
		key = fmt.Sprintf("%x", metaHash)
	}

	var localTarget *LocalTarget
	err = u.retryer.Do(key, func() error {
		var err error
		localTarget, err = u.get(target)
		return err
	})
	if err != nil {
		var rErr *retry.ExcessRetriesError
		if errors.As(err, &rErr) {
			return nil, fmt.Errorf("skipped getting target: %w", err)
		}

		return nil, fmt.Errorf("getting target: %w", err)
	}
	return localTarget, nil
}

func (u *Updater) get(target string) (*LocalTarget, error) {
	if target == "" {
		return nil, errors.New("target is required")
	}

	localTarget, err := u.localTarget(target)
	if err != nil {
		return nil, fmt.Errorf("failed to load local path for target %s: %w", target, err)
	}
	repoPath, err := u.repoPath(target)
	if err != nil {
		return nil, fmt.Errorf("failed to load repository path for target %s: %w", target, err)
	}

	switch stat, err := os.Stat(localTarget.Path); {
	case err == nil:
		if !stat.Mode().IsRegular() {
			return nil, fmt.Errorf("expected %s to be regular file", localTarget.Path)
		}
		meta, err := u.Lookup(target)
		if err != nil {
			return nil, err
		}
		if err := checkFileHash(meta, localTarget.Path); err != nil {
			log.Debug().Str("info", err.Error()).Msg("change detected")
			if err := u.download(target, repoPath, localTarget.Path, localTarget.Info.CustomCheckExec); err != nil {
				return nil, fmt.Errorf("download %q: %w", repoPath, err)
			}
			if strings.HasSuffix(localTarget.Path, ".tar.gz") {
				if err := os.RemoveAll(localTarget.DirPath); err != nil {
					return nil, fmt.Errorf("failed to remove old extracted dir: %q: %w", localTarget.DirPath, err)
				}
			}
			if strings.HasSuffix(localTarget.Path, ".pkg") && runtime.GOOS == "darwin" {
				if err := installPKG(localTarget.Path); err != nil {
					return nil, fmt.Errorf("updating pkg: %w", err)
				}
			}
		} else {
			log.Debug().Str("path", localTarget.Path).Str("target", target).Msg("found expected target locally")
		}
	case errors.Is(err, os.ErrNotExist):
		log.Debug().Err(err).Msg("stat file")
		if err := u.download(target, repoPath, localTarget.Path, localTarget.Info.CustomCheckExec); err != nil {
			return nil, fmt.Errorf("download %q: %w", repoPath, err)
		}
		if strings.HasSuffix(localTarget.Path, ".pkg") && runtime.GOOS == "darwin" {
			if err := installPKG(localTarget.Path); err != nil {
				return nil, fmt.Errorf("installing pkg for the first time: %w", err)
			}
		}
	default:
		return nil, fmt.Errorf("stat %q: %w", localTarget.Path, err)
	}

	if strings.HasSuffix(localTarget.Path, ".tar.gz") {
		s, err := os.Stat(localTarget.ExecPath)
		switch {
		case err == nil:
			// OK
		case errors.Is(err, os.ErrNotExist):
			if err := extractTarGz(localTarget.Path); err != nil {
				return nil, fmt.Errorf("extract %q: %w", localTarget.Path, err)
			}
			s, err = os.Stat(localTarget.ExecPath)
			if err != nil {
				return nil, fmt.Errorf("stat %q: %w", localTarget.ExecPath, err)
			}
		default:
			return nil, fmt.Errorf("stat %q: %w", localTarget.ExecPath, err)
		}
		if !s.Mode().IsRegular() {
			return nil, fmt.Errorf("expected a regular file: %q", localTarget.ExecPath)
		}
	}

	return localTarget, nil
}

// download downloads the target to the provided path. The file is deleted and
// an error is returned if the hash does not match.
func (u *Updater) download(target, repoPath, localPath string, customCheckExec func(execPath string) error) error {
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

	if err := u.checkExec(target, tmp.Name(), customCheckExec); err != nil {
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
	case "linux-arm64":
		return "linux", nil
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
	}
}

func goarchFromPlatform(platform string) ([]string, error) {
	switch platform {
	case "macos", "macos-app":
		return []string{"amd64", "arm64"}, nil
	case "windows":
		return []string{"amd64"}, nil
	case "linux":
		return []string{"amd64"}, nil
	case "linux-arm64":
		return []string{"arm64"}, nil
	default:
		return nil, fmt.Errorf("unknown platform: %s", platform)
	}
}

// checkExec checks/verifies a downloaded executable target by executing it.
func (u *Updater) checkExec(target, tmpPath string, customCheckExec func(execPath string) error) error {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return err
	}
	platformGOOS, err := goosFromPlatform(localTarget.Info.Platform)
	if err != nil {
		return err
	}
	if platformGOOS != runtime.GOOS {
		// Nothing to do, we can't check the executable if running cross-platform.
		// This generally happens when generating a package from a different platform
		// than the target package (e.g. generating an MSI package from macOS).
		return nil
	}

	platformGOARCH, err := goarchFromPlatform(localTarget.Info.Platform)
	if err != nil {
		return err
	}
	var containsArch bool
	for _, arch := range platformGOARCH {
		if arch == runtime.GOARCH {
			containsArch = true
		}
	}
	if !containsArch && strings.HasSuffix(os.Args[0], "fleetctl") {
		// Nothing to do, we can't reliably execute a
		// cross-architecture binary. This happens when cross-building
		// packages
		return nil
	}

	if strings.HasSuffix(tmpPath, ".tar.gz") {
		if err := extractTarGz(tmpPath); err != nil {
			return fmt.Errorf("extract %q: %w", tmpPath, err)
		}
		tmpDirPath := filepath.Join(filepath.Dir(tmpPath), localTarget.Info.ExtractedExecSubPath[0])
		defer os.RemoveAll(tmpDirPath)
		tmpPath = filepath.Join(append([]string{filepath.Dir(tmpPath)}, localTarget.Info.ExtractedExecSubPath...)...)
	}

	if strings.HasSuffix(tmpPath, ".pkg") && runtime.GOOS == "darwin" {
		cmd := exec.Command("pkgutil", "--payload-files", tmpPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("running pkgutil to verify %s: %s: %w", tmpPath, string(out), err)
		}
		return nil
	}

	if customCheckExec != nil {
		if err := customCheckExec(tmpPath); err != nil {
			return fmt.Errorf("custom exec new version failed: %w", err)
		}
	} else {
		// Note that this would fail for any binary that returns nonzero for --help.
		cmd := exec.Command(tmpPath, "--help")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("exec new version: %s: %w", string(out), err)
		}
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

func installPKG(path string) error {
	cmd := exec.Command("installer", "-pkg", path, "-target", "/")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("running pkgutil to install %s: %s: %w", path, string(out), err)
	}
	return nil
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

func CanRun(rootDirPath, targetName string, targetInfo TargetInfo) bool {
	_, binaryPath, _ := LocalTargetPaths(
		rootDirPath,
		targetName,
		targetInfo,
	)

	if _, err := os.Stat(binaryPath); err != nil {
		return false
	}

	return true
}

// HasAccessToNewTUFServer will verify if the agent has access to Fleet's new TUF
// by downloading the metadata and the orbit stable target.
//
// The metadata and the test target files will be downloaded to a temporary directory
// that will be removed before this method returns.
func HasAccessToNewTUFServer(opt Options) bool {
	fp := filepath.Join(opt.RootDirectory, "new-tuf-checked")
	ok, err := file.Exists(fp)
	if err != nil {
		log.Error().Err(err).Msg("failed to check new-tuf-checked file exists")
		return false
	}
	if ok {
		return true
	}
	tmpDir, err := os.MkdirTemp(opt.RootDirectory, "tuf-tmp*")
	if err != nil {
		log.Error().Err(err).Msg("failed to create tuf-tmp directory")
		return false
	}
	defer os.RemoveAll(tmpDir)
	localStore, err := filestore.New(filepath.Join(tmpDir, "tuf-tmp.json"))
	if err != nil {
		log.Error().Err(err).Msg("failed to create tuf-tmp local store")
		return false
	}
	remoteStore, err := createTUFRemoteStore(opt, DefaultURL)
	if err != nil {
		log.Error().Err(err).Msg("failed to create TUF remote store")
		return false
	}
	tufClient := client.NewClient(localStore, remoteStore)
	if err := tufClient.Init([]byte(opt.RootKeys)); err != nil {
		log.Error().Err(err).Msg("failed to pin root keys")
		return false
	}
	if _, err := tufClient.Update(); err != nil {
		// Logging as debug to not fill logs until users allow connections to new TUF server.
		log.Debug().Err(err).Msg("failed to update metadata from new TUF")
		return false
	}
	tmpFile, err := secure.OpenFile(
		filepath.Join(tmpDir, "orbit"),
		os.O_CREATE|os.O_WRONLY,
		constant.DefaultFileMode,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed open temp file for download")
		return false
	}
	defer tmpFile.Close()
	// We are using the orbit stable target as the test target.
	var (
		platform   string
		executable string
	)
	switch runtime.GOOS {
	case "darwin":
		platform = "macos"
		executable = "orbit"
	case "windows":
		platform = "windows"
		executable = "orbit.exe"
	case "linux":
		platform = "linux"
		executable = "orbit"
	}
	if err := tufClient.Download(fmt.Sprintf("orbit/%s/stable/%s", platform, executable), &fileDestination{tmpFile}); err != nil {
		// Logging as debug to not fill logs until users allow connections to new TUF server.
		log.Debug().Err(err).Msg("failed to download orbit from TUF")
		return false
	}

	if err := os.WriteFile(fp, []byte("new-tuf-checked"), constant.DefaultFileMode); err != nil {
		// We log the error and return success below anyway because the access check was successful.
		log.Error().Err(err).Msg("failed to write new-tuf-checked file")
	}
	// We assume access to the whole repository
	// if the orbit macOS stable target is downloaded successfully.
	return true
}

func createTUFRemoteStore(opt Options, serverURL string) (client.RemoteStore, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: opt.InsecureTransport,
	}
	if opt.ServerCertificatePath != "" {
		rootCAs, err := certificate.LoadPEM(opt.ServerCertificatePath)
		if err != nil {
			return nil, fmt.Errorf("loading server root CA: %w", err)
		}
		tlsConfig.RootCAs = rootCAs
	}
	if opt.ClientCertificate != nil {
		tlsConfig.Certificates = []tls.Certificate{*opt.ClientCertificate}
	}
	remoteOpt := &client.HTTPRemoteOptions{
		UserAgent: fmt.Sprintf("orbit/%s (%s %s)", build.Version, runtime.GOOS, runtime.GOARCH),
	}
	httpClient := fleethttp.NewClient(fleethttp.WithTLSClientConfig(tlsConfig))
	remoteStore, err := client.HTTPRemoteStore(serverURL, remoteOpt, httpClient)
	if err != nil {
		return nil, fmt.Errorf("init remote store: %w", err)
	}
	return remoteStore, nil
}
