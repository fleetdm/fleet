package update

import "github.com/fleetdm/fleet/v4/server/fleet"

type OrbitConfigReceiver interface {
	Run(*fleet.OrbitConfig) error
}
