package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
	"github.com/fatih/color"
	"github.com/fleetdm/fleet/v4/pkg/rawjson"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/ghodss/yaml"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"gopkg.in/guregu/null.v3"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	yamlFlagName                = "yaml"
	jsonFlagName                = "json"
	withQueriesFlagName         = "with-queries"
	expiredFlagName             = "expired"
	includeServerConfigFlagName = "include-server-config"
)

type specGeneric struct {
	Kind    string      `json:"kind"`
	Version string      `json:"apiVersion"`
	Spec    interface{} `json:"spec"`
}

func defaultTable(writer io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(writer)
	table.SetRowLine(true)
	return table
}

func borderlessTabularTable(writer io.Writer) *tablewriter.Table {
	table := tablewriter.NewWriter(writer)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	return table
}

func yamlFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  yamlFlagName,
		Usage: "Output in yaml format",
	}
}

func jsonFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  jsonFlagName,
		Usage: "Output in JSON format",
	}
}

func printJSON(spec interface{}, writer io.Writer) error {
	b, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "%s\n", b)
	return nil
}

func printYaml(spec interface{}, writer io.Writer) error {
	b, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}
	fmt.Fprintf(writer, "---\n%s", string(b))
	return nil
}

func printLabel(c *cli.Context, label *fleet.LabelSpec) error {
	spec := specGeneric{
		Kind:    fleet.LabelKind,
		Version: fleet.ApiVersion,
		Spec:    label,
	}

	return printSpec(c, spec)
}

func printQuerySpec(c *cli.Context, query *fleet.QuerySpec) error {
	spec := specGeneric{
		Kind:    fleet.QueryKind,
		Version: fleet.ApiVersion,
		Spec:    query,
	}

	return printSpec(c, spec)
}

func printPack(c *cli.Context, pack *fleet.PackSpec) error {
	spec := specGeneric{
		Kind:    fleet.PackKind,
		Version: fleet.ApiVersion,
		Spec:    pack,
	}

	return printSpec(c, spec)
}

func printSecret(c *cli.Context, secret *fleet.EnrollSecretSpec) error {
	spec := specGeneric{
		Kind:    fleet.EnrollSecretKind,
		Version: fleet.ApiVersion,
		Spec:    secret,
	}

	return printSpec(c, spec)
}

func printHost(c *cli.Context, host *fleet.HostResponse) error {
	spec := specGeneric{
		Kind:    fleet.HostKind,
		Version: fleet.ApiVersion,
		Spec:    host,
	}

	return printSpec(c, spec)
}

func printHostDetail(c *cli.Context, host *service.HostDetailResponse) error {
	spec := specGeneric{
		Kind:    fleet.HostKind,
		Version: fleet.ApiVersion,
		Spec:    host,
	}

	return printSpec(c, spec)
}

type enrichedAppConfigPresenter fleet.EnrichedAppConfig

func (eacp enrichedAppConfigPresenter) MarshalJSON() ([]byte, error) {
	type UpdateIntervalConfigPresenter struct {
		OSQueryDetail string `json:"osquery_detail"`
		OSQueryPolicy string `json:"osquery_policy"`
		*fleet.UpdateIntervalConfig
	}

	type VulnerabilitiesConfigPresenter struct {
		Periodicity               string `json:"periodicity"`
		RecentVulnerabilityMaxAge string `json:"recent_vulnerability_max_age"`
		*fleet.VulnerabilitiesConfig
	}

	enrichedJSON, err := json.Marshal(fleet.EnrichedAppConfig(eacp))
	if err != nil {
		return nil, err
	}

	extraFieldsJSON, err := json.Marshal(&struct {
		UpdateInterval  UpdateIntervalConfigPresenter  `json:"update_interval,omitempty"`
		Vulnerabilities VulnerabilitiesConfigPresenter `json:"vulnerabilities,omitempty"`
	}{
		UpdateInterval: UpdateIntervalConfigPresenter{
			eacp.UpdateInterval.OSQueryDetail.String(),
			eacp.UpdateInterval.OSQueryPolicy.String(),
			eacp.UpdateInterval,
		},
		Vulnerabilities: VulnerabilitiesConfigPresenter{
			eacp.Vulnerabilities.Periodicity.String(),
			eacp.Vulnerabilities.RecentVulnerabilityMaxAge.String(),
			eacp.Vulnerabilities,
		},
	})
	if err != nil {
		return nil, err
	}

	// we need to marshal and combine both groups separately because
	// enrichedAppConfig has a custom marshaler.
	return rawjson.CombineRoots(enrichedJSON, extraFieldsJSON)
}

func printConfig(c *cli.Context, config interface{}) error {
	spec := specGeneric{
		Kind:    fleet.AppConfigKind,
		Version: fleet.ApiVersion,
		Spec:    config,
	}

	return printSpec(c, spec)
}

type UserRoles struct {
	Roles map[string]UserRole `json:"roles"`
}

type TeamRole struct {
	Team string `json:"team"`
	Role string `json:"role"`
}

type UserRole struct {
	GlobalRole *string    `json:"global_role"`
	Teams      []TeamRole `json:"teams"`
}

