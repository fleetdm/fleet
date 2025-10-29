package worker

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	kitlog "github.com/go-kit/log"
)

const SoftwareJobName = "software"

type SoftwareTask string

type Software struct {
	Datastore     fleet.Datastore
	androidModule android.Service
	Log           kitlog.Logger
}

func (s *Software) Name() string {
	return SoftwareJobName
}
