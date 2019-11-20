package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type specMetadata struct {
	Kind    string          `json:"kind"`
	Version string          `json:"apiVersion"`
	Spec    json.RawMessage `json:"spec"`
}

type specGroup struct {
	Queries   []*kolide.QuerySpec
	Packs     []*kolide.PackSpec
	Labels    []*kolide.LabelSpec
	Options   *kolide.OptionsSpec
	AppConfig *kolide.AppConfigPayload
}

func specGroupFromBytes(b []byte) (*specGroup, error) {
	specs := &specGroup{
		Queries: []*kolide.QuerySpec{},
		Packs:   []*kolide.PackSpec{},
		Labels:  []*kolide.LabelSpec{},
	}

	for _, spec := range strings.Split(string(b), "---") {
		if strings.TrimSpace(spec) == "" {
			continue
		}

		var s specMetadata
		if err := yaml.Unmarshal([]byte(spec), &s); err != nil {
			return nil, err
		}

		if s.Spec == nil {
			return nil, errors.Errorf("no spec field on %q document", s.Kind)
		}

		switch strings.ToLower(s.Kind) {
		case "query":
			var querySpec *kolide.QuerySpec
			if err := yaml.Unmarshal(s.Spec, &querySpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling query spec")
			}
			specs.Queries = append(specs.Queries, querySpec)

		case "pack":
			var packSpec *kolide.PackSpec
			if err := yaml.Unmarshal(s.Spec, &packSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling pack spec")
			}
			specs.Packs = append(specs.Packs, packSpec)

		case "label":
			var labelSpec *kolide.LabelSpec
			if err := yaml.Unmarshal(s.Spec, &labelSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling label spec")
			}
			specs.Labels = append(specs.Labels, labelSpec)

		case "options":
			if specs.Options != nil {
				return nil, errors.New("options defined twice in the same file")
			}

			var optionSpec *kolide.OptionsSpec
			if err := yaml.Unmarshal(s.Spec, &optionSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling option spec")
			}
			specs.Options = optionSpec

		case "config":
			if specs.AppConfig != nil {
				return nil, errors.New("config defined twice in the same file")
			}

			var appConfigSpec *kolide.AppConfigPayload
			if err := yaml.Unmarshal(s.Spec, &appConfigSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling config spec")
			}
			specs.AppConfig = appConfigSpec

		default:
			return nil, errors.Errorf("unknown kind %q", s.Kind)
		}
	}

	return specs, nil
}

func applyCommand() cli.Command {
	var (
		flFilename string
		flDebug    bool
	)
	return cli.Command{
		Name:      "apply",
		Usage:     "Apply files to declaratively manage osquery configurations",
		UsageText: `fleetctl apply [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			cli.StringFlag{
				Name:        "f",
				EnvVar:      "FILENAME",
				Value:       "",
				Destination: &flFilename,
				Usage:       "A file to apply",
			},
			cli.BoolFlag{
				Name:        "debug",
				EnvVar:      "DEBUG",
				Destination: &flDebug,
				Usage:       "Whether or not to enable debug logging",
			},
		},
		Action: func(c *cli.Context) error {
			if flFilename == "" {
				return errors.New("-f must be specified")
			}

			b, err := ioutil.ReadFile(flFilename)
			if err != nil {
				return err
			}

			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			specs, err := specGroupFromBytes(b)
			if err != nil {
				return err
			}

			if len(specs.Queries) > 0 {
				if err := fleet.ApplyQueries(specs.Queries); err != nil {
					return errors.Wrap(err, "applying queries")
				}
				fmt.Printf("[+] applied %d queries\n", len(specs.Queries))
			}

			if len(specs.Labels) > 0 {
				if err := fleet.ApplyLabels(specs.Labels); err != nil {
					return errors.Wrap(err, "applying labels")
				}
				fmt.Printf("[+] applied %d labels\n", len(specs.Labels))
			}

			if len(specs.Packs) > 0 {
				if err := fleet.ApplyPacks(specs.Packs); err != nil {
					return errors.Wrap(err, "applying packs")
				}
				fmt.Printf("[+] applied %d packs\n", len(specs.Packs))
			}

			if specs.Options != nil {
				if err := fleet.ApplyOptions(specs.Options); err != nil {
					return errors.Wrap(err, "applying options")
				}
				fmt.Printf("[+] applied options\n")
			}

			if specs.AppConfig != nil {
				if err := fleet.ApplyAppConfig(specs.AppConfig); err != nil {
					return errors.Wrap(err, "applying fleet config")
				}
				fmt.Printf("[+] applied fleet config\n")

			}

			return nil
		},
	}
}
