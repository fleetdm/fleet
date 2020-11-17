package main

import "github.com/urfave/cli"

const (
	outfileFlagName = "outfile"
)

func outfileFlag() cli.Flag {
	return cli.StringFlag{
		Name:   outfileFlagName,
		Value:  "",
		EnvVar: "OUTFILE",
		Usage:  "Path to output file",
	}
}

func getOutfile(c *cli.Context) string {
	return c.String(outfileFlagName)
}
