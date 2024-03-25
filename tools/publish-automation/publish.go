package main

import "fmt"

func main() {
	// var then variable name then variable type
	// Taking input from user
	version := getVersion()

	fmt.Println(version)
}

func getVersion() string {
	for {
		fmt.Println("Enter current version: ")
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
