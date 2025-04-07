package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/urfave/cli/v2"
)

// Defining here for testing purposes
var nowFn = time.Now

const (
	profileExtension = "prof"
	jsonExtension    = "json"
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
			debugArchiveCommand(),
			debugConnectionCommand(),
			debugMigrations(),
			debugDBLocksCommand(),
			debugDBInnodbStatus(),
			debugDBProcessList(),
		},
	}
}

func writeFile(filename string, bytes []byte, mode os.FileMode) error {
	if err := os.WriteFile(filename, bytes, mode); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Output written to %s\n", filename)
	return nil
}

func outfileName(name string) string {
	return fmt.Sprintf("fleet-%s-%s", name, nowFn().Format("20060102150405Z"))
}

func outfileNameWithExt(name string, ext string) string {
	return fmt.Sprintf("%s.%s", outfileName(name), ext)
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
				outfile = outfileNameWithExt("profile", profileExtension)
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
				outfile = outfileNameWithExt(name, profileExtension)
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
				outfile = outfileNameWithExt(name, profileExtension)
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
				outfile = outfileNameWithExt(name, profileExtension)
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
				"errors",
				"goroutine",
				"heap",
				"mutex",
				"profile",
				"threadcreate",
				"trace",
				"db-locks",
				"db-innodb-status",
				"db-process-list",
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileNameWithExt("profiles-archive", "tar.gz")
			}

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
				var ext string

				switch profile {
				case "errors":
					var buf bytes.Buffer
					ext = jsonExtension
					err = fleet.DebugErrors(&buf, false)
					if err == nil {
						res = buf.Bytes()
					}

				case "db-locks":
					ext = jsonExtension
					res, err = fleet.DebugDBLocks()
				case "db-innodb-status":
					ext = jsonExtension
					res, err = fleet.DebugInnoDBStatus()
				case "db-process-list":
					ext = jsonExtension
					res, err = fleet.DebugProcessList()

				default:
					ext = profileExtension
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

				outname := profile
				if ext != "" {
					outname = profile + "." + ext
				}

				tarName := outfile + "/" + outname
				if err := tarwriter.WriteHeader(
					&tar.Header{
						Name: tarName,
						Size: int64(len(res)),
						Mode: defaultFileMode,
					},
				); err != nil {
					return fmt.Errorf("write %s header: %w", tarName, err)
				}

				if _, err := tarwriter.Write(res); err != nil {
					return fmt.Errorf("write %s contents: %w", tarName, err)
				}
			}

			fmt.Fprintf(os.Stderr, "################################################################################\n"+
				"# WARNING:\n"+
				"#   The files in the generated archive may contain sensitive data.\n"+
				"#   Please review them before sharing.\n"+
				"#\n"+
				"#   Archive written to: %s\n"+
				"################################################################################\n",
				outfile)

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

			usingHTTPS := strings.HasPrefix(cc.Address, "https://")

			//
			// Scenarios:
			// 	- If a --fleet-certificate is provided, use it as root CA.
			// 	- If a --fleet-certificate is not provided, but a cc.RootCA is set in the configuration, use it as root CA.
			// 	- If a --fleet-certificate is not provided and there isn't a cc.RootCA set in the configuration, use the embedded certs as root CA.
			//
			usingEmbeddedCA := false
			if usingHTTPS {
				certPath := getFleetCertificate(c)
				if certPath != "" {
					// if a certificate is provided, use it as root CA
					cc.RootCA = certPath
					cc.TLSSkipVerify = false
				} else if cc.RootCA == "" { // --fleet-certificate is not set
					// If a certificate is not provided and a cc.RootCA is not set in the configuration,
					// then use the embedded root CA which is used by osquery to connect to Fleet.
					usingEmbeddedCA = true
					tmpDir, err := os.MkdirTemp("", "")
					if err != nil {
						return fmt.Errorf("failed to create temporary directory: %w", err)
					}
					certPath := filepath.Join(tmpDir, "certs.pem")
					if err := os.WriteFile(certPath, packaging.OsqueryCerts, 0o600); err != nil {
						return fmt.Errorf("failed to create temporary certs.pem file: %s", err)
					}
					defer os.RemoveAll(certPath)
					cc.RootCA = certPath
					cc.TLSSkipVerify = false
				}
			}

			cli, baseURL, err := rawHTTPClientFromConfig(cc)
			if err != nil {
				return err
			}

			// print a summary of the address and TLS context that is investigated
			fmt.Fprintf(c.App.Writer, "Debugging connection to %s; Configuration context: %s; ", baseURL.Hostname(), configContext)

			if usingHTTPS {
				rootCA := cc.RootCA
				if usingEmbeddedCA {
					rootCA += " (embedded certs used by default to generate fleetd packages)"
				}
				fmt.Fprintf(c.App.Writer, "Root CA: %s; ", rootCA)

				tlsMode := "secure"
				if cc.TLSSkipVerify {
					tlsMode = "insecure"
				}
				fmt.Fprintf(c.App.Writer, "TLS: %s.\n", tlsMode)
			}

			// Check that the url's host resolves to an IP address or is otherwise
			// a valid IP address directly.
			if err := resolveHostname(c.Context, timeoutPerCheck, baseURL.Hostname()); err != nil {
				return fmt.Errorf("Fail: resolve host: %w", err)
			}
			fmt.Fprintf(c.App.Writer, "Success: can resolve host %s.\n", baseURL.Hostname())

			// Attempt a raw TCP connection to host:port.
			dialURL := baseURL.Host
			if baseURL.Port() == "" {
				fmt.Fprintf(c.App.Writer, "Assumming port 443.\n")
				dialURL += ":443"
			}
			if err := dialHostPort(c.Context, timeoutPerCheck, dialURL); err != nil {
				return fmt.Errorf("Fail: dial server: %w", err)
			}
			fmt.Fprintf(c.App.Writer, "Success: can dial server at %s.\n", baseURL.Host)

			// Run some validations on the TLS certificate.
			if usingHTTPS {
				if err := checkFleetCert(c.Context, timeoutPerCheck, cc.RootCA, baseURL.Host); err != nil {
					return fmt.Errorf("Fail: certificate: %w", err)
				}
				fmt.Fprintln(c.App.Writer, "Success: TLS certificate seems valid.")
			}

			// Check that the server responds with expected responses (by
			// making a POST to /api/osquery/enroll with an invalid
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
	var (
		name  = "errors"
		flush bool
	)
	return &cli.Command{
		Name:      name,
		Usage:     "Save the recorded fleet server errors to a file.",
		UsageText: "Recording of errors and their retention period is controlled via the --logging_error_retention_period fleet command flag.",
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
			stdoutFlag(),
			&cli.BoolFlag{
				Name:        "flush",
				EnvVars:     []string{"FLUSH"},
				Value:       false,
				Destination: &flush,
				Usage:       "Clear errors from Redis after reading them",
			},
		},
		Action: func(c *cli.Context) error {
			fleet, err := clientFromCLI(c)
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			stdout := getStdout(c)

			if stdout && outfile != "" {
				return errors.New("-stdout and -outfile must not be specified together")
			}

			out := os.Stdout

			if !stdout {
				if outfile == "" {
					outfile = outfileNameWithExt(name, jsonExtension)
				}

				f, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFileMode)
				if err != nil {
					return err
				}
				defer f.Close()
				out = f
			}

			if err := fleet.DebugErrors(out, flush); err != nil {
				return err
			}

			if !stdout {
				if err := out.Close(); err != nil {
					return fmt.Errorf("write errors to file: %w", err)
				}

				fmt.Fprintf(os.Stderr, "################################################################################\n"+
					"# WARNING:\n"+
					"#   The generated file may contain sensitive data.\n"+
					"#   Please review the file before sharing.\n"+
					"#\n"+
					"#   Output written to: %s\n"+
					"################################################################################\n",
					outfile)
			}

			return nil
		},
	}
}

