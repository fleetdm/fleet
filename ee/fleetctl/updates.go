//go:build darwin || linux
// +build darwin linux

package eefleetctl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/theupdateframework/go-tuf"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

const (
	// consistentSnapshots are not needed due to the low update frequency of
	// these repositories.
	consistentSnapshots = false

	decryptionFailedError = "encrypted: decryption failed"

	backupDirectory = ".backup"
)

// The following are defined as string variables so that we can use/set them in tests and test tooling.
var (
	// keyExpirationDuration is used when generating new keys (repository init)
	// or when rotating the root key.
	// ~10 years (10 * 365 * 24 hours)
	keyExpirationDuration = "87600h"

	//
	// Expirations from
	// https://github.com/theupdateframework/notary/blob/e87b31f46cdc5041403c64b7536df236d5e35860/docs/best_practices.md#expiration-prevention
	//
	// They are defined as string so we can modify them at build time for testing purposes.
	//

	// rootExpirationDuration is used to set the expiration of root.json when revoking the current root key.
	// ~10 years (10 * 365 * 24 hours)
	rootExpirationDuration = "87600h"
	// targetsExpirationDuration is used to set the expiration of the targets.json signature.
	// ~3 years (3 * 365 * 24 hours)
	targetsExpirationDuration = "26280h"
	// snapshotExpirationDuration is used to set the expiration of the snapshot.json signature.
	// ~3 years (3 * 365 * 24 hours)
	snapshotExpirationDuration = "26280h"
	// timestampExpirationDuration is used to set the expiration of the timestamp.json signature.
	// 14 days (14 * 24 hours)
	timestampExpirationDuration = "336h"

	keyExpirationDuration_       = mustParseDuration(keyExpirationDuration)
	rootExpirationDuration_      = mustParseDuration(rootExpirationDuration)
	targetsExpirationDuration_   = mustParseDuration(targetsExpirationDuration)
	snapshotExpirationDuration_  = mustParseDuration(snapshotExpirationDuration)
	timestampExpirationDuration_ = mustParseDuration(timestampExpirationDuration)
)

var passHandler = newPassphraseHandler()

func UpdatesCommand() *cli.Command {
	return &cli.Command{
		Name:  "updates",
		Usage: "Manage client updates",
		Description: `fleetctl updates commands provide the initialization and management of a TUF-compliant update repository.

This functionality is licensed under the Fleet EE License. Usage requires a current Fleet EE subscription.`,
		Subcommands: []*cli.Command{
			updatesInitCommand(),
			updatesRootsCommand(),
			updatesAddCommand(),
			updatesTimestampCommand(),
			updatesRotateCommand(),
		},
	}
}

func updatesFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "path",
			Usage: "Path to local repository",
			Value: ".",
		},
	}
}

func updatesInitCommand() *cli.Command {
	return &cli.Command{
		Name:   "init",
		Usage:  "Initialize update repository",
		Flags:  updatesFlags(),
		Action: updatesInitFunc,
	}
}

func mustParseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(err)
	}
	return d
}

