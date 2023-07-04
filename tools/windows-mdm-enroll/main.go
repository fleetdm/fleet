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
		unenroll     = flag.Bool("unenroll", false, "Unenroll from MDM instead of enrolling")
	)
	flag.Parse()

	if *unenroll {
		err := update.RunWindowsMDMUnenrollment(update.WindowsMDMEnrollmentArgs{})
		fmt.Println("unenrollment: ", err)
		return
	}

	err := update.RunWindowsMDMEnrollment(update.WindowsMDMEnrollmentArgs{
		DiscoveryURL: *discoveryURL,
		HostUUID:     *hostUUID,
	})
	fmt.Println("enrollment: ", err)
}
