package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
)

func updatesCommand() cli.Command {
	return cli.Command{
		Name:  "updates",
		Usage: "Manage client updates",
		Subcommands: []cli.Command{
			updatesInitCommand(),
			updatesRootsCommand(),
		},
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
	}
}

func updatesFlags() []cli.Flag {
	return []cli.Flag{
		configFlag(),
		contextFlag(),
		&cli.StringFlag{
			Name:  "path",
			Usage: "Path to local repository",
			Value: ".",
		},
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
	repo, err := updatesRepo(c.String("path"))
	if err != nil {
		return err
	}

	// TODO messaging about using a secure environment

	// TODO check whether this has been completed

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
	repo, err := updatesRepo(c.String("path"))
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

func updatesRepo(path string) (*tuf.Repo, error) {
	repo, err := tuf.NewRepo(tuf.FileSystemStore(path, updatesPassphraseFunc))
	if err != nil {
		return nil, errors.Wrap(err, "open repo")
	}

	return repo, nil
}

func updatesPassphraseFunc(role string, confirm bool) ([]byte, error) {
	// Adapted from
	// https://github.com/theupdateframework/go-tuf/blob/aee6270feb5596036edde4b6d7564fa17db811cb/cmd/tuf/main.go#L125

	if pass := os.Getenv(fmt.Sprintf("FLEET_UPDATE_%s_PASSPHRASE", strings.ToUpper(role))); pass != "" {
		return []byte(pass), nil
	}

	fmt.Printf("Enter %s keys passphrase: ", role)
	passphrase, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return nil, err
	}

	if !confirm {
		return passphrase, nil
	}

	fmt.Printf("Repeat %s keys passphrase: ", role)
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