func updatesInitFunc(c *cli.Context) error {
	path := c.String("path")
	store := tuf.FileSystemStore(path, passHandler.getPassphrase)
	meta, err := store.GetMeta()
	if err != nil {
		return fmt.Errorf("get repo meta: %w", err)
	}
	if len(meta) != 0 {
		return fmt.Errorf("repo already initialized: %s", path)
	}
	// Ensure no existing keys before initializing
	if _, err := os.Stat(filepath.Join(path, "keys")); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return fmt.Errorf("keys directory already exists: %s", filepath.Join(path, "keys"))
		}
		return fmt.Errorf("failed to check existence of keys directory: %w", err)
	}

	repo, err := tuf.NewRepo(store)
	if err != nil {
		return fmt.Errorf("open repo: %w", err)
	}

	// TODO messaging about using a secure environment

	// Explicitly initialize with consistent snapshots turned off.
	if err := repo.Init(consistentSnapshots); err != nil {
		return fmt.Errorf("initialize repo: %w", err)
	}

	// Generate keys
	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		// TODO don't fatal here if passwords don't match
		if err := updatesGenKey(repo, role); err != nil {
			return err
		}
	}

	// Sign roots metadata
	if err := repo.Sign("root.json"); err != nil {
		return fmt.Errorf("sign root metadata: %w", err)
	}

	// Create empty manifests for commit
	if err := repo.AddTargetsWithExpires(
		nil,
		nil,
		time.Now().Add(targetsExpirationDuration_),
	); err != nil {
		return fmt.Errorf("initialize targets: %w", err)
	}
	if err := repo.SnapshotWithExpires(time.Now().Add(snapshotExpirationDuration_)); err != nil {
		return fmt.Errorf("make snapshot: %w", err)
	}
	if err := repo.TimestampWithExpires(time.Now().Add(timestampExpirationDuration_)); err != nil {
		return fmt.Errorf("make timestamp: %w", err)
	}

	// Commit empty manifests
	if err := repo.Commit(); err != nil {
		return fmt.Errorf("commit repo: %w", err)
	}

	// TODO messaging about separating keys -- maybe we can help by splitting
	// things up into separate directories?

	return nil
}

func updatesRootsCommand() *cli.Command {
	return &cli.Command{
		Name:   "roots",
		Usage:  "Get root metadata",
		Flags:  updatesFlags(),
		Action: updatesRootsFunc,
	}
}

func updatesRootsFunc(c *cli.Context) error {
	repo, err := openRepo(c.String("path"))
	if err != nil {
		return err
	}

	meta, err := repo.GetMeta()
	if err != nil {
		return fmt.Errorf("get repo metadata: %w", err)
	}
	rootMeta := meta["root.json"]
	if rootMeta == nil {
		return errors.New("missing root metadata")
	}

	fmt.Println(string(rootMeta))

	return nil
}

func updatesAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Add a new update artifact",
		Flags: append(updatesFlags(),
			&cli.StringFlag{
				Name:     "target",
				Required: true,
				Usage:    "Path to target (required)",
			},
			&cli.StringFlag{
				Name:     "name",
				Required: true,
				Usage:    "Name of target (required)",
			},
			&cli.StringFlag{
				Name:     "platform",
				Required: true,
				Usage:    "Platform name of target (required)",
			},
			&cli.StringFlag{
				Name:     "version",
				Required: true,
				Usage:    "Version of target (required)",
			},
			&cli.StringSliceFlag{
				Name:    "tag",
				Aliases: []string{"t"},
				Usage:   "Tags to apply to the target (multiple may be specified)",
			},
		),
		Action: updatesAddFunc,
	}
}

