package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/guregu/null.v3"
	"io"
	"os"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/ghodss/yaml"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

const (
	yamlFlagName        = "yaml"
	jsonFlagName        = "json"
	withQueriesFlagName = "with-queries"
	expiredFlagName     = "expired"
	stdoutFlagName      = "stdout"
)

type specGeneric struct {
	Kind    string      `json:"kind"`
	Version string      `json:"apiVersion"`
	Spec    interface{} `json:"spec"`
}

func defaultTable() *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetRowLine(true)
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

func printJSON(spec interface{}) error {
	b, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", b)
	return nil
}

func printYaml(spec interface{}) error {
	b, err := yaml.Marshal(spec)
	if err != nil {
		return err
	}
	fmt.Printf("---\n%s", string(b))
	return nil
}

func printLabel(c *cli.Context, label *fleet.LabelSpec) error {
	spec := specGeneric{
		Kind:    fleet.LabelKind,
		Version: fleet.ApiVersion,
		Spec:    label,
	}

	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
	}

	return err
}

func printQuery(c *cli.Context, query *fleet.QuerySpec) error {
	spec := specGeneric{
		Kind:    fleet.QueryKind,
		Version: fleet.ApiVersion,
		Spec:    query,
	}

	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
	}

	return err
}

func printPack(c *cli.Context, pack *fleet.PackSpec) error {
	spec := specGeneric{
		Kind:    fleet.PackKind,
		Version: fleet.ApiVersion,
		Spec:    pack,
	}

	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
	}

	return err
}

func printSecret(c *cli.Context, secret *fleet.EnrollSecretSpec) error {
	spec := specGeneric{
		Kind:    fleet.EnrollSecretKind,
		Version: fleet.ApiVersion,
		Spec:    secret,
	}

	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
	}

	return err
}

func printHost(c *cli.Context, host *fleet.Host) error {
	spec := specGeneric{
		Kind:    fleet.HostKind,
		Version: fleet.ApiVersion,
		Spec:    host,
	}

	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
	}

	return err
}

