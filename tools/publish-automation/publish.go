package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	printIntro()
	version := getVersion()
	makeSureNPMVersionIsCoherent(version)
	if !openReleasesPage(version) {
		fmt.Println("Error - could not publish the Fleet release.")
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

func printIntro() {
	fmt.Println("\n===========================================================")
	fmt.Println("This Script is not full automation.\nIt will guide you through the publishing process step by step.")
	fmt.Println("Hit ENTER when ready.")
	fmt.Scanln()
}

func getVersion() string {
	fmt.Println("\n===========================================================")
	fmt.Println("Step 1: Current version.")
	for {
		fmt.Println("Enter the version being currently released in the format similar to \"v4.47.2\": ")
		var ver string
		fmt.Scanln(&ver)
		fmt.Println("The version you entered: " + ver + ".\n" + "Is that correct? (Y/y)")
		var answer string
		fmt.Scanln(&answer)
		if answer == "Y" || answer == "y" {
			return ver
		}
		fmt.Println("\n     ---> Try again.")
	}
}

func makeSureNPMVersionIsCoherent(version string) bool {
	type NpmPackageJson struct {
		Version string `json:"version"`
	}
	var pck NpmPackageJson

	fmt.Println("\n===========================================================")
	fmt.Println("Step 2: Checking ./tools/fleetctl-npm/package.json for correct version")
	for {
		myJson, err := os.ReadFile("./tools/fleetctl-npm/package.json")
		if err != nil {
			fmt.Println("Error reading JSON file ./tools/fleetctl-npm/package.json:", err)
			return false
		}

		// myJson.close

		err = json.Unmarshal(myJson, &pck)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			// myJson.
			return false
		}
		if pck.Version == version {
			fmt.Println("JSON at ./tools/fleetctl-npm/package.json: " + pck.Version + " is coherent with current version: " + version)
			fmt.Println("Hit ENTER for next step")
			fmt.Scanln()
			// myJson.close
			return true
		}

		// myJson.close
		fmt.Println("JSON at ./tools/fleetctl-npm/package.json shows: " + pck.Version + " which is not coherent with current version: " + version)
		fmt.Println("Please fix ./tools/fleetctl-npm/package.json and hit ENTER.")
		fmt.Scanln()
	}
}

func openReleasesPage(version string) bool {
	fmt.Println("\n===========================================================")
	fmt.Println("Step 3: Publish the release in Fleet Releases page.")
	fmt.Println("Hitting ENTER will take you to the proper editing page (of current release)")
	fmt.Println("Once published, go back here and hit ENTER again")
	fmt.Scanln()
	fmt.Println("Taking you to the edit page ... ")

	url := "https://github.com/fleetdm/fleet/releases/edit/fleet-" + version
	err := openURL(url)
	if err != nil {
		fmt.Println("Error opening URL:", err)
		return false
	}

	fmt.Println("Hit ENTER for next step")
	fmt.Scanln()
	return true
}

func openURL(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func npmPublish() bool {
	fmt.Println("\n===========================================================")
	fmt.Println("Step 4: NPM PUBLISH")
	fmt.Println("Hitting ENTER will take you to another terminal to fleet/tools/fleetctl-npm/")
	fmt.Println("Once there type \"npm publish\". You will need an auth code for that.")
	fmt.Scanln()
	fmt.Println("Taking you to the other terminal ... ")

	relativePath := "./tools/fleetctl-npm"
	err := openTerminalAtPath(relativePath)
	if err != nil {
		fmt.Println("Error opening Terminal:", err)
		return false
	}

	fmt.Println("Hit ENTER for next step")
	fmt.Scanln()
	return true
}

func openTerminalAtPath(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "Terminal", path)
	default:
		return fmt.Errorf("unsupported operating system")
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func checkDocker(version string) bool {
	fmt.Println("\n===========================================================")
	fmt.Println("Step 5: Check hub.docker for existing version")
	fmt.Println("Hitting ENTER will take you to the proper hub.docker page.")
	fmt.Println("Verify that the version exists")
	fmt.Scanln()
	url := "https://hub.docker.com/r/fleetdm/fleet/tags?page=1&name=patch-fleet-" + version
	fmt.Println("Taking you to the docker hub page: " + url)

	err := openURL(url)
	if err != nil {
		fmt.Println("Error opening URL:", err)
		return false
	}

	fmt.Println("Does Docker version exist? (Y/y)")
	var answer string
	fmt.Scanln(&answer)
	if answer == "Y" || answer == "y" {
		return true
	} else {
		return false
	}
}

func deployDogfood(version string) bool {
	fmt.Println("\n===========================================================")
	fmt.Println("Step 6: Deploy " + version + " to Dogfood.")
	fmt.Println("Hitting ENTER will take you to the proper git action page.")
	fmt.Println("Once at the git action page do this:")
	fmt.Println("    1 - Go to #help-infrastructure slack channel and paste this:\"@infrastructure-oncall We have just released " + version + " and are now deploying it to dogfood\"  ")
	fmt.Println("    2 - press the \"Run workflow\" button on the right side of the screen")
	fmt.Println("    3 - paste \"fleetdm/fleet:patch-fleet-" + version + "\" as input.")
	fmt.Println("    4 - press the \"Run workflow\"")
	fmt.Scanln()
	url := "https://github.com/fleetdm/fleet/actions/workflows/dogfood-deploy.yml"
	fmt.Println("Taking you to the docker hub page: " + url)

	err := openURL(url)
	if err != nil {
		fmt.Println("Error opening URL:", err)
		return false
	}

	fmt.Println("Wait for the script to finish (~10 minutes)?")
	fmt.Println("Go to dogfood and verify that it works")
	fmt.Println("SLACK NOTIFICATIONS")
	fmt.Println("Go to #help-engineering slack channel and paste this:")
	fmt.Println("     We have just deployed " + version + " to dogfood")
	fmt.Println("Go to #general slack channel and paste this:")
	fmt.Println("     :cloud: :rocket: The latest version of Fleet is " + version[1:])
	fmt.Println("     More info: https://github.com/fleetdm/fleet/releases/tag/fleet-" + version)
	fmt.Println("     Upgrade now: https://fleetdm.com/docs/deploying/upgrading-fleet")
	fmt.Println("All good?(Y/y)")
	var answer string
	fmt.Scanln(&answer)
	if answer == "Y" || answer == "y" {
		return true
	} else {
		return false
	}
}

func closeMilestoneTickets(version string) bool {
	milestone := version[1:]
	fmt.Println("\n===========================================================")
	fmt.Println("Step 7: Close all tickets with milestone " + milestone)
	fmt.Println("Hitting ENTER will take you to the ZenHub page with all relevant tickets.")
	fmt.Println("Once at ZenHub:")
	fmt.Println(" - for each ticket verify that it was indeed included in the patch release and close it.")
	fmt.Scanln()
	url := "https://github.com/fleetdm/fleet/issues?q=is%3Aissue+is%3Aopen+milestone%3A%22" + milestone + "%22+"
	fmt.Println("Taking you to ZenHub search.")

	err := openURL(url)
	if err != nil {
		fmt.Println("Error opening URL:", err)
		return false
	}

	fmt.Println("Have you closed all tickets?(Y/y)")
	var answer string
	fmt.Scanln(&answer)
	if answer == "Y" || answer == "y" {
		return true
	} else {
		return false
	}
}
