package service

import (
	"context"
	"errors"
	"net/url"

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
	serverURL, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	// scheme present, or "localhost" host. When no scheme, `url.Parse` incorreclty parses "localhost"
	// as the scheme - obviously "localhost" not a scheme
	// Since "localhost" host is the only case we want to allow no included scheme, for now, this
	// works for us:
	if !(serverURL.Scheme == "https" || serverURL.Scheme == "http" || serverURL.Scheme == "localhost") {
		return errors.New(fleet.InvalidServerURLMsg)
	}
	return nil
}