func updatesAddFunc(c *cli.Context) error {
	repo, err := openRepo(c.String("path"))
	if err != nil {
		return err
	}

	if err := checkKeys(c.String("path"),
		"timestamp",
		"snapshot",
		"targets",
	); err != nil {
		return err
	}

	tags := c.StringSlice("tag")
	version := c.String("version")
	platform := c.String("platform")
	name := c.String("name")
	target := c.String("target")

	targetsPath := filepath.Join(c.String("path"), "staged", "targets")

	var paths []string
	for _, tag := range append([]string{version}, tags...) {
		// NOTE(lucas): "updates add" expects the target file to match the target name.
		// E.g.
		// 	- an ".app.tar.gz" file for target=osqueryd is expected to be called "osqueryd.app.tar.gz".
		// 	- an ".exe" file for target=osqueryd is expected to be called "osqueryd.exe".
		var dstPath string
		// check if we are adding extensions, which we namespace as "extensions/<ext_name>"
		if strings.HasPrefix(name, "extensions/") {
			dstPath = filepath.Join(name, platform, tag, strings.TrimPrefix(name, "extensions/"))
		} else {
			dstPath = filepath.Join(name, platform, tag, name)
		}
		switch {
		case name == constant.DesktopTUFTargetName && platform == "windows":
			// This is a special case for the desktop target on Windows.
			dstPath = filepath.Join(filepath.Dir(dstPath), constant.DesktopAppExecName+".exe")
		case name == constant.DesktopTUFTargetName && (platform == "linux" || platform == "linux-arm64"):
			// This is a special case for the desktop target on Linux.
			dstPath += ".tar.gz"
		// The convention for Windows extensions is to use the extension `.ext.exe`
		// All Windows executables must end with `.exe`.
		case strings.HasSuffix(target, ".ext.exe"):
			dstPath += ".ext.exe"
		case strings.HasSuffix(target, ".exe"):
			dstPath += ".exe"
		case strings.HasSuffix(target, ".app.tar.gz"):
			dstPath += ".app.tar.gz"
		case strings.HasSuffix(target, ".pkg"):
			dstPath += ".pkg"
		// osquery extensions require the .ext suffix
		case strings.HasSuffix(target, ".ext"):
			dstPath += ".ext"
		}
		fullPath := filepath.Join(targetsPath, dstPath)
		paths = append(paths, dstPath)
		if err := copyTarget(target, fullPath); err != nil {
			return err
		}
	}

	type customMetadata struct {
		Version string `json:"version"`
	}
	meta, err := json.Marshal(customMetadata{Version: version})
	if err != nil {
		return fmt.Errorf("marshal custom metadata: %w", err)
	}

	if err := repo.AddTargetsWithExpires(
		paths,
		meta,
		time.Now().Add(targetsExpirationDuration_),
	); err != nil {
		return fmt.Errorf("add targets: %w", err)
	}

	if err := repo.SnapshotWithExpires(time.Now().Add(snapshotExpirationDuration_)); err != nil {
		return fmt.Errorf("make snapshot: %w", err)
	}

	if err := repo.TimestampWithExpires(time.Now().Add(timestampExpirationDuration_)); err != nil {
		return fmt.Errorf("make timestamp: %w", err)
	}

	if err := repo.Commit(); err != nil {
		return fmt.Errorf("commit repo: %w", err)
	}

	return nil
}

func updatesTimestampCommand() *cli.Command {
	return &cli.Command{
		Name:   "timestamp",
		Usage:  "Sign a new timestamp manifest",
		Flags:  updatesFlags(),
		Action: updatesTimestampFunc,
	}
}

func updatesTimestampFunc(c *cli.Context) error {
	repo, err := openRepo(c.String("path"))
	if err != nil {
		return err
	}

	if err := checkKeys(c.String("path"),
		"timestamp",
	); err != nil {
		return err
	}

	if err := repo.TimestampWithExpires(
		time.Now().Add(timestampExpirationDuration_),
	); err != nil {
		return fmt.Errorf("make timestamp: %w", err)
	}

	if err := repo.Commit(); err != nil {
		return fmt.Errorf("commit repo: %w", err)
	}

	return nil
}

func updatesRotateCommand() *cli.Command {
	return &cli.Command{
		Name:      "rotate",
		Usage:     "Rotate signing keys",
		ArgsUsage: "<role>",
		Description: `Rotate the signing keys used for updates metadata signing. This should be used when keys are compromised or expiring.

role must be one of ['root', 'targets', 'timestamp', 'snapshot']
		`,
		Flags:  updatesFlags(),
		Action: updatesRotateFunc,
	}
}

