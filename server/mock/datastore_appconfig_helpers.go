package mock

import "github.com/fleetdm/fleet/server/kolide"

func ReturnFakeAppConfig(fake *kolide.AppConfig) AppConfigFunc {
	return func() (*kolide.AppConfig, error) {
		return fake, nil
	}
}
