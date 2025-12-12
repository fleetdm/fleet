package fleetctl

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/urfave/cli/v2"
)

// variable used by tests to set a predictable timestamp
var testUpgradePacksTimestamp time.Time

func upgradePacksCommand() *cli.Command {
	var outputFilename string
	return &cli.Command{
		Name:      "upgrade-packs",
		Usage:     `Generate a config file to assist with converting 2017 "Packs" into portable queries that run on a schedule`,
		UsageText: `fleetctl upgrade-packs [options]`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:        "o",
				EnvVars:     []string{"OUTPUT_FILENAME"},
				Value:       "",
				Destination: &outputFilename,
				Usage:       "The name of the file to output converted results",
				Required:    true,
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			// must be an admin, but reading packs and queries does not require being an admin,
			// so this must be validated separately, before loading the packs.
			user, err := client.Me()
			if err != nil {
				return fmt.Errorf("check user role: %w", err)
			}
			if user.GlobalRole == nil || *user.GlobalRole != fleet.RoleAdmin {
				return errors.New("could not upgrade packs: forbidden: user does not have the admin role")
			}

			// read the server settings so that the packs URL can be printed in the output
			appCfg, err := client.GetAppConfig()
			if err != nil {
				return fmt.Errorf("could not read configuration: %w", err)
			}

			// read the packs and queries
			packsSpecs, err := client.GetPacksSpecs()
			if err != nil {
				return fmt.Errorf("could not list packs specs: %w", err)
			}
			if len(packsSpecs) == 0 {
				log(c, "No 2017 \"Packs\" found.\n")
				return nil
			}

			// must read the DB packs too (not just the specs) in order to get the
			// host targets of the packs, which are not retrieved by GetPacksSpecs
			// (because host targets cannot be set via the apply spec).
			packsDB, err := client.ListPacks()
			if err != nil {
				return fmt.Errorf("could not list packs: %w", err)
			}
			// map the DB packs by ID
			packsByID := make(map[uint]*fleet.Pack, len(packsDB))
			for _, p := range packsDB {
				packsByID[p.ID] = p
			}

			// get global queries (teamID==nil), because 2017 packs reference global queries.
			queries, err := client.GetQueries(nil, nil)
			if err != nil {
				return fmt.Errorf("could not list queries: %w", err)
			}

			// map queries by packs that reference them
			queriesByPack := mapQueriesToPacks(packsSpecs, queries)

			var (
				newSpecs         []*fleet.QuerySpec
				convertedQueries int
			)
			// use a consistent upgrade timestamp for all new queries (used to make name unique)
			upgradeTimestamp := testUpgradePacksTimestamp
			if upgradeTimestamp.IsZero() {
				upgradeTimestamp = time.Now()
			}
			for _, packSpec := range packsSpecs {
				newPackSpecs, convPackQueries := upgradePackToQueriesSpecs(packSpec, packsByID[packSpec.ID], queriesByPack[packSpec], upgradeTimestamp)
				newSpecs = append(newSpecs, newPackSpecs...)
				convertedQueries += convPackQueries
			}

			if err := writeQuerySpecsToFile(outputFilename, newSpecs); err != nil {
				return fmt.Errorf("could not write queries to file: %w", err)
			}

			log(c, fmt.Sprintf(`Converted %d queries from %d 2017 "Packs" into portable queries:
• For any "Packs" targeting teams, duplicate queries were written for each team.
• For any "Packs" targeting labels or individual hosts, a global query was written without scheduling features enabled.

To import these queries to Fleet, you can merge the data in the output file with your existing query configuration and run `+"`fleetctl apply`"+`.

Note that existing 2017 "Packs" have been left intact. To avoid running duplicate queries on your hosts, visit %s/packs/manage and disable all 2017 "Packs" after upgrading. Fleet will continue to support these until the next major version release, when 2017 "Packs" will be automatically converted to queries.
`, convertedQueries, len(packsSpecs), appCfg.ServerSettings.ServerURL))

			return nil
		},
	}
}

func writeQuerySpecsToFile(filename string, specs []*fleet.QuerySpec) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFileMode)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, spec := range specs {
		qYaml := fleet.QueryObject{
			ObjectMetadata: fleet.ObjectMetadata{
				ApiVersion: fleet.ApiVersion,
				Kind:       fleet.QueryKind,
			},
			Spec: *spec,
		}
		yml, err := yaml.Marshal(qYaml)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprint(f, string(yml)+"---\n"); err != nil {
			return err
		}
	}

	return f.Close()
}

