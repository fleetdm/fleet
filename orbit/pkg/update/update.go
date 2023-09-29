// Package update contains the types and functions used by the update system.
package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/rs/zerolog/log"
	"github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
)

const (
	binDir     = "bin"
	stagingDir = "staging"

	defaultURL      = "https://tuf.fleetctl.com"
	defaultRootKeys = `{"signed":{"_type":"root","spec_version":"1.0","version":4,"expires":"2023-10-06T17:47:49Z","keys":{"0cd79ade57d278957069e03a0fca6b975b95c2895fb20bdc3075f71fc19a4474":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"4627d9071a4b4a78c5ee867ea70439583b08dbe4ff23514e3bcb0a292de9406f"}},"1a4d9beb826d1ff4e036d757cfcd6e36d0f041e58d25f99ef3a20ae3f8dd71e3":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"1083b5fedbcaf8f98163f2f7083bbb2761a334b2ba8de44df7be3feb846725f6"}},"3c1fbd1f3b3429d8ccadfb1abfbae5826d0cf74b0a6bcd384c3045d2fe27613c":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"b07555d05d4260410bdf12de7f76be905e288e801071877c7ca3d7f0459bee0f"}},"5003eae9f614f7e2a6c94167d20803eabffc6f65b8731e828e56d068f1b1d834":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"5d91bdfddc381e03d109e3e2f08413ed4ba181d98766eb97802967fb6cf2b87d"}},"5b99d0520321d0177d66b84136f3fc800dde1d36a501c12e28aa12d59a239315":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"8113955a28517752982ed6521e2162cf207605cfd316b8cba637c0a9b7a72856"}},"5f42172605e1a317c6bdae3891b4312a952759185941490624e052889821c929":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"86e26b13b9a64f7de4ad24b47e2bb9779a8628cae0e1afa61e56f2003c2ab586"}},"6c0e404295d4bf8915b46754b5f4546ab0d11ff7d83804d4aa2d178cfc38eafc":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"3782135dcec329bcd0e1eefc1acead987dc6a7d629db62c9fdde8bc8ff3fa781"}},"7cbbc9772d4d6acea33b7edf5a4bc52c85ff283475d428ffee73f9dbd0f62c89":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"f79d0d357aaa534a251abc7b0604725ba7b035eb53d1bdf5cc3173d73e3d9678"}},"7ea5cd46d58ac97ec1424007b7a6b0b3403308bb8aa8de885a75841f6f1d50dd":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"978cdddce95311d56b7fed39419a31019a38a1dab179cddb541ffaf99f442f1b"}},"94ca5921eb097bb871272c1cc3ea2cad833cb8d4c2dea4a826646be656059640":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"6512498c7596f55a23405889539fadbcefecd0909e4af0b54e29f45d49f9b9f7"}},"ae943cb8be8a849b37c66ed46bdd7e905ba3118c0c051a6ee3cd30625855a076":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"e7ffa6355dedd0cd34defc903dfac05a7a8c1855d63be24cecb5577cfde1f990"}},"d940df08b59b12c30f95622a05cc40164b78a11dd7d408395ee4f79773331b30":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"64d15cc3cbaac7eccfd9e0de5a56a0789aadfec3d02e77bf9180b8090a2c48d6"}},"efb4e9bd7a7d9e045edf6f5140c9835dbcbb7770850da44bf15a800b248c810e":{"keytype":"ed25519","scheme":"ed25519","keyid_hash_algorithms":["sha256","sha512"],"keyval":{"public":"0b8b28b30b44ddb733c7457a7c0f75fcbac563208ea1fe7179b5888a4f1d2237"}}},"roles":{"root":{"keyids":["5f42172605e1a317c6bdae3891b4312a952759185941490624e052889821c929"],"threshold":1},"snapshot":{"keyids":["94ca5921eb097bb871272c1cc3ea2cad833cb8d4c2dea4a826646be656059640","1a4d9beb826d1ff4e036d757cfcd6e36d0f041e58d25f99ef3a20ae3f8dd71e3","7ea5cd46d58ac97ec1424007b7a6b0b3403308bb8aa8de885a75841f6f1d50dd","5003eae9f614f7e2a6c94167d20803eabffc6f65b8731e828e56d068f1b1d834"],"threshold":1},"targets":{"keyids":["0cd79ade57d278957069e03a0fca6b975b95c2895fb20bdc3075f71fc19a4474","ae943cb8be8a849b37c66ed46bdd7e905ba3118c0c051a6ee3cd30625855a076","6c0e404295d4bf8915b46754b5f4546ab0d11ff7d83804d4aa2d178cfc38eafc","3c1fbd1f3b3429d8ccadfb1abfbae5826d0cf74b0a6bcd384c3045d2fe27613c"],"threshold":1},"timestamp":{"keyids":["efb4e9bd7a7d9e045edf6f5140c9835dbcbb7770850da44bf15a800b248c810e","d940df08b59b12c30f95622a05cc40164b78a11dd7d408395ee4f79773331b30","7cbbc9772d4d6acea33b7edf5a4bc52c85ff283475d428ffee73f9dbd0f62c89","5b99d0520321d0177d66b84136f3fc800dde1d36a501c12e28aa12d59a239315"],"threshold":1}},"consistent_snapshot":false},"signatures":[{"keyid":"39a1db745ca254d8f8eb27493df8867264d9fb394572ecee76876f4d7e9cb753","sig":"e71ff8ff6da3b4dad325da7f0e3a2a571564a9dd719d6b18a406af8b1d36ac203ae8d77e51afad55f5d948b3d347dd6e7d1ac7d4fd10aa8b5551ae2d570d7507"},{"keyid":"5f42172605e1a317c6bdae3891b4312a952759185941490624e052889821c929","sig":"26899ae7551846e171ec7277877f7615d24c0e22224ce1afdab38978a383fb79d809fdc714d71155a89e41a3515fade58ca86c9b29ae94ef85e8876a4adf2007"},{"keyid":"63b4cb9241c93bca9218c67334da3651394de4cf36c44bb1320bad7111df7bba","sig":"f1b50ac423355495b7896b6a0eee975962edcde497e58c5154045ce911e55799a69c0baa15227c01ee9f4a9bc809432933689de9a1699cdbf415ec1390ed2809"},{"keyid":"656c44011cf8b80a4da765bec1516ee9598ffca5fa7ccb51f0a9feb04e6e6cbd","sig":"166478a8a6602b6a705906845eb6ff873f439173126d184d3c2440b4abd6b0e7d30f42056a5aa1cb52f095414c3e9644a897223778fc343b52ebdb1af384ec0a"},{"keyid":"950097b911794bb554d7e83aa20c8aad11efcdc98f54b775fda76ac39eafa8fb","sig":"aab26209c6ac6a4471eec3df2068451556214079b730e3d4df3d27ad508bc1d1140dccfe83db29e44880bf30b5807c12a0b37496ee5486fb1537e33e3ab3be0c"},{"keyid":"d6e90309d70431729bf722b089a8049efaf449230d94dc90bafa1cfc12d2b36f","sig":"61d2665292aa5475e4ee24e84a668f287a597b59cfdc624867789e9c34412145bd7039405aae6aeae10e875ea9ff5bad74da0a6455dba2d932410aa1b7517209"},{"keyid":"e5d1873c4d5268f650a26ea3c6ffb4bec1e523875888ebb6303fac2bfd578cd0","sig":"d4e3a97e11c91ebd99aaaffb1bf4f6b9ec9fbda624872776fef9aee3c065f8b6a0883e1ca5fbdf91a7f476234539268207c3ed2065d726adbc11608112aa7002"}]}`
)

