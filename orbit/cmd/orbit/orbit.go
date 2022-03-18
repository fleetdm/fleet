package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/insecure"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osquery"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/google/uuid"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	gopsutil_process "github.com/shirou/gopsutil/v3/process"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Flags set by goreleaser during build
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	app := cli.NewApp()
	app.Name = "Orbit osquery"
	app.Usage = "A powered-up, (near) drop-in replacement for osquery"
	app.Commands = []*cli.Command{
		versionCommand,
		shellCommand,
	}
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "root-dir",
			Usage:   "Root directory for Orbit state",
			Value:   update.DefaultOptions.RootDirectory,
			EnvVars: []string{"ORBIT_ROOT_DIR"},
		},
		&cli.BoolFlag{
			Name:    "insecure",
			Usage:   "Disable TLS certificate verification",
			EnvVars: []string{"ORBIT_INSECURE"},
		},
		&cli.StringFlag{
			Name:    "fleet-url",
			Usage:   "URL (host:port) of Fleet server",
			EnvVars: []string{"ORBIT_FLEET_URL"},
		},
		&cli.StringFlag{
			Name:    "fleet-certificate",
			Usage:   "Path to server certificate chain",
			EnvVars: []string{"ORBIT_FLEET_CERTIFICATE"},
		},
		&cli.StringFlag{
			Name:    "update-url",
			Usage:   "URL for update server",
			Value:   "https://tuf.fleetctl.com",
			EnvVars: []string{"ORBIT_UPDATE_URL"},
		},
		&cli.StringFlag{
			Name:    "enroll-secret",
			Usage:   "Enroll secret for authenticating to Fleet server",
			EnvVars: []string{"ORBIT_ENROLL_SECRET"},
		},
		&cli.StringFlag{
			Name:    "enroll-secret-path",
			Usage:   "Path to file containing enroll secret",
			EnvVars: []string{"ORBIT_ENROLL_SECRET_PATH"},
		},
		&cli.StringFlag{
			Name:    "osqueryd-channel",
			Usage:   "Update channel of osqueryd to use",
			Value:   "stable",
			EnvVars: []string{"ORBIT_OSQUERYD_CHANNEL"},
		},
		&cli.StringFlag{
			Name:    "orbit-channel",
			Usage:   "Update channel of Orbit to use",
			Value:   "stable",
			EnvVars: []string{"ORBIT_ORBIT_CHANNEL"},
		},
		&cli.StringFlag{
			Name:    "desktop-channel",
			Usage:   "Update channel of Fleet Desktop to use",
			Value:   "stable",
			EnvVars: []string{"ORBIT_DESKTOP_CHANNEL"},
		},
		&cli.BoolFlag{
			Name:    "disable-updates",
			Usage:   "Disables auto updates",
			EnvVars: []string{"ORBIT_DISABLE_UPDATES"},
		},
		&cli.BoolFlag{
			Name:    "dev-mode",
			Usage:   "Runs in development mode",
			EnvVars: []string{"ORBIT_DEV_MODE"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Enable debug logging",
			EnvVars: []string{"ORBIT_DEBUG"},
		},
		&cli.BoolFlag{
			Name:  "version",
			Usage: "Get Orbit version",
		},
		&cli.StringFlag{
			Name:  "log-file",
			Usage: "Log to this file path in addition to stderr",
		},
		&cli.BoolFlag{
			Name:  "dev-darwin-legacy-targets",
			Usage: "Use darwin legacy target (flag only used on darwin)",
		},
		&cli.BoolFlag{
			Name:    "fleet-desktop",
			Usage:   "Launch Fleet Desktop application (flag currently only used on darwin)",
			EnvVars: []string{"ORBIT_FLEET_DESKTOP"},
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("version") {
			fmt.Println("orbit " + version)
			return nil
		}

		var logFile io.Writer
		if logf := c.String("log-file"); logf != "" {
			if logDir := filepath.Dir(logf); logDir != "." {
				if err := secure.MkdirAll(logDir, constant.DefaultDirMode); err != nil {
					panic(err)
				}
			}
			logFile = &lumberjack.Logger{
				Filename:   logf,
				MaxSize:    25, // megabytes
				MaxBackups: 3,
				MaxAge:     28, // days
			}
			if runtime.GOOS == "windows" {
				// On Windows, Orbit runs as a "Windows Service", which fails to write to os.Stderr with
				// "write /dev/stderr: The handle is invalid" (see #3100). Thus, we log to the logFile only.
				log.Logger = log.Output(zerolog.ConsoleWriter{Out: logFile, TimeFormat: time.RFC3339Nano, NoColor: true})
			} else {
				log.Logger = log.Output(zerolog.MultiLevelWriter(
					zerolog.ConsoleWriter{Out: logFile, TimeFormat: time.RFC3339Nano, NoColor: true},
					zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano, NoColor: true},
				))
			}
		} else {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano, NoColor: true})
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		if c.Bool("insecure") && c.String("fleet-certificate") != "" {
			return errors.New("insecure and fleet-certificate may not be specified together")
		}

		if c.String("enroll-secret-path") != "" {
			if c.String("enroll-secret") != "" {
				return errors.New("enroll-secret and enroll-secret-path may not be specified together")
			}

			b, err := ioutil.ReadFile(c.String("enroll-secret-path"))
			if err != nil {
				return fmt.Errorf("read enroll secret file: %w", err)
			}

			if err := c.Set("enroll-secret", strings.TrimSpace(string(b))); err != nil {
				return fmt.Errorf("set enroll secret from file: %w", err)
			}
		}

		if err := secure.MkdirAll(c.String("root-dir"), constant.DefaultDirMode); err != nil {
			return fmt.Errorf("initialize root dir: %w", err)
		}

		deviceAuthToken, err := loadOrGenerateToken(c.String("root-dir"))
		if err != nil {
			return fmt.Errorf("load identifier file: %w", err)
		}

		localStore, err := filestore.New(filepath.Join(c.String("root-dir"), "tuf-metadata.json"))
		if err != nil {
			log.Fatal().Err(err).Msg("create local metadata store")
		}

		opt := update.DefaultOptions

		if runtime.GOOS == "darwin" && c.Bool("dev-darwin-legacy-targets") {
			opt.Targets = update.DarwinLegacyTargets
		}

		if runtime.GOOS == "darwin" && c.Bool("fleet-desktop") {
			opt.Targets["desktop"] = update.DesktopMacOSTarget
			// Override default channel with the provided value.
			opt.Targets.SetTargetChannel("desktop", c.String("desktop-channel"))
		}

		// Override default channels with the provided values.
		opt.Targets.SetTargetChannel("orbit", c.String("orbit-channel"))
		opt.Targets.SetTargetChannel("osqueryd", c.String("osqueryd-channel"))

		opt.RootDirectory = c.String("root-dir")
		opt.ServerURL = c.String("update-url")
		opt.LocalStore = localStore
		opt.InsecureTransport = c.Bool("insecure")

		var (
			updater        *update.Updater
			osquerydPath   string
			desktopAppPath string
		)

		// NOTE: When running in dev-mode, even if `disable-updates` is set,
		// it fetches osqueryd once as part of initialization.
		if !c.Bool("disable-updates") || c.Bool("dev-mode") {
			updater, err = update.New(opt)
			if err != nil {
				return fmt.Errorf("create updater: %w", err)
			}
			if err := updater.UpdateMetadata(); err != nil {
				log.Info().Err(err).Msg("update metadata. using saved metadata.")
			}
			osquerydLocalTarget, err := updater.Get("osqueryd")
			if err != nil {
				return fmt.Errorf("get osqueryd target: %w", err)
			}
			osquerydPath = osquerydLocalTarget.ExecPath
			if runtime.GOOS == "darwin" && c.Bool("fleet-desktop") {
				fleetDesktopLocalTarget, err := updater.Get("desktop")
				if err != nil {
					return fmt.Errorf("get desktop target: %w", err)
				}
				desktopAppPath = fleetDesktopLocalTarget.DirPath
			}
		} else {
			log.Info().Msg("running with auto updates disabled")
			updater = update.NewDisabled(opt)
			osquerydPath, err = updater.ExecutableLocalPath("osqueryd")
			if err != nil {
				log.Fatal().Err(err).Msg("locate osqueryd")
			}
			if runtime.GOOS == "darwin" && c.Bool("fleet-desktop") {
				desktopAppPath, err = updater.DirLocalPath("desktop")
				if err != nil {
					return fmt.Errorf("get desktop target: %w", err)
				}
			}
		}

		// Clear leftover files from updates
		if err := filepath.Walk(c.String("root-dir"), func(path string, info fs.FileInfo, err error) error {
			// Ignore anything not containing .old extension
			if !strings.HasSuffix(path, ".old") {
				return nil
			}

			if err := os.RemoveAll(path); err != nil {
				log.Info().Err(err).Msg("remove .old")
				return nil
			}
			log.Debug().Str("path", path).Msg("cleaned up old")

			return nil
		}); err != nil {
			return fmt.Errorf("cleanup old files: %w", err)
		}

		var g run.Group

		if !c.Bool("disable-updates") {
			updateRunner, err := update.NewRunner(updater, update.RunnerOptions{
				CheckInterval: 10 * time.Second,
				Targets:       []string{"orbit", "osqueryd", "desktop"},
			})
			if err != nil {
				return err
			}
			g.Add(updateRunner.Execute, updateRunner.Interrupt)
		}

		var options []osquery.Option
		options = append(options, osquery.WithDataPath(c.String("root-dir")))
		options = append(options, osquery.WithLogPath(filepath.Join(c.String("root-dir"), "osquery_log")))

		if logFile != nil {
			// If set, redirect osqueryd's stderr to the logFile.
			options = append(options, osquery.WithStderr(logFile))
		}

		fleetURL := c.String("fleet-url")
		if !strings.HasPrefix(fleetURL, "http") {
			fleetURL = "https://" + fleetURL
		}

		enrollSecret := c.String("enroll-secret")
		if enrollSecret != "" {
			const enrollSecretEnvName = "ENROLL_SECRET"
			options = append(options,
				osquery.WithEnv([]string{enrollSecretEnvName + "=" + enrollSecret}),
				osquery.WithFlags([]string{"--enroll_secret_env", enrollSecretEnvName}),
			)
		}

		if fleetURL != "https://" && c.Bool("insecure") {
			proxy, err := insecure.NewTLSProxy(fleetURL)
			if err != nil {
				return fmt.Errorf("create TLS proxy: %w", err)
			}

			g.Add(
				func() error {
					log.Info().
						Str("addr", fmt.Sprintf("localhost:%d", proxy.Port)).
						Str("target", c.String("fleet-url")).
						Msg("using insecure TLS proxy")
					err := proxy.InsecureServeTLS()
					return err
				},
				func(error) {
					if err := proxy.Close(); err != nil {
						log.Error().Err(err).Msg("close proxy")
					}
				},
			)

			certPath := filepath.Join(os.TempDir(), "fleet.crt")

			// Write cert that proxy uses
			err = ioutil.WriteFile(certPath, []byte(insecure.ServerCert), os.ModePerm)
			if err != nil {
				return fmt.Errorf("write server cert: %w", err)
			}

			// Rewrite URL to the proxy URL. Note the proxy handles any URL
			// prefix so we don't need to carry that over here.
			parsedURL := &url.URL{
				Scheme: "https",
				Host:   fmt.Sprintf("localhost:%d", proxy.Port),
			}

			// Check and log if there are any errors with TLS connection.
			pool, err := certificate.LoadPEM(certPath)
			if err != nil {
				return fmt.Errorf("load certificate: %w", err)
			}
			if err := certificate.ValidateConnection(pool, fleetURL); err != nil {
				log.Info().Err(err).Msg("Failed to connect to Fleet server. Osquery connection may fail.")
			}

			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(parsedURL)),
				osquery.WithFlags([]string{"--tls_server_certs", certPath}),
			)
		} else if fleetURL != "https://" {
			if enrollSecret == "" {
				return errors.New("enroll secret must be specified to connect to Fleet server")
			}

			parsedURL, err := url.Parse(fleetURL)
			if err != nil {
				return fmt.Errorf("parse URL: %w", err)
			}

			options = append(options,
				osquery.WithFlags(osquery.FleetFlags(parsedURL)),
			)

			if certPath := c.String("fleet-certificate"); certPath != "" {
				// Check and log if there are any errors with TLS connection.
				pool, err := certificate.LoadPEM(certPath)
				if err != nil {
					return fmt.Errorf("load certificate: %w", err)
				}
				if err := certificate.ValidateConnection(pool, fleetURL); err != nil {
					log.Info().Err(err).Msg("Failed to connect to Fleet server. Osquery connection may fail.")
				}

				options = append(options,
					osquery.WithFlags([]string{"--tls_server_certs", certPath}),
				)
			} else {
				certPath := filepath.Join(c.String("root-dir"), "certs.pem")
				if exists, err := file.Exists(certPath); err == nil && exists {
					_, err = certificate.LoadPEM(certPath)
					if err != nil {
						return fmt.Errorf("load certs.pem: %w", err)
					}
					options = append(options, osquery.WithFlags([]string{"--tls_server_certs", certPath}))
				} else {
					log.Info().Msg("No cert chain available. Relying on system store.")
				}
			}
		}

		// --force is sometimes needed when an older osquery process has not
		// exited properly
		options = append(options, osquery.WithFlags([]string{"--force"}))

		if c.Bool("debug") {
			options = append(options,
				osquery.WithFlags([]string{"--verbose", "--tls_dump"}),
			)
		}

		// Provide the flagfile to osquery if it exists. This comes after the other flags set by
		// Orbit so that users can override those flags. Note this means users may unintentionally
		// break things by overriding Orbit flags in incompatible ways. That's the price to pay for
		// flexibility.
		flagfilePath := filepath.Join(c.String("root-dir"), "osquery.flags")
		if exists, err := file.Exists(flagfilePath); err == nil && exists {
			options = append(options, osquery.WithFlags([]string{"--flagfile", flagfilePath}))
		}

		// Handle additional args after '--' in the command line. These are added last and should
		// override all other flags and flagfile entries.
		options = append(options, osquery.WithFlags(c.Args().Slice()))

		// Create an osquery runner with the provided options.
		r, err := osquery.NewRunner(osquerydPath, options...)
		if err != nil {
			return fmt.Errorf("create osquery runner: %w", err)
		}
		g.Add(r.Execute, r.Interrupt)

		ext := table.NewRunner(r.ExtensionSocketPath(), table.WithExtension(orbitInfoExtension{
			deviceAuthToken: deviceAuthToken,
		}))
		g.Add(ext.Execute, ext.Interrupt)

		if runtime.GOOS == "darwin" && c.Bool("fleet-desktop") {
			desktopRunner := newDesktopRunner(desktopAppPath, fleetURL, deviceAuthToken)
			g.Add(desktopRunner.execute, desktopRunner.interrupt)
		}

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(run.SignalHandler(ctx, os.Interrupt, os.Kill))

		if err := g.Run(); err != nil {
			log.Error().Err(err).Msg("unexpected exit")
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("run orbit failed")
	}
}

