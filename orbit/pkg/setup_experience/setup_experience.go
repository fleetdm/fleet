package setupexperience

import (
	"context"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/swiftdialog"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

type Client interface {
	SetupExperienceReady() error
}

type SetupExperiencer struct {
	OrbitClient Client
}

func NewSetupExperiencer(oc Client) *SetupExperiencer {
	return &SetupExperiencer{OrbitClient: oc}
}

func (s *SetupExperiencer) Run(oc *fleet.OrbitConfig) error {
	log.Debug().Bool("shouldRunSetupExperience", oc.Notifications.RunSetupExperienceInstalls).Msg("JVE_LOG: in Run for setup experiencer")

	// We should only launch swiftDialog if we get the notification from Fleet.
	if !oc.Notifications.RunSetupExperienceInstalls {
		log.Debug().Msg("skipping setup experience")
		return nil
	}

	log.Debug().Msg("JVE_LOG: going to launch swift dialog zzzzzz")
	sdPath := "/opt/orbit/bin/swiftDialog/macos/stable/Dialog.app/Contents/MacOS/Dialog"
	if _, err := os.Stat(sdPath); err != nil {
		log.Error().Msg("couldn't find swiftDialog on orbit startup")
	} else {

		sd, err := swiftdialog.Create(context.Background(), sdPath, &swiftdialog.SwiftDialogOptions{Title: "Hello from setup experiencer", Height: "650"})
		if err != nil {
			log.Error().Err(err).Msg("failed to create swiftDialog")
		}
		sd.ShowProgress()
		for i := 1; i < 11; i++ {
			time.Sleep(2 * time.Second)
			sd.IncrementProgress()
		}
		if _, err := sd.Wait(); err != nil {
			log.Error().Err(err).Msg("failed to wait on swiftDialog")
		}
		log.Info().Msg("swiftDialog in setup assistant closed!")
		if err := s.OrbitClient.SetupExperienceReady(); err != nil {
			log.Error().Err(err).Msg("failed to mark as ready for setup experience")
		}

		log.Info().Msg("DEP device released")
	}

	return nil
}
