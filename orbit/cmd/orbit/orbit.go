package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"

	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/orbit/pkg/insecure"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osquery"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osservice"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
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
			Value:   "", // need to check if explicitly set
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
		&cli.DurationFlag{
			Name:    "update-interval",
			Usage:   "How often to check for updates",
			Value:   15 * time.Minute,
			EnvVars: []string{"ORBIT_UPDATE_INTERVAL"},
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
			Name:    "fleet-desktop",
			Usage:   "Launch Fleet Desktop application (flag currently only used on darwin)",
			EnvVars: []string{"ORBIT_FLEET_DESKTOP"},
		},
	}
	app.Before = func(c *cli.Context) error {
		// handle old installations, which had default root dir set to /var/lib/orbit
		if c.String("root-dir") == "" {
			rootDir := update.DefaultOptions.RootDirectory

			executable, err := os.Executable()
			if err != nil {
				return fmt.Errorf("failed to get orbit executable: %w", err)
			}
			if strings.HasPrefix(executable, "/var/lib/orbit") {
				rootDir = "/var/lib/orbit"
			}
			c.Set("root-dir", rootDir)
		}

		return nil
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("version") {
			fmt.Println("orbit " + build.Version)
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

			b, err := os.ReadFile(c.String("enroll-secret-path"))
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

		if c.Bool("fleet-desktop") {
			switch runtime.GOOS {
			case "darwin":
				opt.Targets["desktop"] = update.DesktopMacOSTarget
			case "windows":
				opt.Targets["desktop"] = update.DesktopWindowsTarget
			case "linux":
				opt.Targets["desktop"] = update.DesktopLinuxTarget
			default:
				log.Fatal().Str("GOOS", runtime.GOOS).Msg("unsupported GOOS for desktop target")
			}
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
			osquerydPath string
			desktopPath  string
			g            run.Group
		)

		// List of interrupt functions to call during service teardown
		var interruptFunctions []func(err error)

		// Setting up the system service management early on the process lifetime
		go osservice.SetupServiceManagement(constant.SystemServiceName, c.Bool("fleet-desktop"), &interruptFunctions)

		// NOTE: When running in dev-mode, even if `disable-updates` is set,
		// it fetches osqueryd once as part of initialization.
		if !c.Bool("disable-updates") || c.Bool("dev-mode") {
			updater, err := update.NewUpdater(opt)
			if err != nil {
				return fmt.Errorf("create updater: %w", err)
			}
			if err := updater.UpdateMetadata(); err != nil {
				log.Info().Err(err).Msg("update metadata. using saved metadata")
			}

			targets := []string{"orbit", "osqueryd"}
			if c.Bool("fleet-desktop") {
				targets = append(targets, "desktop")
			}
			if c.Bool("dev-mode") {
				targets = targets[1:] // exclude orbit itself on dev-mode.
			}
			updateRunner, err := update.NewRunner(updater, update.RunnerOptions{
				CheckInterval: c.Duration("update-interval"),
				Targets:       targets,
			})
			if err != nil {
				return err
			}

			// Perform early check for updates before starting any sub-system.
			// This is to prevent bugs in other sub-systems to mess up with
			// the download of available updates.
			didUpdate, err := updateRunner.UpdateAction()
			if err != nil {
				log.Info().Err(err).Msg("early update check failed")
			}
			if didUpdate && !c.Bool("dev-mode") {
				log.Info().Msg("exiting due to successful early update")
				return nil
			}

			g.Add(updateRunner.Execute, updateRunner.Interrupt)

			osquerydLocalTarget, err := updater.Get("osqueryd")
			if err != nil {
				return fmt.Errorf("get osqueryd target: %w", err)
			}
			osquerydPath = osquerydLocalTarget.ExecPath
			if c.Bool("fleet-desktop") {
				fleetDesktopLocalTarget, err := updater.Get("desktop")
				if err != nil {
					return fmt.Errorf("get desktop target: %w", err)
				}
				if runtime.GOOS == "darwin" {
					desktopPath = fleetDesktopLocalTarget.DirPath
				} else {
					desktopPath = fleetDesktopLocalTarget.ExecPath
				}
			}
		} else {
			log.Info().Msg("running with auto updates disabled")
			updater := update.NewDisabled(opt)
			osquerydPath, err = updater.ExecutableLocalPath("osqueryd")
			if err != nil {
				log.Fatal().Err(err).Msg("locate osqueryd")
			}
			if c.Bool("fleet-desktop") {
				if runtime.GOOS == "darwin" {
					desktopPath, err = updater.DirLocalPath("desktop")
					if err != nil {
						return fmt.Errorf("get desktop target: %w", err)
					}
				} else {
					desktopPath, err = updater.ExecutableLocalPath("desktop")
					if err != nil {
						return fmt.Errorf("get desktop target: %w", err)
					}
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

		log.Debug().Msg("running single query (SELECT uuid FROM system_info)")
		uuidStr, err := getUUID(osquerydPath)
		if err != nil {
			return fmt.Errorf("get UUID: %w", err)
		}
		log.Debug().Msg("UUID is " + uuidStr)

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

		var certPath string
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

			// Directory to store proxy related assets
			proxyDirectory := filepath.Join(c.String("root-dir"), "proxy")
			if err := secure.MkdirAll(proxyDirectory, constant.DefaultDirMode); err != nil {
				return fmt.Errorf("there was a problem creating the proxy directory: %w", err)
			}

			certPath = filepath.Join(proxyDirectory, "fleet.crt")

			// Write cert that proxy uses
			err = os.WriteFile(certPath, []byte(insecure.ServerCert), os.ModePerm)
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

			if certPath = c.String("fleet-certificate"); certPath != "" {
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
				certPath = filepath.Join(c.String("root-dir"), "certs.pem")
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

		capabilities := fleet.CapabilityMap{}

		orbitClient, err := service.NewOrbitClient(fleetURL, c.String("fleet-certificate"), c.Bool("insecure"), enrollSecret, uuidStr, capabilities)
		if err != nil {
			return fmt.Errorf("error new orbit client: %w", err)
		}

		// ping the server to get the latest capabilities
		if err := orbitClient.Ping(); err != nil {
			return fmt.Errorf("error pinging the server: %w", err)
		}

		if orbitClient.GetServerCapabilities().Has(fleet.CapabilityOrbitEndpoints) {
			log.Info().Msg("Orbit endpoints are enabled")
			orbitNodeKey, err := getOrbitNodeKeyOrEnroll(orbitClient, c.String("root-dir"))
			if err != nil {
				return fmt.Errorf("error enroll: %w", err)
			}

			const orbitFlagsUpdateInterval = 30 * time.Second
			flagRunner, err := update.NewFlagRunner(orbitClient, update.FlagUpdateOptions{
				CheckInterval: orbitFlagsUpdateInterval,
				RootDir:       c.String("root-dir"),
				OrbitNodeKey:  orbitNodeKey,
			})
			if err != nil {
				return err
			}
			// do the initial flags update
			_, err = flagRunner.DoFlagsUpdate()
			if err != nil {
				// just log, OK to continue, since we will retry
				log.Info().Err(err).Msg("Initial flags update failed")
			}
			g.Add(flagRunner.Execute, flagRunner.Interrupt)
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

		// Only osquery runner is being interrupted
		// This ends up forcing the rest of the interrupt functions in the runner group to get called
		interruptFunctions = append(interruptFunctions, r.Interrupt)

		registerExtensionRunner(&g, r.ExtensionSocketPath(), deviceAuthToken)

		checkerClient, err := service.NewOrbitClient(fleetURL, c.String("fleet-certificate"), c.Bool("insecure"), enrollSecret, uuidStr, capabilities)
		if err != nil {
			return fmt.Errorf("new client for capabilities checker: %w", err)
		}
		capabilitiesChecker := newCapabilitiesChecker(checkerClient)
		g.Add(capabilitiesChecker.actor())

		if c.Bool("fleet-desktop") {
			desktopRunner := newDesktopRunner(desktopPath, fleetURL, deviceAuthToken, c.String("fleet-certificate"), c.Bool("insecure"))
			g.Add(desktopRunner.actor())
		}

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(signalHandler(ctx))

		if err := g.Run(); err != nil {
			log.Error().Err(err).Msg("unexpected exit")
		}

		return nil
	}

	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("run orbit failed")
	}
}

func registerExtensionRunner(g *run.Group, extSockPath, deviceAuthToken string) {
	ext := table.NewRunner(extSockPath, table.WithExtension(orbitInfoExtension{
		deviceAuthToken: deviceAuthToken,
	}))
	g.Add(ext.Execute, ext.Interrupt)
}

type desktopRunner struct {
	desktopPath     string
	fleetURL        string
	deviceAuthToken string
	fleetRootCA     string
	insecure        bool
	interruptCh     chan struct{} // closed when interrupt is triggered
	executeDoneCh   chan struct{} // closed when execute returns
}

func newDesktopRunner(desktopPath, fleetURL, deviceAuthToken, fleetRootCA string, insecure bool) *desktopRunner {
	return &desktopRunner{
		desktopPath:     desktopPath,
		fleetURL:        fleetURL,
		deviceAuthToken: deviceAuthToken,
		fleetRootCA:     fleetRootCA,
		insecure:        insecure,
		interruptCh:     make(chan struct{}),
		executeDoneCh:   make(chan struct{}),
	}
}

func (d *desktopRunner) actor() (func() error, func(error)) {
	return d.execute, d.interrupt
}

// execute makes sure the fleet-desktop application is running.
//
// We have to support the scenario where the user closes its sessions (log out).
// To support this, we add retries to execuser.Run. Basically retry execuser.Run until it succeeds,
// which will happen when the user logs in.
// Once fleet-desktop is started, the process is monitored (user processes get killed when the user
// closes all its sessions).
//
// NOTE(lucas): This logic could be improved to detect if there's a valid session or not first.
func (d *desktopRunner) execute() error {
	defer close(d.executeDoneCh)

	log.Info().Msg("killing any pre-existing fleet-desktop instances")
	if err := platform.KillProcessByName(constant.DesktopAppExecName); err != nil && !errors.Is(err, platform.ErrProcessNotFound) {
		log.Error().Err(err).Msg("killProcess")
	}

	log.Info().Str("path", d.desktopPath).Msg("opening")
	url, err := url.Parse(d.fleetURL)
	if err != nil {
		return fmt.Errorf("invalid fleet-url: %w", err)
	}
	url.Path = path.Join(url.Path, "device", d.deviceAuthToken)
	opts := []execuser.Option{
		execuser.WithEnv("FLEET_DESKTOP_DEVICE_URL", url.String()),
	}
	if d.fleetRootCA != "" {
		opts = append(opts, execuser.WithEnv("FLEET_DESKTOP_FLEET_ROOT_CA", d.fleetRootCA))
	}
	if d.insecure {
		opts = append(opts, execuser.WithEnv("FLEET_DESKTOP_INSECURE", "1"))
	}

	for {
		// First retry logic to start fleet-desktop.
		if done := retry(30*time.Second, false, d.interruptCh, func() bool {
			// Orbit runs as root user on Unix and as SYSTEM (Windows Service) user on Windows.
			// To be able to run the desktop application (mostly to register the icon in the system tray)
			// we need to run the application as the login user.
			// Package execuser provides multi-platform support for this.
			if err := execuser.Run(d.desktopPath, opts...); err != nil {
				log.Debug().Err(err).Msg("execuser.Run")
				return true
			}
			return false
		}); done {
			return nil
		}

		// Second retry logic to monitor fleet-desktop.
		// Call with waitFirst=true to give some time for the process to start.
		if done := retry(30*time.Second, true, d.interruptCh, func() bool {
			switch _, err := platform.GetProcessByName(constant.DesktopAppExecName); {
			case err == nil:
				return true // all good, process is running, retry.
			case errors.Is(err, platform.ErrProcessNotFound):
				log.Debug().Msgf("%s process not running", constant.DesktopAppExecName)
				return false // process is not running, do not retry.
			default:
				log.Debug().Err(err).Msg("getProcessByName")
				return true // failed to get process by name, retry.
			}
		}); done {
			return nil
		}
	}
}

func retry(d time.Duration, waitFirst bool, done chan struct{}, fn func() bool) bool {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		if !waitFirst {
			if retry := fn(); !retry {
				return false
			}
		}
		waitFirst = false
		select {
		case <-done:
			return true
		case <-ticker.C:
			// OK
		}
	}
}

func (d *desktopRunner) interrupt(err error) {
	log.Debug().Err(err).Msg("interrupt desktopRunner")

	close(d.interruptCh) // Signal execute to return.
	<-d.executeDoneCh    // Wait for execute to return.

	if err := platform.KillProcessByName(constant.DesktopAppExecName); err != nil {
		log.Error().Err(err).Msg("killProcess")
	}
}

// shell out to osquery (on Linux and macOS) or to wmic (on Windows), and get the system uuid
func getUUID(osqueryPath string) (string, error) {
	if runtime.GOOS == "windows" {
		args := []string{"/C", "wmic csproduct get UUID"}
		out, err := exec.Command("cmd", args...).Output()
		if err != nil {
			return "", err
		}
		uuidOutputStr := string(out)
		if len(uuidOutputStr) == 0 {
			return "", errors.New("get UUID: output from wmi is empty")
		}
		outputByLines := strings.Split(strings.TrimRight(uuidOutputStr, "\n"), "\n")
		if len(outputByLines) < 2 {
			return "", errors.New("get UUID: unexpected output")
		}
		return strings.TrimSpace(outputByLines[1]), nil
	}
	type UuidOutput struct {
		UuidString string `json:"uuid"`
	}

	args := []string{"-S", "--json", "select uuid from system_info"}
	out, err := exec.Command(osqueryPath, args...).Output()
	if err != nil {
		return "", err
	}
	var uuids []UuidOutput
	err = json.Unmarshal(out, &uuids)
	if err != nil {
		return "", err
	}

	if len(uuids) != 1 {
		return "", fmt.Errorf("invalid number of rows from system_info query: %d", len(uuids))
	}
	return uuids[0].UuidString, nil
}

// getOrbitNodeKeyOrEnroll attempts to read the orbit node key if the file exists on disk
// otherwise it enrolls the host with Fleet and saves the node key to disk
func getOrbitNodeKeyOrEnroll(orbitClient *service.OrbitClient, rootDir string) (string, error) {
	nodeKeyFilePath := filepath.Join(rootDir, constant.OrbitNodeKeyFileName)
	orbitNodeKey, err := ioutil.ReadFile(nodeKeyFilePath)
	switch {
	case err == nil:
		return string(orbitNodeKey), nil
	case errors.Is(err, fs.ErrNotExist):
		// OK
	default:
		return "", fmt.Errorf("read orbit node key file: %w", err)
	}
	for retries := 0; retries < constant.OrbitEnrollMaxRetries; retries++ {
		orbitNodeKey, err := enrollAndWriteNodeKeyFile(orbitClient, nodeKeyFilePath)
		if err != nil {
			log.Info().Err(err).Msg("enroll failed, retrying")
			time.Sleep(constant.OrbitEnrollRetrySleep)
			continue
		}
		return orbitNodeKey, nil
	}
	return "", fmt.Errorf("orbit node key enroll failed, attempts=%d", constant.OrbitEnrollMaxRetries)
}

func enrollAndWriteNodeKeyFile(orbitClient *service.OrbitClient, nodeKeyFilePath string) (string, error) {
	orbitNodeKey, err := orbitClient.DoEnroll()
	if err != nil {
		return "", fmt.Errorf("enroll request: %w", err)
	}
	if err := os.WriteFile(nodeKeyFilePath, []byte(orbitNodeKey), constant.DefaultFileMode); err != nil {
		return "", fmt.Errorf("write orbit node key file: %w", err)
	}
	return orbitNodeKey, nil
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
		if err := os.WriteFile(filePath, []byte(id.String()), constant.DefaultFileMode); err != nil {
			return "", fmt.Errorf("write identifier file %q: %w", filePath, err)
		}
		return id.String(), nil
	default:
		return "", fmt.Errorf("load identifier file %q: %w", filePath, err)
	}
}

var versionCommand = &cli.Command{
	Name:  "version",
	Usage: "Get the orbit version",
	Flags: []cli.Flag{},
	Action: func(c *cli.Context) error {
		fmt.Println("orbit " + build.Version)
		fmt.Println("commit - " + build.Commit)
		fmt.Println("date - " + build.Date)
		return nil
	},
}

// capabilitiesChecker is a helper to restart Orbit as soon as certain capabilities
// are changed in the server.
//
// This struct and its methods are designed to play nicely with `oklog.Group`.
type capabilitiesChecker struct {
	client        *service.OrbitClient
	interruptCh   chan struct{} // closed when interrupt is triggered
	executeDoneCh chan struct{} // closed when execute returns
}

func newCapabilitiesChecker(client *service.OrbitClient) *capabilitiesChecker {
	return &capabilitiesChecker{
		client:        client,
		interruptCh:   make(chan struct{}),
		executeDoneCh: make(chan struct{}),
	}
}

func (f *capabilitiesChecker) actor() (func() error, func(error)) {
	return f.execute, f.interrupt
}

// execute will poll the server for capabilities and emit a stop signal to restart
// Orbit if certain capabilities are enabled.
//
// You need to add an explicit check for each capability you want to watch for
func (f *capabilitiesChecker) execute() error {
	defer close(f.executeDoneCh)
	capabilitiesCHeckTicker := time.NewTicker(5 * time.Minute)

	for {
		select {
		case <-capabilitiesCHeckTicker.C:
			oldCapabilities := f.client.GetServerCapabilities()
			// ping the server to get the latest capabilities
			if err := f.client.Ping(); err != nil {
				log.Error().Err(err).Msg("pinging the server")
				continue
			}
			newCapabilities := f.client.GetServerCapabilities()

			if oldCapabilities.Has(fleet.CapabilityOrbitEndpoints) != newCapabilities.Has(fleet.CapabilityOrbitEndpoints) {
				log.Info().Msg("orbit endpoints capability changed, restarting")
				return nil
			}
		case <-f.interruptCh:
			return nil
		}
	}
}

func (f *capabilitiesChecker) interrupt(err error) {
	log.Debug().Err(err).Msg("interrupt capabilitiesChecker")
	close(f.interruptCh) // Signal execute to return.
	<-f.executeDoneCh    // Wait for execute to return.
}
