package main

import (
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

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
				queries, err := fleet.GetQuerySpecs()
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
				fmt.Println("[+] Getting information on a specific query is not currently supported")
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
				packs, err := fleet.GetPackSpecs()
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
				fmt.Println("[+] Getting information on a specific pack is not currently supported")
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
				labels, err := fleet.GetLabelSpecs()
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
				fmt.Println("[+] Getting information on a specific label is not currently supported")
				return nil
			}
		},
	}
}
