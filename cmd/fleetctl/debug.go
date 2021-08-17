package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
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
			debugConnectionCommand(),
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

func debugConnectionCommand() *cli.Command {
	// TODO: do we want to allow setting this via a flag?
	const timeoutPerCheck = 10 * time.Second

	return &cli.Command{
		Name:  "connection",
		Usage: "Investigate the cause of a connection failure to the Fleet server.",
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			debugFlag(),
			fleetCertificateFlag(),
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			baseURL := fleet.BaseURL()

			// 1. Check that the url's host resolves to an IP address or is otherwise
			// a valid IP address directly. The ips may be used in a later check to
			// verify if the certificate is for one of them instead of the hostname.
			ips, err := resolveHostname(c.Context, timeoutPerCheck, baseURL.Hostname())
			if err != nil {
				return errors.Wrap(err, "Fail")
			}
			fmt.Fprintf(os.Stdout, "Success: can resolve host %s.\n", baseURL.Hostname())
			_ = ips

			// 2. Attempt a raw TCP connection to host:port.
			if err := dialHostPort(c.Context, timeoutPerCheck, baseURL.Host); err != nil {
				return errors.Wrap(err, "Fail")
			}
			fmt.Fprintf(os.Stdout, "Success: can dial server at %s.\n", baseURL.Host)

			if cert := getFleetCertificate(c); cert != "" {
				// TODO: check to make sure clientFromCLI can work without a working connection.
				// The command's steps could be:
				// 3. Is the certificate valid at all (x509.Certificate.ParseCertificate?)
				// 4. Is the certificate valid for the hostname/IP address (x509.Certificate.VerifyHostname?)
			}

			// 5. Check that the server responds with expected responses (by
			// making a POST to /api/v1/osquery/enroll with an invalid
			// secret).
			var enrollRes struct {
				Error       string `json:"error"`
				NodeInvalid bool   `json:"node_invalid"`
			}
			res, err := fleet.Do("POST", "/api/v1/osquery/enroll", "", map[string]string{
				"enroll_secret": "--invalid--",
			})
			if err != nil {
				return errors.Wrap(err, "Fail")
			}
			defer res.Body.Close()
			if err := json.NewDecoder(res.Body).Decode(&enrollRes); err != nil {
				// TODO: handle invalid json response
			}
			if enrollRes.Error == "" || !enrollRes.NodeInvalid {
				// TODO: handle unexpected response, should have failed
			}
			fmt.Fprintf(os.Stdout, "%#v\n", enrollRes)
			fmt.Fprintln(os.Stdout, "Success: agent API endpoints are available.")
			return nil
		},
	}
}

func resolveHostname(ctx context.Context, timeout time.Duration, host string) ([]net.IP, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var r net.Resolver
	return r.LookupIP(ctx, "ip", host)
}

func dialHostPort(ctx context.Context, timeout time.Duration, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err == nil {
		conn.Close()
	}
	return err
}
