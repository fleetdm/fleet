package main

import (
	"flag"
	"fmt"

	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
)

func main() {
	var (
		discoveryURL = flag.String("discovery-url", "", "The Windows MDM discovery URL")
		hostUUID     = flag.String("host-uuid", "", "The Host UUID")
	)
	flag.Parse()

	err := update.RunWindowsMDMEnrollment(update.WindowsMDMEnrollmentArgs{
		DiscoveryURL: *discoveryURL,
		HostUUID:     *hostUUID,
	})
	fmt.Println(err)
}
