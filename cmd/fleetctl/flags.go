package main

import "github.com/urfave/cli"

const (
	outfileFlagName = "outfile"
	debugFlagName   = "debug"
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

func debugFlag() cli.Flag {
	return cli.BoolFlag{
		Name:   debugFlagName,
		EnvVar: "DEBUG",
		Usage:  "Enable debug http request logging",
	}
}

func getDebug(c *cli.Context) bool {
	return c.Bool(debugFlagName)
}
