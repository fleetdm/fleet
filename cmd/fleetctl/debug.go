package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
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
			debugErrorsCommand(),
			debugDBLocksCommand(),
			debugArchiveCommand(),
			debugConnectionCommand(),
			debugMigrations(),
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
				return fmt.Errorf("write profile to file: %w", err)
			}

			return nil
		},
	}
}

func joinCmdline(cmdline string) string {
	var tokens []string
	for _, token := range strings.Split(cmdline, "\x00") {
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
					return fmt.Errorf("write cmdline to file: %w", err)
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
				return fmt.Errorf("write %s to file: %w", name, err)
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
				return fmt.Errorf("write %s to file: %w", name, err)
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
				return fmt.Errorf("write %s to file: %w", name, err)
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
				"db-locks",
				"errors",
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
				return fmt.Errorf("open archive for output: %w", err)
			}
			defer f.Close()
			gzwriter := gzip.NewWriter(f)
			defer gzwriter.Close()
			tarwriter := tar.NewWriter(gzwriter)
			defer tarwriter.Close()

			for _, profile := range profiles {
				var res []byte

				switch profile {
				case "errors":
					var buf bytes.Buffer
					err = fleet.DebugErrors(&buf)
					if err == nil {
						res = buf.Bytes()
					}

				case "db-locks":
					res, err = fleet.DebugDBLocks()

				default:
					res, err = fleet.DebugPprof(profile)
				}

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
					return fmt.Errorf("write %s header: %w", profile, err)
				}

				if _, err := tarwriter.Write(res); err != nil {
					return fmt.Errorf("write %s contents: %w", profile, err)
				}
			}

			fmt.Fprintf(os.Stderr, "Archive written to %s\n", outfile)

			return nil
		},
	}
}

func debugConnectionCommand() *cli.Command {
	const timeoutPerCheck = 10 * time.Second

	return &cli.Command{
		Name:      "connection",
		ArgsUsage: "[<address>]",
		Usage:     "Investigate the cause of a connection failure to the Fleet server.",
		Description: `Run a number of checks to debug a connection failure to the Fleet
server.

If <address> is provided, this is the address that is investigated,
otherwise the address of the provided context is used, with
the default context used if none is explicitly specified.`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
			fleetCertificateFlag(),
		},
		Action: func(c *cli.Context) error {
			var addr string
			if narg := c.NArg(); narg > 0 {
				if narg > 1 {
					return errors.New("too many arguments")
				}
				addr = c.Args().First()

				// when an address is provided, the --config and --context flags
				// cannot be set.
				if c.IsSet("config") {
					return errors.New("the --config flag cannot be set when an <address> is provided")
				}
				if c.IsSet("context") {
					return errors.New("the --context flag cannot be set when an <address> is provided")
				}
			} else if cert := getFleetCertificate(c); cert != "" {
				return errors.New("the --fleet-certificate flag can only be set when an <address> is provided")
			}

			// ensure there is an address to debug (either from the config's context,
			// or explicit)
			cc, err := clientConfigFromCLI(c)
			if err != nil {
				return err
			}
			configContext := c.String("context")
			if addr != "" {
				cc.Address = addr

				// when an address is explicitly provided, we don't use any of the
				// config's context values.
				configContext = "none - using provided address"
				cc.TLSSkipVerify = false
				cc.RootCA = ""
			}
			if cc.Address == "" {
				return errors.New(`set the Fleet API address with: fleetctl config set --address https://localhost:8080
or provide an <address> argument to debug: fleetctl debug connection localhost:8080`)
			}

			// it's ok if there is no scheme specified, add it automatically
			if !strings.Contains(cc.Address, "://") {
				cc.Address = "https://" + cc.Address
			}

			if certPath := getFleetCertificate(c); certPath != "" {
				// if a certificate is provided, use it as root CA
				cc.RootCA = certPath
				cc.TLSSkipVerify = false
			}

			cli, baseURL, err := rawHTTPClientFromConfig(cc)
			if err != nil {
				return err
			}

			// print a summary of the address and TLS context that is investigated
			fmt.Fprintf(c.App.Writer, "Debugging connection to %s; Configuration context: %s; ", baseURL.Hostname(), configContext)
			rootCA := "(system)"
			if cc.RootCA != "" {
				rootCA = cc.RootCA
			}
			fmt.Fprintf(c.App.Writer, "Root CA: %s; ", rootCA)
			tlsMode := "secure"
			if cc.TLSSkipVerify {
				tlsMode = "insecure"
			}
			fmt.Fprintf(c.App.Writer, "TLS: %s.\n", tlsMode)

			// Check that the url's host resolves to an IP address or is otherwise
			// a valid IP address directly.
			if err := resolveHostname(c.Context, timeoutPerCheck, baseURL.Hostname()); err != nil {
				return fmt.Errorf("Fail: resolve host: %w", err)
			}
			fmt.Fprintf(c.App.Writer, "Success: can resolve host %s.\n", baseURL.Hostname())

			// Attempt a raw TCP connection to host:port.
			if err := dialHostPort(c.Context, timeoutPerCheck, baseURL.Host); err != nil {
				return fmt.Errorf("Fail: dial server: %w", err)
			}
			fmt.Fprintf(c.App.Writer, "Success: can dial server at %s.\n", baseURL.Host)

			if cert := getFleetCertificate(c); cert != "" {
				// Run some validations on the TLS certificate.
				if err := checkFleetCert(c.Context, timeoutPerCheck, cert, baseURL.Host); err != nil {
					return fmt.Errorf("Fail: certificate: %w", err)
				}
				fmt.Fprintln(c.App.Writer, "Success: TLS certificate seems valid.")
			}

			// Check that the server responds with expected responses (by
			// making a POST to /api/v1/osquery/enroll with an invalid
			// secret).
			if err := checkAPIEndpoint(c.Context, timeoutPerCheck, baseURL, cli); err != nil {
				return fmt.Errorf("Fail: agent API endpoint: %w", err)
			}
			fmt.Fprintln(c.App.Writer, "Success: agent API endpoints are available.")

			return nil
		},
	}
}

