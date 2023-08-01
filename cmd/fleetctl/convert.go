package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/urfave/cli/v2"
)

func specGroupFromPack(name string, inputPack fleet.PermissivePackContent) (*spec.Group, error) {
	specs := &spec.Group{
		Queries: []*fleet.QuerySpec{},
	}

	// this ensures order is consistent in output
	keys := make([]string, len(inputPack.Queries))
	i := 0
	for k := range inputPack.Queries {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	for _, name := range keys {
		query := inputPack.Queries[name]

		// get the interval as uint from a variety of possible types
		interval := uint(0)
		switch i := query.Interval.(type) {
		case string:
			u64, err := strconv.ParseUint(i, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("converting interval from string to uint: %w", err)
			}
			interval = uint(u64)
		case uint:
			interval = i
		case float64:
			interval = uint(i)
		}

		spec := &fleet.QuerySpec{
			Name:        name,
			Description: query.Description,
			Query:       query.Query,
			Interval:    interval,
		}

		specs.Queries = append(specs.Queries, spec)
	}

	return specs, nil
}

func convertCommand() *cli.Command {
	var (
		flFilename     string
		outputFilename string
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
			&cli.StringFlag{
				Name:        "o",
				EnvVars:     []string{"OUTPUT_FILENAME"},
				Value:       "",
				Destination: &outputFilename,
				Usage:       "The name of the file to output converted results",
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

			var specs *spec.Group

			var pack fleet.PermissivePackContent
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

			var w io.Writer = os.Stdout
			if outputFilename != "" {
				file, err := os.Create(outputFilename)
				if err != nil {
					return err
				}
				defer file.Close()
				w = file
			}

			for _, query := range specs.Queries {
				specBytes, err := json.Marshal(query)
				if err != nil {
					return err
				}

				meta := spec.Metadata{
					Kind:    fleet.QueryKind,
					Version: fleet.ApiVersion,
					Spec:    specBytes,
				}

				out, err := yaml.Marshal(meta)
				if err != nil {
					return err
				}

				fmt.Fprintln(w, "---")
				fmt.Fprint(w, string(out))
			}

			return nil
		},
	}
}
