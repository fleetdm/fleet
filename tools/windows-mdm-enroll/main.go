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
		err := update.RunMicrosoftMDMUnenrollment(update.MicrosoftMDMEnrollmentArgs{})
		fmt.Println("unenrollment: ", err)
		return
	}

	err := update.RunMicrosoftMDMEnrollment(update.MicrosoftMDMEnrollmentArgs{
		DiscoveryURL: *discoveryURL,
		HostUUID:     *hostUUID,
	})
	fmt.Println("enrollment: ", err)
}
