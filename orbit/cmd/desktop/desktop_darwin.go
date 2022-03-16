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

	if len(os.Args) < 2 {
		fmt.Println("Missing URL argument")
		os.Exit(1)
	}
	url, err := url.Parse(os.Args[1])
	if err != nil {
		fmt.Printf("Invalid URL argument: %s\n", err)
		os.Exit(1)
	}

	onReady := func() {
		systray.SetIcon(icoBytes)
		systray.SetTooltip("Fleet Device Management Desktop Application")
		myDeviceItem := systray.AddMenuItem("My Device", "Access My Device Details")

		go func() {
			for range myDeviceItem.ClickedCh {
				if err := open.Browser(url.String()); err != nil {
					log.Printf("open browser: %s", err)
				}
			}
		}()
	}
	systray.Run(onReady, nil)
}