func mapQueriesToPacks(packs []*fleet.PackSpec, queries []fleet.Query) map[*fleet.PackSpec][]*fleet.Query {
	queriesByName := make(map[string]*fleet.Query, len(queries))
	for _, q := range queries {
		q := q // avoid taking address of iteration var
		queriesByName[q.Name] = &q
	}

	queriesByPack := make(map[*fleet.PackSpec][]*fleet.Query, len(packs))
	for _, pack := range packs {
		for _, sq := range pack.Queries {
			if q := queriesByName[sq.QueryName]; q != nil {
				queriesByPack[pack] = append(queriesByPack[pack], queriesByName[sq.QueryName])
			}
		}
	}
	return queriesByPack
}

// upgrades the pack to the new query format, duplicating queries as needed
// (pack queries targeting teams are duplicated for each team, pack queries
// targeting labels or hosts are duplicated as a global query). Returns the
// generated new query specs and the number of pack queries that were
// converted.
func upgradePackToQueriesSpecs(packSpec *fleet.PackSpec, packDB *fleet.Pack, packQueries []*fleet.Query, ts time.Time) ([]*fleet.QuerySpec, int) {
	if len(packQueries) == 0 {
		// if the pack has no query, there's nothing to convert
		return nil, 0
	}

	var targetsHosts bool
	if packDB != nil {
		targetsHosts = len(packDB.Hosts) > 0
	}

	schedByName := make(map[string]*fleet.PackSpecQuery, len(packSpec.Queries))
	for _, sq := range packSpec.Queries {
		sq := sq // avoid taking the address of iteration var
		schedByName[sq.QueryName] = &sq
	}

	var (
		newSpecs         []*fleet.QuerySpec
		convertedQueries int
	)

	for _, pq := range packQueries {
		sched := schedByName[pq.Name]
		if sched == nil {
			continue
		}

		desc := pq.Description
		if desc != "" && !strings.HasSuffix(desc, "\n") {
			desc += "\n"
		}
		desc += fmt.Sprintf("(converted from pack %q, query %q)", packSpec.Name, pq.Name)

		var loggingType string
		if sched.Snapshot != nil && *sched.Snapshot {
			loggingType = "snapshot"
		} else if sched.Removed != nil {
			if *sched.Removed {
				loggingType = "differential"
			} else {
				loggingType = "differential_ignore_removals"
			}
		}

		var (
			schedPlatform string
			schedVersion  string
			converted     bool
		)
		if sched.Platform != nil {
			schedPlatform = *sched.Platform
		}
		if sched.Version != nil {
			schedVersion = *sched.Version
		}

		for _, tm := range packSpec.Targets.Teams {
			converted = true

			// duplicate the query for each targeted team
			newQueryName := fmt.Sprintf("%s - %s - %s - %s", packSpec.Name, pq.Name, tm, ts.Format("Jan _2 15:04:05.000"))
			newSpecs = append(newSpecs, &fleet.QuerySpec{
				Name:               newQueryName,
				Description:        desc,
				Query:              pq.Query,
				TeamName:           tm,
				Interval:           sched.Interval,
				ObserverCanRun:     pq.ObserverCanRun,
				Platform:           schedPlatform,
				MinOsqueryVersion:  schedVersion,
				AutomationsEnabled: !packSpec.Disabled,
				Logging:            loggingType,
			})
		}

		if len(packSpec.Targets.Labels) > 0 || targetsHosts {
			converted = true

			// write a global query without scheduling features
			newQueryName := fmt.Sprintf("%s - %s - %s", packSpec.Name, pq.Name, ts.Format("Jan _2 15:04:05.000"))
			newSpecs = append(newSpecs, &fleet.QuerySpec{
				Name:               newQueryName,
				Description:        desc,
				Query:              pq.Query,
				TeamName:           "",
				Interval:           0,
				ObserverCanRun:     pq.ObserverCanRun,
				Platform:           schedPlatform,
				MinOsqueryVersion:  schedVersion,
				AutomationsEnabled: false,
				Logging:            loggingType,
			})
		}

		if converted {
			convertedQueries++
		}
	}

	return newSpecs, convertedQueries
}