func updatesRotateFunc(c *cli.Context) error {
	if c.NArg() != 1 {
		return errors.New("role must be provided")
	}
	role := c.Args().Get(0)

	repoPath := c.String("path")
	repo, err := openRepo(repoPath)
	if err != nil {
		return err
	}
	store, err := openLocalStore(repoPath)
	if err != nil {
		return err
	}

	if err := checkKeys(repoPath,
		"root",
		"targets",
		"snapshot",
		"timestamp",
	); err != nil {
		return err
	}

	// Get old keys for role
	keys, err := store.GetSigners(role)
	if err != nil {
		return fmt.Errorf("get keys for role: %w", err)
	}

	// Prepare to roll back in case of error.
	success := false
	commit, rollback, err := startRotatePseudoTx(repoPath)
	if err != nil {
		return err
	}
	defer func() {
		if success {
			if err := commit(); err != nil {
				fmt.Println("Warning: failure during commit:", err)
			}
		} else {
			fmt.Println("Rolling back changes.")
			if err := rollback(); err != nil {
				fmt.Println("Warning: failure during rollback:", err)
			}
		}
	}()

	// Delete old keys for role
	for _, key := range keys {
		id := key.PublicData().IDs()[0]
		err := repo.RevokeKeyWithExpires(role, id, time.Now().Add(rootExpirationDuration_))
		if err != nil {
			// go-tuf keeps keys around even after they are revoked from the manifest. We can skip
			// tuf.ErrKeyNotFound as these represent keys that are not present in the manifest and
			// so do not need to be revoked.
			if !errors.As(err, &tuf.ErrKeyNotFound{}) {
				return fmt.Errorf("revoke key: %w", err)
			}
		}
	}

	// TODO change passphrase for new key:
	// Waiting on https://github.com/theupdateframework/go-tuf/pull/163

	// Generate new key for role
	if err := updatesGenKey(repo, role); err != nil {
		return err
	}

	// Re-sign the root metadata
	if err := repo.Sign("root.json"); err != nil {
		return fmt.Errorf("sign root.json: %w", err)
	}

	// Generate new metadata for each role (technically some of these may not need regeneration
	// depending on which key was rotated, but there should be no harm in generating new ones for each).
	if err := repo.AddTargetsWithExpires(nil, nil, time.Now().Add(targetsExpirationDuration_)); err != nil {
		return fmt.Errorf("generate targets: %w", err)
	}

	if err := repo.SnapshotWithExpires(time.Now().Add(snapshotExpirationDuration_)); err != nil {
		return fmt.Errorf("generate snapshot: %w", err)
	}

	if err := repo.TimestampWithExpires(time.Now().Add(timestampExpirationDuration_)); err != nil {
		return fmt.Errorf("generate timestamp: %w", err)
	}

	// Commit the changes.
	if err := repo.Commit(); err != nil {
		return fmt.Errorf("commit repo: %w", err)
	}

	success = true
	return nil
}

// startRotatePseudoTx starts a "transaction" for the rotation routine, preparing a commit and
// rollback function for the metadata files that are modified by the rotation process.
func startRotatePseudoTx(repoPath string) (commit, rollback func() error, err error) {
	repositoryDir := filepath.Join(repoPath, "repository")
	if err := createBackups(repositoryDir); err != nil {
		return nil, nil, fmt.Errorf("backup repository: %w", err)
	}
	keysDir := filepath.Join(repoPath, "keys")
	if err := createBackups(keysDir); err != nil {
		return nil, nil, fmt.Errorf("backup keys: %w", err)
	}

	commit = func() error {
		// Remove the backups on successful rotation.
		if err := os.RemoveAll(filepath.Join(repositoryDir, backupDirectory)); err != nil {
			return fmt.Errorf("remove repository backup directory: %w", err)
		}
		if err := os.RemoveAll(filepath.Join(keysDir, backupDirectory)); err != nil {
			return fmt.Errorf("remove keys backup directory: %w", err)
		}
		return nil
	}

	rollback = func() error {
		// Restore the backups on failure.
		if err := restoreBackups(repositoryDir); err != nil {
			return fmt.Errorf("restore repository backup: %w", err)
		}
		if err := restoreBackups(keysDir); err != nil {
			return fmt.Errorf("restore keys backup: %w", err)
		}
		return nil
	}

	return commit, rollback, nil
}

