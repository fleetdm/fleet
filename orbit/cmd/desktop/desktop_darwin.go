package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/fleetdm/fleet/v4/pkg/open"
	"github.com/getlantern/systray"
)

//go:embed favicon.ico
var icoBytes []byte

func main() {
	// Our TUF provided targets must support launching with "--help".
	if len(os.Args) > 1 && os.Args[1] == "--help" {
		fmt.Println("Fleet Desktop application executable")
		return
	}

	devURL := os.Getenv("FLEET_DESKTOP_DEVICE_URL")
	if devURL == "" {
		fmt.Println("Missing URL environment FLEET_DESKTOP_DEVICE_URL")
		os.Exit(1)
	}
	deviceURL, err := url.Parse(devURL)
	if err != nil {
		fmt.Printf("Invalid URL argument: %s\n", err)
		os.Exit(1)
	}

	onReady := func() {
		systray.SetIcon(icoBytes)
		systray.SetTooltip("Fleet Device Management Menu.")
		myDeviceItem := systray.AddMenuItem("My device", "")
		transparencyItem := systray.AddMenuItem("Transparency", "")

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
	systray.Run(onReady, nil)
}