func printConfig(c *cli.Context, config *fleet.AppConfigPayload) error {
	spec := specGeneric{
		Kind:    fleet.AppConfigKind,
		Version: fleet.ApiVersion,
		Spec:    config,
	}
	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
	}

	return err
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
		roles[u.Name] = UserRole{
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

	var err error

	if c.Bool(jsonFlagName) {
		err = printJSON(spec)
	} else {
		err = printYaml(spec)
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
					return errors.Wrap(err, "could not list queries")
				}

				if len(queries) == 0 {
					fmt.Println("No queries found")
					return nil
				}

				if c.Bool(yamlFlagName) || c.Bool(jsonFlagName) {
					for _, query := range queries {
						if err := printQuery(c, query); err != nil {
							return errors.Wrap(err, "unable to print query")
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

					table := defaultTable()
					table.SetHeader([]string{"name", "description", "query"})
					table.AppendBulk(data)
					table.Render()
				}
				return nil
			}

			query, err := client.GetQuery(name)
			if err != nil {
				return err
			}

			if err := printQuery(c, query); err != nil {
				return errors.Wrap(err, "unable to print query")
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
					return errors.Wrap(err, "could not list queries")
				}

				// Getting all queries then filtering is usually faster than getting
				// one query at a time.
				for _, query := range queries {
					if !queriesToPrint[query.Name] {
						continue
					}

					if err := printQuery(c, query); err != nil {
						return errors.Wrap(err, "unable to print query")
					}
				}

				return nil
			}

			// if name wasn't provided, list all packs
			if name == "" {
				packs, err := client.GetPacks()
				if err != nil {
					return errors.Wrap(err, "could not list packs")
				}

				if c.Bool(yamlFlagName) {
					for _, pack := range packs {
						if err := printPack(c, pack); err != nil {
							return errors.Wrap(err, "unable to print pack")
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

				table := defaultTable()
				table.SetHeader([]string{"name", "platform", "description", "disabled"})
				table.AppendBulk(data)
				table.Render()

				return nil
			}

			// Name was specified
			pack, err := client.GetPack(name)
			if err != nil {
				return err
			}

			addQueries(pack)

			if err := printPack(c, pack); err != nil {
				return errors.Wrap(err, "unable to print pack")
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
					return errors.Wrap(err, "could not list labels")
				}

				if c.Bool(yamlFlagName) || c.Bool(jsonFlagName) {
					for _, label := range labels {
						printLabel(c, label)
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

				table := defaultTable()
				table.SetHeader([]string{"name", "platform", "description", "query"})
				table.AppendBulk(data)
				table.Render()

				return nil
			}

			// Label name was specified
			label, err := client.GetLabel(name)
			if err != nil {
				return err
			}

			printLabel(c, label)
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
		Usage: "Retrieve the Fleet configuration",
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

			config, err := client.GetAppConfig()
			if err != nil {
				return err
			}

			err = printConfig(c, config)
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

			identifier := c.Args().First()

			if identifier == "" {
				hosts, err := client.GetHosts()
				if err != nil {
					return errors.Wrap(err, "could not list hosts")
				}

				if len(hosts) == 0 {
					fmt.Println("No hosts found")
					return nil
				}

				if c.Bool(jsonFlagName) || c.Bool(yamlFlagName) {
					for _, host := range hosts {
						err = printHost(c, host.Host)
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

				table := defaultTable()
				table.SetHeader([]string{"uuid", "hostname", "platform", "osquery_version", "status"})
				table.AppendBulk(data)
				table.Render()
			} else {
				host, err := client.HostByIdentifier(identifier)
				if err != nil {
					return errors.Wrap(err, "could not get host")
				}
				b, err := yaml.Marshal(host)
				if err != nil {
					return err
				}

				fmt.Print(string(b))
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

				data = append(data, []string{
					strconv.FormatInt(c.ID, 10),
					c.CreatedAt.Local().String(),
					c.RequestId,
					strconv.FormatInt(c.CarveSize, 10),
					completion,
				})
			}

			table := defaultTable()
			table.SetHeader([]string{"id", "created_at", "request_id", "carve_size", "completion"})
			table.AppendBulk(data)
			table.Render()

			return nil
		},
	}
}

func getCarveCommand() *cli.Command {
	return &cli.Command{
		Name:  "carve",
		Usage: "Retrieve details for a carve by ID",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  stdoutFlagName,
				Usage: "Print carve contents to stdout",
			},
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
				return errors.Errorf("must provide carve ID as first argument")
			}

			id, err := strconv.ParseInt(idString, 10, 64)
			if err != nil {
				return errors.Wrap(err, "unable to parse carve ID as int")
			}

			outFile := getOutfile(c)
			stdout := c.Bool(stdoutFlagName)

			if stdout && outFile != "" {
				return errors.Errorf("-stdout and -outfile must not be specified together")
			}

			if stdout || outFile != "" {
				out := os.Stdout
				if outFile != "" {
					f, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY, defaultFileMode)
					if err != nil {
						return errors.Wrap(err, "open out file")
					}
					defer f.Close()
					out = f
				}

				reader, err := client.DownloadCarve(id)
				if err != nil {
					return err
				}

				if _, err := io.Copy(out, reader); err != nil {
					return errors.Wrap(err, "download carve contents")
				}

				return nil
			}

			carve, err := client.GetCarve(id)
			if err != nil {
				return err
			}

			if err := printYaml(carve); err != nil {
				return errors.Wrap(err, "print carve yaml")
			}

			return nil
		},
	}
}

func getUserRolesCommand() *cli.Command {
	return &cli.Command{
		Name:    "user_roles",
		Aliases: []string{"user_roles", "ur"},
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
				return errors.Wrap(err, "could not list users")
			}

			if len(users) == 0 {
				fmt.Println("No users found")
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

			table := defaultTable()
			table.SetHeader([]string{"User", "Global Role"})
			table.AppendBulk(data)
			table.Render()

			return nil
		},
	}
}
