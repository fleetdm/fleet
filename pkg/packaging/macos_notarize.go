package packaging

import (
	"context"
	"os"
	"sync"

	"github.com/fatih/color"
	"github.com/mitchellh/gon/notarize"
	"github.com/mitchellh/gon/staple"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

func notarizePkg(pkgPath string) error {
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
			File:     pkgPath,
			BundleId: "com.fleetdm.orbit",
			Username: username,
			Password: password,
			Status: &statusHuman{
				Lock: &sync.Mutex{},
			},
		},
	)
	if err != nil {
		return errors.Wrap(err, "notarize")
	}

	log.Info().Str("logs", info.LogFileURL).Msg("notarization completed")

	if err := staple.Staple(context.Background(), &staple.Options{File: pkgPath}); err != nil {
		return errors.Wrap(err, "staple notarization")
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
