package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw validationMiddleware) NewAppConfig(ctx context.Context, payload fleet.AppConfig) (*fleet.AppConfig, error) {
	invalid := &fleet.InvalidArgumentError{}
	var serverURLString string
	if payload.ServerSettings.ServerURL == "" {
		invalid.Append("server_url", "missing required argument")
	} else {
		serverURLString = cleanupURL(payload.ServerSettings.ServerURL)
	}
	if err := ValidateServerURL(serverURLString); err != nil {
		invalid.Append("server_url", err.Error())
	}
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.NewAppConfig(ctx, payload)
}

func ValidateServerURL(urlString string) error {
	// TODO - implement more robust URL validation here

	// no valid scheme provided
	if !(strings.HasPrefix(urlString, "http://") || strings.HasPrefix(urlString, "https://")) {
		parsed, err := url.Parse("https://" + urlString)
		if err != nil {
			return err
		}
		// localhost only acceptable host if no schem (protocol) provided. .Host will include port (e.g.
		// 8080), so we check for "localhost" prefix
		// by checking for prefix, we also invalidate unsupported schemes like "ftp://"
		if !strings.HasPrefix(parsed.Host, "localhost") {
			return errors.New(fleet.InvalidServerURLMsg)
		}
		return nil
	}

	// scheme provided
	parsed, err := url.Parse(urlString)
	if err != nil {
		return err
	}
	// host required with scheme if provided
	if parsed.Host == "" {
		return errors.New(fleet.InvalidServerURLMsg)
	}

	return nil
}