type desktopRunner struct {
	appPath         string
	fleetURL        string
	deviceAuthToken string
	done            chan struct{}
}

func newDesktopRunner(appPath, fleetURL, deviceAuthToken string) *desktopRunner {
	return &desktopRunner{
		appPath:         appPath,
		fleetURL:        fleetURL,
		deviceAuthToken: deviceAuthToken,
		done:            make(chan struct{}),
	}
}

func (d *desktopRunner) execute() error {
	log.Info().Str("path", d.appPath).Msg("opening")
	url := path.Join(d.fleetURL, "device", d.deviceAuthToken)
	if err := open.App(d.appPath, open.AppWithEnv("FLEET_DESKTOP_DEVICE_URL", url)); err != nil {
		return fmt.Errorf("open desktop app: %w", err)
	}
	// TODO(lucas): Orbit currently does not monitor the Fleet Desktop
	// application (e.g. in case of crashes).
	<-d.done
	return nil
}

func (d *desktopRunner) interrupt(err error) {
	log.Debug().Err(err).Msg("interrupt desktopRunner")
	defer close(d.done)

	if err := killProcessByName(d.appPath); err != nil {
		log.Error().Err(err).Msg("killProcess")
	}
}

func loadOrGenerateToken(rootDir string) (string, error) {
	filePath := filepath.Join(rootDir, "identifier")
	id, err := ioutil.ReadFile(filePath)
	switch {
	case err == nil:
		return string(id), nil
	case errors.Is(err, os.ErrNotExist):
		id, err := uuid.NewRandom()
		if err != nil {
			return "", fmt.Errorf("generate identifier: %w", err)
		}
		if err := ioutil.WriteFile(filePath, []byte(id.String()), constant.DefaultFileMode); err != nil {
			return "", fmt.Errorf("write identifier file %q: %w", filePath, err)
		}
		return id.String(), nil
	default:
		return "", fmt.Errorf("load identifier file %q: %w", filePath, err)
	}
}

func killProcessByName(name string) error {
	processes, err := gopsutil_process.Processes()
	if err != nil {
		return err
	}
	var foundProcess *gopsutil_process.Process
	for _, process := range processes {
		processName, err := process.Name()
		if err != nil {
			log.Debug().Err(err).Int32("pid", process.Pid).Msg("get process name")
			continue
		}
		if processName == "fleet-desktop" {
			foundProcess = process
			break
		}
	}
	if foundProcess == nil {
		return fmt.Errorf("process not found: %s", name)
	}
	if err := foundProcess.Kill(); err != nil {
		return fmt.Errorf("kill process %d: %w", foundProcess.Pid, err)
	}
	return nil
}

var versionCommand = &cli.Command{
	Name:  "version",
	Usage: "Get the orbit version",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		fmt.Println("orbit " + version)
		fmt.Println("commit - " + commit)
		fmt.Println("date - " + date)
		return nil
	},
}
