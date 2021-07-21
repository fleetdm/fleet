package main

import (
	"encoding/json"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var (
	yamlSeparator = regexp.MustCompile(`(?m:^---[\t ]*)`)
)

type specMetadata struct {
	Kind    string          `json:"kind"`
	Version string          `json:"apiVersion"`
	Spec    json.RawMessage `json:"spec"`
}

type specGroup struct {
	Queries      []*fleet.QuerySpec
	Teams        []*fleet.TeamSpec
	Packs        []*fleet.PackSpec
	Labels       []*fleet.LabelSpec
	AppConfig    *fleet.AppConfigPayload
	EnrollSecret *fleet.EnrollSecretSpec
	UsersRoles   *fleet.UsersRoleSpec
}

type TeamSpec struct {
	Team *fleet.TeamSpec `json:"team"`
}

func specGroupFromBytes(b []byte) (*specGroup, error) {
	specs := &specGroup{
		Queries: []*fleet.QuerySpec{},
		Packs:   []*fleet.PackSpec{},
		Labels:  []*fleet.LabelSpec{},
	}

	for _, spec := range splitYaml(string(b)) {
		var s specMetadata
		if err := yaml.Unmarshal([]byte(spec), &s); err != nil {
			return nil, err
		}

		if s.Spec == nil {
			return nil, errors.Errorf("no spec field on %q document", s.Kind)
		}

		kind := strings.ToLower(s.Kind)

		switch kind {
		case fleet.QueryKind:
			var querySpec *fleet.QuerySpec
			if err := yaml.Unmarshal(s.Spec, &querySpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.Queries = append(specs.Queries, querySpec)

		case fleet.PackKind:
			var packSpec *fleet.PackSpec
			if err := yaml.Unmarshal(s.Spec, &packSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.Packs = append(specs.Packs, packSpec)

		case fleet.LabelKind:
			var labelSpec *fleet.LabelSpec
			if err := yaml.Unmarshal(s.Spec, &labelSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.Labels = append(specs.Labels, labelSpec)

		case fleet.AppConfigKind:
			if specs.AppConfig != nil {
				return nil, errors.New("config defined twice in the same file")
			}

			var appConfigSpec *fleet.AppConfigPayload
			if err := yaml.Unmarshal(s.Spec, &appConfigSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.AppConfig = appConfigSpec

		case fleet.EnrollSecretKind:
			if specs.AppConfig != nil {
				return nil, errors.New("enroll_secret defined twice in the same file")
			}

			var enrollSecretSpec *fleet.EnrollSecretSpec
			if err := yaml.Unmarshal(s.Spec, &enrollSecretSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.EnrollSecret = enrollSecretSpec

		case fleet.UserRolesKind:
			var userRoleSpec *fleet.UsersRoleSpec
			if err := yaml.Unmarshal(s.Spec, &userRoleSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.UsersRoles = userRoleSpec

		case fleet.TeamKind:
			var teamSpec TeamSpec
			if err := yaml.Unmarshal(s.Spec, &teamSpec); err != nil {
				return nil, errors.Wrap(err, "unmarshaling "+kind+" spec")
			}
			specs.Teams = append(specs.Teams, teamSpec.Team)

		default:
			return nil, errors.Errorf("unknown kind %q", s.Kind)
		}
	}

	return specs, nil
}

func applyCommand() *cli.Command {
	var (
		flFilename string
	)
	return &cli.Command{
		Name:      "apply",
		Usage:     "Apply files to declaratively manage osquery configurations",
		UsageText: `fleetctl apply [options]`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "f",
				EnvVars:     []string{"FILENAME"},
				Value:       "",
				Destination: &flFilename,
				Usage:       "A file to apply",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			if flFilename == "" {
				return errors.New("-f must be specified")
			}

			b, err := ioutil.ReadFile(flFilename)
			if err != nil {
				return err
			}

			fleetClient, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			specs, err := specGroupFromBytes(b)
			if err != nil {
				return err
			}

			if len(specs.Queries) > 0 {
				if err := fleetClient.ApplyQueries(specs.Queries); err != nil {
					return errors.Wrap(err, "applying queries")
				}
				logf(c, "[+] applied %d queries\n", len(specs.Queries))
			}

			if len(specs.Labels) > 0 {
				if err := fleetClient.ApplyLabels(specs.Labels); err != nil {
					return errors.Wrap(err, "applying labels")
				}
				logf(c, "[+] applied %d labels\n", len(specs.Labels))
			}

			if len(specs.Packs) > 0 {
				if err := fleetClient.ApplyPacks(specs.Packs); err != nil {
					return errors.Wrap(err, "applying packs")
				}
				logf(c, "[+] applied %d packs\n", len(specs.Packs))
			}

			if specs.AppConfig != nil {
				if err := fleetClient.ApplyAppConfig(specs.AppConfig); err != nil {
					return errors.Wrap(err, "applying fleet config")
				}
				log(c, "[+] applied fleet config\n")

			}

			if specs.EnrollSecret != nil {
				if err := fleetClient.ApplyEnrollSecretSpec(specs.EnrollSecret); err != nil {
					return errors.Wrap(err, "applying enroll secrets")
				}
				log(c, "[+] applied enroll secrets\n")
			}

			if len(specs.Teams) > 0 {
				if err := fleetClient.ApplyTeams(specs.Teams); err != nil {
					return errors.Wrap(err, "applying queries")
				}
				logf(c, "[+] applied %d teams\n", len(specs.Teams))
			}

			if specs.UsersRoles != nil {
				if err := fleetClient.ApplyUsersRoleSecretSpec(specs.UsersRoles); err != nil {
					return errors.Wrap(err, "applying user roles")
				}
				log(c, "[+] applied user roles\n")
			}

			return nil
		},
	}
}
