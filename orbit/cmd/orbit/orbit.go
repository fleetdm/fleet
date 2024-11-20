package main

import (
	"bytes"
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
	"github.com/fleetdm/fleet/v4/orbit/pkg/installer"
	"github.com/fleetdm/fleet/v4/orbit/pkg/keystore"
	"github.com/fleetdm/fleet/v4/orbit/pkg/logging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osquery"
	"github.com/fleetdm/fleet/v4/orbit/pkg/osservice"
	"github.com/fleetdm/fleet/v4/orbit/pkg/platform"
	"github.com/fleetdm/fleet/v4/orbit/pkg/profiles"
	setupexperience "github.com/fleetdm/fleet/v4/orbit/pkg/setup_experience"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/fleetd_logs"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/orbit_info"
	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update/filestore"
	"github.com/fleetdm/fleet/v4/orbit/pkg/user"
	"github.com/fleetdm/fleet/v4/pkg/certificate"
	"github.com/fleetdm/fleet/v4/pkg/file"
	retrypkg "github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/pkg/secure"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/google/uuid"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"gopkg.in/natefinch/lumberjack.v2"
)

// unusedFlagKeyword is used by the MSI installer to populate parameters, which cannot be empty
const unusedFlagKeyword = "dummy"

type logError string

const (
	logErrorLaunchServicesSubstr logError = "error=Error Domain=NSOSStatusErrorDomain Code=-10822"
	logErrorLaunchServicesMsg    logError = "LaunchServices kLSServerCommunicationErr (-10822)"
	logErrorMissingExecSubstr    logError = "The application cannot be opened because its executable is missing."
	logErrorMissingExecMsg       logError = "bad desktop executable"
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
			Usage:   "Path to file containing enroll secret. On macOS and Windows, this file will be deleted and secret will be stored in the system keystore",
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
			Usage:   "How often to check for updates. Note: fleetd checks for updates at startup. The next update check adds some randomization and may take up to 10 minutes longer",
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
			Name:    "log-file",
			Usage:   "Log to this file path in addition to stderr",
			EnvVars: []string{"ORBIT_LOG_FILE"},
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
		&cli.StringFlag{
			Name:    "host-identifier",
			Usage:   "Sets the host identifier that orbit and osquery will use when enrolling to Fleet. Options: 'uuid' and 'instance' (requires Fleet >= v4.42.0)",
			EnvVars: []string{"ORBIT_HOST_IDENTIFIER"},
			Value:   "uuid",
		},
		&cli.StringFlag{
			Name:    "end-user-email",
			Hidden:  true, // experimental feature, we don't want to show it for now
			Usage:   "Sets the email address of the user associated with the host when enrolling to Fleet. (requires Fleet >= v4.43.0)",
			EnvVars: []string{"ORBIT_END_USER_EMAIL"},
		},
		&cli.BoolFlag{
			Name:    "disable-keystore",
			Usage:   "Disables the use of the keychain on macOS and Credentials Manager on Windows",
			EnvVars: []string{"ORBIT_DISABLE_KEYSTORE"},
		},
		&cli.StringFlag{
			Name:    "osquery-db",
			Usage:   "Sets a custom osquery database directory, it must be an absolute path",
			EnvVars: []string{"ORBIT_OSQUERY_DB"},
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
		startTime := time.Now()

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
				// "write /dev/stderr: The handle is invalid" (see
				// #3100). Thus, we log to the logFile only.
				log.Logger = log.Output(zerolog.MultiLevelWriter(
					zerolog.ConsoleWriter{Out: logFile, TimeFormat: time.RFC3339Nano, NoColor: true},
					&fleetd_logs.Logger,
				))
			} else {
				log.Logger = log.Output(zerolog.MultiLevelWriter(
					zerolog.ConsoleWriter{Out: logFile, TimeFormat: time.RFC3339Nano, NoColor: true},
					zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano, NoColor: true},
					&fleetd_logs.Logger,
				))
			}
		} else {
			log.Logger = log.Output(zerolog.MultiLevelWriter(
				zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano, NoColor: true},
				&fleetd_logs.Logger,
			))
		}

		zerolog.SetGlobalLevel(zerolog.InfoLevel)

		if c.Bool("debug") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		// Override flags with values retrieved from Fleet.
		fallbackServerOverridesCfg := setServerOverrides(c)
		if !fallbackServerOverridesCfg.empty() {
			log.Debug().Msgf("fallback settings: %+v", fallbackServerOverridesCfg)
		}

		if c.Bool("insecure") && c.String("fleet-certificate") != "" {
			return errors.New("insecure and fleet-certificate may not be specified together")
		}

		if c.Bool("insecure") && c.String("update-tls-certificate") != "" {
			return errors.New("insecure and update-tls-certificate may not be specified together")
		}

		if odb := c.String("osquery-db"); odb != "" && !filepath.IsAbs(odb) {
			return fmt.Errorf("the osquery database must be an absolute path: %q", odb)
		}

		readEnrollSecretFromFile := func(enrollSecretPath string) error {
			// Read secret from file. If secret is found and keystore enabled, write/overwrite the secret to the keystore and delete the file.
			b, err := os.ReadFile(enrollSecretPath)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) || !keystore.Supported() || c.Bool("disable-keystore") {
					return fmt.Errorf("read enroll secret file: %w", err)
				}
			} else {
				secret := strings.TrimSpace(string(b))
				if err = c.Set("enroll-secret", secret); err != nil {
					return fmt.Errorf("set enroll secret from file: %w", err)
				}
				if keystore.Supported() && !c.Bool("disable-keystore") {
					// Check if secret is already in the keystore.
					secretFromKeystore, err := keystore.GetSecret()
					if err != nil { //nolint:gocritic // ignore ifElseChain
						log.Warn().Err(err).Msgf("failed to retrieve enroll secret from %v", keystore.Name())
					} else if secretFromKeystore == "" {
						// Keystore secret not found, so we will add it to the keystore.
						if err = keystore.AddSecret(secret); err != nil {
							log.Warn().Err(err).Msgf("failed to add enroll secret to %v", keystore.Name())
						} else {
							// Sanity check that the secret was added to the keystore.
							checkSecret, err := keystore.GetSecret()
							if err != nil { //nolint:gocritic // ignore ifElseChain
								log.Warn().Err(err).Msgf("failed to check that enroll secret was saved in %v", keystore.Name())
							} else if checkSecret != secret {
								log.Warn().Msgf("enroll secret was not saved correctly in %v", keystore.Name())
							} else {
								log.Info().Msgf("added enroll secret to keystore: %v", keystore.Name())
								deleteSecretPathIfExists(enrollSecretPath)
							}
						}
					} else if secretFromKeystore != secret {
						// Keystore secret found, but needs to be updated.
						if err = keystore.UpdateSecret(secret); err != nil {
							log.Warn().Err(err).Msgf("failed to update enroll secret in %v", keystore.Name())
						} else {
							// Sanity check that the secret was updated in the keystore.
							checkSecret, err := keystore.GetSecret()
							if err != nil { //nolint:gocritic // ignore ifElseChain
								log.Warn().Err(err).Msgf("failed to check that enroll secret was updated in %v", keystore.Name())
							} else if checkSecret != secret {
								log.Warn().Msgf("enroll secret was not updated correctly in %v", keystore.Name())
							} else {
								log.Info().Msgf("updated enroll secret in keystore: %v", keystore.Name())
								deleteSecretPathIfExists(enrollSecretPath)
							}
						}
					} else {
						// Keystore secret found, and it matches the secret from the file.
						deleteSecretPathIfExists(enrollSecretPath)
					}
				}
			}
			return nil
		}
		enrollSecretPath := c.String("enroll-secret-path")
		if enrollSecretPath != "" {
			if c.String("enroll-secret") != "" {
				return errors.New("enroll-secret and enroll-secret-path may not be specified together")
			}
			if err := readEnrollSecretFromFile(enrollSecretPath); err != nil {
				return err
			}
		}
		tryReadEnrollSecretFromKeystore := func() error {
			if c.String("enroll-secret") == "" && keystore.Supported() && !c.Bool("disable-keystore") {
				secret, err := keystore.GetSecret()
				if err != nil || secret == "" {
					return fmt.Errorf("failed to retrieve enroll secret from %v: %w", keystore.Name(), err)
				}
				log.Info().Msgf("found enroll secret in keystore: %v", keystore.Name())
				if err = c.Set("enroll-secret", secret); err != nil {
					return fmt.Errorf("set enroll secret from keystore: %w", err)
				}
			}
			return nil
		}
		if !(runtime.GOOS == "darwin" && c.Bool("use-system-configuration")) {
			if err := tryReadEnrollSecretFromKeystore(); err != nil {
				return err
			}
		}

		if hostIdentifier := c.String("host-identifier"); hostIdentifier != "uuid" && hostIdentifier != "instance" {
			return fmt.Errorf("--host-identifier=%s is not supported, currently supported values are 'uuid' and 'instance'", hostIdentifier)
		}

		if email := c.String("end-user-email"); email != "" && email != unusedFlagKeyword && !fleet.IsLooseEmail(email) {
			return fmt.Errorf("the provided end-user email address %q is not a valid email address", email)
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
					if err := writeFleetURL(config.FleetURL, c.String("root-dir")); err != nil {
						return fmt.Errorf("write fleet URL: %w", err)
					}
				}

				if c.String("fleet-url") != "" && c.String("enroll-secret") != "" {
					log.Info().Msg("found configuration values in system profile")
					break
				}

				// If we didn't find the configuration values, try to read them from the stored files.
				// First, get Fleet URL
				b, err := os.ReadFile(path.Join(c.String("root-dir"), constant.FleetURLFileName))
				if err != nil {
					if !errors.Is(err, os.ErrNotExist) {
						return fmt.Errorf("read fleet URL file: %w", err)
					}
				} else {
					fleetURL := strings.TrimSpace(string(b))
					if err = c.Set("fleet-url", fleetURL); err != nil {
						return fmt.Errorf("set fleet URL from file: %w", err)
					}
				}
				// Now, get enroll secret
				if err := readEnrollSecretFromFile(path.Join(c.String("root-dir"), constant.OsqueryEnrollSecretFileName)); err != nil {
					return err
				}
				// Since the normal enroll secret flow supports keychain, we can use it here as well.
				// The story to remove the enroll secret from macOS MDM profile is: https://github.com/fleetdm/fleet/issues/16118
				if err := tryReadEnrollSecretFromKeystore(); err != nil {
					// Log the error but don't return it, as we want to keep trying to read the configuration
					// from the system profile.
					log.Error().Err(err).Msg("failed to read enroll secret from keystore")
				}
				if c.String("fleet-url") != "" && c.String("enroll-secret") != "" {
					log.Info().Msg("found configuration values in local files")
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
				opt.Targets[constant.DesktopTUFTargetName] = update.DesktopMacOSTarget
			case "windows":
				opt.Targets[constant.DesktopTUFTargetName] = update.DesktopWindowsTarget
			case "linux":
				if runtime.GOARCH == "arm64" {
					opt.Targets[constant.DesktopTUFTargetName] = update.DesktopLinuxArm64Target
				} else {
					opt.Targets[constant.DesktopTUFTargetName] = update.DesktopLinuxTarget
				}
			default:
				log.Fatal().Str("GOOS", runtime.GOOS).Msg("unsupported GOOS for desktop target")
			}
			// Override default channel with the provided value.
			opt.Targets.SetTargetChannel(constant.DesktopTUFTargetName, c.String("desktop-channel"))
		}

		// Override default channels with the provided values.
		opt.Targets.SetTargetChannel(constant.OrbitTUFTargetName, c.String("orbit-channel"))
		opt.Targets.SetTargetChannel(constant.OsqueryTUFTargetName, c.String("osqueryd-channel"))

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

		// Initializing windows service runner and system service manager.
		if runtime.GOOS == "windows" {
			systemChecker := newSystemChecker()
			addSubsystem(&g, "system checker", systemChecker)
			go osservice.SetupServiceManagement(constant.SystemServiceName, systemChecker.svcInterruptCh, appDoneCh)
		}

		// sofwareupdated is a macOS daemon that automatically updates Apple software.
		if c.Bool("disable-kickstart-softwareupdated") && runtime.GOOS == "darwin" {
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
				log.Info().Err(err).Msg("update metadata, using saved metadata")
			}

			signaturesExpiredAtStartup := updater.SignaturesExpired()
			if signaturesExpiredAtStartup {
				log.Info().Err(err).Msg("detected signatures expired at startup")
			}

			targets := []string{constant.OrbitTUFTargetName, constant.OsqueryTUFTargetName}

			if c.Bool("fleet-desktop") {
				targets = append(targets, constant.DesktopTUFTargetName)
			}
			if c.Bool("dev-mode") {
				targets = targets[1:] // exclude orbit itself on dev-mode.
			}
			updateRunner, err = update.NewRunner(updater, update.RunnerOptions{
				CheckInterval:              c.Duration("update-interval"),
				Targets:                    targets,
				SignaturesExpiredAtStartup: signaturesExpiredAtStartup,
			})
			if err != nil {
				return err
			}

			// Get current version of osquery
			log.Info().Msgf("orbit version: %s", build.Version)
			osquerydPath, err = updater.ExecutableLocalPath(constant.OsqueryTUFTargetName)
			if err != nil {
				log.Info().Err(err).Msg("Could not find local osqueryd executable")
			} else {
				version, err := update.GetVersion(osquerydPath)
				if err == nil && version != "" {
					log.Info().Msgf("Found osquery version: %s", version)
					updateRunner.OsqueryVersion = version
				}
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

			addSubsystem(&g, "update runner", updateRunner)

			// if getting any of the targets fails, keep on
			// retrying, the `updater.Get` method has built-in backoff functionality.
			//
			// NOTE: it used to be the case that we would return an
			// error on the first attempt here, causing orbit to
			// restart. This was changed to have control over
			// how/when we want to retry to download the packages.
			err = retrypkg.Do(func() error {
				var err error
				osquerydPath, desktopPath, err = getFleetdComponentPaths(c, updater, fallbackServerOverridesCfg)
				if err != nil {
					return err
				}
				return nil
			},
				// retry every 5 minutes to not flood the logs,
				// but actual pings to the remote server are
				// handled by `updater.Get`
				retrypkg.WithInterval(5*time.Minute),
			)
			if err != nil {
				// this should never happen because `retry.Do` is
				// executed without a defined number of max attempts
				return fmt.Errorf("getting targets after retry: %w", err)
			}
		} else {
			log.Info().Msg("running with auto updates disabled")
			updater = update.NewDisabled(opt)
			osquerydPath, err = updater.ExecutableLocalPath(constant.OsqueryTUFTargetName)
			if err != nil {
				log.Fatal().Err(err).Msgf("locate %s", constant.OsqueryTUFTargetName)
			}
			if c.Bool("fleet-desktop") {
				if runtime.GOOS == "darwin" {
					desktopPath, err = updater.DirLocalPath(constant.DesktopTUFTargetName)
					if err != nil {
						return fmt.Errorf("get %s target: %w", constant.DesktopTUFTargetName, err)
					}
				} else {
					desktopPath, err = updater.ExecutableLocalPath(constant.DesktopTUFTargetName)
					if err != nil {
						return fmt.Errorf("get %s target: %w", constant.DesktopTUFTargetName, err)
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

		// Kill any pre-existing instances of osqueryd (otherwise getHostInfo will fail
		// because of osqueryd's lock over its database).
		//
		// This can happen for instance when orbit is killed via the Windows Task Manager
		// (there's no SIGTERM on Windows) and the osqueryd child processes are left orphaned.
		killedProcesses, err := platform.KillAllProcessByName("osqueryd")
		if err != nil {
			log.Error().Err(err).Msg("failed to kill pre-existing instances of osqueryd")
		} else if len(killedProcesses) > 0 {
			log.Debug().Str("processes", fmt.Sprintf("%+v", killedProcesses)).Msg("existing osqueryd processes killed")
		}

		osqueryDB := filepath.Join(c.String("root-dir"), "osquery.db")
		if odb := c.String("osquery-db"); odb != "" {
			osqueryDB = odb
		}

		osqueryHostInfo, err := getHostInfo(osquerydPath, osqueryDB)
		if err != nil {
			return fmt.Errorf("get UUID: %w", err)
		}
		log.Debug().Str("info", fmt.Sprint(osqueryHostInfo)).Msg("retrieved host info from osquery")
		orbitHostInfo := fleet.OrbitHostInfo{
			HardwareSerial: osqueryHostInfo.HardwareSerial,
			HardwareUUID:   osqueryHostInfo.HardwareUUID,
			Hostname:       osqueryHostInfo.Hostname,
			Platform:       osqueryHostInfo.Platform,
			ComputerName:   osqueryHostInfo.ComputerName,
			HardwareModel:  osqueryHostInfo.HardwareModel,
		}

		if runtime.GOOS == "darwin" {
			// Get the hardware UUID. We use a temporary osquery DB location in order to guarantee that
			// we're getting true UUID, not a cached UUID. See
			// https://github.com/fleetdm/fleet/issues/17934 and
			// https://github.com/osquery/osquery/issues/7509 for more details.

			tmpDBPath := filepath.Join(os.TempDir(), strings.Join([]string{uuid.NewString(), "tmp-db"}, "-"))
			oi, err := getHostInfo(osquerydPath, tmpDBPath)
			if err != nil {
				return fmt.Errorf("get UUID from temp db: %w", err)
			}

			if err := os.RemoveAll(tmpDBPath); err != nil {
				log.Info().Err(err).Msg("failed to remove temporary osquery db")
			}

			if oi.HardwareUUID != orbitHostInfo.HardwareUUID {
				// Then we have moved to a new physical machine, so we should restart!
				// Removing the osquery DB should trigger a re-enrollment when fleetd is restarted.
				if err := os.RemoveAll(osqueryDB); err != nil {
					return fmt.Errorf("removing old osquery.db: %w", err)
				}

				// We can remove these because we want them to be regenerated during the re-enrollment.
				if err := os.RemoveAll(filepath.Join(c.String("root-dir"), constant.OrbitNodeKeyFileName)); err != nil {
					return fmt.Errorf("removing old orbit node key file: %w", err)
				}
				if err := os.RemoveAll(filepath.Join(c.String("root-dir"), constant.DesktopTokenFileName)); err != nil {
					return fmt.Errorf("removing old Fleet Desktop identifier file: %w", err)
				}

				return errors.New("found a new hardware uuid, restarting")
			}
		}

		// Only send osquery's `instance_id` if the user is running orbit with `--host-identifier=instance`.
		// When not set, orbit and osquery will be matched using the hardware UUID (orbitHostInfo.HardwareUUID).
		if c.String("host-identifier") == "instance" {
			orbitHostInfo.OsqueryIdentifier = osqueryHostInfo.InstanceID
		}

		var (
			options []osquery.Option
			// optionsAfterFlagfile is populated with options that will be set after the '--flagfile' argument
			// to not allow users to change their values on their flagfiles.
			optionsAfterFlagfile []osquery.Option
		)
		options = append(options, osquery.WithDataPath(c.String("root-dir"), ""))
		options = append(options, osquery.WithLogPath(filepath.Join(c.String("root-dir"), "osquery_log")))
		optionsAfterFlagfile = append(optionsAfterFlagfile, osquery.WithFlags(
			[]string{"--database_path", osqueryDB},
		))

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

			addSubsystem(&g, "insecure proxy", &wrapSubsystem{
				execute: func() error {
					log.Info().
						Str("addr", fmt.Sprintf("localhost:%d", proxy.Port)).
						Str("target", c.String("fleet-url")).
						Msg("using insecure TLS proxy")
					err := proxy.InsecureServeTLS()
					return err
				},
				interrupt: func(err error) {
					if err := proxy.Close(); err != nil {
						log.Error().Err(err).Msg("close proxy")
					}
				},
			})

			// Directory to store proxy related assets
			proxyDirectory := filepath.Join(c.String("root-dir"), "proxy")
			if err := secure.MkdirAll(proxyDirectory, constant.DefaultDirMode); err != nil {
				return fmt.Errorf("there was a problem creating the proxy directory: %w", err)
			}

			certPath = filepath.Join(proxyDirectory, "fleet.crt")

			// Write cert that proxy uses
			err = os.WriteFile(certPath, []byte(insecure.ServerCert), os.FileMode(0o644))
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
			&service.OnGetConfigErrFuncs{
				DebugErrFunc: func(err error) {
					log.Debug().Err(err).Msg("get config")
				},
				OnNetErrFunc: func(err error) {
					log.Info().Err(err).Msg("network error")
				},
			},
		)
		if err != nil {
			return fmt.Errorf("error new orbit client: %w", err)
		}

		// create the notifications middleware that wraps the orbit client
		// (must be shared by all runners that use a ConfigFetcher).
		const (
			renewEnrollmentProfileCommandFrequency = 3 * time.Minute
			windowsMDMEnrollmentCommandFrequency   = time.Hour
			windowsMDMBitlockerCommandFrequency    = time.Hour
		)

		scriptConfigReceiver, scriptsEnabledFn := update.ApplyRunScriptsConfigFetcherMiddleware(
			c.Bool("enable-scripts"), orbitClient, c.String("root-dir"),
		)
		orbitClient.RegisterConfigReceiver(scriptConfigReceiver)

		switch runtime.GOOS {
		case "darwin":
			orbitClient.RegisterConfigReceiver(update.ApplyRenewEnrollmentProfileConfigFetcherMiddleware(
				orbitClient, renewEnrollmentProfileCommandFrequency, fleetURL))
			const nudgeLaunchInterval = 30 * time.Minute
			orbitClient.RegisterConfigReceiver(update.ApplyNudgeConfigReceiverMiddleware(update.NudgeConfigFetcherOptions{
				UpdateRunner: updateRunner, RootDir: c.String("root-dir"), Interval: nudgeLaunchInterval,
			}))
			setupExperiencer := setupexperience.NewSetupExperiencer(orbitClient, c.String("root-dir"))
			orbitClient.RegisterConfigReceiver(setupExperiencer)
			orbitClient.RegisterConfigReceiver(update.ApplySwiftDialogDownloaderMiddleware(updateRunner))

		case "windows":
			orbitClient.RegisterConfigReceiver(update.ApplyWindowsMDMEnrollmentFetcherMiddleware(windowsMDMEnrollmentCommandFrequency, orbitHostInfo.HardwareUUID, orbitClient))
			orbitClient.RegisterConfigReceiver(update.ApplyWindowsMDMBitlockerFetcherMiddleware(windowsMDMBitlockerCommandFrequency, orbitClient))
		}

		flagUpdateReceiver := update.NewFlagReceiver(orbitClient.TriggerOrbitRestart, update.FlagUpdateOptions{
			RootDir: c.String("root-dir"),
		})
		orbitClient.RegisterConfigReceiver(flagUpdateReceiver)

		if !c.Bool("disable-updates") {
			serverOverridesReceiver := newServerOverridesReceiver(
				c.String("root-dir"),
				fallbackServerOverridesConfig{
					OsquerydPath: osquerydPath,
					DesktopPath:  desktopPath,
				},
				c.Bool("fleet-desktop"),
				orbitClient.TriggerOrbitRestart,
			)

			orbitClient.RegisterConfigReceiver(serverOverridesReceiver)
		}

		// only setup extensions autoupdate if we have enabled updates
		// for extensions autoupdate, we can only proceed after orbit is enrolled in fleet
		// and all relevant things for it (like certs, enroll secrets, tls proxy, etc) is configured
		if !c.Bool("disable-updates") || c.Bool("dev-mode") {
			extRunner := update.NewExtensionConfigUpdateRunner(update.ExtensionUpdateOptions{
				RootDir: c.String("root-dir"),
			}, updateRunner, orbitClient.TriggerOrbitRestart)

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
					// We set this option after the --flagfile to prevent users from changing it on their flagfiles.
					optionsAfterFlagfile = append(optionsAfterFlagfile, osquery.WithFlags([]string{"--extensions_autoload", extensionAutoLoadFile}))
				} else {
					// OK, expected as well when extensions are unloaded, just debug log
					log.Debug().Msg("found empty extensions.load file at " + extensionAutoLoadFile)
				}
			case errors.Is(err, os.ErrNotExist):
				// OK, nothing to do.
			default:
				logging.LogErrIfEnvNotSet(constant.SilenceEnrollLogErrorEnvVar, err, "error with extensions.load file at "+extensionAutoLoadFile)
			}

			orbitClient.RegisterConfigReceiver(extRunner)
		}

		// Run a early check of fleetd configuration to check if orbit needs to
		// restart before proceeding to start the sub-systems.
		//
		// E.g. the administrator has updated the following agent options for this device:
		//	- `update_channels`
		//	- `extensions` were removed/unset
		//	- `command_line_flags` (osquery startup flags)
		if err := orbitClient.RunConfigReceivers(); err != nil {
			log.Error().Msgf("failed initial config fetch: %s", err)
		} else if orbitClient.RestartTriggered() {
			log.Info().Msg("exiting after early config fetch")
			return nil
		}

		addSubsystem(&g, "config receivers", &wrapSubsystem{
			execute:   orbitClient.ExecuteConfigReceivers,
			interrupt: orbitClient.InterruptConfigReceivers,
		})

		var trw *token.ReadWriter
		var deviceClient *service.DeviceClient
		if c.Bool("fleet-desktop") {
			trw = token.NewReadWriter(filepath.Join(c.String("root-dir"), constant.DesktopTokenFileName))
			if err := trw.LoadOrGenerate(); err != nil {
				return fmt.Errorf("initializing token read writer: %w", err)
			}

			log.Info().Msg("token rotation is enabled")

			// we enable remote updates only if the server supports them by setting
			// this function.
			trw.SetRemoteUpdateFunc(
				func(token string) error {
					return orbitClient.SetOrUpdateDeviceToken(token)
				},
			)

			// Note that the deviceClient used by orbit must not define a retry on
			// invalid token, because its goal is to detect invalid tokens when
			// making requests with this client.
			deviceClient, err = service.NewDeviceClient(
				fleetURL,
				c.Bool("insecure"),
				c.String("fleet-certificate"),
				fleetClientCertificate,
				c.String("fleet-desktop-alternative-browser-host"),
			)
			if err != nil {
				return fmt.Errorf("initializing client: %w", err)
			}

			// Check if token is not expired and still good.
			// If not, rotate the token.
			expired, _ := trw.HasExpired()
			if expired || deviceClient.CheckToken(trw.GetCached()) != nil {
				if err := trw.Rotate(); err != nil {
					return fmt.Errorf("rotating token: %w", err)
				}
			}

			go func() {
				// This timer is used to check if the token should be rotated if at
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
						rotationTicker.Reset(rotationDuration)

						log.Debug().Msgf("checking if token has changed or expired, cached mtime: %s", trw.GetMtime())
						hasChanged, err := trw.HasChanged()
						if err != nil {
							log.Error().Err(err).Msg("error checking if token has changed")
						}

						exp, remain := trw.HasExpired()

						// rotate if the token file has been modified, if the token is
						// expired or if it is very close to expire.
						if hasChanged || exp || remain <= time.Second {
							log.Info().Msg("token TTL expired, rotating token")

							if err := trw.Rotate(); err != nil {
								log.Error().Err(err).Msg("error rotating token")
							}
						} else if remain > 0 && remain < rotationDuration {
							// check again when the token will expire, which will happen
							// before the next rotation check
							rotationTicker.Reset(remain)
							log.Debug().Msgf("token will expire soon, checking again in: %s", remain)
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

		// These options must go after '--flagfile' to not allow users to change their values
		// on their flagfiles.
		hostIdentifier := c.String("host-identifier")
		options = append(options, osquery.WithFlags([]string{"--host-identifier", hostIdentifier}))
		options = append(options, optionsAfterFlagfile...)

		// Handle additional args after '--' in the command line. These are added last and should
		// override all other flags and flagfile entries.
		options = append(options, osquery.WithFlags(c.Args().Slice()))

		// Create an osquery runner with the provided options.
		r, err := osquery.NewRunner(osquerydPath, options...)
		if err != nil {
			return fmt.Errorf("create osquery runner: %w", err)
		}
		addSubsystem(&g, "osqueryd runner", r)

		// rootDir string, addr string, rootCA string, insecureSkipVerify bool, enrollSecret, uuid string
		checkerClient, err := service.NewOrbitClient(
			c.String("root-dir"),
			fleetURL,
			c.String("fleet-certificate"),
			c.Bool("insecure"),
			enrollSecret,
			fleetClientCertificate,
			orbitHostInfo,
			&service.OnGetConfigErrFuncs{
				DebugErrFunc: func(err error) {
					log.Debug().Err(err).Msg("get config")
				},
				OnNetErrFunc: func(err error) {
					log.Info().Err(err).Msg("network error")
				},
			},
		)
		if err != nil {
			return fmt.Errorf("new client for capabilities checker: %w", err)
		}
		capabilitiesChecker := newCapabilitiesChecker(checkerClient)
		// We populate the known capabilities so that the capability checker does not need to do the initial check on startup.
		checkerClient.GetServerCapabilities().Copy(orbitClient.GetServerCapabilities())
		addSubsystem(&g, "capabilities checker", capabilitiesChecker)

		var desktopVersion string
		if c.Bool("fleet-desktop") {
			runPath := desktopPath
			if runtime.GOOS == "darwin" {
				runPath = filepath.Join(desktopPath, "Contents", "MacOS", constant.DesktopAppExecName)
			}
			desktopVersion, err = update.GetVersion(runPath)
			if err == nil && desktopVersion != "" {
				log.Info().Msgf("Found fleet-desktop version: %s", desktopVersion)
			} else {
				desktopVersion = "unknown"
			}
		}

		registerExtensionRunner(
			&g,
			r.ExtensionSocketPath(),
			table.WithExtension(orbit_info.New(
				orbitClient,
				c.String("orbit-channel"),
				c.String("osqueryd-channel"),
				c.String("desktop-channel"),
				desktopVersion,
				trw,
				startTime,
				scriptsEnabledFn,
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
			go func() {
				for {
					msg := <-desktopRunner.errorNotifyCh
					log.Error().Err(errors.New(msg)).Msg("fleet-desktop runner error")
					// Vital errors are always sent to Fleet, regardless of the error reporting setting FLEET_ENABLE_POST_CLIENT_DEBUG_ERRORS.
					fleetdErr := fleet.FleetdError{
						Vital:              true,
						ErrorSource:        "fleet-desktop",
						ErrorSourceVersion: desktopVersion,
						ErrorTimestamp:     time.Now(),
						ErrorMessage:       msg,
						ErrorAdditionalInfo: map[string]interface{}{
							"orbit_version":   build.Version,
							"osquery_version": osqueryHostInfo.OsqueryVersion,
							"os_platform":     osqueryHostInfo.Platform,
							"os_version":      osqueryHostInfo.OSVersion,
						},
					}
					if err = deviceClient.ReportError(trw.GetCached(), fleetdErr); err != nil {
						log.Error().Err(err).Msg(fmt.Sprintf("failed to send error report to Fleet: %s", msg))
					}
				}
			}()
			addSubsystem(&g, "desktop runner", desktopRunner)
		}

		// --end-user-email is only supported on Windows and Linux (for macOS it gets the
		// email from the enrollment profile)
		endUserEmail := c.String("end-user-email")
		if (runtime.GOOS == "windows" || runtime.GOOS == "linux") && endUserEmail != "" && endUserEmail != unusedFlagKeyword {
			if orbitClient.GetServerCapabilities().Has(fleet.CapabilityEndUserEmail) {
				log.Debug().Msg("sending end-user email to Fleet")
				if err := orbitClient.SetOrUpdateDeviceMappingEmail(endUserEmail); err != nil {
					log.Error().Err(err).Msg("error sending end-user email to Fleet")
				}
			} else {
				log.Info().Msg("an end-user email is provided, but the Fleet server doesn't have the capability to set it.")
			}
		}

		// For macOS hosts, check if MDM enrollment profile is present and if it contains the
		// custom end user email field. If so, report it to the server.
		if runtime.GOOS == "darwin" {
			log.Info().Msg("checking for custom mdm enrollment profile with end user email")
			email, err := profiles.GetCustomEnrollmentProfileEndUserEmail()
			if err != nil {
				if errors.Is(err, profiles.ErrNotFound) {
					// This is fine. Many hosts will not have this profile so just log and continue.
					log.Info().Msg(fmt.Sprintf("get custom enrollment profile end user email: %s", err))
				} else {
					log.Error().Err(err).Msg("get custom enrollment profile end user email")
				}
			}

			if email != "" {
				log.Info().Msg(fmt.Sprintf("found custom end user email: %s", email))
				if err := orbitClient.SetOrUpdateDeviceMappingEmail(email); err != nil {
					log.Error().Err(err).Msg(fmt.Sprintf("set or update device mapping: %s", email))
				}
			}
		}

		softwareRunner := installer.NewRunner(orbitClient, r.ExtensionSocketPath(), scriptsEnabledFn, c.String("root-dir"))
		orbitClient.RegisterConfigReceiver(softwareRunner)

		if runtime.GOOS == "darwin" {
			log.Info().Msgf("orbitClient.GetServerCapabilities() %+v", orbitClient.GetServerCapabilities())
			if orbitClient.GetServerCapabilities().Has(fleet.CapabilityEscrowBuddy) {
				orbitClient.RegisterConfigReceiver(update.NewEscrowBuddyRunner(updateRunner, 5*time.Minute))
			} else {
				orbitClient.RegisterConfigReceiver(
					update.ApplyDiskEncryptionRunnerMiddleware(
						orbitClient.GetServerCapabilities,
						orbitClient.TriggerOrbitRestart,
					),
				)
			}
		}

		// Install a signal handler
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		signalHandlerExecute, signalHandlerInterrupt := signalHandler(ctx)
		addSubsystem(&g, "signal handler", &wrapSubsystem{
			execute:   signalHandlerExecute,
			interrupt: signalHandlerInterrupt,
		})

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

func deleteSecretPathIfExists(enrollSecretPath string) {
	// Since the secret is in the keystore, we can delete the original secret file if it exists
	if _, err := os.Stat(enrollSecretPath); err == nil {
		log.Info().Msgf("deleting enroll secret file: %v", enrollSecretPath)
		err = os.Remove(enrollSecretPath)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to delete enroll secret file: %v", enrollSecretPath)
		}
	}
}

// setServerOverrides overrides specific variables in c with values fetched from Fleet.
func setServerOverrides(c *cli.Context) fallbackServerOverridesConfig {
	overrideCfg, err := loadServerOverrides(c.String("root-dir"))
	if err != nil {
		log.Error().Err(err).Msg("failed to load server overrides")
		return fallbackServerOverridesConfig{}
	}
	if overrideCfg.OrbitChannel != "" {
		if err := c.Set("orbit-channel", overrideCfg.OrbitChannel); err != nil {
			log.Error().Err(err).Str("component", constant.OrbitTUFTargetName).Msg("failed to set server overrides")
		}
	}
	if overrideCfg.OsquerydChannel != "" {
		if err := c.Set("osqueryd-channel", overrideCfg.OsquerydChannel); err != nil {
			log.Error().Err(err).Str("component", constant.OsqueryTUFTargetName).Msg("failed to set server overrides")
		}
	}
	if overrideCfg.DesktopChannel != "" {
		if err := c.Set("desktop-channel", overrideCfg.DesktopChannel); err != nil {
			log.Error().Err(err).Str("component", constant.DesktopTUFTargetName).Msg("failed to set server overrides")
		}
	}

	return overrideCfg.fallbackServerOverridesConfig
}

// getFleetdComponentPaths returns the paths of the fleetd components.
// If the path to the component cannot be fetched using the updater (e.g. channel doesn't exist yet)
// then it will use the fallbackCfg's paths (if set).
func getFleetdComponentPaths(
	c *cli.Context,
	updater *update.Updater,
	fallbackCfg fallbackServerOverridesConfig,
) (osquerydPath string, desktopPath string, err error) {
	if err := updater.UpdateMetadata(); err != nil {
		log.Error().Err(err).Msg("update metadata before getting components")
	}

	// "root", "targets", or "snapshot" signatures have expired, thus
	// we attempt to get local paths for the targets (updater.Get will fail
	// because of the expired signatures).
	if updater.SignaturesExpired() {
		log.Error().Err(err).Msg("expired metadata, using local targets")

		// Attempt to get local path of osqueryd.
		osquerydPath, err = updater.ExecutableLocalPath(constant.OsqueryTUFTargetName)
		if err != nil {
			log.Info().Err(err).Msgf("failed to get local path for %s target", constant.OsqueryTUFTargetName)
			// Attempt to use fallback path.
			if fallbackCfg.OsquerydPath == "" {
				log.Info().Err(err).Msgf("no fallback local path for %s", constant.OsqueryTUFTargetName)
				return "", "", fmt.Errorf("get local %s target: %w", constant.OsqueryTUFTargetName, err)
			}
			log.Info().Err(err).Msgf("get local %s target failed, fallback to using %s", constant.OsqueryTUFTargetName, fallbackCfg.OsquerydPath)
			osquerydPath = fallbackCfg.OsquerydPath
		}
		// Attempt to get local path of Fleet Desktop.
		if c.Bool("fleet-desktop") {
			if runtime.GOOS == "darwin" {
				desktopPath, err = updater.DirLocalPath(constant.DesktopTUFTargetName)
			} else {
				desktopPath, err = updater.ExecutableLocalPath(constant.DesktopTUFTargetName)
			}
			if err != nil {
				log.Info().Err(err).Msgf("failed to get local path for %s target", constant.DesktopTUFTargetName)
				// Attempt to use fallback path.
				if fallbackCfg.DesktopPath == "" {
					log.Info().Err(err).Msgf("no fallback local path for %s", constant.DesktopTUFTargetName)
					return "", "", fmt.Errorf("get local %s target: %w", constant.DesktopTUFTargetName, err)
				}
				log.Info().Err(err).Msgf("get local %s target failed, fallback to using %s", constant.DesktopTUFTargetName, fallbackCfg.DesktopPath)
				desktopPath = fallbackCfg.DesktopPath
			}
		}

		return osquerydPath, desktopPath, nil
	}

	// osqueryd
	osquerydLocalTarget, err := updater.Get(constant.OsqueryTUFTargetName)
	if err != nil {
		if fallbackCfg.OsquerydPath == "" {
			log.Info().Err(err).Msgf("get %s target failed", constant.OsqueryTUFTargetName)
			return "", "", fmt.Errorf("get %s target: %w", constant.OsqueryTUFTargetName, err)
		}
		log.Info().Err(err).Msgf("get %s target failed, fallback to using %s", constant.OsqueryTUFTargetName, fallbackCfg.OsquerydPath)
		osquerydPath = fallbackCfg.OsquerydPath
	} else {
		osquerydPath = osquerydLocalTarget.ExecPath
	}

	// Fleet Desktop
	if c.Bool("fleet-desktop") {
		fleetDesktopLocalTarget, err := updater.Get(constant.DesktopTUFTargetName)
		if err != nil {
			if fallbackCfg.DesktopPath == "" {
				log.Info().Err(err).Msgf("get %s target failed", constant.DesktopTUFTargetName)
				return "", "", fmt.Errorf("get %s target: %w", constant.DesktopTUFTargetName, err)
			}
			log.Info().Err(err).Msgf("get %s target failed, fallback to using %s", constant.DesktopTUFTargetName, fallbackCfg.DesktopPath)
			desktopPath = fallbackCfg.DesktopPath
		} else {
			if runtime.GOOS == "darwin" {
				desktopPath = fleetDesktopLocalTarget.DirPath
			} else {
				desktopPath = fleetDesktopLocalTarget.ExecPath
			}
		}
	}

	return osquerydPath, desktopPath, nil
}

func registerExtensionRunner(g *run.Group, extSockPath string, opts ...table.Opt) {
	ext := table.NewRunner(extSockPath, opts...)
	addSubsystem(g, "osqueryd extension runner", ext)
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
	// errorNotifyCh is used to notify errors to the main orbit process.
	errorNotifyCh chan string
	// errorsReported is used to keep track of errors already reported to the main orbit process.
	errorsReported map[string]struct{}
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
		errorNotifyCh:               make(chan string),
	}
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
func (d *desktopRunner) Execute() error {
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
			// On MacOS, if we attempt to run Fleet Desktop while the user is not logged in through
			// the GUI, MacOS returns an error. See https://github.com/fleetdm/fleet/issues/14698
			// for more details.
			loggedInGui, err := user.IsUserLoggedInViaGui()
			if err != nil {
				log.Debug().Err(err).Msg("desktop.IsUserLoggedInGui")
				return true
			}

			if !loggedInGui {
				return true
			}

			// Orbit runs as root user on Unix and as SYSTEM (Windows Service) user on Windows.
			// To be able to run the desktop application (mostly to register the icon in the system tray)
			// we need to run the application as the login user.
			// Package execuser provides multi-platform support for this.
			if lastLogs, err := execuser.Run(d.desktopPath, opts...); err != nil {
				log.Debug().Err(err).Msg("execuser.Run")
				d.processLog(lastLogs)
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

func (d *desktopRunner) Interrupt(err error) {
	close(d.interruptCh) // Signal execute to return.
	<-d.executeDoneCh    // Wait for execute to return.

	if err := platform.SignalProcessBeforeTerminate(constant.DesktopAppExecName); err != nil {
		log.Error().Err(err).Msg("SignalProcessBeforeTerminate")
	}
}

func (d *desktopRunner) processLog(log string) {
	if len(log) == 0 {
		return
	}
	// Important: make sure msg does not contain sensitive information since it is used for analytics.
	var msg string
	switch {
	case strings.Contains(log, string(logErrorLaunchServicesSubstr)):
		// https://github.com/fleetdm/fleet/issues/19172
		msg = string(logErrorLaunchServicesMsg)
	case strings.Contains(log, string(logErrorMissingExecSubstr)):
		// For manual testing.
		// To get this message, delete Fleet Desktop.app directory, make an empty Fleet Desktop.app directory,
		// and kill the fleet-desktop process. Orbit will try to re-start Fleet Desktop and log this message.
		msg = string(logErrorMissingExecMsg)
	}
	if msg == "" {
		return
	}
	if d.errorsReported == nil {
		d.errorsReported = make(map[string]struct{})
	}
	if _, ok := d.errorsReported[msg]; !ok {
		d.errorsReported[msg] = struct{}{}
		d.errorNotifyCh <- msg
	}
}

// osqueryHostInfo is used to parse osquery JSON output from system tables.
type osqueryHostInfo struct {
	// HardwareUUID is the unique identifier for this device (extracted from `system_info` osquery table).
	HardwareUUID string `json:"uuid"`
	// HardwareSerial is the unique serial number for this device (extracted from `system_info` osquery table).
	HardwareSerial string `json:"hardware_serial"`
	// Hostname is the device's hostname (extracted from `system_info` osquery table).
	Hostname string `json:"hostname"`
	// ComputerName is the friendly computer name (optional) (extracted from `system_info` osquery table).
	ComputerName string `json:"computer_name"`
	// HardwareModel is the device's hardware model (extracted from `system_info` osquery table).
	HardwareModel string `json:"hardware_model"`
	// Platform is the device's platform as defined by osquery (extracted from `os_version` osquery table).
	Platform string `json:"platform"`
	// InstanceID is the osquery's randomly generated instance ID
	// (extracted from `osquery_info` osquery table).
	InstanceID string `json:"instance_id"`
	// OSVersion is the device's OS version as defined by osquery (extracted from `os_version` osquery table).
	OSVersion string `json:"os_version"`
	// OsqueryVersion is the version of osquery running on the device (extracted from `osquery_info` osquery table).
	OsqueryVersion string `json:"osquery_version"`
}

// getHostInfo retrieves system information about the host by shelling out to `osqueryd -S` and performing a `SELECT` query.
func getHostInfo(osqueryPath string, osqueryDBPath string) (*osqueryHostInfo, error) {
	// Make sure parent directory exists (`osqueryd -S` doesn't create the parent directories).
	if err := os.MkdirAll(filepath.Dir(osqueryDBPath), constant.DefaultDirMode); err != nil {
		return nil, err
	}
	const systemQuery = `
	SELECT
		si.uuid,
		si.hardware_serial,
		si.hostname,
		si.computer_name,
		si.hardware_model,
		os.platform,
		os.version as os_version,
		oi.instance_id,
		oi.version as osquery_version
	FROM system_info si, os_version os, osquery_info oi`
	args := []string{
		"-S",
		"--database_path", osqueryDBPath,
		"--json", systemQuery,
	}
	log.Debug().Str("query", systemQuery).Msg("running single query")
	cmd := exec.Command(osqueryPath, args...)
	var (
		osquerydStdout bytes.Buffer
		osquerydStderr bytes.Buffer
	)
	cmd.Stdout = &osquerydStdout
	cmd.Stderr = &osquerydStderr
	var info []osqueryHostInfo
	if err := cmd.Run(); err != nil {
		// osquery may return correct data with an exit status 78, in which case we only log the error
		// Related issue: https://github.com/osquery/osquery/issues/6566
		unmarshalErr := json.Unmarshal(osquerydStdout.Bytes(), &info)
		// Note: Unmarshal will fail on an empty buffer output.
		if unmarshalErr != nil {
			// Since the original command failed, we log the original error and the output for debugging purposes.
			log.Error().Str(
				"output", osquerydStdout.String(),
			).Str(
				"stderr", osquerydStderr.String(),
			).Msg("getHostInfo via osquery")
			return nil, err
		}
		log.Warn().Str("status", err.Error()).Msg("getHostInfo via osquery returned data, but with a non-zero exit status")
	}
	if len(info) == 0 {
		if err := json.Unmarshal(osquerydStdout.Bytes(), &info); err != nil {
			return nil, err
		}
	}
	if len(info) != 1 {
		return nil, fmt.Errorf("invalid number of rows from system info query: %d", len(info))
	}
	return &info[0], nil
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

// execute will poll the server for capabilities and emit a stop signal to restart
// Orbit if certain capabilities are enabled.
//
// You need to add an explicit check for each capability you want to watch for
func (f *capabilitiesChecker) Execute() error {
	defer close(f.executeDoneCh)
	capabilitiesCheckTicker := time.NewTicker(5 * time.Minute)

	// Do an initial ping to store the initial capabilities if needed
	if len(f.client.GetServerCapabilities()) == 0 {
		if err := f.client.Ping(); err != nil {
			logging.LogErrIfEnvNotSetDebug(constant.SilenceEnrollLogErrorEnvVar, err, "pinging the server")
		}
	}

	for {
		select {
		case <-capabilitiesCheckTicker.C:
			oldCapabilities := f.client.GetServerCapabilities()
			// ping the server to get the latest capabilities
			if err := f.client.Ping(); err != nil {
				logging.LogErrIfEnvNotSetDebug(constant.SilenceEnrollLogErrorEnvVar, err, "pinging the server")
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
			if oldCapabilities.Has(fleet.CapabilityEndUserEmail) !=
				newCapabilities.Has(fleet.CapabilityEndUserEmail) {
				log.Info().Msgf("%s capability changed, restarting", fleet.CapabilityEndUserEmail)
				return nil
			}
		case <-f.interruptCh:
			return nil

		}
	}
}

func (f *capabilitiesChecker) Interrupt(err error) {
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
	return writeOrbitFile(enrollSecret, orbitRoot, constant.OsqueryEnrollSecretFileName)
}

func writeOrbitFile(contents string, orbitRoot string, fileName string) error {
	filePath := filepath.Join(orbitRoot, fileName)
	if err := secure.MkdirAll(filepath.Dir(filePath), constant.DefaultDirMode); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(contents), constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// writeFleetURL writes the Fleet URL to the designated file. This is needed in case the
// Fleet URL originally came from a config profile, which was subsequently removed.
func writeFleetURL(contents string, orbitRoot string) error {
	return writeOrbitFile(contents, orbitRoot, constant.FleetURLFileName)
}

// serverOverridesRunner is a oklog.Group runner that polls for configuration overrides from Fleet.
type serverOverridesRunner struct {
	rootDir             string
	fallbackCfg         fallbackServerOverridesConfig
	desktopEnabled      bool
	cancel              chan struct{}
	triggerOrbitRestart func(reason string)
}

// newServerOverridesReveiver creates a runner for updating server overrides configuration with values fetched from Fleet.
func newServerOverridesReceiver(
	rootDir string,
	fallbackCfg fallbackServerOverridesConfig,
	desktopEnabled bool,
	triggerOrbitRestart func(reason string),
) *serverOverridesRunner {
	return &serverOverridesRunner{
		rootDir:             rootDir,
		fallbackCfg:         fallbackCfg,
		desktopEnabled:      desktopEnabled,
		cancel:              make(chan struct{}),
		triggerOrbitRestart: triggerOrbitRestart,
	}
}

func (r *serverOverridesRunner) Run(orbitCfg *fleet.OrbitConfig) error {
	overrideCfg, err := loadServerOverrides(r.rootDir)
	if err != nil {
		return err
	}

	if orbitCfg.UpdateChannels == nil {
		// Server is not setting or doesn't know of
		// this feature (old server version), so nothing to do.
		return nil
	}

	if cfgsDiffer(overrideCfg, orbitCfg, r.desktopEnabled) {
		if err := r.updateServerOverrides(orbitCfg); err != nil {
			return err
		}
		r.triggerOrbitRestart("server overrides updated")
		return nil
	}

	return nil
}

// cfgsDiffer returns whether the local server overrides differ from the fetched remotely.
func cfgsDiffer(overrideCfg *serverOverridesConfig, orbitCfg *fleet.OrbitConfig, desktopEnabled bool) bool {
	localUpdateChannelsCfg := &fleet.OrbitUpdateChannels{
		Orbit:    overrideCfg.OrbitChannel,
		Osqueryd: overrideCfg.OsquerydChannel,
		Desktop:  overrideCfg.DesktopChannel,
	}
	remoteUpdateChannelsCfg := orbitCfg.UpdateChannels

	setStableAsDefault := func(cfg *fleet.OrbitUpdateChannels) {
		if cfg.Orbit == "" {
			cfg.Orbit = "stable"
		}
		if cfg.Osqueryd == "" {
			cfg.Osqueryd = "stable"
		}
		if cfg.Desktop == "" {
			cfg.Desktop = "stable"
		}
	}
	setStableAsDefault(localUpdateChannelsCfg)
	setStableAsDefault(remoteUpdateChannelsCfg)

	local := *localUpdateChannelsCfg
	remote := *remoteUpdateChannelsCfg

	if !desktopEnabled {
		local.Desktop = ""
		remote.Desktop = ""
	}

	return local != remote
}

// serverOverridesConfig holds the currently supported fields that can be
// overriden by server configuration.
type serverOverridesConfig struct {
	// OrbitChannel defines the override for the orbit's channel.
	OrbitChannel string `json:"orbit-channel"`
	// OsquerydChannel defines the override for the osqueryd's channel.
	OsquerydChannel string `json:"osqueryd-channel"`
	// DesktopChannel defines the override for the Fleet Desktop's channel.
	DesktopChannel string `json:"desktop-channel"`

	fallbackServerOverridesConfig
}

// fallbackServerOverridesConfig contains fallback configuration in case the server
// settings are invalid (e.g. invalid update channels that don't exist).
// Whenever the user sets an invalid channel, then the fallback paths are used to get
// fleetd up and running with the last known good configuration.
//
// NOTE: We don't need orbit's path because the `orbit` component is a special case that uses
// a symlink to define the last known valid version.
type fallbackServerOverridesConfig struct {
	// OsquerydPath contains the path of the osqueryd executable last known to be valid.
	OsquerydPath string `json:"fallback-osqueryd-path"`
	// DesktopPath contains the path of the Fleet Desktop executable last known to be valid.
	DesktopPath string `json:"fallback-desktop-path"`
}

func (f fallbackServerOverridesConfig) empty() bool {
	return f.OsquerydPath == "" && f.DesktopPath == ""
}

// updateServerOverrides updates the server override local file with the configuration fetched from Fleet.
func (r *serverOverridesRunner) updateServerOverrides(remoteCfg *fleet.OrbitConfig) error {
	overrideCfg := serverOverridesConfig{
		OrbitChannel:                  remoteCfg.UpdateChannels.Orbit,
		OsquerydChannel:               remoteCfg.UpdateChannels.Osqueryd,
		DesktopChannel:                remoteCfg.UpdateChannels.Desktop,
		fallbackServerOverridesConfig: r.fallbackCfg,
	}

	data, err := json.MarshalIndent(overrideCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal override config: %w", err)
	}
	serverOverridesPath := filepath.Join(r.rootDir, constant.ServerOverridesFileName)
	if err := os.WriteFile(serverOverridesPath, data, constant.DefaultFileMode); err != nil {
		return fmt.Errorf("write override config: %w", err)
	}
	return nil
}

// loadServerOverrides loads the server overrides from the local file.
func loadServerOverrides(rootDir string) (*serverOverridesConfig, error) {
	serverOverridesPath := filepath.Join(rootDir, constant.ServerOverridesFileName)
	data, err := os.ReadFile(serverOverridesPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &serverOverridesConfig{}, nil
		}
		return nil, err
	}
	var cfg serverOverridesConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// subSystem is an interface that implements the methods needed for oklog/run.Group.
type subSystem interface {
	// Execute partially implements the interface needed for oklog/run.Group.Add.
	Execute() error
	// Interrupt partially implements the interface needed for oklog/run.Group.Add.
	Interrupt(err error)
}

// addSubsystem adds a new subsystem to the oklog/run.Group.
func addSubsystem(g *run.Group, name string, s subSystem) {
	g.Add(
		func() error {
			log.Debug().Msgf("start %s", name)

			return s.Execute()
		}, func(err error) {
			log.Info().Err(err).Msgf("interrupt %s", name)

			s.Interrupt(err)
		},
	)
}

// wrapSubsystem wraps functions to implement the subSystem interface.
type wrapSubsystem struct {
	execute   func() error
	interrupt func(err error)
}

// Execute partially implements subSystem.
func (w *wrapSubsystem) Execute() error {
	return w.execute()
}

// Interrupt partially implements subSystem.
func (w *wrapSubsystem) Interrupt(err error) {
	w.interrupt(err)
}