// Updater is responsible for managing update state.
//
// Updater supports updating plain executables and
// .tar.gz compressed executables.
type Updater struct {
	opt    Options
	client *client.Client
	mu     sync.Mutex
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

	httpClient := fleethttp.NewClient(fleethttp.WithTLSClientConfig(tlsConfig))

	remoteOpt := &client.HTTPRemoteOptions{
		UserAgent: fmt.Sprintf("orbit/%s (%s %s)", build.Version, runtime.GOOS, runtime.GOARCH),
	}
	remoteStore, err := client.HTTPRemoteStore(opt.ServerURL, remoteOpt, httpClient)
	if err != nil {
		return nil, fmt.Errorf("init remote store: %w", err)
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
		// NOTE: This path is currently only used when (1) packaging Orbit (`fleetctl package`) and
		// (2) in the edge-case when Orbit's metadata JSON local file is removed for some reason.
		// When edge-case (2) happens, Orbit will attempt to use Fleet DM's root JSON
		// (which may be unexpected on custom TUF Orbit deployments).
		if err := tufClient.Init([]byte(opt.RootKeys)); err != nil {
			return nil, fmt.Errorf("client init with configuration metadata: %w", err)
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
		return fmt.Errorf("update metadata: %w", err)
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
	return localTarget.ExecPath, nil
}

// DirLocalPath returns the configured root directory local path of a tar.gz target.
//
// Returns empty for a non tar.gz target.
func (u *Updater) DirLocalPath(target string) (string, error) {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return "", err
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
		} else {
			log.Debug().Str("path", localTarget.Path).Str("target", target).Msg("found expected target locally")
		}
	case errors.Is(err, os.ErrNotExist):
		log.Debug().Err(err).Msg("stat file")
		if err := u.download(target, repoPath, localTarget.Path, localTarget.Info.CustomCheckExec); err != nil {
			return nil, fmt.Errorf("download %q: %w", repoPath, err)
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
	default:
		return "", fmt.Errorf("unknown platform: %s", platform)
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

	if strings.HasSuffix(tmpPath, ".tar.gz") {
		if err := extractTarGz(tmpPath); err != nil {
			return fmt.Errorf("extract %q: %w", tmpPath, err)
		}
		tmpDirPath := filepath.Join(filepath.Dir(tmpPath), localTarget.Info.ExtractedExecSubPath[0])
		defer os.RemoveAll(tmpDirPath)
		tmpPath = filepath.Join(append([]string{filepath.Dir(tmpPath)}, localTarget.Info.ExtractedExecSubPath...)...)
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