func debugDBLocksCommand() *cli.Command {
	name := "db-locks"
	usage := "Save the current database transaction locking information to a file."
	usageText := "Saves transaction locking information with queries that are waiting on or blocking other transactions."
	return bytesCommand(name, usage, usageText, func(c *cli.Context) (func() ([]byte, error), error) {
		client, err := clientFromCLI(c)
		if err != nil {
			return nil, err
		}

		return client.DebugDBLocks, nil
	})
}

func debugDBInnodbStatus() *cli.Command {
	name := "db-innodb-status"
	usage := "Save the current database InnoDB status information to a file."
	usageText := usage
	return bytesCommand(name, usage, usageText, func(c *cli.Context) (func() ([]byte, error), error) {
		client, err := clientFromCLI(c)
		if err != nil {
			return nil, err
		}

		return client.DebugInnoDBStatus, nil
	})
}

func debugDBProcessList() *cli.Command {
	name := "db-process-list"
	usage := "Save the current running processes (queries, etc) in the database to a file."
	usageText := usage
	return bytesCommand(name, usage, usageText, func(c *cli.Context) (func() ([]byte, error), error) {
		client, err := clientFromCLI(c)
		if err != nil {
			return nil, err
		}

		return client.DebugProcessList, nil
	})
}

func bytesCommand(name, usage, usageText string, bytesFuncGenerator func(c *cli.Context) (func() ([]byte, error), error)) *cli.Command {
	return &cli.Command{
		Name:      name,
		Usage:     usage,
		UsageText: usageText,
		Flags: []cli.Flag{
			outfileFlag(),
			configFlag(),
			contextFlag(),
			debugFlag(),
		},
		Action: func(c *cli.Context) error {
			bytesFunc, err := bytesFuncGenerator(c)
			if err != nil {
				return err
			}

			bytesData, err := bytesFunc()
			if err != nil {
				return err
			}

			outfile := getOutfile(c)
			if outfile == "" {
				outfile = outfileName(name)
			}

			if err := writeFile(outfile, bytesData, defaultFileMode); err != nil {
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

	baseURL.Path = "/api/osquery/enroll"
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
