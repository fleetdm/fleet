package main

import (
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/kolide/fleet/server/kolide"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
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

func getQueriesCommand() cli.Command {
	return cli.Command{
		Name:    "queries",
		Aliases: []string{"query", "q"},
		Usage:   "List information about one or more queries",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			name := c.Args().First()

			// if name wasn't provided, list all queries
			if name == "" {
				queries, err := fleet.GetQueries()
				if err != nil {
					return errors.Wrap(err, "could not list queries")
				}

				if len(queries) == 0 {
					fmt.Println("no queries found")
					return nil
				}

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

				return nil
			} else {
				query, err := fleet.GetQuery(name)
				if err != nil {
					return err
				}

				spec := specGeneric{
					Kind:    "query",
					Version: kolide.ApiVersion,
					Spec:    query,
				}

				b, err := yaml.Marshal(spec)
				if err != nil {
					return err
				}

				fmt.Print(string(b))
				return nil
			}
		},
	}
}

func getPacksCommand() cli.Command {
	return cli.Command{
		Name:    "packs",
		Aliases: []string{"pack", "p"},
		Usage:   "List information about one or more packs",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			name := c.Args().First()

			// if name wasn't provided, list all packs
			if name == "" {
				packs, err := fleet.GetPacks()
				if err != nil {
					return errors.Wrap(err, "could not list packs")
				}

				if len(packs) == 0 {
					fmt.Println("no packs found")
					return nil
				}

				data := [][]string{}

				for _, pack := range packs {
					data = append(data, []string{
						pack.Name,
						pack.Platform,
						pack.Description,
					})
				}

				table := defaultTable()
				table.SetHeader([]string{"name", "platform", "description"})
				table.AppendBulk(data)
				table.Render()

				return nil
			} else {
				pack, err := fleet.GetPack(name)
				if err != nil {
					return err
				}

				spec := specGeneric{
					Kind:    "pack",
					Version: kolide.ApiVersion,
					Spec:    pack,
				}

				b, err := yaml.Marshal(spec)
				if err != nil {
					return err
				}

				fmt.Print(string(b))
				return nil
			}
		},
	}
}

func getLabelsCommand() cli.Command {
	return cli.Command{
		Name:    "labels",
		Aliases: []string{"label", "l"},
		Usage:   "List information about one or more labels",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			name := c.Args().First()

			// if name wasn't provided, list all labels
			if name == "" {
				labels, err := fleet.GetLabels()
				if err != nil {
					return errors.Wrap(err, "could not list labels")
				}

				if len(labels) == 0 {
					fmt.Println("no labels found")
					return nil
				}

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
			} else {
				label, err := fleet.GetLabel(name)
				if err != nil {
					return err
				}

				spec := specGeneric{
					Kind:    "label",
					Version: kolide.ApiVersion,
					Spec:    label,
				}

				b, err := yaml.Marshal(spec)
				if err != nil {
					return err
				}

				fmt.Print(string(b))

				return nil
			}
		},
	}
}

func getOptionsCommand() cli.Command {
	return cli.Command{
		Name:  "options",
		Usage: "Retrieve the osquery configuration",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			options, err := fleet.GetOptions()
			if err != nil {
				return err
			}

			spec := specGeneric{
				Kind:    "options",
				Version: kolide.ApiVersion,
				Spec:    options,
			}

			b, err := yaml.Marshal(spec)
			if err != nil {
				return err
			}

			fmt.Print(string(b))
			return nil
		},
	}
}

func getEnrollSecretCommand() cli.Command {
	return cli.Command{
		Name:  "enroll-secret",
		Usage: "Retrieve the osquery enroll secret",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			settings, err := fleet.GetServerSettings()
			if err != nil {
				return err
			}
			if settings == nil {
				return errors.New("error: server setting were nil")
			}

			fmt.Println(*settings.EnrollSecret)

			return nil
		},
	}
}

func getHostsCommand() cli.Command {
	return cli.Command{
		Name:    "hosts",
		Aliases: []string{"host", "h"},
		Usage:   "List information about one or more hosts",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			hosts, err := fleet.GetHosts()
			if err != nil {
				return errors.Wrap(err, "could not list hosts")
			}

			if len(hosts) == 0 {
				fmt.Println("no hosts found")
				return nil
			}

			data := [][]string{}

			for _, host := range hosts {
				data = append(data, []string{
					host.Host.UUID,
					host.DisplayText,
					host.Host.Platform,
					host.Status,
				})
			}

			table := defaultTable()
			table.SetHeader([]string{"uuid", "hostname", "platform", "status"})
			table.AppendBulk(data)
			table.Render()

			return nil
		},
	}
}
