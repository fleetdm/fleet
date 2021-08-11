package eefleetctl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/secure"
	"github.com/pkg/errors"
	"github.com/theupdateframework/go-tuf"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	// consistentSnapshots are not needed due to the low update frequency of
	// these repositories.
	consistentSnapshots = false

	// ~10 years
	keyExpirationDuration = 10 * 365 * 24 * time.Hour

	// Expirations from
	// https://github.com/theupdateframework/notary/blob/e87b31f46cdc5041403c64b7536df236d5e35860/docs/best_practices.md#expiration-prevention
	// ~10 years
	rootExpirationDuration = 10 * 365 * 24 * time.Hour
	// ~3 years
	targetsExpirationDuration = 3 * 365 * 24 * time.Hour
	// ~3 years
	snapshotExpirationDuration = 3 * 365 * 24 * time.Hour
	// 14 days
	timestampExpirationDuration = 14 * 24 * time.Hour

	decryptionFailedError = "encrypted: decryption failed"
)

var (
	passHandler = newPassphraseHandler()
)

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

func updatesInitFunc(c *cli.Context) error {
	path := c.String("path")
	store := tuf.FileSystemStore(path, passHandler.getPassphrase)
	meta, err := store.GetMeta()
	if err != nil {
		return errors.Wrap(err, "get repo meta")
	}
	if len(meta) != 0 {
		return errors.Errorf("repo already initialized: %s", path)
	}
	// Ensure no existing keys before initializing
	if _, err := os.Stat(filepath.Join(path, "keys")); !errors.Is(err, os.ErrNotExist) {
		if err == nil {
			return errors.Errorf("keys directory already exists: %s", filepath.Join(path, "keys"))
		} else {
			return errors.Wrap(err, "failed to check existence of keys directory")
		}
	}

	repo, err := tuf.NewRepo(store)
	if err != nil {
		return errors.Wrap(err, "open repo")
	}

	// TODO messaging about using a secure environment

	// Explicitly initialize with consistent snapshots turned off.
	if err := repo.Init(consistentSnapshots); err != nil {
		return errors.Wrap(err, "initialize repo")
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
		return errors.Wrap(err, "sign root metadata")
	}

	// Create empty manifests for commit
	if err := repo.AddTargetsWithExpires(
		nil,
		nil,
		time.Now().Add(targetsExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "initialize targets")
	}
	if err := repo.SnapshotWithExpires(
		tuf.CompressionTypeNone,
		time.Now().Add(snapshotExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "make snapshot")
	}
	if err := repo.TimestampWithExpires(
		time.Now().Add(timestampExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "make timestamp")
	}

	// Commit empty manifests
	if err := repo.Commit(); err != nil {
		return errors.Wrap(err, "commit repo")
	}

	// TODO messaging about separating keys -- maybe we can help by splitting
	// things up into separate directories?

	return nil
}

func updatesRootsCommand() *cli.Command {
	return &cli.Command{
		Name:   "roots",
		Usage:  "Get root keys metadata",
		Flags:  updatesFlags(),
		Action: updatesRootsFunc,
	}
}

func updatesRootsFunc(c *cli.Context) error {
	repo, err := openRepo(c.String("path"))
	if err != nil {
		return err
	}

	keys, err := repo.RootKeys()
	if err != nil {
		return errors.Wrap(err, "get root metadata")
	}

	if err := json.NewEncoder(os.Stdout).Encode(keys); err != nil {
		return errors.Wrap(err, "encode root metadata")
	}

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
		dstPath := filepath.Join(name, platform, tag, name)
		if strings.HasSuffix(target, ".exe") {
			dstPath += ".exe"
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
		return errors.Wrap(err, "marshal custom metadata")
	}

	if err := repo.AddTargetsWithExpires(
		paths,
		meta,
		time.Now().Add(targetsExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "add targets")
	}

	if err := repo.SnapshotWithExpires(
		tuf.CompressionTypeNone,
		time.Now().Add(snapshotExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "make snapshot")
	}

	if err := repo.TimestampWithExpires(
		time.Now().Add(timestampExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "make timestamp")
	}

	if err := repo.Commit(); err != nil {
		return errors.Wrap(err, "commit repo")
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
		time.Now().Add(timestampExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "make timestamp")
	}

	if err := repo.Commit(); err != nil {
		return errors.Wrap(err, "commit repo")
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
		return errors.Wrap(err, "open src for copy")
	}
	defer src.Close()

	if err := secure.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return errors.Wrap(err, "create dst dir for copy")
	}

	dst, err := secure.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrap(err, "open dst for copy")
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copy src to dst")
	}

	return nil
}

func updatesGenKey(repo *tuf.Repo, role string) error {
	keyids, err := repo.GenKeyWithExpires(role, time.Now().Add(keyExpirationDuration))
	if err != nil {
		return errors.Wrapf(err, "generate %s key", role)
	}

	if len(keyids) != 1 {
		return errors.Errorf("expected 1 keyid for %s key: got %d", role, len(keyids))
	}
	fmt.Printf("Generated %s key with ID: %s\n", role, keyids[0])

	return nil
}

func openRepo(path string) (*tuf.Repo, error) {
	store := tuf.FileSystemStore(path, passHandler.getPassphrase)
	meta, err := store.GetMeta()
	if err != nil {
		return nil, errors.Wrap(err, "get repo meta")
	}
	if len(meta) == 0 {
		return nil, errors.Errorf("repo not initialized: %s", path)
	}

	repo, err := tuf.NewRepo(store)
	if err != nil {
		return nil, errors.Wrap(err, "new repo from store")
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

func (p *passphraseHandler) getPassphrase(role string, confirm bool) ([]byte, error) {
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
		passphrase, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, errors.Wrap(err, "read password")
		}

		if !confirm {
			return passphrase, nil
		}

		fmt.Printf("Repeat %s key passphrase: ", role)
		confirmation, err := terminal.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return nil, errors.Wrap(err, "read password confirmation")
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
		keys, err := store.GetSigningKeys(role)
		if err != nil {
			// TODO it would be helpful if we could upstream a new error type in
			// go-tuf and use errors.Is instead of comparing the text of the
			// error as we do currently.
			if errors.Cause(err).Error() != decryptionFailedError {
				return err
			} else if err != nil {
				if p.getPassphraseFromEnv(role) != nil {
					// Fatal error if environment variable passphrase is
					// incorrect
					return errors.Errorf("%s passphrase from %s is invalid", role, p.passphraseEnvName(role))
				}

				fmt.Printf("Failed to decrypt %s key. Try again.\n", role)
				delete(p.cache, role)
			}
			continue
		} else if len(keys) == 0 {
			return errors.Errorf("%s key not found", role)
		} else {
			return nil
		}
	}
}
