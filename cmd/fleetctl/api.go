package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func unauthenticatedClientFromCLI(c *cli.Context) (*service.Client, error) {
	if flag.Lookup("test.v") != nil {
		return service.NewClient(os.Getenv("FLEET_SERVER_ADDRESS"), true, "", "")
	}

	if err := makeConfigIfNotExists(c.String("config")); err != nil {
		return nil, errors.Wrapf(err, "error verifying that config exists at %s", c.String("config"))
	}

	config, err := readConfig(c.String("config"))
	if err != nil {
		return nil, err
	}

	cc, ok := config.Contexts[c.String("context")]
	if !ok {
		return nil, fmt.Errorf("context %q is not found", c.String("context"))
	}

	if cc.Address == "" {
		return nil, errors.New("set the Fleet API address with: fleetctl config set --address https://localhost:8080")
	}

	if runtime.GOOS == "windows" && cc.RootCA == "" && !cc.TLSSkipVerify {
		return nil, errors.New("Windows clients must configure rootca (secure) or tls-skip-verify (insecure)")
	}

	var options []service.ClientOption
	if getDebug(c) {
		options = append(options, service.EnableClientDebug())
	}

	fleet, err := service.NewClient(
		cc.Address,
		cc.TLSSkipVerify,
		cc.RootCA,
		cc.URLPrefix,
		options...,
	)
	if err != nil {
		return nil, errors.Wrap(err, "error creating Fleet API client handler")
	}

	return fleet, nil
}

func clientFromCLI(c *cli.Context) (*service.Client, error) {
	fleet, err := unauthenticatedClientFromCLI(c)
	if err != nil {
		return nil, err
	}

	configPath, context := c.String("config"), c.String("context")

	if flag.Lookup("test.v") != nil {
		fleet.SetToken("AAAA")
		return fleet, nil
	}

	// Add authentication token
	t, err := getConfigValue(configPath, context, "token")
	if err != nil {
		return nil, errors.Wrap(err, "error getting token from the config")
	}

	if token, ok := t.(string); ok {
		if token == "" {
			return nil, errors.New("Please log in with: fleetctl login")
		}
		fleet.SetToken(token)
	} else {
		return nil, errors.Errorf("token config value was not a string: %+v", t)
	}

	return fleet, nil
}
