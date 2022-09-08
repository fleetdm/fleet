package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/token"
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

	identifierPath := os.Getenv("FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH")
	if identifierPath == "" {
		log.Fatal().Msg("missing URL environment FLEET_DESKTOP_DEVICE_IDENTIFIER_PATH")
	}

	fleetURL := os.Getenv("FLEET_DESKTOP_FLEET_URL")
	if fleetURL == "" {
		log.Fatal().Msg("missing URL environment FLEET_DESKTOP_FLEET_URL")
	}

	onReady := func() {
		log.Info().Msg("ready")

		systray.SetTemplateIcon(icoBytes, icoBytes)
		systray.SetTooltip("Fleet Device Management Menu.")

		// Add a disabled menu item with the current version
		versionItem := systray.AddMenuItem(fmt.Sprintf("Fleet Desktop v%s", version), "")
		versionItem.Disable()
		systray.AddSeparator()

		myDeviceItem := systray.AddMenuItem("Connecting...", "")
		myDeviceItem.Disable()
		transparencyItem := systray.AddMenuItem("Transparency", "")
		transparencyItem.Disable()

		tokenReader := token.Reader{Path: identifierPath}
		if _, err := tokenReader.Read(); err != nil {
			log.Fatal().Err(err).Msg("error reading device token from file")
		}

		var insecureSkipVerify bool
		if os.Getenv("FLEET_DESKTOP_INSECURE") != "" {
			insecureSkipVerify = true
		}
		rootCA := os.Getenv("FLEET_DESKTOP_FLEET_ROOT_CA")

		client, err := service.NewDeviceClient(
			fleetURL,
			insecureSkipVerify,
			rootCA,
		)
		if err != nil {
			log.Fatal().Err(err).Msg("unable to initialize request client")
		}

		refetchToken := func() {
			if _, err := tokenReader.Read(); err != nil {
				log.Error().Err(err).Msg("refetch token")
			}
			log.Debug().Msg("successfully refetched the token from disk")
		}

		disableTray := func() {
			log.Debug().Msg("disabling tray items")
			myDeviceItem.SetTitle("Connecting...")
			myDeviceItem.Disable()
			transparencyItem.Disable()
		}

		// checkToken performs API test calls to enable the "My device" item as
		// soon as the device auth token is registered by Fleet.
		checkToken := func() <-chan interface{} {
			done := make(chan interface{})

			go func() {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()
				defer close(done)

				for {
					refetchToken()
					_, err := client.ListDevicePolicies(tokenReader.GetCached())
					if err == nil || errors.Is(err, service.ErrMissingLicense) {
						log.Debug().Msg("enabling tray items")
						myDeviceItem.SetTitle("My device")
						myDeviceItem.Enable()
						myDeviceItem.SetTooltip("")
						transparencyItem.Enable()
						return
					}

					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
					log.Error().Err(err).Msg("get device URL")

					<-ticker.C
				}
			}()

			return done
		}

		// start a check as soon as the app starts
		deviceEnabledChan := checkToken()

		// this loop checks the `mtime` value of the token file and:
		// 1. if the token file was modified, it disables the tray items until we
		// verify the token is valid
		// 2. calls (blocking) `checkToken` to verify the token is valid
		go func() {
			<-deviceEnabledChan
			tic := time.NewTicker(1 * time.Second)
			defer tic.Stop()

			for {
				<-tic.C
				expired, err := tokenReader.HasChanged()
				switch {
				case err != nil:
					log.Error().Err(err).Msg("check token file")
				case expired:
					log.Info().Msg("token file changed, rechecking")
					disableTray()
					<-checkToken()
				}
			}
		}()

		go func() {
			<-deviceEnabledChan
			tic := time.NewTicker(5 * time.Minute)
			defer tic.Stop()

			for {
				policies, err := client.ListDevicePolicies(tokenReader.GetCached())
				switch {
				case err == nil:
					// OK
				case errors.Is(err, service.ErrMissingLicense):
					myDeviceItem.SetTitle("My device")
					continue
				case errors.Is(err, service.ErrUnauthenticated):
					disableTray()
					<-checkToken()
					continue
				default:
					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
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
					myDeviceItem.SetTitle(fmt.Sprintf("🔴 My device (%d)", failedPolicyCount))
				} else {
					myDeviceItem.SetTitle("🟢 My device")
				}
				myDeviceItem.Enable()

				<-tic.C
			}
		}()

		go func() {
			for {
				select {
				case <-myDeviceItem.ClickedCh:
					if err := open.Browser(client.DeviceURL(tokenReader.GetCached())); err != nil {
						log.Error().Err(err).Msg("open browser my device")
					}
				case <-transparencyItem.ClickedCh:
					if err := open.Browser(client.TransparencyURL(tokenReader.GetCached())); err != nil {
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
