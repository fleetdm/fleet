package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"gopkg.in/guregu/null.v3"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/ghodss/yaml"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
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

func printQuery(c *cli.Context, query *fleet.QuerySpec) error {
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

	return json.Marshal(&struct {
		fleet.EnrichedAppConfig
		UpdateInterval  UpdateIntervalConfigPresenter  `json:"update_interval,omitempty"`
		Vulnerabilities VulnerabilitiesConfigPresenter `json:"vulnerabilities,omitempty"`
	}{
		EnrichedAppConfig: fleet.EnrichedAppConfig(eacp),
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
		},
	}
}

func getQueriesCommand() *cli.Command {
	return &cli.Command{
		Name:    "queries",
		Aliases: []string{"query", "q"},
		Usage:   "List information about one or more queries",
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

			// if name wasn't provided, list all queries
			if name == "" {
				queries, err := client.GetQueries()
				if err != nil {
					return fmt.Errorf("could not list queries: %w", err)
				}

				if len(queries) == 0 {
					fmt.Println("No queries found")
					return nil
				}

				if c.Bool(yamlFlagName) || c.Bool(jsonFlagName) {
					for _, query := range queries {
						if err := printQuery(c, query); err != nil {
							return fmt.Errorf("unable to print query: %w", err)
						}
					}
				} else {
					// Default to printing as a table
					data := [][]string{}

					for _, query := range queries {
						data = append(data, []string{
							query.Name,
							query.Description,
							query.Query,
						})
					}

					columns := []string{"name", "description", "query"}
					printTable(c, columns, data)
				}
				return nil
			}

			query, err := client.GetQuery(name)
			if err != nil {
				return err
			}

			if err := printQuery(c, query); err != nil {
				return fmt.Errorf("unable to print query: %w", err)
			}

			return nil
		},
	}
}

func getPacksCommand() *cli.Command {
	return &cli.Command{
		Name:    "packs",
		Aliases: []string{"pack", "p"},
		Usage:   "List information about one or more packs",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  withQueriesFlagName,
				Usage: "Output queries included in pack(s) too",
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

				queries, err := client.GetQueries()
				if err != nil {
					return fmt.Errorf("could not list queries: %w", err)
				}

				// Getting all queries then filtering is usually faster than getting
				// one query at a time.
				for _, query := range queries {
					if !queriesToPrint[query.Name] {
						continue
					}

					if err := printQuery(c, query); err != nil {
						return fmt.Errorf("unable to print query: %w", err)
					}
				}

				return nil
			}

			// if name wasn't provided, list all packs
			if name == "" {
				packs, err := client.GetPacks()
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
					fmt.Println("No packs found")
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

				return nil
			}

			// Name was specified
			pack, err := client.GetPack(name)
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
		Usage:   "List information about one or more labels",
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
		Usage:   "List information about one or more hosts",
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
					appCfg, err := client.GetAppConfig()
					if err != nil {
						return err
					}
					if !appCfg.MDM.EnabledAndConfigured {
						return errors.New("MDM features aren't turned on. Use `fleetctl generate mdm-apple` and then `fleet serve` with `mdm` configuration to turn on MDM features.")
					}

					// --mdm and --mdm-pending are mutually exclusive, return an error if
					// both are set (one returns the enrolled hosts, the other the pending
					// to be enrolled, so it would always return an empty list).
					if c.Bool("mdm") && c.Bool("mdm-pending") {
						return errors.New("cannot use --mdm and --mdm-pending together")
					}

					if c.Bool("mdm") {
						// hosts enrolled (automatic or manual) in Fleet's MDM server
						query.Set("mdm_name", fleet.WellKnownMDMFleet)
						query.Set("mdm_enrollment_status", string(fleet.MDMEnrollStatusEnrolled))
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

			for _, team := range teams {
				data = append(data, []string{
					team.Name,
					fmt.Sprintf("%d", team.HostCount),
					fmt.Sprintf("%d", team.UserCount),
				})
			}
			columns := []string{"Team name", "Host count", "User count"}
			printTable(c, columns, data)

			return nil
		},
	}
}

func getSoftwareCommand() *cli.Command {
	return &cli.Command{
		Name:    "software",
		Aliases: []string{"s"},
		Usage:   "List software",
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:  teamFlagName,
				Usage: "Only list software of hosts that belong to the specified team",
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

			software, err := client.ListSoftware(query.Encode())
			if err != nil {
				return fmt.Errorf("could not list software: %w", err)
			}

			if len(software) == 0 {
				log(c, "No software found")
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
					s.GenerateCPE,
					fmt.Sprint(len(s.Vulnerabilities)),
				})
			}
			columns := []string{"Name", "Version", "Source", "CPE", "# of CVEs"}
			printTable(c, columns, data)

			return nil
		},
	}
}

func getMDMAppleCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm_apple",
		Hidden:  true, // TODO: temporary, until the MDM feature is officially released
		Aliases: []string{"mdm-apple"},
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
				color.New(color.FgRed).Fprintln(c.App.Writer, "\nERROR: Your Apple Push Notification service (APNs) certificate is expired. MDM features are turned off. To renew your APNs certificate, follow these instructions: https://fleetdm.com/docs/using-fleet/mdm-setup#apple-push-notification-service-apns")
			} else if mdm.RenewDate.Before(warnDate) {
				// certificate will soon expire, print a warning
				color.New(color.FgYellow).Fprintln(c.App.Writer, "\nWARNING: Your Apple Push Notification service (APNs) certificate is less than 30 days from expiration. If it expires, MDM features will be turned off. To renew your APNs certificate, follow these instructions: https://fleetdm.com/docs/using-fleet/mdm-setup#renewing-apns")
			}

			return nil
		},
	}
}

func getMDMAppleBMCommand() *cli.Command {
	return &cli.Command{
		Name:    "mdm_apple_bm",
		Hidden:  true, // TODO: temporary, until the MDM feature is officially released
		Aliases: []string{"mdm-apple-bm"},
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
