package mock

import "github.com/fleetdm/fleet/v4/server/fleet"

func ReturnFakeAppConfig(fake *fleet.AppConfig) AppConfigFunc {
	return func() (*fleet.AppConfig, error) {
		return fake, nil
	}
}
