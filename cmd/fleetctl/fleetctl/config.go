package fleetctl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/ghodss/yaml"
	"github.com/urfave/cli/v2"
)

const (
	configFilePerms = 0600
)

type configFile struct {
	Contexts map[string]Context `json:"contexts"`
}

type Context struct {
	Address       string            `json:"address"`
	Email         string            `json:"email"`
	Token         string            `json:"token"`
	TLSSkipVerify bool              `json:"tls-skip-verify"`
	RootCA        string            `json:"rootca"`
	URLPrefix     string            `json:"url-prefix"`
	CustomHeaders map[string]string `json:"custom-headers"`
}

func configFlag() cli.Flag {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "~"
	}
	defaultConfigPath := filepath.Join(homeDir, ".fleet", "config")
	return &cli.StringFlag{
		Name:    "config",
		Value:   defaultConfigPath,
		EnvVars: []string{"CONFIG"},
		Usage:   "Path to the fleetctl config file",
	}
}

func contextFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "context",
		Value:   "default",
		EnvVars: []string{"CONTEXT"},
		Usage:   "Name of fleetctl config context to use",
	}
}

func makeConfigIfNotExists(fp string) error {
	if _, err := os.Stat(filepath.Dir(fp)); errors.Is(err, os.ErrNotExist) {
		if err := secure.MkdirAll(filepath.Dir(fp), 0700); err != nil {
			return err
		}
	}

	f, err := secure.OpenFile(fp, os.O_RDONLY|os.O_CREATE, configFilePerms)
	if err == nil {
		f.Close()
	}
	return err
}

func readConfig(fp string) (configFile, error) {
	var c configFile
	b, err := os.ReadFile(fp)
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("unmarshal config: %w", err)
	}

	if c.Contexts == nil {
		c.Contexts = map[string]Context{
			"default": {},
		}
	}
	return c, nil
}

func writeConfig(fp string, c configFile) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(fp, b, configFilePerms)
}

func getConfigValue(configPath, context, key string) (interface{}, error) {
	if err := makeConfigIfNotExists(configPath); err != nil {
		return nil, fmt.Errorf("error verifying that config exists at %s: %w", configPath, err)
	}

	config, err := readConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config at %s: %w", configPath, err)
	}

	currentContext, ok := config.Contexts[context]
	if !ok {
		fmt.Printf("[+] Context %q not found, creating it with default values\n", context)
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
		}
		return false, nil
	case "url-prefix":
		return currentContext.URLPrefix, nil
	case "custom-headers":
		return currentContext.CustomHeaders, nil
	default:
		return nil, fmt.Errorf("%q is an invalid key", key)
	}
}

func setConfigValue(configPath, context, key string, value interface{}) error {
	if err := makeConfigIfNotExists(configPath); err != nil {
		return fmt.Errorf("error verifying that config exists at %s: %w", configPath, err)
	}

	config, err := readConfig(configPath)
	if err != nil {
		return fmt.Errorf("error reading config at %s: %w", configPath, err)
	}

	currentContext, ok := config.Contexts[context]
	if !ok {
		fmt.Printf("[+] Context %q not found, creating it with default values\n", context)
		currentContext = Context{}
	}

	var strVal string
	switch key {
	case "address", "email", "token", "rootca", "tls-skip-verify", "url-prefix":
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("error setting %q, string value expected, got %T", key, value)
		}
		strVal = s
	}

	switch key {
	case "address":
		currentContext.Address = strVal
	case "email":
		currentContext.Email = strVal
	case "token":
		currentContext.Token = strVal
	case "rootca":
		currentContext.RootCA = strVal
	case "tls-skip-verify":
		boolValue, err := strconv.ParseBool(strVal)
		if err != nil {
			return fmt.Errorf("error parsing %q as bool: %w", value, err)
		}
		currentContext.TLSSkipVerify = boolValue
	case "url-prefix":
		currentContext.URLPrefix = strVal

	case "custom-headers":
		vals, ok := value.([]string)
		if !ok {
			return fmt.Errorf("error setting %q, []string value expected, got %T", key, value)
		}

		hdrs := make(map[string]string, len(vals))
		for _, v := range vals {
			parts := strings.SplitN(v, ":", 2)
			if len(parts) < 2 {
				parts = append(parts, "")
			}
			hdrs[parts[0]] = parts[1]
		}
		currentContext.CustomHeaders = hdrs

	default:
		return fmt.Errorf("%q is an invalid option", key)
	}

	config.Contexts[context] = currentContext

	if err := writeConfig(configPath, config); err != nil {
		return fmt.Errorf("error saving config file: %w", err)
	}

	return nil
}

