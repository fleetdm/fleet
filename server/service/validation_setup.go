package service

import (
	"context"
	"fmt"
	"net/url"

	"github.com/kolide/kolide/server/kolide"
)

func (mw validationMiddleware) NewAppConfig(ctx context.Context, payload kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	invalid := &invalidArgumentError{}
	var serverURLString string
	if payload.ServerSettings == nil {
		invalid.Append("kolide_server_url", "missing required argument")
	} else {
		serverURLString = cleanupURL(*payload.ServerSettings.KolideServerURL)
	}
	if err := validateKolideServerURL(serverURLString); err != nil {
		invalid.Append("kolide_server_url", err.Error())
	}
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.NewAppConfig(ctx, payload)
}

func validateKolideServerURL(urlString string) error {
	serverURL, err := url.Parse(urlString)
	if err != nil {
		return err
	}
	if serverURL.Scheme != "https" {
		return fmt.Errorf("url scheme must be https")
	}
	return nil
}