func usersToUserRoles(users []fleet.User) UserRoles {
	roles := make(map[string]UserRole)
	for _, u := range users {
		var teams []TeamRole
		for _, t := range u.Teams {
			teams = append(teams, TeamRole{
				Team: t.Name,
				Role: t.Role,
			})
		}
		roles[u.Email] = UserRole{
			GlobalRole: u.GlobalRole,
			Teams:      teams,
		}
	}
	return UserRoles{Roles: roles}
}

func printUserRoles(c *cli.Context, users []fleet.User) error {
	spec := specGeneric{
		Kind:    fleet.UserRolesKind,
		Version: fleet.ApiVersion,
		Spec:    usersToUserRoles(users),
	}

	return printSpec(c, spec)
}

func printTeams(c *cli.Context, teams []fleet.Team) error {
	for _, team := range teams {
		var teamItem interface{} = team
		if c.Bool(yamlFlagName) {
			teamSpec, err := fleet.TeamSpecFromTeam(&team)
			if err != nil {
				return err
			}
			teamItem = teamSpec
		}
		spec := specGeneric{
			Kind:    fleet.TeamKind,
			Version: fleet.ApiVersion,
			Spec: map[string]interface{}{
				"team": teamItem,
			},
		}

		if err := printSpec(c, spec); err != nil {
			return err
		}
	}
	return nil
}

func printSpec(c *cli.Context, spec specGeneric) error {
	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec, c.App.Writer)
	} else {
		err = printYaml(spec, c.App.Writer)
	}
	return err
}

func getCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "Get/list resources",
		Subcommands: []*cli.Command{
			getQueriesCommand(),
			getPacksCommand(),
			getLabelsCommand(),
			getHostsCommand(),
			getEnrollSecretCommand(),
			getAppConfigCommand(),
			getCarveCommand(),
			getCarvesCommand(),
			getUserRolesCommand(),
			getTeamsCommand(),
			getSoftwareCommand(),
			getMDMAppleCommand(),
			getMDMAppleBMCommand(),
			getMDMCommandResultsCommand(),
			getMDMCommandsCommand(),
		},
	}
}

func queryToTableRow(query fleet.Query, teamName string) []string {
	platform := "all"
	if query.Platform != "" {
		platform = query.Platform
	}

	minOsqueryVersion := "all"
	if query.MinOsqueryVersion != "" {
		minOsqueryVersion = query.MinOsqueryVersion
	}

	scheduleInfo := fmt.Sprintf("interval: %d\nplatform: %s\nmin_osquery_version: %s\nautomations_enabled: %t\nlogging: %s",
		query.Interval,
		platform,
		minOsqueryVersion,
		query.AutomationsEnabled,
		query.Logging,
	)

	teamNameOut := teamName
	if teamName == "" {
		teamNameOut = "All teams"
	}

	return []string{
		query.Name,
		query.Description,
		query.Query,
		teamNameOut,
		scheduleInfo,
	}
}

func printInheritedQueriesMsg(client *service.Client, teamID *uint) error {
	if teamID != nil {
		globalQueries, err := client.GetQueries(nil, nil)
		if err != nil {
			return fmt.Errorf("could not list global queries: %w", err)
		}

		if len(globalQueries) > 0 {
			fmt.Printf("Not showing %d inherited queries. To see global queries, run this command without the `--team` flag.\n", len(globalQueries))
		}
		return nil
	}

	return nil
}

func printNoQueriesFoundMsg(teamID *uint) {
	if teamID != nil {
		fmt.Println("No team queries found.")
		return
	}
	fmt.Println("No global queries found.")
	fmt.Println("To see team queries, run this command with the --team flag.")
}

