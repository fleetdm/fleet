package packaging

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/mitchellh/gon/notarize"
	"github.com/mitchellh/gon/staple"
	"github.com/rs/zerolog/log"
)

// Notarize will do the Notarization step. Note that the provided path must be a .zip or a .dmg.
func Notarize(path, bundleIdentifier string) error {
	username, ok := os.LookupEnv("AC_USERNAME")
	if !ok {
		return errors.New("AC_USERNAME must be set in environment")
	}

	password, ok := os.LookupEnv("AC_PASSWORD")
	if !ok {
		return errors.New("AC_PASSWORD must be set in environment")
	}

	info, err := notarize.Notarize(
		context.Background(),
		&notarize.Options{
			File:     path,
			BundleId: bundleIdentifier,
			Username: username,
			Password: password,
			Status: &statusHuman{
				Lock: &sync.Mutex{},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("notarize: %w", err)
	}

	log.Info().Str("logs", info.LogFileURL).Msg("notarization completed")

	return nil
}

// Staple will do the "stapling" step of Notarization. Note that this only works on .app and .dmg
// (not .zip or plain binaries).
func Staple(path string) error {
	if err := staple.Staple(context.Background(), &staple.Options{File: path}); err != nil {
		return fmt.Errorf("staple notarization: %w", err)
	}

	return nil
}

// NotarizeStaple will notarize and staple a macOS artifact.
func NotarizeStaple(path, bundleIdentifier string) error {
	if err := Notarize(path, bundleIdentifier); err != nil {
		return err
	}

	if err := Staple(path); err != nil {
		return err
	}

	return nil
}

// This status plugin copied from
// https://github.com/mitchellh/gon/blob/v0.2.3/cmd/gon/status_human.go since it
// is not exposed from the gon library.

// statusHuman implements notarize.Status and outputs information to
// the CLI for human consumption.
type statusHuman struct {
	Prefix string
	Lock   *sync.Mutex

	lastStatus string
}

func (s *statusHuman) Submitting() {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	color.New().Fprintf(os.Stdout, "    %sSubmitting file for notarization...\n", s.Prefix)
}

func (s *statusHuman) Submitted(uuid string) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	color.New().Fprintf(os.Stdout, "    %sSubmitted. Request UUID: %s\n", s.Prefix, uuid)
	color.New().Fprintf(
		os.Stdout, "    %sWaiting for results from Apple. This can take minutes to hours.\n", s.Prefix)
}

func (s *statusHuman) Status(info notarize.Info) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	if info.Status != s.lastStatus {
		s.lastStatus = info.Status
		color.New().Fprintf(os.Stdout, "    %sStatus: %s\n", s.Prefix, info.Status)
	}
}
