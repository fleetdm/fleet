package main

import (
	"crypto/tls"
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/getlantern/systray"
)

//go:embed icon_white.png
var icoBytes []byte

func main() {
	// Our TUF provided targets must support launching with "--help".
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Fleet Desktop application executable")
		return
	}

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
	devTestPath := os.Getenv("FLEET_DESKTOP_DEVICE_API_TEST_PATH")
	if devTestPath == "" {
		log.Println("missing URL environment FLEET_DESKTOP_DEVICE_API_TEST_PATH")
		os.Exit(1)
	}
	devTestURL := *deviceURL
	devTestURL.Path = devTestPath

	onReady := func() {
		log.Println("ready")

		systray.SetIcon(icoBytes)
		systray.SetTooltip("Fleet Device Management Menu.")
		myDeviceItem := systray.AddMenuItem("Initializing...", "")
		myDeviceItem.Disable()
		transparencyItem := systray.AddMenuItem("Transparency", "")

		// Perform API test call to enable the "My device" item as soon
		// as the device auth token is registered by Fleet.
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()

			tr := http.DefaultTransport.(*http.Transport)
			if os.Getenv("FLEET_DESKTOP_INSECURE") != "" {
				tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
			client := &http.Client{
				Transport: tr,
			}

			for {
				resp, err := client.Get(devTestURL.String())
				if err != nil {
					// To ease troubleshooting we set the tooltip as the error.
					myDeviceItem.SetTooltip(err.Error())
					log.Printf("get device URL: %s", err)
				} else {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						myDeviceItem.SetTitle("My device")
						myDeviceItem.Enable()
						myDeviceItem.SetTooltip("")
						return
					}
				}
				<-ticker.C
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