func getQueriesCommand() *cli.Command {
	return &cli.Command{
		Name:    "queries",
		Aliases: []string{"query", "q"},
		Usage:   "List information about queries",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:  teamFlagName,
				Usage: "filter queries by team_id (0 means global)",
			},
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			name := c.Args().First()

			var teamID *uint
			var teamName string

			if tid := c.Uint(teamFlagName); tid != 0 {
				teamID = &tid
				team, err := client.GetTeam(*teamID)
				if err != nil {
					var notFoundErr service.NotFoundErr
					if errors.As(err, &notFoundErr) {
						// Do not error out, just inform the user and 'gracefully' exit.
						fmt.Println("Team not found.")
						return nil
					}
					return fmt.Errorf("get team: %w", err)
				}
				teamName = team.Name
			}

			// if name wasn't provided, list either all global queries or all team queries...
			if name == "" {
				queries, err := client.GetQueries(teamID, nil)
				if err != nil {
					return fmt.Errorf("could not list queries: %w", err)
				}

				me, err := client.Me()
				if err != nil {
					return err
				}
				if me == nil {
					return errors.New("/api/latest/fleet/me returned an empty user")
				}
				ok, err := userIsObserver(*me)
				if err != nil {
					return err
				}
				if ok {
					// Filter out queries (in-place) that a observer user
					// cannot execute (this behavior matches the UI).
					n := 0
					for _, query := range queries {
						if query.ObserverCanRun {
							queries[n] = query
							n++
						}
					}
					queries = queries[:n]
				}

				if len(queries) == 0 {
					printNoQueriesFoundMsg(teamID)
					if err := printInheritedQueriesMsg(client, teamID); err != nil {
						return err
					}
					return nil
				}

				if c.Bool(yamlFlagName) || c.Bool(jsonFlagName) {
					for _, query := range queries {
						if err := printQuerySpec(c, &fleet.QuerySpec{
							Name:        query.Name,
							Description: query.Description,
							Query:       query.Query,

							TeamName:           teamName,
							Interval:           query.Interval,
							ObserverCanRun:     query.ObserverCanRun,
							Platform:           query.Platform,
							MinOsqueryVersion:  query.MinOsqueryVersion,
							AutomationsEnabled: query.AutomationsEnabled,
							Logging:            query.Logging,
							DiscardData:        query.DiscardData,
						}); err != nil {
							return fmt.Errorf("unable to print query: %w", err)
						}
					}
				} else {
					// Default to printing as a table
					rows := [][]string{}

					columns := []string{"name", "description", "query", "team", "schedule"}
					for _, query := range queries {
						rows = append(rows, queryToTableRow(query, teamName))
					}

					printQueryTable(c, columns, rows)
					if err := printInheritedQueriesMsg(client, teamID); err != nil {
						return err
					}
				}
				return nil
			}

			query, err := client.GetQuerySpec(teamID, name)
			if err != nil {
				return err
			}

			if err := printQuerySpec(c, query); err != nil {
				return fmt.Errorf("unable to print query: %w", err)
			}

			return nil
		},
	}
}

var errUserNoRoles = errors.New("user does not have roles")

// userIsObserver returns whether the user is a global/team observer/observer+.
// In the case of user belonging to multiple teams, a user is considered observer
// if it is observer of all teams.
//
// Returns errUserNoRoles if the user does not have any roles.
func userIsObserver(user fleet.User) (bool, error) {
	if user.GlobalRole != nil {
		return *user.GlobalRole == fleet.RoleObserver || *user.GlobalRole == fleet.RoleObserverPlus, nil
	} // Team user
	if len(user.Teams) == 0 {
		return false, errUserNoRoles
	}
	for _, team := range user.Teams {
		if team.Role != fleet.RoleObserver && team.Role != fleet.RoleObserverPlus {
			return false, nil
		}
	}
	return true, nil
}

func getPacksCommand() *cli.Command {
	return &cli.Command{
		Name:    "packs",
		Aliases: []string{"pack", "p"},
		Usage:   `Retrieve 2017 "Packs" data for migration into modern osquery packs`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  withQueriesFlagName,
				Usage: "Output queries included in pack(s) too, when used alongside --yaml or --json",
			},
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			name := c.Args().First()
			shouldPrintQueries := c.Bool(withQueriesFlagName)
			queriesToPrint := make(map[string]bool)

			addQueries := func(pack *fleet.PackSpec) {
				if shouldPrintQueries {
					for _, q := range pack.Queries {
						queriesToPrint[q.QueryName] = true
					}
				}
			}

			printQueries := func() error {
				if !shouldPrintQueries {
					return nil
				}

				// Get global queries (teamID==nil), because 2017 packs reference global queries.
				queries, err := client.GetQueries(nil, nil)
				if err != nil {
					return fmt.Errorf("could not list queries: %w", err)
				}

				// Getting all queries then filtering is usually faster than getting
				// one query at a time.
				for _, query := range queries {
					if !queriesToPrint[query.Name] {
						continue
					}

					if err := printQuerySpec(c, &fleet.QuerySpec{
						Name:        query.Name,
						Description: query.Description,
						Query:       query.Query,
					}); err != nil {
						return fmt.Errorf("unable to print query: %w", err)
					}
				}

				return nil
			}

			// if name wasn't provided, list all packs
			if name == "" {
				packs, err := client.GetPacksSpecs()
				if err != nil {
					return fmt.Errorf("could not list packs: %w", err)
				}

				if c.Bool(yamlFlagName) || c.Bool(jsonFlagName) {
					for _, pack := range packs {
						if err := printPack(c, pack); err != nil {
							return fmt.Errorf("unable to print pack: %w", err)
						}
						addQueries(pack)
					}
					return printQueries()
				}

				if len(packs) == 0 {
					log(c, "No 2017 \"Packs\" found.\n")
					return nil
				}

				data := [][]string{}

				for _, pack := range packs {
					data = append(data, []string{
						pack.Name,
						pack.Platform,
						pack.Description,
						strconv.FormatBool(pack.Disabled),
					})
				}

				columns := []string{"name", "platform", "description", "disabled"}
				printTable(c, columns, data)
				log(c, fmt.Sprintf(`Found %d 2017 "Packs".

Querying in Fleet is becoming more powerful. To learn more, visit:
https://fleetdm.com/handbook/company/why-this-way#why-does-fleet-support-query-packs

To retrieve "Pack" data in a portable format for upgrading, run `+"`fleetctl upgrade-packs`"+`.
`, len(packs)))

				return nil
			}

			// Name was specified
			pack, err := client.GetPackSpec(name)
			if err != nil {
				return err
			}

			addQueries(pack)

			if err := printPack(c, pack); err != nil {
				return fmt.Errorf("unable to print pack: %w", err)
			}

			return printQueries()
		},
	}
}

