package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/getlantern/systray"
	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// version is set at compile time via -ldflags
var version = "unknown"

func setupRunners(channelID string) {
	var runnerGroup run.Group

	// Setting up a watcher for the communication channel
	if runtime.GOOS == "windows" {
		runnerGroup.Add(
			func() error {
				// block wait on the communication channel
				if err := blockWaitForStopEvent(channelID); err != nil {
					log.Error().Err(err).Msg("There was an error on the desktop communication channel")
					return err
				}

				log.Info().Msg("Shutdown was requested!")
				return nil
			},
			func(err error) {
				systray.Quit()
			},
		)
	}

	// Setting up a signal handler runner
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var execFn func() error
	var interrFn func(error)

	if runtime.GOOS == "windows" {
		execFn, interrFn = run.SignalHandler(ctx, os.Interrupt, os.Kill)
	} else {
		execFn, interrFn = run.SignalHandler(ctx, os.Interrupt, os.Kill, syscall.SIGTERM)
	}

	runnerGroup.Add(
		execFn,
		func(err error) {
			interrFn(err)
			systray.Quit()
		},
	)

	if err := runnerGroup.Run(); err != nil {
		log.Error().Err(err).Msg("Fleet Desktop runners terminated")
		return
	}
}

func main() {
	setupLogs()

	// Our TUF provided targets must support launching with "--help".
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Fleet Desktop application executable")
		return
	}
	log.Info().Msgf("fleet-desktop version=%s", version)

	commChannelID := os.Getenv("FLEET_DESKTOP_CHANNEL_ID")
	if commChannelID == "" {
		log.Fatal().Msg("missing Channel ID environment FLEET_DESKTOP_CHANNEL_ID")
	}

	devURL := os.Getenv("FLEET_DESKTOP_DEVICE_URL")
	if devURL == "" {
		log.Fatal().Msg("missing URL environment FLEET_DESKTOP_DEVICE_URL")
	}
	deviceURL, err := url.Parse(devURL)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid URL argument")
	}

	basePath := deviceURL.Scheme + "://" + deviceURL.Host
	deviceToken := path.Base(deviceURL.Path)
	transparencyURL := basePath + "/api/latest/fleet/device/" + deviceToken + "/transparency"

	// Setting up working runners such as signalHandler runner
	go setupRunners(commChannelID)

	onReady := func() {
		log.Info().Msg("ready")

		systray.SetTooltip("Fleet Desktop")
		// Default to dark theme icon because this seems to be a better fit on Linux (Ubuntu at
		// least). On macOS this is used as a template icon anyway.
		systray.SetTemplateIcon(iconDark, iconDark)

		// Theme detection is currently only on Windows. On macOS we use template icons (which
		// automatically change), and on Linux we don't handle it yet (Ubuntu doesn't seem to change
		// systray colors in the default configuration when toggling light/dark).
		if runtime.GOOS == "windows" {
			// Set the initial theme, and watch for theme changes.
			theme, err := getSystemTheme()
			if err != nil {
				log.Error().Err(err).Msg("get system theme")
			}
			iconManager := newIconManager(theme)
			go func() {
				watchSystemTheme(iconManager)
			}()
		}

		// Add a disabled menu item with the current version
		versionItem := systray.AddMenuItem(fmt.Sprintf("Fleet Desktop v%s", version), "")
		versionItem.Disable()
		systray.AddSeparator()

		myDeviceItem := systray.AddMenuItem("Initializing...", "")
		myDeviceItem.Disable()
		transparencyItem := systray.AddMenuItem("Transparency", "")
		transparencyItem.Disable()

		var insecureSkipVerify bool
		if os.Getenv("FLEET_DESKTOP_INSECURE") != "" {
			insecureSkipVerify = true
		}
		rootCA := os.Getenv("FLEET_DESKTOP_FLEET_ROOT_CA")

		capabilities := fleet.CapabilityMap{}

		client, err := service.NewDeviceClient(basePath, deviceToken, insecureSkipVerify, rootCA, capabilities)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to initialize request client")
		}

		// Perform API test call to enable the "My device" item as soon
		// as the device auth token is registered by Fleet.
		deviceEnabledChan := func() <-chan interface{} {
			done := make(chan interface{})

			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				defer close(done)

				for {
					_, err := client.ListDevicePolicies()

					if err == nil || errors.Is(err, service.ErrMissingLicense) {
						myDeviceItem.SetTitle("My device")
						myDeviceItem.Enable()
						transparencyItem.Enable()
						return
					}

					log.Error().Err(err).Msg("get device URL")

					<-ticker.C
				}
			}()

			return done
		}()

		go func() {
			<-deviceEnabledChan
			tic := time.NewTicker(5 * time.Minute)
			defer tic.Stop()

			for {
				<-tic.C

				policies, err := client.ListDevicePolicies()
				switch {
				case err == nil:
					// OK
				case errors.Is(err, service.ErrMissingLicense):
					myDeviceItem.SetTitle("My device")
					continue
				default:
					log.Error().Err(err).Msg("get device URL")
					continue
				}

				failedPolicyCount := 0
				for _, policy := range policies {
					if policy.Response != "pass" {
						failedPolicyCount++
					}
				}

				if failedPolicyCount > 0 {
					if runtime.GOOS == "windows" {
						// Windows (or maybe just the systray library?) doesn't support color emoji
						// in the system tray menu, so we use text as an alternative.
						if failedPolicyCount == 1 {
							myDeviceItem.SetTitle("My device (1 issue)")
						} else {
							myDeviceItem.SetTitle(fmt.Sprintf("My device (%d issues)", failedPolicyCount))
						}
					} else {
						myDeviceItem.SetTitle(fmt.Sprintf("🔴 My device (%d)", failedPolicyCount))
					}
				} else {
					if runtime.GOOS == "windows" {
						myDeviceItem.SetTitle("My device")
					} else {
						myDeviceItem.SetTitle("🟢 My device")
					}
				}
				myDeviceItem.Enable()
			}
		}()

		go func() {
			for {
				select {
				case <-myDeviceItem.ClickedCh:
					if err := open.Browser(deviceURL.String()); err != nil {
						log.Error().Err(err).Msg("open browser my device")
					}
				case <-transparencyItem.ClickedCh:
					if err := open.Browser(transparencyURL); err != nil {
						log.Error().Err(err).Msg("open browser transparency")
					}
				}
			}
		}()
	}
	onExit := func() {
		log.Info().Msg("exit")
	}

	systray.Run(onReady, onExit)
}

