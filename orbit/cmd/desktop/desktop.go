package main

import (
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/getlantern/systray"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

// version is set at compile time via -ldflags
var version = "unknown"

func main() {
	setupLogs()

	// Our TUF provided targets must support launching with "--help".
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Fleet Desktop application executable")
		return
	}
	log.Info().Msgf("fleet-desktop version=%s", version)

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

	onReady := func() {
		log.Info().Msg("ready")

		systray.SetTemplateIcon(icoBytes, icoBytes)
		systray.SetTooltip("Fleet Desktop")

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

		client, err := service.NewDeviceClient(basePath, deviceToken, insecureSkipVerify, rootCA)
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
					_, err := client.GetDesktopPayload()

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

				res, err := client.GetDesktopPayload()
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

				if res.FailingPolicies > 0 {
					myDeviceItem.SetTitle(fmt.Sprintf("🔴 My device (%d)", res.FailingPolicies))
				} else {
					myDeviceItem.SetTitle("🟢 My device")
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
