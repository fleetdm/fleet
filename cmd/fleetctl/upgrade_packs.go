package main

import (
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
			_ = client
			return nil
		},
	}
}
