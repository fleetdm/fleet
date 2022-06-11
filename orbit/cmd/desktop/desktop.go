package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/getlantern/systray"
)

// version is set at compile time via -ldflags
var version = "unknown"

func main() {
	// Our TUF provided targets must support launching with "--help".
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Fleet Desktop application executable")
		return
	}
	log.Printf("fleet-desktop version=%s\n", version)

	devURL := os.Getenv("FLEET_DESKTOP_DEVICE_URL")
	if devURL == "" {
		log.Println("missing URL environment FLEET_DESKTOP_DEVICE_URL")
		os.Exit(1)
	}
	deviceURL, err := url.Parse(devURL)
	if err != nil {
		log.Printf("invalid URL argument: %s\n", err)
		os.Exit(1)
	}

	basePath := deviceURL.Scheme + "://" + deviceURL.Host
	deviceToken := path.Base(deviceURL.Path)

	onReady := func() {
		log.Println("ready")

		systray.SetTemplateIcon(icoBytes, icoBytes)
		systray.SetTooltip("Fleet Device Management Menu.")

		// Add a disabled menu item with the current version
		versionItem := systray.AddMenuItem(fmt.Sprintf("Fleet Desktop v%s", version), "")
		versionItem.Disable()
		systray.AddSeparator()

		myDeviceItem := systray.AddMenuItem("Initializing...", "")
		myDeviceItem.Disable()
		transparencyItem := systray.AddMenuItem("Transparency", "")

		var insecureSkipVerify bool
		if os.Getenv("FLEET_DESKTOP_INSECURE") != "" {
			insecureSkipVerify = true
		}

		// TODO: figure out the right rootCA to pass to the client
		client, err := service.NewDeviceClient(basePath, deviceToken, insecureSkipVerify, "")

		if err != nil {
			log.Printf("unable to initialize request client: %s", err)
			os.Exit(1)
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
						myDeviceItem.SetTooltip("")
						return
					}

					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
					log.Printf("get device URL: %s", err)

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
					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
					log.Printf("get device URL: %s", err)
					continue
				}

				status := "🟢"
				for _, policy := range policies {
					if policy.Response != "pass" {
						status = "🔴"
						break
					}
				}

				myDeviceItem.SetTitle(status + " My device")
				myDeviceItem.Enable()
			}
		}()

		go func() {
			for {
				select {
				case <-myDeviceItem.ClickedCh:
					if err := open.Browser(deviceURL.String()); err != nil {
						log.Printf("open browser my device: %s", err)
					}
				case <-transparencyItem.ClickedCh:
					if err := open.Browser("https://fleetdm.com/transparency"); err != nil {
						log.Printf("open browser transparency: %s", err)
					}
				}
			}
		}()
	}
	onExit := func() {
		log.Println("exit")
	}

	systray.Run(onReady, onExit)
}