func configSetCommand() *cli.Command {
	var (
		flAddress       string
		flEmail         string
		flToken         string
		flTLSSkipVerify bool
		flRootCA        string
		flURLPrefix     string
		flCustomHeaders cli.StringSlice
	)
	return &cli.Command{
		Name:      "set",
		Usage:     "Set config options",
		UsageText: `fleetctl config set [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			&cli.StringFlag{
				Name:        "address",
				EnvVars:     []string{"ADDRESS"},
				Value:       "",
				Destination: &flAddress,
				Usage:       "Address of the Fleet server",
			},
			&cli.StringFlag{
				Name:        "email",
				EnvVars:     []string{"EMAIL"},
				Value:       "",
				Destination: &flEmail,
				Usage:       "Email to use when connecting to the Fleet server",
			},
			&cli.StringFlag{
				Name:        "token",
				EnvVars:     []string{"TOKEN"},
				Value:       "",
				Destination: &flToken,
				Usage:       "Fleet API token",
			},
			&cli.BoolFlag{
				Name:        "tls-skip-verify",
				EnvVars:     []string{"INSECURE"},
				Destination: &flTLSSkipVerify,
				Usage:       "Skip TLS certificate validation",
			},
			&cli.StringFlag{
				Name:        "rootca",
				EnvVars:     []string{"ROOTCA"},
				Value:       "",
				Destination: &flRootCA,
				Usage:       "Specify RootCA chain used to communicate with Fleet",
			},
			&cli.StringFlag{
				Name:        "url-prefix",
				EnvVars:     []string{"URL_PREFIX"},
				Value:       "",
				Destination: &flURLPrefix,
				Usage:       "Specify URL Prefix to use with Fleet server (copy from server configuration)",
			},
			&cli.StringSliceFlag{
				Name:        "custom-header",
				EnvVars:     []string{"CUSTOM_HEADER"},
				Value:       nil,
				Destination: &flCustomHeaders,
				Usage:       "Specify a custom header as 'Header:Value' to be set on every request to the Fleet server (can be specified multiple times for multiple headers, note that this replaces any existing custom headers). Note that when using the environment variable to set this option, it must be set like so: 'CUSTOM_HEADER=Header:Value,Header:Value', and the value cannot contain commas.",
			},
		},
		Action: func(c *cli.Context) error {
			set := false

			configPath, context := c.String("config"), c.String("context")

			if flAddress != "" {
				set = true
				if err := setConfigValue(configPath, context, "address", flAddress); err != nil {
					return fmt.Errorf("error setting address: %w", err)
				}
				fmt.Printf("[+] Set the address config key to %q in the %q context\n", flAddress, c.String("context"))
			}

			if flEmail != "" {
				set = true
				if err := setConfigValue(configPath, context, "email", flEmail); err != nil {
					return fmt.Errorf("error setting email: %w", err)
				}
				fmt.Printf("[+] Set the email config key to %q in the %q context\n", flEmail, c.String("context"))
			}

			if flToken != "" {
				set = true
				if err := setConfigValue(configPath, context, "token", flToken); err != nil {
					return fmt.Errorf("error setting token: %w", err)
				}
				fmt.Printf("[+] Set the token config key to %q in the %q context\n", flToken, c.String("context"))
			}

			if flTLSSkipVerify {
				set = true
				if err := setConfigValue(configPath, context, "tls-skip-verify", "true"); err != nil {
					return fmt.Errorf("error setting tls-skip-verify: %w", err)
				}
				fmt.Printf("[+] Set the tls-skip-verify config key to \"true\" in the %q context\n", c.String("context"))
			}

			if flRootCA != "" {
				set = true
				if err := setConfigValue(configPath, context, "rootca", flRootCA); err != nil {
					return fmt.Errorf("error setting rootca: %w", err)
				}
				fmt.Printf("[+] Set the rootca config key to %q in the %q context\n", flRootCA, c.String("context"))
			}

			if flURLPrefix != "" {
				set = true
				if err := setConfigValue(configPath, context, "url-prefix", flURLPrefix); err != nil {
					return fmt.Errorf("error setting URL Prefix: %w", err)
				}
				fmt.Printf("[+] Set the url-prefix config key to %q in the %q context\n", flURLPrefix, c.String("context"))
			}

			if len(flCustomHeaders.Value()) > 0 {
				set = true
				if err := setConfigValue(configPath, context, "custom-headers", flCustomHeaders.Value()); err != nil {
					return fmt.Errorf("error setting custom headers: %w", err)
				}
				fmt.Printf("[+] Set the custom-headers config key to %v in the %q context\n", flCustomHeaders.Value(), c.String("context"))
			}

			if !set {
				return cli.ShowCommandHelp(c, "set")
			}

			return nil
		},
	}
}

func configGetCommand() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Get a config option",
		UsageText: `fleetctl config get [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() != 1 {
				return cli.ShowCommandHelp(c, "get")
			}

			key := c.Args().Get(0)

			// validate key
			switch key {
			case "address", "email", "token", "tls-skip-verify", "rootca", "url-prefix", "custom-headers":
			default:
				return cli.ShowCommandHelp(c, "get")
			}

			configPath, context := c.String("config"), c.String("context")

			value, err := getConfigValue(configPath, context, key)
			if err != nil {
				return fmt.Errorf("error getting config value: %w", err)
			}

			fmt.Printf("  %s.%s => %v\n", c.String("context"), key, value)

			return nil
		},
	}
}
