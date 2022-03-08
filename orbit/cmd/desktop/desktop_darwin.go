package main

import (
	_ "embed"
	"fmt"
	"log"
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

	onReady := func() {
		systray.SetIcon(icoBytes)
		systray.SetTooltip("Fleet Device Management Desktop Application")
		myDeviceItem := systray.AddMenuItem("My Device", "Access My Device Details")

		go func() {
			for range myDeviceItem.ClickedCh {
				// TODO(lucas): Just using dummy URL for testing.
				if err := open.Browser("https://localhost:8080"); err != nil {
					log.Printf("open browser: %s", err)
				}
			}
		}()
	}
	systray.Run(onReady, nil)
}
