package main

import (
	"errors"

	"github.com/urfave/cli/v2"
)

const (
	hostsFlagName       = "hosts"
	labelFlagName       = "label"
	statusFlagName      = "status"
	searchQueryFlagName = "search_query"
)

func hostsCommand() *cli.Command {
	return &cli.Command{
		Name:  "hosts",
		Usage: "Manage Fleet hosts",
		Subcommands: []*cli.Command{
			transferCommand(),
		},
	}
}

func transferCommand() *cli.Command {
	return &cli.Command{
		Name:      "transfer",
		Usage:     "Transfer one or more hosts to a team",
		UsageText: `This command will gather the set of hosts specified and transfer them to the team.`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     teamFlagName,
				Usage:    "Team name hosts will be transferred to. Use '' for No team",
				Required: true,
			},
			&cli.StringSliceFlag{
				Name:  hostsFlagName,
				Usage: "Comma-separated hostnames to transfer",
			},
			&cli.StringFlag{
				Name:  labelFlagName,
				Usage: "Label name to transfer",
			},
			&cli.StringFlag{
				Name:  statusFlagName,
				Usage: "Status to use when filtering hosts",
			},
			&cli.StringFlag{
				Name:  searchQueryFlagName,
				Usage: "A search query that returns matching hostnames to be transferred",
			},
			configFlag(),
			contextFlag(),
			yamlFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			team := c.String(teamFlagName)
			hosts := c.StringSlice(hostsFlagName)
			label := c.String(labelFlagName)
			status := c.String(statusFlagName)
			searchQuery := c.String(searchQueryFlagName)

			if hosts != nil {
				if label != "" || searchQuery != "" || status != "" {
					return errors.New("--hosts cannot be used along side any other flag")
				}
			} else {
				if label == "" && searchQuery == "" && status == "" {
					return errors.New("You need to define either --hosts, or one or more of --label, --status, --search_query")
				}
			}

			return client.TransferHosts(hosts, label, status, searchQuery, team)
		},
	}
}
