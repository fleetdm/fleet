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

	// TODO(lucas): Before merging to `main` update DefaultURL to "https://updates.fleetctl.com".
	DefaultURL       = `https://tuf.fleetctl.com`
	MetadataFileName = "updates-metadata.json"
	// TODO(lucas): Before merging to `main` update defaultRootMetadata to the root.json when the new repository is ready.
	defaultRootMetadata = `{"signed":{"_type":"root","spec_version":"1.0","version":5,"expires":"2034-10-05T15:57:51-05:00","keys":{"0cd79ade57d278957069e03a0fca6b975b95c2895fb20bdc3075f71fc19a4474":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"4627d9071a4b4a78c5ee867ea70439583b08dbe4ff23514e3bcb0a292de9406f"}},"1a4d9beb826d1ff4e036d757cfcd6e36d0f041e58d25f99ef3a20ae3f8dd71e3":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"1083b5fedbcaf8f98163f2f7083bbb2761a334b2ba8de44df7be3feb846725f6"}},"3c1fbd1f3b3429d8ccadfb1abfbae5826d0cf74b0a6bcd384c3045d2fe27613c":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"b07555d05d4260410bdf12de7f76be905e288e801071877c7ca3d7f0459bee0f"}},"5003eae9f614f7e2a6c94167d20803eabffc6f65b8731e828e56d068f1b1d834":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"5d91bdfddc381e03d109e3e2f08413ed4ba181d98766eb97802967fb6cf2b87d"}},"5b99d0520321d0177d66b84136f3fc800dde1d36a501c12e28aa12d59a239315":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"8113955a28517752982ed6521e2162cf207605cfd316b8cba637c0a9b7a72856"}},"6c0e404295d4bf8915b46754b5f4546ab0d11ff7d83804d4aa2d178cfc38eafc":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"3782135dcec329bcd0e1eefc1acead987dc6a7d629db62c9fdde8bc8ff3fa781"}},"7cbbc9772d4d6acea33b7edf5a4bc52c85ff283475d428ffee73f9dbd0f62c89":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"f79d0d357aaa534a251abc7b0604725ba7b035eb53d1bdf5cc3173d73e3d9678"}},"7ea5cd46d58ac97ec1424007b7a6b0b3403308bb8aa8de885a75841f6f1d50dd":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"978cdddce95311d56b7fed39419a31019a38a1dab179cddb541ffaf99f442f1b"}},"94ca5921eb097bb871272c1cc3ea2cad833cb8d4c2dea4a826646be656059640":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"6512498c7596f55a23405889539fadbcefecd0909e4af0b54e29f45d49f9b9f7"}},"ae943cb8be8a849b37c66ed46bdd7e905ba3118c0c051a6ee3cd30625855a076":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"e7ffa6355dedd0cd34defc903dfac05a7a8c1855d63be24cecb5577cfde1f990"}},"d940df08b59b12c30f95622a05cc40164b78a11dd7d408395ee4f79773331b30":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"64d15cc3cbaac7eccfd9e0de5a56a0789aadfec3d02e77bf9180b8090a2c48d6"}},"e4594c3f5ef66488fe3787be822677b5401c1ed866422182f50b9b64d075af82":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"27b118c2f3d8c954c747f41aea339da021fcb4d5fdda859a15199d48ff89b01d"}},"efb4e9bd7a7d9e045edf6f5140c9835dbcbb7770850da44bf15a800b248c810e":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"0b8b28b30b44ddb733c7457a7c0f75fcbac563208ea1fe7179b5888a4f1d2237"}}},"roles":{"root":{"keyids":["e4594c3f5ef66488fe3787be822677b5401c1ed866422182f50b9b64d075af82"],"threshold":1},"snapshot":{"keyids":["94ca5921eb097bb871272c1cc3ea2cad833cb8d4c2dea4a826646be656059640","1a4d9beb826d1ff4e036d757cfcd6e36d0f041e58d25f99ef3a20ae3f8dd71e3","7ea5cd46d58ac97ec1424007b7a6b0b3403308bb8aa8de885a75841f6f1d50dd","5003eae9f614f7e2a6c94167d20803eabffc6f65b8731e828e56d068f1b1d834"],"threshold":1},"targets":{"keyids":["0cd79ade57d278957069e03a0fca6b975b95c2895fb20bdc3075f71fc19a4474","ae943cb8be8a849b37c66ed46bdd7e905ba3118c0c051a6ee3cd30625855a076","6c0e404295d4bf8915b46754b5f4546ab0d11ff7d83804d4aa2d178cfc38eafc","3c1fbd1f3b3429d8ccadfb1abfbae5826d0cf74b0a6bcd384c3045d2fe27613c"],"threshold":1},"timestamp":{"keyids":["efb4e9bd7a7d9e045edf6f5140c9835dbcbb7770850da44bf15a800b248c810e","d940df08b59b12c30f95622a05cc40164b78a11dd7d408395ee4f79773331b30","7cbbc9772d4d6acea33b7edf5a4bc52c85ff283475d428ffee73f9dbd0f62c89","5b99d0520321d0177d66b84136f3fc800dde1d36a501c12e28aa12d59a239315"],"threshold":1}},"consistent_snapshot":false},"signatures":[{"keyid":"39a1db745ca254d8f8eb27493df8867264d9fb394572ecee76876f4d7e9cb753","sig":"88898e3b2bea874b1145d556608ad542797c754bd126428f18ff351dad4e53fc9504ae07d0c07bceab59963c5014f7ecf85c54beb0b62daa66444b180575e603"},{"keyid":"5f42172605e1a317c6bdae3891b4312a952759185941490624e052889821c929","sig":"5b94def2749db440cfb3ab5a7964f658e24c8fadd811c36abfe8143d877935f28681f918eed9db43dbe075d26cbd26c874cb36dec9eaf450ea3429c28dcd1403"},{"keyid":"63b4cb9241c93bca9218c67334da3651394de4cf36c44bb1320bad7111df7bba","sig":"0cb61afc4add58634b2e2d8e50816e9fe44514e6c34a50bac9097ead9452559770ab0f01562eb9faac8509f90d4baa49d05d84c5a2bdf1f2ead1aacad3122802"},{"keyid":"656c44011cf8b80a4da765bec1516ee9598ffca5fa7ccb51f0a9feb04e6e6cbd","sig":"1bb724481b05c40596254e63172f142577fbb1f92251818cd0e91d35aa0bef31a3b499d7dcd36dc9fc5d4e4a465b1a9ce8a9adfb76cf080eda276156c9103804"},{"keyid":"950097b911794bb554d7e83aa20c8aad11efcdc98f54b775fda76ac39eafa8fb","sig":"a5bcc34e724241a098304220233bdba78d5d8dc8d874d76ce74a4ec683990ea11c1fab02727a23c8d8d5000a53736f2565b00982e182a555bd9c5420d868d40f"},{"keyid":"d6e90309d70431729bf722b089a8049efaf449230d94dc90bafa1cfc12d2b36f","sig":"f8da9e3a9cc9abd31ccb9f299e9946717ef881f243f39aaec3f6b3e085142bcbacd29f0c2a276d557d54b482ef12354e09d2e4fe226558047e8b0e9f1acd3e04"},{"keyid":"e4594c3f5ef66488fe3787be822677b5401c1ed866422182f50b9b64d075af82","sig":"149940db1b91063742f4e5cdd28df3ee7c4ecbdad8d64a9585fd8755468f0faf80bbe3061d56da8d391c4b1d5a39bc82e6ed40353b0bea5b38d0740113129e0a"},{"keyid":"e5d1873c4d5268f650a26ea3c6ffb4bec1e523875888ebb6303fac2bfd578cd0","sig":"6da6fac86f62ca3264eb4d898ff298227362c5c486fd88c708225ed4e525f8e8eb8459cc4ef74e0a5ab3cc046046be992f1fe98855946274b4bfeff08b85ce00"}]}`

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
