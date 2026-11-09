package service

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func validateSetupServerURL(ctx context.Context, serverURL string) error {
	invalid := &fleet.InvalidArgumentError{}
	var serverURLString string
	if serverURL == "" {
		invalid.Append("server_url", "missing required argument")
	} else {
		serverURLString = cleanupURL(serverURL)
	}
	if err := ValidateServerURL(serverURLString); err != nil {
		invalid.Append("server_url", err.Error())
	}
	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}
	return nil
}

func ValidateServerURL(urlString string) error {
	// TODO - implement more robust URL validation here

	// no valid scheme provided
	if !(strings.HasPrefix(urlString, "http://") || strings.HasPrefix(urlString, "https://")) {
		return errors.New(fleet.InvalidServerURLMsg)
	}

	// valid scheme provided - require host
	parsed, err := url.Parse(urlString)
	if err != nil {
		return err
	}
	if parsed.Host == "" {
		return errors.New(fleet.InvalidServerURLMsg)
	}

	return nil
}
