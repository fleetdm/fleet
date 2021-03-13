package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func specGroupFromPack(name string, inputPack kolide.PermissivePackContent) (*specGroup, error) {
	specs := &specGroup{
		Queries: []*kolide.QuerySpec{},
		Packs:   []*kolide.PackSpec{},
		Labels:  []*kolide.LabelSpec{},
	}

	pack := &kolide.PackSpec{
		Name: name,
	}

	for name, query := range inputPack.Queries {
		spec := &kolide.QuerySpec{
			Name:        name,
			Description: query.Description,
			Query:       query.Query,
		}

		interval := uint(0)
		switch i := query.Interval.(type) {
		case string:
			u64, err := strconv.ParseUint(i, 10, 32)
			if err != nil {
				return nil, errors.Wrap(err, "converting interval from string to uint")
			}
			interval = uint(u64)
		case uint:
			interval = i
		case float64:
			interval = uint(i)
		}

		specs.Queries = append(specs.Queries, spec)
		pack.Queries = append(pack.Queries, kolide.PackSpecQuery{
			Name:        name,
			QueryName:   name,
			Interval:    interval,
			Description: query.Description,
			Snapshot:    query.Snapshot,
			Removed:     query.Removed,
			Shard:       query.Shard,
			Platform:    query.Platform,
			Version:     query.Version,
		})
	}

	specs.Packs = append(specs.Packs, pack)

	return specs, nil
}

func convertCommand() *cli.Command {
	var (
		flFilename string
	)
	return &cli.Command{
		Name:      "convert",
		Usage:     "Convert osquery packs into decomposed fleet configs",
		UsageText: `fleetctl convert [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			&cli.StringFlag{
				Name:        "f",
				EnvVars:     []string{"FILENAME"},
				Value:       "",
				Destination: &flFilename,
				Usage:       "A file to apply",
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

			// Remove any literal newlines (because they are not
			// valid JSON but osquery accepts them) and replace
			// with \n so that we get them in the YAML output where
			// they are allowed.
			re := regexp.MustCompile(`\s*\\\n`)
			b = re.ReplaceAll(b, []byte(`\n`))

			var specs *specGroup

			var pack kolide.PermissivePackContent
			if err := json.Unmarshal(b, &pack); err != nil {
				return err
			}

			base := filepath.Base(flFilename)
			specs, err = specGroupFromPack(strings.TrimSuffix(base, filepath.Ext(base)), pack)
			if err != nil {
				return err
			}

			if specs == nil {
				return errors.New("could not parse files")
			}

			for _, pack := range specs.Packs {
				spec, err := json.Marshal(pack)
				if err != nil {
					return err
				}

				meta := specMetadata{
					Kind:    kolide.PackKind,
					Version: kolide.ApiVersion,
					Spec:    spec,
				}

				out, err := yaml.Marshal(meta)
				if err != nil {
					return err
				}

				fmt.Println("---")
				fmt.Print(string(out))
			}

			for _, query := range specs.Queries {
				spec, err := json.Marshal(query)
				if err != nil {
					return err
				}

				meta := specMetadata{
					Kind:    kolide.QueryKind,
					Version: kolide.ApiVersion,
					Spec:    spec,
				}

				out, err := yaml.Marshal(meta)
				if err != nil {
					return err
				}

				fmt.Println("---")
				fmt.Print(string(out))
			}

			return nil
		},
	}
}
