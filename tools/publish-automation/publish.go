package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	// var then variable name then variable type
	// Taking input from user
	version := getVersion()
	makeSureNPMVersionIsCoherent(version)

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