func getLabelsCommand() *cli.Command {
	return &cli.Command{
		Name:    "labels",
		Aliases: []string{"label", "l"},
		Usage:   "List information about labels",
		Flags: []cli.Flag{
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			name := c.Args().First()

			// if name wasn't provided, list all labels
			if name == "" {
				labels, err := client.GetLabels()
				if err != nil {
					return fmt.Errorf("could not list labels: %w", err)
				}

				if c.Bool(yamlFlagName) || c.Bool(jsonFlagName) {
					for _, label := range labels {
						printLabel(c, label) //nolint:errcheck
					}
					return nil
				}

				if len(labels) == 0 {
					fmt.Println("No labels found")
					return nil
				}

				// Default to printing as a table
				data := [][]string{}

				for _, label := range labels {
					data = append(data, []string{
						label.Name,
						label.Platform,
						label.Description,
						label.Query,
					})
				}

				columns := []string{"name", "platform", "description", "query"}
				printTable(c, columns, data)

				return nil
			}

			// Label name was specified
			label, err := client.GetLabel(name)
			if err != nil {
				return err
			}

			printLabel(c, label) //nolint:errcheck
			return nil
		},
	}
}

func getEnrollSecretCommand() *cli.Command {
	return &cli.Command{
		Name:    "enroll_secret",
		Aliases: []string{"enroll_secrets", "enroll-secret", "enroll-secrets"},
		Usage:   "Retrieve the osquery enroll secrets",
		Flags: []cli.Flag{
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			secrets, err := client.GetEnrollSecretSpec()
			if err != nil {
				return err
			}

			err = printSecret(c, secrets)
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func getAppConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Retrieve the Fleet app configuration",
		Flags: []cli.Flag{
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.BoolFlag{
				Name:  includeServerConfigFlagName,
				Usage: "Include the server configuration in the output",
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			config, err := client.GetAppConfig()
			if err != nil {
				return err
			}

			if c.Bool(includeServerConfigFlagName) {
				err = printConfig(c, enrichedAppConfigPresenter(*config))
			} else {
				err = printConfig(c, config.AppConfig)
			}
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func getHostsCommand() *cli.Command {
	return &cli.Command{
		Name:    "hosts",
		Aliases: []string{"host", "h"},
		Usage:   "List information about hosts",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:     "team",
				Usage:    "filter hosts by team_id",
				Required: false,
			},
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.BoolFlag{
				Name:  "mdm",
				Usage: "Filters hosts by hosts that have MDM turned on in Fleet and are connected to Fleet's MDM server.",
			},
			&cli.BoolFlag{
				Name:  "mdm-pending",
				Usage: "Filters hosts by hosts ordered via Apple Business Manager (ABM). These will automatically enroll to Fleet and turn on MDM when they're unboxed.",
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			identifier := c.Args().First()

			if identifier == "" {
				query := url.Values{}
				query.Set("additional_info_filters", "*")
				if teamID := c.Uint("team"); teamID > 0 {
					query.Set("team_id", strconv.FormatUint(uint64(teamID), 10))
				}

				if c.Bool("mdm") || c.Bool("mdm-pending") {
					// print an error if MDM is not configured
					if err := client.CheckAnyMDMEnabled(); err != nil {
						return err
					}

					// --mdm and --mdm-pending are mutually exclusive, return an error if
					// both are set (one returns the enrolled hosts, the other the pending
					// to be enrolled, so it would always return an empty list).
					if c.Bool("mdm") && c.Bool("mdm-pending") {
						return errors.New("cannot use --mdm and --mdm-pending together")
					}

					if c.Bool("mdm") {
						query.Set("connected_to_fleet", "true")
					}
					if c.Bool("mdm-pending") {
						// hosts pending enrollment in Fleet's MDM server
						query.Set("mdm_name", fleet.WellKnownMDMFleet)
						query.Set("mdm_enrollment_status", string(fleet.MDMEnrollStatusPending))
					}
				}
				queryStr := query.Encode()

				hosts, err := client.GetHosts(queryStr)
				if err != nil {
					return fmt.Errorf("could not list hosts: %w", err)
				}

				if len(hosts) == 0 {
					fmt.Println("No hosts found")
					return nil
				}

				if c.Bool(jsonFlagName) || c.Bool(yamlFlagName) {
					for _, host := range hosts {
						err = printHost(c, &host)
						if err != nil {
							return err
						}
					}
					return nil
				}

				// Default to printing as table
				data := [][]string{}

				for _, host := range hosts {
					data = append(data, []string{
						host.Host.UUID,
						host.DisplayText,
						host.Host.Platform,
						host.OsqueryVersion,
						string(host.Status),
					})
				}

				columns := []string{"uuid", "hostname", "platform", "osquery_version", "status"}
				printTable(c, columns, data)
			} else {
				host, err := client.HostByIdentifier(identifier)
				if err != nil {
					return fmt.Errorf("could not get host: %w", err)
				}
				err = printHostDetail(c, host)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func getCarvesCommand() *cli.Command {
	return &cli.Command{
		Name:  "carves",
		Usage: "Retrieve the file carving sessions",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  expiredFlagName,
				Usage: "Include expired carves",
			},
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			expired := c.Bool(expiredFlagName)

			carves, err := client.ListCarves(fleet.CarveListOptions{Expired: expired})
			if err != nil {
				return err
			}

			if len(carves) == 0 {
				fmt.Println("No carves found")
				return nil
			}

			data := [][]string{}
			for _, c := range carves {
				completion := fmt.Sprintf(
					"%d%%",
					int64((float64(c.MaxBlock+1)/float64(c.BlockCount))*100),
				)
				if c.Expired {
					completion = "Expired"
				}

				errored := "no"
				if c.Error != nil {
					errored = "yes"
				}

				data = append(data, []string{
					strconv.FormatInt(c.ID, 10),
					c.CreatedAt.String(),
					c.RequestId,
					strconv.FormatInt(c.CarveSize, 10),
					completion,
					errored,
				})
			}

			columns := []string{"id", "created_at", "request_id", "carve_size", "completion", "errored"}
			printTable(c, columns, data)

			return nil
		},
	}
}

func getCarveCommand() *cli.Command {
	return &cli.Command{
		Name:  "carve",
		Usage: "Retrieve details for a carve by ID",
		Flags: []cli.Flag{
			stdoutFlag(),
			configFlag(),
			contextFlag(),
			outfileFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			idString := c.Args().First()

			if idString == "" {
				return errors.New("must provide carve ID as first argument")
			}

			id, err := strconv.ParseInt(idString, 10, 64)
			if err != nil {
				return fmt.Errorf("unable to parse carve ID as int: %w", err)
			}

			outFile := getOutfile(c)
			stdout := c.Bool(stdoutFlagName)

			if stdout && outFile != "" {
				return errors.New("-stdout and -outfile must not be specified together")
			}

			carve, err := client.GetCarve(id)
			if err != nil {
				return err
			}

			if carve.Error != nil {
				return errors.New(*carve.Error)
			}

			if stdout || outFile != "" {
				out := os.Stdout
				if outFile != "" {
					f, err := secure.OpenFile(outFile, os.O_CREATE|os.O_WRONLY, defaultFileMode)
					if err != nil {
						return fmt.Errorf("open out file: %w", err)
					}
					defer f.Close()
					out = f
				}

				reader, err := client.DownloadCarve(id)
				if err != nil {
					return err
				}

				if _, err := io.Copy(out, reader); err != nil {
					return fmt.Errorf("download carve contents: %w", err)
				}

				return nil
			}

			if err := printYaml(carve, c.App.Writer); err != nil {
				return fmt.Errorf("print carve yaml: %w", err)
			}

			return nil
		},
	}
}

func log(c *cli.Context, msg ...interface{}) {
	fmt.Fprint(c.App.Writer, msg...)
}

func getUserRolesCommand() *cli.Command {
	return &cli.Command{
		Name:    "user_roles",
		Aliases: []string{"user_role", "ur"},
		Usage:   "List global and team roles for users",
		Flags: []cli.Flag{
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			users, err := client.ListUsers()
			if err != nil {
				return fmt.Errorf("could not list users: %w", err)
			}

			if len(users) == 0 {
				log(c, "No users found")
				return nil
			}

			if c.Bool(jsonFlagName) || c.Bool(yamlFlagName) {
				err = printUserRoles(c, users)
				if err != nil {
					return err
				}
				return nil
			}

			// Default to printing as table
			data := [][]string{}

			for _, u := range users {
				data = append(data, []string{
					u.Name,
					null.StringFromPtr(u.GlobalRole).ValueOrZero(),
				})
			}
			columns := []string{"User", "Global Role"}
			printTable(c, columns, data)

			return nil
		},
	}
}

func printQueryTable(c *cli.Context, columns []string, data [][]string) {
	table := defaultTable(c.App.Writer)
	table.SetHeader(columns)
	table.SetReflowDuringAutoWrap(false)
	table.AppendBulk(data)
	table.Render()
}

func printTable(c *cli.Context, columns []string, data [][]string) {
	table := defaultTable(c.App.Writer)
	table.SetHeader(columns)
	table.AppendBulk(data)
	table.Render()
}

func printKeyValueTable(c *cli.Context, rows [][]string) {
	table := borderlessTabularTable(c.App.Writer)
	table.AppendBulk(rows)
	table.Render()
}

func printTableWithXML(c *cli.Context, columns []string, data [][]string) {
	table := defaultTable(c.App.Writer)
	table.SetHeader(columns)
	table.SetReflowDuringAutoWrap(false)
	table.SetAutoWrapText(false)
	table.AppendBulk(data)
	table.Render()
}

func getTeamsJSONFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  jsonFlagName,
		Usage: "Output all team information in JSON format",
	}
}

func getTeamsYAMLFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  yamlFlagName,
		Usage: "Output team configuration in yaml format. Intended for use with \"fleetctl apply -f\"",
	}
}

func getTeamsCommand() *cli.Command {
	return &cli.Command{
		Name:    "teams",
		Aliases: []string{"t"},
		Usage:   "List teams",
		Flags: []cli.Flag{
			getTeamsJSONFlag(),
			getTeamsYAMLFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:  nameFlagName,
				Usage: "filter by name",
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			query := url.Values{}
			if name := c.String(nameFlagName); name != "" {
				query.Set("query", name)
			}
			queryStr := query.Encode()

			teams, err := client.ListTeams(queryStr)
			if err != nil {
				return fmt.Errorf("could not list teams: %w", err)
			}

			if len(teams) == 0 {
				log(c, "No teams found")
				return nil
			}

			if c.Bool(jsonFlagName) || c.Bool(yamlFlagName) {
				err = printTeams(c, teams)
				if err != nil {
					return err
				}
				return nil
			}

			// Default to printing as table
			data := [][]string{}

			sort.Slice(teams, func(i, j int) bool {
				return teams[i].Name < teams[j].Name
			})

			for _, team := range teams {
				data = append(data, []string{
					team.Name,
					strconv.Itoa(int(team.ID)),
					fmt.Sprintf("%d", team.HostCount),
					fmt.Sprintf("%d", team.UserCount),
				})
			}
			columns := []string{"Team name", "Team ID", "Host count", "User count"}
			printTable(c, columns, data)

			return nil
		},
	}
}

func getSoftwareCommand() *cli.Command {
	return &cli.Command{
		Name:    "software",
		Aliases: []string{"s"},
		Usage:   "List software titles",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:  teamFlagName,
				Usage: "Only list software of hosts that belong to the specified team",
			},
			&cli.BoolFlag{
				Name:  "versions",
				Usage: "List all software versions",
			},
			jsonFlag(),
			yamlFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			if c.Bool(yamlFlagName) && c.Bool(jsonFlagName) {
				return errors.New("Can't specify both yaml and json flags.")
			}

			query := url.Values{}

			teamID := c.Uint(teamFlagName)
			if teamID != 0 {
				query.Set("team_id", strconv.FormatUint(uint64(teamID), 10))
			}

			if c.Bool("versions") {
				return printSoftwareVersions(c, client, query)
			}
			return printSoftwareTitles(c, client, query)
		},
	}
}

