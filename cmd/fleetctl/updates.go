package main

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

	"github.com/pkg/errors"
	"github.com/theupdateframework/go-tuf"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	// consistentSnapshots are not needed due to the low update frequency of
	// these repositories.
	consistentSnapshots = false
	// default to 10 year expiration on keys
	keyExpirationDuration = 10 * 365 * 24 * time.Hour

	// Expirations from
	// https://github.com/theupdateframework/notary/blob/e87b31f46cdc5041403c64b7536df236d5e35860/docs/best_practices.md#expiration-prevention
	// 10 years
	rootExpirationDuration = 10 * 365 * 24 * time.Hour
	// 3 years
	targetsExpirationDuration = 3 * 365 * 24 * time.Hour
	// 3 years
	snapshotExpirationDuration = 3 * 365 * 24 * time.Hour
	// 14 days
	timestampExpirationDuration = 14 * 24 * time.Hour
)

func updatesCommand() cli.Command {
	return cli.Command{
		Name:  "updates",
		Usage: "Manage client updates",
		Subcommands: []cli.Command{
			updatesInitCommand(),
			updatesRootsCommand(),
			updatesAddCommand(),
		},
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
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
		configFlag(),
		contextFlag(),
	}
}

func updatesInitCommand() cli.Command {
	return cli.Command{
		Name:   "init",
		Usage:  "Initialize update repository",
		Flags:  updatesFlags(),
		Action: updatesInitFunc,
	}
}

func updatesInitFunc(c *cli.Context) error {
	path := c.String("path")
	store := tuf.FileSystemStore(path, updatesPassphraseFunc)
	meta, err := store.GetMeta()
	if err != nil {
		return errors.Wrap(err, "get repo meta")
	}
	if len(meta) != 0 {
		return errors.Errorf("repo already initialized: %s", path)
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

	// TODO messaging about separating keys -- maybe we can help by splitting
	// things up into separate directories?

	return nil
}

func updatesRootsCommand() cli.Command {
	return cli.Command{
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

func updatesAddCommand() cli.Command {
	return cli.Command{
		Name:  "add",
		Usage: "Add a new update artifact",
		Flags: append(updatesFlags(),
			&cli.StringFlag{
				Name:     "target",
				Required: true,
				Usage:    "Path to target to add (required)",
			},
			&cli.StringSliceFlag{
				Name:     "version",
				Required: true,
				Usage:    "Version of target (required)",
			},
			&cli.StringSliceFlag{
				Name:  "tag,t",
				Usage: "Tags to apply to the target (multiple may be specified)",
			},
		),
		Action: updatesAddFunc,
	}
}

type customMetadata struct {
	Version string `json:"version"`
}

func updatesAddFunc(c *cli.Context) error {
	repo, err := openRepo(c.String("path"))
	if err != nil {
		return err
	}

	// TODO verify all passwords here
	// store := tuf.FileSystemStore(c.String("path"), updatesPassphraseFunc)
	// keys, err := store.GetSigningKeys("timestamp")

	tags := c.StringSlice("tag")
	version := c.String("version")

	targetsPath := filepath.Join(c.String("path"), "staged", "targets")

	var paths []string
	for _, tag := range append([]string{version}, tags...) {
		dstPath := filepath.Join("osqueryd", "linux", tag, "osqueryd")
		fullPath := filepath.Join(targetsPath, dstPath)
		paths = append(paths, dstPath)
		if err := copyTarget(c.String("target"), fullPath); err != nil {
			return err
		}
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
		time.Now().Add(snapshotExpirationDuration),
	); err != nil {
		return errors.Wrap(err, "make timestamp")
	}

	if err := repo.Commit(); err != nil {
		return errors.Wrap(err, "commit repo")
	}

	return nil
}

func copyTarget(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return errors.Wrap(err, "open src for copy")
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return errors.Wrap(err, "create dst dir for copy")
	}

	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE, 0644)
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
	store := tuf.FileSystemStore(path, updatesPassphraseFunc)
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

func updatesPassphraseFunc(role string, confirm bool) ([]byte, error) {
	// Adapted from
	// https://github.com/theupdateframework/go-tuf/blob/aee6270feb5596036edde4b6d7564fa17db811cb/cmd/tuf/main.go#L125

	if pass := os.Getenv(fmt.Sprintf("FLEET_UPDATE_%s_PASSPHRASE", strings.ToUpper(role))); pass != "" {
		return []byte(pass), nil
	}

	fmt.Printf("Enter %s key passphrase: ", role)
	passphrase, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	if !confirm {
		return passphrase, nil
	}

	fmt.Printf("Repeat %s key passphrase: ", role)
	confirmation, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(passphrase, confirmation) {
		return nil, errors.New("The entered passphrases do not match")
	}
	return passphrase, nil
}
