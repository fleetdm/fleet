package mock

import "github.com/fleetdm/fleet/server/fleet"

func ReturnFakeAppConfig(fake *fleet.AppConfig) AppConfigFunc {
	return func() (*fleet.AppConfig, error) {
		return fake, nil
	}
}