// createBackups creates backups for metadata and key files during the key rotation process,
// allowing for rollback if necessary.
func createBackups(dirPath string) error {
	// Only *.json files need to be backed up (other files are not modified)
	backupPath := filepath.Join(dirPath, backupDirectory)
	if err := os.Mkdir(backupPath, os.ModeDir|0o744); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("backup directory already exists: %w", err)
		}
		return fmt.Errorf("create backup directory: %w", err)
	}

	// Copy each of the *.json files into a backup file.
	files, err := filepath.Glob(filepath.Join(dirPath, "*.json"))
	if err != nil {
		return fmt.Errorf("glob for backup: %w", err)
	}
	for _, path := range files {
		if err := file.CopyWithPerms(
			path,
			filepath.Join(backupPath, filepath.Base(path)),
		); err != nil {
			return fmt.Errorf("copy for backup: %w", err)
		}
	}

	return nil
}

// restoreBackups restores the directory from the backups created by createBackups.
func restoreBackups(dirPath string) error {
	backupDir := filepath.Join(dirPath, backupDirectory)
	info, err := os.Stat(backupDir)
	if err != nil {
		return fmt.Errorf("stat backup path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("backup is not directory: %s", backupDir)
	}

	// Remove files that did not exist at backup time (determined by no corresponding backup).
	files, err := filepath.Glob(filepath.Join(dirPath, "*.json"))
	if err != nil {
		return fmt.Errorf("glob for restore: %w", err)
	}
	for _, path := range files {
		backupPath := filepath.Join(backupDir, filepath.Base(path))
		exists, err := file.Exists(backupPath)
		if err != nil {
			return fmt.Errorf("check exists for restore: %w", err)
		}

		// File does not exist in the backup, remove it because this implies that the file was added
		// since the backup was taken.
		if !exists {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove for restore: %w", err)
			}
		}
	}

	// Restore files from backups.
	backupFiles, err := filepath.Glob(filepath.Join(backupDir, "*.json"))
	if err != nil {
		return fmt.Errorf("glob for restore: %w", err)
	}
	for _, path := range backupFiles {
		originalPath := filepath.Join(dirPath, filepath.Base(path))

		// Replace with the backed up file, copying the previous permissions.
		if err := file.CopyWithPerms(path, originalPath); err != nil {
			return fmt.Errorf("copy for restore: %w", err)
		}
	}

	// Remove the backups now that we are finished with the restore.
	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("remove backup directory: %w", err)
	}

	return nil
}

func checkKeys(repoPath string, keys ...string) error {
	// Verify we can decrypt necessary role keys
	store := tuf.FileSystemStore(repoPath, passHandler.getPassphrase)
	for _, role := range keys {
		if err := passHandler.checkPassphrase(store, role); err != nil {
			return err
		}
	}

	return nil
}

func copyTarget(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open src for copy: %w", err)
	}
	defer src.Close()

	if err := secure.MkdirAll(filepath.Dir(dstPath), 0o700); err != nil {
		return fmt.Errorf("create dst dir for copy: %w", err)
	}

	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open dst for copy: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy src to dst: %w", err)
	}

	return nil
}

func updatesGenKey(repo *tuf.Repo, role string) error {
	keyids, err := repo.GenKeyWithExpires(role, time.Now().Add(keyExpirationDuration_))
	if err != nil {
		return fmt.Errorf("generate %s key: %w", role, err)
	}

	if len(keyids) != 1 {
		return fmt.Errorf("expected 1 keyid for %s key: got %d", role, len(keyids))
	}
	fmt.Printf("Generated %s key with ID: %s\n", role, keyids[0])

	return nil
}

func openLocalStore(path string) (tuf.LocalStore, error) {
	store := tuf.FileSystemStore(path, passHandler.getPassphrase)
	meta, err := store.GetMeta()
	if err != nil {
		return nil, fmt.Errorf("get repo meta: %w", err)
	}
	if len(meta) == 0 {
		return nil, fmt.Errorf("repo not initialized: %s", path)
	}
	return store, nil
}

