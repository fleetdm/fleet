package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func main() {
	version := getVersion()
	makeSureNPMVersionIsCoherent(version)
	if !openReleasesPage(version) {
		fmt.Println("Script Failed.")
		return
	}

	fmt.Println("Script ended successfully.")
}

func getVersion() string {
	for {
		fmt.Println("Enter current version in the format similar to \"v4.47.2\": ")
		var ver string
		fmt.Scanln(&ver)
		fmt.Println("The version you entered: " + ver + ".\n" + "Is that correct? (Y/N)")
		var answer string
		fmt.Scanln(&answer)
		if answer == "Y" || answer == "y" {
			return ver
		}
		fmt.Println("Try again.")
	}
}

func makeSureNPMVersionIsCoherent(version string) bool {
	type NpmPackageJson struct {
		Version string `json:"version"`
	}
	var pck NpmPackageJson

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
	url := "https://github.com/fleetdm/fleet/releases/edit/fleet-" + version

	fmt.Println("===============================================")
	fmt.Println("Publishing the release on Fleet Releases page")
	fmt.Println("Hit ENTER to edit current release")
	fmt.Scanln()
	fmt.Println("Taking you to the edit page")

	err := openURL(url)
	if err != nil {
		fmt.Println("Error opening URL:", err)
		return false
	}

	fmt.Println("Once published, hit ENTER again")
	fmt.Println("===============================================")
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
