package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

const (
	configFilePerms = 0600
)

type configFile struct {
	Contexts map[string]Context `json:"contexts"`
}

type Context struct {
	Address       string `json:"address"`
	Email         string `json:"email"`
	Token         string `json:"token"`
	TLSSkipVerify bool   `json:"tls-skip-verify"`
	RootCA        string `json:"rootca"`
	URLPrefix     string `json:"url-prefix"`
}

func configFlag() cli.Flag {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}
	defaultConfigPath := filepath.Join(homeDir, ".fleet", "config")
	return cli.StringFlag{
		Name:   "config",
		Value:  defaultConfigPath,
		EnvVar: "CONFIG",
		Usage:  "Path to the Fleet config file",
	}
}

func contextFlag() cli.Flag {
	return cli.StringFlag{
		Name:   "context",
		Value:  "default",
		EnvVar: "CONTEXT",
		Usage:  "Name of Fleet config context to use",
	}
}

func makeConfigIfNotExists(fp string) error {
	if _, err := os.Stat(filepath.Dir(fp)); os.IsNotExist(err) {
		if err := os.Mkdir(filepath.Dir(fp), 0700); err != nil {
			return err
		}
	}

	_, err := os.OpenFile(fp, os.O_RDONLY|os.O_CREATE, configFilePerms)
	return err
}

func readConfig(fp string) (c configFile, err error) {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		return
	}

	err = yaml.Unmarshal(b, &c)

	if c.Contexts == nil {
		c.Contexts = map[string]Context{
			"default": Context{},
		}
	}
	return
}

func writeConfig(fp string, c configFile) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fp, b, configFilePerms)
}

func getConfigValue(c *cli.Context, key string) (interface{}, error) {
	var (
		flContext string
		flConfig  string
	)

	flConfig = c.String("config")
	flContext = c.String("context")

	if err := makeConfigIfNotExists(flConfig); err != nil {
		return nil, errors.Wrapf(err, "error verifying that config exists at %s", flConfig)
	}

	config, err := readConfig(flConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading config at %s", flConfig)
	}

	currentContext, ok := config.Contexts[flContext]
	if !ok {
		fmt.Printf("[+] Context %q not found, creating it with default values\n", flContext)
		currentContext = Context{}
	}

	switch key {
	case "address":
		return currentContext.Address, nil
	case "email":
		return currentContext.Email, nil
	case "token":
		return currentContext.Token, nil
	case "rootca":
		return currentContext.RootCA, nil
	case "tls-skip-verify":
		if currentContext.TLSSkipVerify {
			return true, nil
		} else {
			return false, nil
		}
	case "url-prefix":
		return currentContext.URLPrefix, nil
	default:
		return nil, fmt.Errorf("%q is an invalid key", key)
	}
}

func setConfigValue(c *cli.Context, key, value string) error {
	var (
		flContext string
		flConfig  string
	)

	flConfig = c.String("config")
	flContext = c.String("context")

	if err := makeConfigIfNotExists(flConfig); err != nil {
		return errors.Wrapf(err, "error verifying that config exists at %s", flConfig)
	}

	config, err := readConfig(flConfig)
	if err != nil {
		return errors.Wrapf(err, "error reading config at %s", flConfig)
	}

	currentContext, ok := config.Contexts[flContext]
	if !ok {
		fmt.Printf("[+] Context %q not found, creating it with default values\n", flContext)
		currentContext = Context{}
	}

	switch key {
	case "address":
		currentContext.Address = value
	case "email":
		currentContext.Email = value
	case "token":
		currentContext.Token = value
	case "rootca":
		currentContext.RootCA = value
	case "tls-skip-verify":
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return errors.Wrapf(err, "error parsing %q as bool", value)
		}
		currentContext.TLSSkipVerify = boolValue
	case "url-prefix":
		currentContext.URLPrefix = value
	default:
		return fmt.Errorf("%q is an invalid option", key)
	}

	config.Contexts[flContext] = currentContext

	if err := writeConfig(flConfig, config); err != nil {
		return errors.Wrap(err, "error saving config file")
	}

	return nil
}