func openRepo(path string) (*tuf.Repo, error) {
	store, err := openLocalStore(path)
	if err != nil {
		return nil, err
	}

	repo, err := tuf.NewRepo(store)
	if err != nil {
		return nil, fmt.Errorf("new repo from store: %w", err)
	}

	return repo, nil
}

// passphraseHandler will cache passphrases so that they can be checked prior to
// usage without requiring the user to enter them more than once.
type passphraseHandler struct {
	cache map[string][]byte
}

func newPassphraseHandler() *passphraseHandler {
	return &passphraseHandler{cache: make(map[string][]byte)}
}

// TODO #4145 make use of recently added `change` argument
func (p *passphraseHandler) getPassphrase(role string, confirm, change bool) ([]byte, error) {
	// Check cache
	if pass, ok := p.cache[role]; ok {
		return pass, nil
	}

	// Get passphrase
	var err error
	passphrase, err := p.readPassphrase(role, confirm)
	if err != nil {
		return nil, err
	}
	if len(passphrase) == 0 {
		return nil, errors.New("passphrase must not be empty")
	}

	// Store cache
	p.cache[role] = passphrase

	return passphrase, nil
}

// export FLEET_TIMESTAMP_PASSPHRASE=insecure FLEET_SNAPSHOT_PASSPHRASE=insecure FLEET_TARGETS_PASSPHRASE=insecure FLEET_ROOT_PASSPHASE=insecure
func (p *passphraseHandler) passphraseEnvName(role string) string {
	return fmt.Sprintf("FLEET_%s_PASSPHRASE", strings.ToUpper(role))
}

func (p *passphraseHandler) getPassphraseFromEnv(role string) []byte {
	if pass, ok := os.LookupEnv(p.passphraseEnvName(role)); ok {
		return []byte(pass)
	}

	return nil
}

// Read input. Adapted from
// https://github.com/theupdateframework/go-tuf/blob/aee6270feb5596036edde4b6d7564fa17db811cb/cmd/tuf/main.go#L125
func (p *passphraseHandler) readPassphrase(role string, confirm bool) ([]byte, error) {
	// Loop until error reading or successful confirmation (if needed)
	for {
		if passphrase := p.getPassphraseFromEnv(role); passphrase != nil {
			return passphrase, nil
		}

		fmt.Printf("Enter %s key passphrase: ", role)
		// the int(...) conversion is required as on Windows syscall.Stdin is of type Handle.
		passphrase, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("read password: %w", err)
		}

		if !confirm {
			return passphrase, nil
		}

		fmt.Printf("Repeat %s key passphrase: ", role)
		// the int(...) conversion is required as on Windows syscall.Stdin is of type Handle.
		confirmation, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert
		fmt.Println()
		if err != nil {
			return nil, fmt.Errorf("read password confirmation: %w", err)
		}

		if bytes.Equal(passphrase, confirmation) {
			return passphrase, nil
		}

		fmt.Println("The entered passphrases do not match")
	}
}

func (p *passphraseHandler) checkPassphrase(store tuf.LocalStore, role string) error {
	// It seems the only way to check the passphrase is to try decrypting the
	// key and see if it is successful. Loop until successful decryption or
	// non-decryption error.
	for {
		keys, err := store.GetSigners(role)
		if err != nil {
			// TODO it would be helpful if we could upstream a new error type in
			// go-tuf and use errors.Is instead of comparing the text of the
			// error as we do currently.
			if ctxerr.Cause(err).Error() != decryptionFailedError {
				return err
			} else if err != nil {
				if p.getPassphraseFromEnv(role) != nil {
					// Fatal error if environment variable passphrase is
					// incorrect
					return fmt.Errorf("%s passphrase from %s is invalid", role, p.passphraseEnvName(role))
				}

				fmt.Printf("Failed to decrypt %s key. Try again.\n", role)
				delete(p.cache, role)
			}
			continue
		} else if len(keys) == 0 {
			return fmt.Errorf("%s key not found", role)
		}

		return nil
	}
}
