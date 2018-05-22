package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
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

func convertCommand() cli.Command {
	var (
		flFilename string
		flDebug    bool
	)
	return cli.Command{
		Name:      "convert",
		Usage:     "Convert osquery packs into decomposed fleet configs",
		UsageText: `fleetctl convert [options]`,
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

			var specs *specGroup

			var pack kolide.PermissivePackContent
			packErr := json.Unmarshal(b, &pack)
			if packErr == nil {
				base := filepath.Base(flFilename)
				specs, err = specGroupFromPack(strings.TrimSuffix(base, filepath.Ext(base)), pack)
				if err != nil {
					return err
				}
			} else {
				return packErr
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
					Kind:    "pack",
					Version: "v1",
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
					Kind:    "query",
					Version: "v1",
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
