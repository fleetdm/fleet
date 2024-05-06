package writetoslack

import (
	"fmt"
)

func main() {
	printIntro()
	version := getVersion()
	makeSureNPMVersionIsCoherent(version)
	if !openReleasesPage(version) {
		fmt.Println("Error - could not publish the fleat release.")
		fmt.Println("Try manually at https://github.com/fleetdm/fleet/releases/edit/fleet-" + version)
		return
	}
	if !npmPublish() {
		fmt.Println("Error - npm publish failed.")
		return
	}
	if !checkDocker(version) {
		fmt.Println("Error - no docker image found.")
		return
	}
	if !deployDogfood(version) {
		fmt.Println("Error -  could not deploy to dogfood.")
		return
	}
	if !closeMilestoneTickets(version) {
		fmt.Println("Script Failed.")
		return
	}

	fmt.Println("Script ended successfully.")
}