func configSetCommand() cli.Command {
	var (
		flAddress       string
		flEmail         string
		flToken         string
		flTLSSkipVerify bool
		flRootCA        string
		flURLPrefix     string
	)
	return cli.Command{
		Name:      "set",
		Usage:     "Set config options",
		UsageText: `fleetctl config set [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			cli.StringFlag{
				Name:        "address",
				EnvVar:      "ADDRESS",
				Value:       "",
				Destination: &flAddress,
				Usage:       "Address of the Fleet server",
			},
			cli.StringFlag{
				Name:        "email",
				EnvVar:      "EMAIL",
				Value:       "",
				Destination: &flEmail,
				Usage:       "Email to use when connecting to the Fleet server",
			},
			cli.StringFlag{
				Name:        "token",
				EnvVar:      "TOKEN",
				Value:       "",
				Destination: &flToken,
				Usage:       "Fleet API token",
			},
			cli.BoolFlag{
				Name:        "tls-skip-verify",
				EnvVar:      "INSECURE",
				Destination: &flTLSSkipVerify,
				Usage:       "Skip TLS certificate validation",
			},
			cli.StringFlag{
				Name:        "rootca",
				EnvVar:      "ROOTCA",
				Value:       "",
				Destination: &flRootCA,
				Usage:       "Specify RootCA chain used to communicate with fleet",
			},
			cli.StringFlag{
				Name:        "url-prefix",
				EnvVar:      "URL_PREFIX",
				Value:       "",
				Destination: &flURLPrefix,
				Usage:       "Specify URL Prefix to use with Fleet server (copy from server configuration)",
			},
		},
		Action: func(c *cli.Context) error {
			set := false

			if flAddress != "" {
				set = true
				if err := setConfigValue(c, "address", flAddress); err != nil {
					return errors.Wrap(err, "error setting address")
				}
				fmt.Printf("[+] Set the address config key to %q in the %q context\n", flAddress, c.String("context"))
			}

			if flEmail != "" {
				set = true
				if err := setConfigValue(c, "email", flEmail); err != nil {
					return errors.Wrap(err, "error setting email")
				}
				fmt.Printf("[+] Set the email config key to %q in the %q context\n", flEmail, c.String("context"))
			}

			if flToken != "" {
				set = true
				if err := setConfigValue(c, "token", flToken); err != nil {
					return errors.Wrap(err, "error setting token")
				}
				fmt.Printf("[+] Set the token config key to %q in the %q context\n", flToken, c.String("context"))
			}

			if flTLSSkipVerify {
				set = true
				if err := setConfigValue(c, "tls-skip-verify", "true"); err != nil {
					return errors.Wrap(err, "error setting tls-skip-verify")
				}
				fmt.Printf("[+] Set the tls-skip-verify config key to \"true\" in the %q context\n", c.String("context"))
			}

			if flRootCA != "" {
				set = true
				if err := setConfigValue(c, "rootca", flRootCA); err != nil {
					return errors.Wrap(err, "error setting rootca")
				}
				fmt.Printf("[+] Set the rootca config key to %q in the %q context\n", flRootCA, c.String("context"))
			}

			if flURLPrefix != "" {
				set = true
				if err := setConfigValue(c, "url-prefix", flURLPrefix); err != nil {
					return errors.Wrap(err, "error setting URL Prefix")
				}
				fmt.Printf("[+] Set the url-prefix config key to %q in the %q context\n", flURLPrefix, c.String("context"))
			}

			if !set {
				return cli.ShowCommandHelp(c, "set")
			}

			return nil
		},
	}
}

func configGetCommand() cli.Command {
	return cli.Command{
		Name:      "get",
		Usage:     "Get a config option",
		UsageText: `fleetctl config get [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			if len(c.Args()) != 1 {
				return cli.ShowCommandHelp(c, "get")
			}

			key := c.Args()[0]

			// validate key
			switch key {
			case "address", "email", "token", "tls-skip-verify", "rootca":
			default:
				return cli.ShowCommandHelp(c, "get")
			}

			value, err := getConfigValue(c, key)
			if err != nil {
				return errors.Wrap(err, "error getting config value")
			}

			fmt.Printf("  %s.%s => %s\n", c.String("context"), key, value)

			return nil
		},
	}
}