func debugMigrations() *cli.Command {
	return &cli.Command{
		Name:  "migrations",
		Usage: "Run a check of database migrations",
		Description: `Run a check for database migrations on the fleet server.

It returns the list of migrations that are missing.
Such migrations can be applied via "fleet prepare db" before running "fleet serve".
`,
		Flags: []cli.Flag{
			configFlag(),
			contextFlag(),
		},
		Action: func(c *cli.Context) error {
			client, err := clientFromCLI(c)
			if err != nil {
				return err
			}
			migrationStatus, err := client.DebugMigrations()
			if err != nil {
				return err
			}
			switch migrationStatus.StatusCode {
			case fleet.NoMigrationsCompleted:
				// Currently shouldn't happen, because this command requires authentication, and therefore
				// requires the sessions table. Leaving this here in case we remove authentication from this endpoint.
				fmt.Println("Your Fleet database is not initialized. Fleet cannot start up.\n" +
					"Fleet server must be run with \"prepare db\" to perform the migrations.")
			case fleet.AllMigrationsCompleted:
				fmt.Println("Migrations up-to-date.")
			case fleet.UnknownMigrations:
				fmt.Printf("Unknown migrations detected: tables=%v, data=%v.\n",
					migrationStatus.UnknownTable, migrationStatus.UnknownData)
			case fleet.SomeMigrationsCompleted:
				fmt.Printf("Missing migrations detected: tables=%v, data=%v.\n"+
					"Fleet server must be run with \"prepare db\" to perform the migrations.\n",
					migrationStatus.MissingTable, migrationStatus.MissingData)
			}
			return nil
		},
	}
}

func debugErrorsCommand() *cli.Command {
	name := "errors"
	return &cli.Command{
		Name:      name,
		Usage:     "Save the recorded fleet server errors to a file.",
		UsageText: "Recording of errors and their retention period is controlled via the --logging_error_retention_period fleet command flag.",
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

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName(name)
			}

			f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFileMode)
			if err != nil {
				return err
			}
			defer f.Close()

			if err := fleet.DebugErrors(f); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("write errors to file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "Output written to %s\n", outfile)

			return nil
		},
	}
}

func debugDBLocksCommand() *cli.Command {
	name := "db-locks"
	return &cli.Command{
		Name:      name,
		Usage:     "Save the current database transaction locking information to a file.",
		UsageText: "Saves transaction locking information with queries that are waiting on or blocking other transactions.",
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

			locks, err := fleet.DebugDBLocks()
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName(name)
			}

			if err := writeFile(outfile, locks, defaultFileMode); err != nil {
				return fmt.Errorf("write %s to file: %w", name, err)
			}

			return nil
		},
	}
}

func resolveHostname(ctx context.Context, timeout time.Duration, host string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var r net.Resolver
	ips, err := r.LookupIP(ctx, "ip", host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return errors.New("no address found for host")
	}
	return nil
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

func checkAPIEndpoint(ctx context.Context, timeout time.Duration, baseURL *url.URL, client *http.Client) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// make an enroll request with a deliberately invalid secret,
	// to see if we get the expected error json payload.
	var enrollRes struct {
		Error       string `json:"error"`
		NodeInvalid bool   `json:"node_invalid"`
	}
	headers := map[string]string{
		"Content-type": "application/json",
		"Accept":       "application/json",
	}

	baseURL.Path = "/api/v1/osquery/enroll"
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		baseURL.String(),
		bytes.NewBufferString(`{"enroll_secret": "--invalid--"}`),
	)
	if err != nil {
		return fmt.Errorf("creating request object: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	if err := json.NewDecoder(res.Body).Decode(&enrollRes); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if res.StatusCode != http.StatusUnauthorized || enrollRes.Error == "" || !enrollRes.NodeInvalid {
		return fmt.Errorf("unexpected %d response", res.StatusCode)
	}
	return nil
}

func checkFleetCert(ctx context.Context, timeout time.Duration, certPath, addr string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	certPool, err := certificate.LoadPEM(certPath)
	if err != nil {
		return err
	}
	if err := certificate.ValidateConnectionContext(ctx, certPool, "https://"+addr); err != nil {
		return err
	}

	return nil
}