// setupLogs configures our logging system to write logs to rolling files and
// stderr, if for some reason we can't write a log file the logs are still
// printed to stderr.
func setupLogs() {
	stderrOut := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano, NoColor: true}

	dir, err := logDir()
	if err != nil {
		log.Logger = log.Output(stderrOut)
		log.Error().Err(err).Msg("find directory for logs")
		return
	}

	dir = filepath.Join(dir, "Fleet")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Logger = log.Output(stderrOut)
		log.Error().Err(err).Msg("make directories for log files")
		return
	}

	logFile := &lumberjack.Logger{
		Filename:   filepath.Join(dir, "fleet-desktop.log"),
		MaxSize:    25, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
	}

	log.Logger = log.Output(zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: logFile, TimeFormat: time.RFC3339Nano, NoColor: true},
		stderrOut,
	))
}

// logDir returns the default root directory to use for application-level logs.
//
// On Unix systems, it returns $XDG_STATE_HOME as specified by
// https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if
// non-empty, else $HOME/.local/state.
// On Darwin, it returns $HOME/Library/Logs.
// On Windows, it returns %LocalAppData%
//
// If the location cannot be determined (for example, $HOME is not defined),
// then it will return an error.
func logDir() (string, error) {
	var dir string

	switch runtime.GOOS {
	case "windows":
		dir = os.Getenv("LocalAppData")
		if dir == "" {
			return "", errors.New("%LocalAppData% is not defined")
		}

	case "darwin":
		dir = os.Getenv("HOME")
		if dir == "" {
			return "", errors.New("$HOME is not defined")
		}
		dir += "/Library/Logs"

	default: // Unix
		dir = os.Getenv("XDG_STATE_HOME")
		if dir == "" {
			dir = os.Getenv("HOME")
			if dir == "" {
				return "", errors.New("neither $XDG_STATE_HOME nor $HOME are defined")
			}
			dir += "/.local/state"
		}
	}

	return dir, nil
}

type iconManager struct {
	theme theme
}

func newIconManager(theme theme) *iconManager {
	m := &iconManager{
		theme: theme,
	}
	m.UpdateTheme(theme)
	return m
}

func (m *iconManager) UpdateTheme(theme theme) {
	m.theme = theme
	switch theme {
	case themeDark:
		systray.SetIcon(iconDark)
	case themeLight:
		systray.SetIcon(iconLight)
	case themeUnknown:
		log.Debug().Msg("theme unknown, using dark theme")
	default:
		log.Error().Str("theme", string(theme)).Msg("tried to set invalid theme")
	}
}