func printSoftwareVersions(c *cli.Context, client *service.Client, query url.Values) error {
	software, err := client.ListSoftwareVersions(query.Encode())
	if err != nil {
		return fmt.Errorf("could not list software versions: %w", err)
	}

	if len(software) == 0 {
		log(c, "No software versions found")
		return nil
	}

	if c.Bool(jsonFlagName) || c.Bool(yamlFlagName) {
		spec := specGeneric{
			Kind:    "software",
			Version: "1",
			Spec:    software,
		}
		err = printSpec(c, spec)
		if err != nil {
			return err
		}
		return nil
	}

	// Default to printing as table
	data := [][]string{}

	for _, s := range software {
		data = append(data, []string{
			s.Name,
			s.Version,
			s.Source,
			fmt.Sprintf("%d vulnerabilities", len(s.Vulnerabilities)),
			fmt.Sprint(s.HostsCount),
		})
	}
	columns := []string{"Name", "Version", "Type", "Vulnerabilities", "Hosts"}
	printTable(c, columns, data)
	return nil
}

func printSoftwareTitles(c *cli.Context, client *service.Client, query url.Values) error {
	software, err := client.ListSoftwareTitles(query.Encode())
	if err != nil {
		return fmt.Errorf("could not list software titles: %w", err)
	}

	if len(software) == 0 {
		log(c, "No software titles found")
		return nil
	}

	if c.Bool(jsonFlagName) || c.Bool(yamlFlagName) {
		spec := specGeneric{
			Kind:    "software_title",
			Version: "1",
			Spec:    software,
		}
		err = printSpec(c, spec)
		if err != nil {
			return err
		}
		return nil
	}

	// Default to printing as table
	data := [][]string{}

	for _, s := range software {
		vulns := make(map[string]bool)
		for _, ver := range s.Versions {
			if ver.Vulnerabilities != nil {
				for _, vuln := range *ver.Vulnerabilities {
					vulns[vuln] = true
				}
			}
		}
		data = append(data, []string{
			s.Name,
			fmt.Sprintf("%d versions", s.VersionsCount),
			s.Source,
			fmt.Sprintf("%d vulnerabilities", len(vulns)),
			fmt.Sprint(s.HostsCount),
		})
	}
	columns := []string{"Name", "Versions", "Type", "Vulnerabilities", "Hosts"}
	printTable(c, columns, data)
	return nil
}

func getMDMAppleCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-apple",
		Aliases: []string{"mdm_apple"},
		Usage:   "Show Apple Push Notification Service (APNs) information",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			const expirationWarning = 30 * 24 * time.Hour // 30 days

			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			mdm, err := client.GetAppleMDM()
			if err != nil {
				var nfe service.NotFoundErr
				if errors.As(err, &nfe) {
					log(c, "Error: No Apple Push Notification service (APNs) certificate found. Use `fleetctl generate mdm-apple` and then `fleet serve` with `mdm` configuration to turn on MDM features.\n")
					return nil
				}
				return fmt.Errorf("could not get Apple MDM information: %w", err)
			}

			printKeyValueTable(c, [][]string{
				{"Common name (CN):", mdm.CommonName},
				{"Serial number:", mdm.SerialNumber},
				{"Issuer:", mdm.Issuer},
				{"Renew date:", mdm.RenewDate.Format("January 2, 2006")},
			})

			warnDate := time.Now().Add(expirationWarning)
			if mdm.RenewDate.Before(time.Now()) {
				// certificate is expired, print an error
				color.New(color.FgRed).Fprintln(c.App.Writer, "\nERROR: Your Apple Push Notification service (APNs) certificate is expired. MDM features are turned off. To renew your APNs certificate, follow these instructions: https://fleetdm.com/learn-more-about/renew-apns")
			} else if mdm.RenewDate.Before(warnDate) {
				// certificate will soon expire, print a warning
				color.New(color.FgYellow).Fprintln(c.App.Writer, "\nWARNING: Your Apple Push Notification service (APNs) certificate is less than 30 days from expiration. If it expires, MDM features will be turned off. To renew your APNs certificate, follow these instructions: https://fleetdm.com/learn-more-about/renew-apns")
			}

			return nil
		},
	}
}

func getMDMAppleBMCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-apple-bm",
		Aliases: []string{"mdm_apple_bm"},
		Usage:   "Show information about Apple Business Manager for automatic enrollment",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			const expirationWarning = 30 * 24 * time.Hour // 30 days

			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			bm, err := client.GetAppleBM()
			if err != nil {
				var nfe service.NotFoundErr
				if errors.As(err, &nfe) {
					log(c, "Error: No Apple Business Manager server token found. Use `fleetctl generate mdm-apple-bm` and then `fleet serve` with `mdm` configuration to automatically enroll macOS hosts to Fleet.\n")
					return nil
				}
				return fmt.Errorf("could not get Apple BM information: %w", err)
			}

			defaultTeam := bm.DefaultTeam
			if defaultTeam == "" {
				defaultTeam = "No team"
			}
			printKeyValueTable(c, [][]string{
				{"Apple ID:", bm.AppleID},
				{"Organization name:", bm.OrgName},
				{"MDM server URL:", bm.MDMServerURL},
				{"Renew date:", bm.RenewDate.Format("January 2, 2006")},
				{"Default team:", defaultTeam},
			})

			warnDate := time.Now().Add(expirationWarning)
			if bm.RenewDate.Before(time.Now()) {
				// certificate is expired, print an error
				color.New(color.FgRed).Fprintln(c.App.Writer, "\nERROR: Your Apple Business Manager (ABM) server token is expired. Laptops newly purchased via ABM will not automatically enroll in Fleet. To renew your ABM server token, follow these instructions: https://fleetdm.com/docs/using-fleet/faq#how-can-i-renew-my-apple-business-manager-server-token")
			} else if bm.RenewDate.Before(warnDate) {
				// certificate will soon expire, print a warning
				color.New(color.FgYellow).Fprintln(c.App.Writer, "\nWARNING: Your Apple Business Manager (ABM) server token is less than 30 days from expiration. If it expires, laptops newly purchased via ABM will not automatically enroll in Fleet. To renew your ABM server token, follow these instructions: https://fleetdm.com/docs/using-fleet/faq#how-can-i-renew-my-apple-business-manager-server-token")
			}

			return nil
		},
	}
}

const useHighPerformanceRenderer = false

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type vpmodel struct {
	content  string
	ready    bool
	viewport viewport.Model
}

func (m vpmodel) Init() tea.Cmd {
	return nil
}

func (m vpmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this program is using the full size of the viewport we
			// need to wait until we've received the window dimensions before
			// we can initialize the viewport. The initial dimensions come in
			// quickly, though asynchronously, which is why we wait for them
			// here.
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.HighPerformanceRendering = useHighPerformanceRenderer
			m.viewport.SetContent(m.content)
			m.ready = true

			// This is only necessary for high performance rendering, which in
			// most cases you won't need.
			//
			// Render the viewport one line below the header.
			m.viewport.YPosition = headerHeight + 1
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		if useHighPerformanceRenderer {
			// Render (or re-render) the whole viewport. Necessary both to
			// initialize the viewport and when the window is resized.
			//
			// This is needed for high-performance rendering only.
			cmds = append(cmds, viewport.Sync(m.viewport))
		}
	}

	// Handle keyboard and mouse events in the viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m vpmodel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}
	return fmt.Sprintf("%s\n%s\n%s", m.headerView(), m.viewport.View(), m.footerView())
}

func (m vpmodel) headerView() string {
	title := titleStyle.Render("Mr. Pager")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(title)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line)
}

