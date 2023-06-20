package main

import (
	"flag"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
)

func main() {
	var discoveryURL = flag.String("discovery-url", "", "The Windows MDM discovery URL")
	flag.Parse()

	err := update.RunWindowsMDMEnrollment(*discoveryURL)
	fmt.Println(err)
}
