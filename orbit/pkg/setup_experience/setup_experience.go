package setupexperience

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/rs/zerolog/log"
)

// SetupExperiencer is the type that manages the Fleet setup experience flow during macOS Setup
// Assistant. It uses swiftDialog as a UI for showing the status of software installations and
// script execution that are configured to run before the user has full access to the device.
type SetupExperiencer struct{}

func NewSetupExperiencer() *SetupExperiencer {
	return &SetupExperiencer{}
}

func (s *SetupExperiencer) Run(oc *fleet.OrbitConfig) error {
	// We should only launch swiftDialog if we get the notification from Fleet.
	if !oc.Notifications.RunSetupExperience {
		log.Debug().Msg("skipping setup experience")
		return nil
	}

	log.Debug().Msg("TODO: launch swiftDialog here!")

	return nil
}