func (m vpmodel) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)))
	return lipgloss.JoinHorizontal(lipgloss.Center, line, info)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type model struct {
	table     table.Model
	currIndex int
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			return m, tea.Batch(
				tea.Printf("Result:\n%s\n", m.table.SelectedRow()[len(m.table.Columns())-1]),
			)
		case tea.KeyLeft.String():
			m.currIndex--
			if m.currIndex < 0 {
				m.currIndex = 0
			}
		case "p":
			return m, tea.Batch(
				tea.Printf("Command payload: \n%s\n", m.table.SelectedRow()[len(m.table.Columns())-2]),
			)

		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func printTableWithBubbleTea(c *cli.Context, columns []string, data [][]string) {
	var btCols []table.Column
	for _, c := range columns {
		btCols = append(btCols, table.Column{Title: c, Width: len(c) + 20})
	}

	var btRows []table.Row
	for _, d := range data {
		btRows = append(btRows, d)
	}

	t := table.New(
		table.WithColumns(btCols),
		table.WithRows(btRows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	m := model{t, 0}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func getMDMCommandResultsCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-command-results",
		Aliases: []string{"mdm_command_results"},
		Usage:   "Retrieve results for a specific MDM command.",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
			&cli.StringFlag{
				Name:     "id",
				Usage:    "Filter MDM commands by ID.",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			// print an error if MDM is not configured
			if err := client.CheckAnyMDMEnabled(); err != nil {
				return err
			}

			res, err := client.MDMGetCommandResults(c.String("id"))
			if err != nil {
				var nfe service.NotFoundErr
				if errors.As(err, &nfe) {
					return errors.New("The command doesn't exist. Please provide a valid command ID. To see a list of commands that were run, run `fleetctl get mdm-commands`.")
				}

				var sce kithttp.StatusCoder
				if errors.As(err, &sce) {
					if sce.StatusCode() == http.StatusForbidden {
						return fmt.Errorf("Permission denied. You don't have permission to view the results of this MDM command for at least one of the hosts: %w", err)
					}
				}
				return err
			}

			// print the results as a table
			data := [][]string{}
			for _, r := range res {
				formattedResult, err := formatXML(r.Result)
				// if we get an error, just log it and use the
				// unformatted command
				if err != nil {
					if getDebug(c) {
						log(c, fmt.Sprintf("error formatting command result: %s\n", err))
					}
					formattedResult = r.Result
				}
				formattedPayload, err := formatXML(r.Payload)
				// if we get an error, just log it and use the
				// unformatted payload
				if err != nil {
					if getDebug(c) {
						log(c, fmt.Sprintf("error formatting command payload: %s\n", err))
					}
					formattedPayload = r.Payload
				}
				reqType := r.RequestType
				if len(reqType) == 0 {
					reqType = "InstallProfile"
				}
				data = append(data, []string{
					r.CommandUUID,
					r.UpdatedAt.Format(time.RFC3339),
					reqType,
					r.Status,
					r.Hostname,
					string(formattedPayload),
					string(formattedResult),
				})
			}
			columns := []string{"ID", "TIME", "TYPE", "STATUS", "HOSTNAME", "PAYLOAD", "RESULTS"}
			printTableWithBubbleTea(c, columns, data)

			return nil
		},
	}
}

func getMDMCommandsCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm-commands",
		Aliases: []string{"mdm_commands"},
		Usage:   "List information about MDM commands that were run.",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
			byHostIdentifier(),
			byMDMCommandRequestType(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			// print an error if MDM is not configured
			if err := client.CheckAnyMDMEnabled(); err != nil {
				return err
			}

			opts := fleet.MDMCommandListOptions{
				Filters: fleet.MDMCommandFilters{
					HostIdentifier: c.String("host"),
					RequestType:    c.String("type"),
				},
			}

			results, err := client.MDMListCommands(opts)
			if err != nil {
				if strings.Contains(err.Error(), fleet.HostIdentiferNotFound) {
					return errors.New(fleet.HostIdentiferNotFound)
				}
				return err
			}
			if len(results) == 0 && opts.Filters.HostIdentifier == "" && opts.Filters.RequestType == "" {
				log(c, "You haven't run any MDM commands. Run MDM commands with the `fleetctl mdm run-command` command.\n")
				return nil
			}

			// print the results as a table
			data := [][]string{}
			for _, r := range results {
				reqType := r.RequestType
				if len(reqType) == 0 {
					reqType = "InstallProfile"
				}
				data = append(data, []string{
					r.CommandUUID,
					r.UpdatedAt.Format(time.RFC3339),
					reqType,
					r.Status,
					r.Hostname,
				})
			}
			columns := []string{"UUID", "TIME", "TYPE", "STATUS", "HOSTNAME"}
			fmt.Fprintf(c.App.Writer, "\nThe list of %d most recent commands:\n\n", len(results))
			printTable(c, columns, data)

			return nil
		},
	}
}

func formatXML(in []byte) ([]byte, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(in); err != nil {
		return nil, err
	}
	doc.Indent(2)
	return doc.WriteToBytes()
}
