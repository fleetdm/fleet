package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc"
	msrc_io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
)

func main() {
	now := time.Now()

	fmt.Println("Downloading feed...")
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	msrcAPI := msrc_io.NewMSRCClient(http.DefaultClient, wd, nil)
	f, err := msrcAPI.GetFeed(now.Month(), now.Year())
	if err != nil {
		panic(err)
	}

	fmt.Println("Parsing feed...")
	_, err = msrc.ParseFeed(f)
	if err != nil {
		panic(err)
	}

	fmt.Println("Parsed .")
	fmt.Println("Done.")
}
