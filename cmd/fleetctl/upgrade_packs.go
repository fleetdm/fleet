package main

import (
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
)

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
			queries, err := client.GetQueries(nil)
			if err != nil {
				return fmt.Errorf("could not list queries: %w", err)
			}

			// map queries by packs that reference them
			queriesByPack := mapQueriesToPacks(packsSpecs, queries)

			var newSpecs []*fleet.QuerySpec
			for _, packSpec := range packsSpecs {
				newSpecs = append(newSpecs,
					upgradePackToQueriesSpecs(packSpec, packsByID[packSpec.ID], queriesByPack[packSpec])...)
			}

			log(c, fmt.Sprintf(`Converted %d queries from %d 2017 "Packs" into portable queries:
• For any "Packs" targeting teams, duplicate queries were written for each team.
• For any "Packs" targeting labels or individual hosts, a global query was written without scheduling features enabled.

To import these queries to Fleet, you can merge the data in the output file with your existing query configuration and run `+"`fleetctl apply`"+`.

Note that existing 2017 "Packs" have been left intact. To avoid running duplicate queries on your hosts, visit %s/packs/manage and disable all 2017 "Packs" after upgrading. Fleet will continue to support these until the next major version release, when 2017 "Packs" will be automatically converted to queries.
`, len(newSpecs), len(packsSpecs), appCfg.ServerSettings.ServerURL))

			return nil
		},
	}
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

func upgradePackToQueriesSpecs(packSpec *fleet.PackSpec, packDB *fleet.Pack, packQueries []*fleet.Query) []*fleet.QuerySpec {
	if len(packQueries) == 0 {
		// if the pack has no query, there's nothing to convert
		return nil
	}

	var targetsHosts bool
	if packDB != nil {
		targetsHosts = len(packDB.HostIDs) > 0
	}

	schedByName := make(map[string]*fleet.PackSpecQuery, len(packSpec.Queries))
	for _, sq := range packSpec.Queries {
		sq := sq // avoid taking the address of iteration var
		schedByName[sq.QueryName] = &sq
	}

	var newSpecs []*fleet.QuerySpec
	for _, pq := range packQueries {
		sched := schedByName[pq.Name]

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
		)
		if sched.Platform != nil {
			schedPlatform = *sched.Platform
		}
		if sched.Version != nil {
			schedVersion = *sched.Version
		}

		for _, tm := range packSpec.Targets.Teams {
			// duplicate the query for each targeted team
			newSpecs = append(newSpecs, &fleet.QuerySpec{
				Name:               pq.Name, // TODO(mna): unique name as we did in migration
				Description:        pq.Description + fmt.Sprintf("\n\n(Converted from pack %q.)", packSpec.Name),
				Query:              pq.Query,
				TeamName:           tm,
				Interval:           sched.Interval,
				ObserverCanRun:     pq.ObserverCanRun,
				Platform:           schedPlatform,
				MinOsqueryVersion:  schedVersion,
				AutomationsEnabled: !packSpec.Disabled, // TODO(mna): confirm this
				Logging:            loggingType,
			})
		}

		if len(packSpec.Targets.Labels) > 0 || targetsHosts {
			// write a global query without scheduling features
			newSpecs = append(newSpecs, &fleet.QuerySpec{
				Name:               pq.Name, // TODO(mna): unique name as we did in migration
				Description:        pq.Description + fmt.Sprintf("\n\n(Converted from pack %q.)", packSpec.Name),
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
	}

	return newSpecs
}
