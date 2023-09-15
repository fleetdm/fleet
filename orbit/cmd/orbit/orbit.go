package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/augeas"
	"github.com/fleetdm/fleet/v4/orbit/pkg/build"
	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/orbit/pkg/execuser"
	"github.com/fleetdm/fleet/v4/orbit/pkg/insecure"
	"github.com/fleetdm/fleet/v4/orbit/pkg/logging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osquery"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osservice"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/orbit_info"
	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
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
			Usage:   "Path to the Fleet server certificate chain",
			EnvVars: []string{"ORBIT_FLEET_CERTIFICATE"},
		},
		&cli.StringFlag{
			Name:    "fleet-desktop-alternative-browser-host",
			Usage:   "Alternative host:port to use for Fleet Desktop in the browser (this may be required when using TLS client authentication in the Fleet server)",
			EnvVars: []string{"ORBIT_FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST"},
		},
		&cli.StringFlag{
			Name:    "update-url",
			Usage:   "URL for update server",
			Value:   "https://tuf.fleetctl.com",
			EnvVars: []string{"ORBIT_UPDATE_URL"},
		},
		&cli.StringFlag{
			Name:    "update-tls-certificate",
			Usage:   "Path to the update server TLS certificate chain",
			EnvVars: []string{"ORBIT_UPDATE_TLS_CERTIFICATE"},
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
		// Note: this flag doesn't have any effect anymore. I'm keeping
		// it just for backwards compatibility since some users were
		// using it because softwareupdated was causing problems and I
		// don't want to break their setups.
		//
		// For more context please check out: https://github.com/fleetdm/fleet/issues/11777
		&cli.BoolFlag{
			Name:    "disable-kickstart-softwareupdated",
			Usage:   "(Deprecated) Disable periodic execution of 'launchctl kickstart -k softwareupdated' on macOS",
			EnvVars: []string{"ORBIT_FLEET_DISABLE_KICKSTART_SOFTWAREUPDATED"},
			Hidden:  true,
		},
		&cli.BoolFlag{
			Name:    "use-system-configuration",
			Usage:   "Try to read --fleet-url and --enroll-secret using configuration in the host (currently only macOS profiles are supported)",
			EnvVars: []string{"ORBIT_USE_SYSTEM_CONFIGURATION"},
			Hidden:  true,
		},
		&cli.BoolFlag{
			Name:    "enable-scripts",
			Usage:   "Enable script execution",
			EnvVars: []string{"ORBIT_ENABLE_SCRIPTS"},
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
			if err := c.Set("root-dir", rootDir); err != nil {
				return fmt.Errorf("failed to set root-dir: %w", err)
			}
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

		if c.Bool("insecure") && c.String("update-tls-certificate") != "" {
			return errors.New("insecure and update-tls-certificate may not be specified together")
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

		// if neither are set, this might be an agent deployed via MDM, try to read
		// both configs from a configuration profile
		if runtime.GOOS == "darwin" && c.Bool("use-system-configuration") {
			log.Info().Msg("trying to read fleet-url and enroll-secret from a configuration profile")
			for {
				config, err := profiles.GetFleetdConfig()
				switch {
				// handle these errors separately as debug messages to not raise false
				// alarms when users look into the orbit logs, it's perfectly normal to
				// not have a configuration profile, or to get into this situation in
				// operating systems that don't have profile support.
				case err != nil:
					log.Error().Err(err).Msg("reading configuration profile")
				case config.EnrollSecret == "" || config.FleetURL == "":
					log.Debug().Msg("enroll secret or fleet url are empty in configuration profile, not setting either")
				default:
					log.Info().Msg("setting enroll-secret and fleet-url configs from configuration profile")
					if err := c.Set("enroll-secret", config.EnrollSecret); err != nil {
						return fmt.Errorf("set enroll secret from configuration profile: %w", err)
					}
					if err := c.Set("fleet-url", config.FleetURL); err != nil {
						return fmt.Errorf("set fleet URL from configuration profile: %w", err)
					}
					if err := writeSecret(config.EnrollSecret, c.String("root-dir")); err != nil {
						return fmt.Errorf("write enroll secret: %w", err)
					}
				}

				if c.String("fleet-url") != "" && c.String("enroll-secret") != "" {
					log.Info().Msg("found configuration values in system profile")
					break
				}

				log.Info().Msg("didn't find configuration values in system profile, trying again in 30 seconds")
				time.Sleep(30 * time.Second)
			}
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
		opt.ServerCertificatePath = c.String("update-tls-certificate")

		var (
			osquerydPath string
			desktopPath  string
			g            run.Group
			appDoneCh    chan struct{} // closed when runner run.group.Run() returns
		)

		// Setting up the system service management early on the process lifetime
		appDoneCh = make(chan struct{})

		// Initializing service runner and system service manager
		systemChecker := newSystemChecker()
		g.Add(systemChecker.Execute, systemChecker.Interrupt)
		go osservice.SetupServiceManagement(constant.SystemServiceName, systemChecker.svcInterruptCh, appDoneCh)

		// sofwareupdated is a macOS daemon that automatically updates Apple software.
		if !c.Bool("disable-kickstart-softwareupdated") && runtime.GOOS == "darwin" {
			log.Warn().Msg("fleetd no longer automatically kickstarts softwareupdated. The --disable-kickstart-softwareupdated flag, which was previously used to disable this behavior, has been deprecated and will be removed in a future version")
		}

		updateClientCrtPath := filepath.Join(c.String("root-dir"), constant.UpdateTLSClientCertificateFileName)
		updateClientKeyPath := filepath.Join(c.String("root-dir"), constant.UpdateTLSClientKeyFileName)
		updateClientCrt, err := certificate.LoadClientCertificateFromFiles(updateClientCrtPath, updateClientKeyPath)
		if err != nil {
			return fmt.Errorf("error loading update client certificate: %w", err)
		}
		if updateClientCrt != nil {
			log.Info().Msg("Found TLS client certificate and key. Using them to authenticate to the update server.")
			opt.ClientCertificate = &updateClientCrt.Crt
		}

		// NOTE: When running in dev-mode, even if `disable-updates` is set,
		// it fetches osqueryd once as part of initialization.
		var updater *update.Updater
		var updateRunner *update.Runner
		if !c.Bool("disable-updates") || c.Bool("dev-mode") {
			updater, err = update.NewUpdater(opt)
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
			updateRunner, err = update.NewRunner(updater, update.RunnerOptions{
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
			updater = update.NewDisabled(opt)
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

		orbitHostInfo, err := getHostInfo(osquerydPath, filepath.Join(c.String("root-dir"), "osquery.db"))
		if err != nil {
			return fmt.Errorf("get UUID: %w", err)
		}
		log.Debug().Str("info", fmt.Sprint(orbitHostInfo)).Msg("retrieved host info")

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

		fleetClientCertPath := filepath.Join(c.String("root-dir"), constant.FleetTLSClientCertificateFileName)
		fleetClientKeyPath := filepath.Join(c.String("root-dir"), constant.FleetTLSClientKeyFileName)
		fleetClientCrt, err := certificate.LoadClientCertificateFromFiles(fleetClientCertPath, fleetClientKeyPath)
		if err != nil {
			return fmt.Errorf("error loading fleet client certificate: %w", err)
		}

		var fleetClientCertificate *tls.Certificate
		if fleetClientCrt != nil {
			log.Info().Msg("Found TLS client certificate and key. Using them to authenticate to Fleet.")
			fleetClientCertificate = &fleetClientCrt.Crt
			options = append(options, osquery.WithFlags([]string{
				"--tls_client_cert", fleetClientCertPath,
				"--tls_client_key", fleetClientKeyPath,
			}))
		}

		orbitClient, err := service.NewOrbitClient(
			c.String("root-dir"),
			fleetURL,
			c.String("fleet-certificate"),
			c.Bool("insecure"),
			enrollSecret,
			fleetClientCertificate,
			orbitHostInfo,
		)
		if err != nil {
			return fmt.Errorf("error new orbit client: %w", err)
		}

		// create the notifications middleware that wraps the orbit client
		// (must be shared by all runners that use a ConfigFetcher).
		const (
			renewEnrollmentProfileCommandFrequency = time.Hour
			windowsMDMEnrollmentCommandFrequency   = time.Hour
			windowsMDMEBitlockerCommandFrequency   = time.Hour
		)
		configFetcher := update.ApplyRenewEnrollmentProfileConfigFetcherMiddleware(orbitClient, renewEnrollmentProfileCommandFrequency, fleetURL)
		configFetcher = update.ApplyRunScriptsConfigFetcherMiddleware(configFetcher, c.Bool("enable-scripts"), orbitClient)

		switch runtime.GOOS {
		case "darwin":
			// add middleware to handle nudge installation and updates
			const nudgeLaunchInterval = 30 * time.Minute
			configFetcher = update.ApplyNudgeConfigFetcherMiddleware(configFetcher, update.NudgeConfigFetcherOptions{
				UpdateRunner: updateRunner, RootDir: c.String("root-dir"), Interval: nudgeLaunchInterval,
			})

			configFetcher = update.ApplyDiskEncryptionRunnerMiddleware(configFetcher)
			configFetcher = update.ApplySwiftDialogDownloaderMiddleware(configFetcher, updateRunner)
		case "windows":
			configFetcher = update.ApplyWindowsMDMEnrollmentFetcherMiddleware(configFetcher, windowsMDMEnrollmentCommandFrequency, orbitHostInfo.HardwareUUID, orbitClient)
			configFetcher = update.ApplyWindowsMDMBitlockerFetcherMiddleware(configFetcher, windowsMDMEnrollmentCommandFrequency, orbitHostInfo.HardwareUUID, orbitClient)
		}

		const orbitFlagsUpdateInterval = 30 * time.Second
		flagRunner := update.NewFlagRunner(configFetcher, update.FlagUpdateOptions{
			CheckInterval: orbitFlagsUpdateInterval,
			RootDir:       c.String("root-dir"),
		})
		// Try performing a flags update to use latest configured osquery flags from get-go.
		if _, err := flagRunner.DoFlagsUpdate(); err != nil {
			// Just log, OK to continue, since flagRunner will retry
			// in flagRunner.Execute.
			log.Debug().Err(err).Msg("initial flags update failed")
		}
		g.Add(flagRunner.Execute, flagRunner.Interrupt)

		// only setup extensions autoupdate if we have enabled updates
		// for extensions autoupdate, we can only proceed after orbit is enrolled in fleet
		// and all relevant things for it (like certs, enroll secrets, tls proxy, etc) is configured
		if !c.Bool("disable-updates") || c.Bool("dev-mode") {
			const orbitExtensionUpdateInterval = 60 * time.Second
			extRunner := update.NewExtensionConfigUpdateRunner(configFetcher, update.ExtensionUpdateOptions{
				CheckInterval: orbitExtensionUpdateInterval,
				RootDir:       c.String("root-dir"),
			}, updateRunner)

			if _, err := extRunner.DoExtensionConfigUpdate(); err != nil {
				// just log, OK to continue since this will get retry
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "initial update to fetch extensions from /config API failed")
			}

			// call UpdateAction on the updateRunner after we have fetched extensions from Fleet
			_, err := updateRunner.UpdateAction()
			if err != nil {
				// OK, initial call may fail, ok to continue
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "initial extensions update action failed")
			}

			extensionAutoLoadFile := filepath.Join(c.String("root-dir"), "extensions.load")
			stat, err := os.Stat(extensionAutoLoadFile)
			// we only want to add the extensions_autoload flag to osquery, if the file exists and size > 0
			switch {
			case err == nil:
				if stat.Size() > 0 {
					log.Debug().Msg("adding --extensions_autoload flag for file " + extensionAutoLoadFile)
					options = append(options, osquery.WithFlags([]string{"--extensions_autoload", extensionAutoLoadFile}))
				} else {
					// OK, expected as well when extensions are unloaded, just debug log
					log.Debug().Msg("found empty extensions.load file at " + extensionAutoLoadFile)
				}
			case errors.Is(err, os.ErrNotExist):
				// OK, nothing to do.
			default:
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "error with extensions.load file at "+extensionAutoLoadFile)
			}
			g.Add(extRunner.Execute, extRunner.Interrupt)
		}

		trw := token.NewReadWriter(filepath.Join(c.String("root-dir"), "identifier"))

		if err := trw.LoadOrGenerate(); err != nil {
			return fmt.Errorf("initializing token read writer: %w", err)
		}

		// note: the initial flags fetch above already populated the capabilities
		// of the server based on the response header
		if orbitClient.GetServerCapabilities().Has(fleet.CapabilityOrbitEndpoints) &&
			orbitClient.GetServerCapabilities().Has(fleet.CapabilityTokenRotation) {
			log.Info().Msg("token rotation is enabled")

			// we enable remote updates only if the server supports them by setting
			// this function.
			trw.SetRemoteUpdateFunc(func(token string) error {
				return orbitClient.SetOrUpdateDeviceToken(token)
			})

			// ensure the token value is written to the remote server, we might have
			// a token on disk that wasn't written to the server yet
			if err := trw.Write(trw.GetCached()); err != nil {
				return fmt.Errorf("writing token: %w", err)
			}

			deviceClient, err := service.NewDeviceClient(
				fleetURL,
				c.Bool("insecure"),
				c.String("fleet-certificate"),
				fleetClientCertificate,
				c.String("fleet-desktop-alternative-browser-host"),
			)
			if err != nil {
				return fmt.Errorf("initializing client: %w", err)
			}

			// perform an initial check to see if the token
			// has not been revoked by the server
			if err := deviceClient.CheckToken(trw.GetCached()); err != nil {
				if err := trw.Rotate(); err != nil {
					return fmt.Errorf("rotating token: %w", err)
				}
			}

			go func() {
				// This timer is used to check if the token should be rotated if  at
				// least one hour has passed since the last modification of the token
				// file.
				//
				// This is better than using a ticker that ticks every hour because the
				// we can't ensure the tick actually runs every hour (eg: the computer is
				// asleep).
				rotationDuration := 30 * time.Second
				rotationTicker := time.NewTicker(rotationDuration)
				defer rotationTicker.Stop()

				// This timer is used to periodically check if the token is valid. The
				// server might deem a toked as invalid for reasons out of our control,
				// for example if the database is restored to a back-up or if somebody
				// manually invalidates the token in the db.
				remoteCheckDuration := 5 * time.Minute
				remoteCheckTicker := time.NewTicker(remoteCheckDuration)
				defer remoteCheckTicker.Stop()

				for {
					select {
					case <-rotationTicker.C:
						log.Debug().Msgf("checking if token has changed or expired, cached mtime: %s", trw.GetMtime())
						hasChanged, err := trw.HasChanged()
						if err != nil {
							log.Error().Err(err).Msg("error checking if token has changed")
						}
						if hasChanged || trw.HasExpired() {
							log.Info().Msg("token TTL expired, rotating token")

							if err := trw.Rotate(); err != nil {
								log.Error().Err(err).Msg("error rotating token")
							}
						}
					case <-remoteCheckTicker.C:
						log.Debug().Msgf("initiating remote token check after %s", remoteCheckDuration)
						if err := deviceClient.CheckToken(trw.GetCached()); err != nil {
							log.Info().Err(err).Msg("periodic check of token failed, initiating rotation")

							if err := trw.Rotate(); err != nil {
								log.Error().Err(err).Msg("error rotating token")
							}
						}
					}
				}
			}()
		}

		// On Windows, where augeas doesn't work, we have a stubbed CopyLenses that always returns
		// `"", nil`. Therefore there's no platform-specific stuff required here
		augeasPath, err := augeas.CopyLenses(c.String("root-dir"))
		if err != nil {
			log.Warn().Err(err).Msg("failed to copy augeas lenses, augeas may not be available")
		} else if augeasPath != "" {
			options = append(options, osquery.WithFlags([]string{"--augeas_lenses", augeasPath}))
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

		// rootDir string, addr string, rootCA string, insecureSkipVerify bool, enrollSecret, uuid string
		checkerClient, err := service.NewOrbitClient(
			c.String("root-dir"),
			fleetURL,
			c.String("fleet-certificate"),
			c.Bool("insecure"),
			enrollSecret,
			fleetClientCertificate,
			orbitHostInfo,
		)
		if err != nil {
			return fmt.Errorf("new client for capabilities checker: %w", err)
		}
		capabilitiesChecker := newCapabilitiesChecker(checkerClient)
		g.Add(capabilitiesChecker.actor())

		registerExtensionRunner(
			&g,
			r.ExtensionSocketPath(),
			table.WithExtension(orbit_info.New(
				orbitClient,
				c.String("orbit-channel"),
				c.String("osqueryd-channel"),
				c.String("desktop-channel"),
				trw,
			)),
		)

		if c.Bool("fleet-desktop") {
			var (
				rawClientCrt []byte
				rawClientKey []byte
			)
			if fleetClientCrt != nil {
				rawClientCrt = fleetClientCrt.RawCrt
				rawClientKey = fleetClientCrt.RawKey
			}
			desktopRunner := newDesktopRunner(
				desktopPath,
				fleetURL,
				c.String("fleet-certificate"),
				c.Bool("insecure"),
				trw,
				rawClientCrt,
				rawClientKey,
				c.String("fleet-desktop-alternative-browser-host"),
				opt.RootDirectory,
			)
			g.Add(desktopRunner.actor())
		}

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		g.Add(signalHandler(ctx))

		go sigusrListener(c.String("root-dir"))

		if err := g.Run(); err != nil {
			log.Error().Err(err).Msg("unexpected exit")
		}

		close(appDoneCh) // Signal to indicate runners have just ended
		return nil
	}

	if len(os.Args) == 2 && os.Args[1] == "--help" {
		platform.PreUpdateQuirks()
	}

	if err := app.Run(os.Args); err != nil {
		log.Error().Err(err).Msg("run orbit failed")
	}
}

func registerExtensionRunner(g *run.Group, extSockPath string, opts ...table.Opt) {
	ext := table.NewRunner(extSockPath, opts...)
	g.Add(ext.Execute, ext.Interrupt)
}

// desktopRunner runs the Fleet Desktop application.
type desktopRunner struct {
	// desktopPath is the path to the desktop executable.
	desktopPath string
	updateRoot  string
	// fleetURL is the URL of the Fleet server.
	fleetURL string
	// trw is the Fleet Desktop token reader and writer (implements token rotation).
	trw *token.ReadWriter
	// fleetRootCA is the path to a certificate to use for server TLS verification.
	fleetRootCA string
	// insecure disables all TLS verification.
	insecure bool
	// fleetClientCrt is the raw TLS client certificate (in PEM format)
	// to use for authenticating to the Fleet server.
	fleetClientCrt []byte
	// fleetClientKey is the raw TLS client private key (in PEM format)
	// to use for authenticating to the Fleet server.
	fleetClientKey []byte
	// fleetAlternativeBrowserHost is an alternative host:port to use for
	// the browser URLs in Fleet Desktop.
	fleetAlternativeBrowserHost string
	// interruptCh is closed when interrupt is triggered.
	interruptCh chan struct{} //
	// executeDoneCh is closed when execute returns.
	executeDoneCh chan struct{}
}

func newDesktopRunner(
	desktopPath, fleetURL, fleetRootCA string,
	insecure bool,
	trw *token.ReadWriter,
	fleetClientCrt []byte, fleetClientKey []byte,
	fleetAlternativeBrowserHost string,
	updateRoot string,
) *desktopRunner {
	return &desktopRunner{
		desktopPath:                 desktopPath,
		updateRoot:                  updateRoot,
		fleetURL:                    fleetURL,
		trw:                         trw,
		fleetRootCA:                 fleetRootCA,
		insecure:                    insecure,
		fleetClientCrt:              fleetClientCrt,
		fleetClientKey:              fleetClientKey,
		fleetAlternativeBrowserHost: fleetAlternativeBrowserHost,
		interruptCh:                 make(chan struct{}),
		executeDoneCh:               make(chan struct{}),
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

	if err := platform.SignalProcessBeforeTerminate(constant.DesktopAppExecName); err != nil &&
		!errors.Is(err, platform.ErrProcessNotFound) &&
		!errors.Is(err, platform.ErrComChannelNotFound) {
		log.Error().Err(err).Msg("desktop early terminate")
	}

	log.Info().Str("path", d.desktopPath).Msg("opening")
	url, err := url.Parse(d.fleetURL)
	if err != nil {
		return fmt.Errorf("invalid fleet-url: %w", err)
	}
	deviceURL, err := url.Parse(d.fleetURL)
	if err != nil {
		return fmt.Errorf("invalid fleet-url: %w", err)
	}
	deviceURL.Path = path.Join(url.Path, "device", d.trw.GetCached())
	opts := []execuser.Option{
		execuser.WithEnv("FLEET_DESKTOP_FLEET_URL", url.String()),
		execuser.WithEnv("FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH", d.trw.Path),

		// TODO(roperzh): this env var is keept only for backwards compatibility,
		// we should remove it once we think is safe
		execuser.WithEnv("FLEET_DESKTOP_DEVICE_URL", deviceURL.String()),

		execuser.WithEnv("FLEET_DESKTOP_FLEET_TLS_CLIENT_CERTIFICATE", string(d.fleetClientCrt)),
		execuser.WithEnv("FLEET_DESKTOP_FLEET_TLS_CLIENT_KEY", string(d.fleetClientKey)),

		execuser.WithEnv("FLEET_DESKTOP_ALTERNATIVE_BROWSER_HOST", d.fleetAlternativeBrowserHost),
		execuser.WithEnv("FLEET_DESKTOP_TUF_UPDATE_ROOT", d.updateRoot),
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

	if err := platform.SignalProcessBeforeTerminate(constant.DesktopAppExecName); err != nil {
		log.Error().Err(err).Msg("SignalProcessBeforeTerminate")
	}
}

// hostInfo is used to parse osquery JSON output from `system_info` and `os_version` tables.
type hostInfo struct {
	// HardwareUUID is the unique identifier for this device (extracted from `system_info` osquery table).
	HardwareUUID string `json:"uuid"`
	// HardwareSerial is the unique serial number for this device (extracted from `system_info` osquery table).
	HardwareSerial string `json:"hardware_serial"`
	// Hostname is the device's hostname (extracted from `system_info` osquery table).
	Hostname string `json:"hostname"`
	// Platform is the device's platform as defined by osquery (extracted from `os_version` osquery table).
	Platform string `json:"platform"`
}

// getHostInfo retrieves system information about the host.
//
// On macOS and Linux it shells out to osqueryd to retrieve the information.
//
// On Windows:
//
//   - HardwareUUID is retrieved by shelling out to wmic, if that fails
//     then the windows API are used.
//   - HardwareSerial is currently not retrieved for Windows devices.
//   - Hostname is retrieved using stdlib method.
//   - Platform is always "windows" for windows hosts.
//
// NOTE: Windows uses a different approach to retrieve the device information
// as there were issues at the time with shelling out to osquery - from what the
// team remembers it would sometimes fail due to the osquery process not being ready yet.
// A recent CI run without the Windows special-case did succeed, but since we don't
// need the serial number for Windows at the moment, we opted to keep the
// code as it is.
func getHostInfo(osqueryPath string, osqueryDBPath string) (fleet.OrbitHostInfo, error) {
	if runtime.GOOS == "windows" {
		uuidData, uuidSource, err := platform.GetSMBiosUUID()
		if err != nil {
			return fleet.OrbitHostInfo{}, err
		}
		log.Debug().Str("source", string(uuidSource)).Msg("UUID")
		// Hostname might differ from the one provided by osquery but we are sending it
		// for troubleshooting purposes and to avoid empty host entries in the UI.
		hostname, err := os.Hostname()
		if err != nil {
			return fleet.OrbitHostInfo{}, err
		}
		return fleet.OrbitHostInfo{
			HardwareUUID:   uuidData,
			HardwareSerial: "", // currently not needed for Windows.
			Hostname:       hostname,
			Platform:       "windows",
		}, nil
	}
	const systemQuery = "SELECT si.uuid, si.hardware_serial, si.hostname, os.platform FROM system_info si, os_version os"
	args := []string{
		"-S",
		"--database_path", osqueryDBPath,
		"--json", systemQuery,
	}
	log.Debug().Str("query", systemQuery).Msg("running single query")
	out, err := exec.Command(osqueryPath, args...).Output()
	if err != nil {
		return fleet.OrbitHostInfo{}, err
	}
	var info []hostInfo
	err = json.Unmarshal(out, &info)
	if err != nil {
		return fleet.OrbitHostInfo{}, err
	}
	if len(info) != 1 {
		return fleet.OrbitHostInfo{}, fmt.Errorf("invalid number of rows from system info query: %d", len(info))
	}
	return fleet.OrbitHostInfo{
		HardwareSerial: info[0].HardwareSerial,
		HardwareUUID:   info[0].HardwareUUID,
		Hostname:       info[0].Hostname,
		Platform:       info[0].Platform,
	}, nil
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

// serviceChecker is a helper to gracefully shutdown the runners group when a
// system service stop request was received.
//
// This struct and its methods are designed to play nicely with `oklog.Group`.
type serviceChecker struct {
	svcInterruptCh   chan struct{} // closed when system service stop is requested
	localInterruptCh chan struct{} // closed when serviceChecker interrupt is called
}

func newSystemChecker() *serviceChecker {
	return &serviceChecker{
		svcInterruptCh:   make(chan struct{}),
		localInterruptCh: make(chan struct{}),
	}
}

// execute will just return when required locally or by the system service
func (s *serviceChecker) Execute() error {
	for {
		select {
		case <-s.svcInterruptCh:
			return errors.New("os service stop request")
		case <-s.localInterruptCh:
			return errors.New("internal service interrupt")
		}
	}
}

func (s *serviceChecker) Interrupt(err error) {
	log.Debug().Err(err).Msg("interrupt serviceChecker")
	close(s.localInterruptCh) // Signal execute to return.
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
	capabilitiesCheckTicker := time.NewTicker(5 * time.Minute)

	// do an initial ping to store the initial capabilities
	if err := f.client.Ping(); err != nil {
		logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "pinging the server")
	}

	for {
		select {
		case <-capabilitiesCheckTicker.C:
			oldCapabilities := f.client.GetServerCapabilities()
			// ping the server to get the latest capabilities
			if err := f.client.Ping(); err != nil {
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "pinging the server")
				continue
			}
			newCapabilities := f.client.GetServerCapabilities()

			if oldCapabilities.Has(fleet.CapabilityOrbitEndpoints) !=
				newCapabilities.Has(fleet.CapabilityOrbitEndpoints) {
				log.Info().Msgf("%s capability changed, restarting", fleet.CapabilityOrbitEndpoints)
				return nil
			}
			if oldCapabilities.Has(fleet.CapabilityTokenRotation) !=
				newCapabilities.Has(fleet.CapabilityTokenRotation) {
				log.Info().Msgf("%s capability changed, restarting", fleet.CapabilityTokenRotation)
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

// writeSecret writes the orbit enroll secret to the designated file. We do
// this at runtime for packages that are using --use-system-config, since they
// don't contain a secret file in their payload.
//
// This implementation is very similar to the one in orbit/pkg/packaging but
// intentionally kept separate to prevent issues since the writes happen at two
// completely different circumstances.
func writeSecret(enrollSecret string, orbitRoot string) error {
	path := filepath.Join(orbitRoot, constant.OsqueryEnrollSecretFileName)
	if err := secure.MkdirAll(filepath.Dir(path), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := os.WriteFile(path, []byte(enrollSecret), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
