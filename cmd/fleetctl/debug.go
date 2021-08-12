package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/fleetdm/fleet/v4/secure"
)

func debugCommand() *cli.Command {
	return &cli.Command{
		Name:  "debug",
		Usage: "Tools for debugging Fleet",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Subcommands: []*cli.Command{
			debugProfileCommand(),
			debugCmdlineCommand(),
			debugHeapCommand(),
			debugGoroutineCommand(),
			debugTraceCommand(),
			debugArchiveCommand(),
		},
	}
}

func writeFile(filename string, bytes []byte, mode os.FileMode) error {
	if err := ioutil.WriteFile(filename, bytes, mode); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Output written to %s\n", filename)
	return nil
}

func outfileName(name string) string {
	return fmt.Sprintf("fleet-%s-%s", name, time.Now().Format(time.RFC3339))
}

func debugProfileCommand() *cli.Command {
	return &cli.Command{
		Name:      "profile",
		Usage:     "Record a CPU profile from the Fleet server.",
		UsageText: "Record a 30-second CPU profile. The output can be analyzed with go tool pprof.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			profile, err := fleet.DebugPprof("profile")
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName("profile")
			}

			if err := writeFile(outfile, profile, defaultFileMode); err != nil {
				return errors.Wrap(err, "write profile to file")
			}

			return nil
		},
	}
}

func joinCmdline(cmdline string) string {
	var tokens []string
	for _, token := range strings.Split(string(cmdline), "\x00") {
		tokens = append(tokens, fmt.Sprintf("'%s'", token))
	}
	return fmt.Sprintf("[%s]", strings.Join(tokens, ", "))
}

func debugCmdlineCommand() *cli.Command {
	return &cli.Command{
		Name:  "cmdline",
		Usage: "Get the command line used to invoke the Fleet server.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			cmdline, err := fleet.DebugPprof("cmdline")
			if err != nil {
				return err
			}

			out := joinCmdline(string(cmdline))

			if outfile := getOutfile(c); outfile != "" {
				if err := writeFile(outfile, []byte(out), defaultFileMode); err != nil {
					return errors.Wrap(err, "write cmdline to file")
				}
				return nil
			}

			fmt.Println(out)

			return nil
		},
	}
}

func debugHeapCommand() *cli.Command {
	name := "heap"
	return &cli.Command{
		Name:      name,
		Usage:     "Report the allocated memory in the Fleet server.",
		UsageText: "Report the heap-allocated memory. The output can be analyzed with go tool pprof.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			profile, err := fleet.DebugPprof(name)
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName(name)
			}

			if err := writeFile(outfile, profile, defaultFileMode); err != nil {
				return errors.Wrapf(err, "write %s to file", name)
			}

			return nil
		},
	}
}

func debugGoroutineCommand() *cli.Command {
	name := "goroutine"
	return &cli.Command{
		Name:      name,
		Usage:     "Get stack traces of all goroutines (threads) in the Fleet server.",
		UsageText: "Get stack traces of all current goroutines (threads). The output can be analyzed with go tool pprof.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			profile, err := fleet.DebugPprof(name)
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName(name)
			}

			if err := writeFile(outfile, profile, defaultFileMode); err != nil {
				return errors.Wrapf(err, "write %s to file", name)
			}

			return nil
		},
	}
}

func debugTraceCommand() *cli.Command {
	name := "trace"
	return &cli.Command{
		Name:      name,
		Usage:     "Record an execution trace on the Fleet server.",
		UsageText: "Record a 1 second execution trace. The output can be analyzed with go tool trace.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			profile, err := fleet.DebugPprof(name)
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName(name)
			}

			if err := writeFile(outfile, profile, defaultFileMode); err != nil {
				return errors.Wrapf(err, "write %s to file", name)
			}

			return nil
		},
	}
}

func debugArchiveCommand() *cli.Command {
	return &cli.Command{
		Name:  "archive",
		Usage: "Create an archive with the entire suite of debug profiles.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			profiles := []string{
				"allocs",
				"block",
				"cmdline",
				"goroutine",
				"heap",
				"mutex",
				"profile",
				"threadcreate",
				"trace",
			}

			outpath := getOutfile(c)
			if outpath == "" {
				outpath = outfileName("profiles-archive")
			}
			outfile := outpath + ".tar.gz"

			f, err := secure.OpenFile(outfile, os.O_CREATE|os.O_WRONLY, defaultFileMode)
			if err != nil {
				return errors.Wrap(err, "open archive for output")
			}
			defer f.Close()
			gzwriter := gzip.NewWriter(f)
			defer gzwriter.Close()
			tarwriter := tar.NewWriter(gzwriter)
			defer tarwriter.Close()

			for _, profile := range profiles {
				res, err := fleet.DebugPprof(profile)
				if err != nil {
					// Don't fail the entire process on errors. We'll take what
					// we can get if the servers are in a bad state and not
					// responding to all requests.
					fmt.Fprintf(os.Stderr, "Failed %s: %v\n", profile, err)
					continue
				}
				fmt.Fprintf(os.Stderr, "Ran %s\n", profile)

				if err := tarwriter.WriteHeader(
					&tar.Header{
						Name: outpath + "/" + profile,
						Size: int64(len(res)),
						Mode: defaultFileMode,
					},
				); err != nil {
					return errors.Wrapf(err, "write %s header", profile)
				}

				if _, err := tarwriter.Write(res); err != nil {
					return errors.Wrapf(err, "write %s contents", profile)
				}
			}

			fmt.Fprintf(os.Stderr, "Archive written to %s\n", outfile)

			return nil
		},
	}
}
