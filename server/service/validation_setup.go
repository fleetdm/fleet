package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw validationMiddleware) NewAppConfig(ctx context.Context, payload fleet.AppConfigPayload) (*fleet.AppConfig, error) {
	invalid := &fleet.InvalidArgumentError{}
	var serverURLString string
	if payload.ServerSettings == nil || payload.ServerSettings.ServerURL == nil ||
		*payload.ServerSettings.ServerURL == "" {
		invalid.Append("server_url", "missing required argument")
	} else {
		serverURLString = cleanupURL(*payload.ServerSettings.ServerURL)
	}
	if err := validateServerURL(serverURLString); err != nil {
		invalid.Append("server_url", err.Error())
	}
	if invalid.HasErrors() {
		return nil, invalid
	}
	return mw.Service.NewAppConfig(ctx, payload)
}

func validateServerURL(urlString string) error {
	serverURL, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	if serverURL.Scheme != "https" && !strings.Contains(serverURL.Host, "localhost") {
		return errors.New("url scheme must be https")
	}

	return nil
}
