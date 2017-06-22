package mock

import "github.com/kolide/fleet/server/kolide"

func ReturnFakeAppConfig(fake *kolide.AppConfig) AppConfigFunc {
	return func() (*kolide.AppConfig, error) {
		return fake, nil
	}
}
