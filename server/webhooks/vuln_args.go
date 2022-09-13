package webhooks

import (
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type VulnArgs struct {
	Vulnerablities []fleet.Vulnerability
	Meta           map[string]fleet.CVEMeta
	AppConfig      *fleet.AppConfig
	Time           time.Time
}
