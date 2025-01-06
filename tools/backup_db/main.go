package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/manifoldco/promptui"
)

func main() {
	// Get the home directory so we can get the backups dir.
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Could not determine home directory: %v\n", err)
		return
	}

	backupsDir := filepath.Join(homedir, ".fleet", "snapshots")
	_, err = os.Lstat(backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("You don't currently have any snapshots.\n")
		} else {
			// Handle other PathError-specific cases
			fmt.Printf("Error reading snapshots directory (%s): %v\n", backupsDir, err)
		}
		return
	}

	// Walk the ~/.fleet/db-backups directory if it exists.

	prompt := promptui.Select{
		Label: "Select Day",
		Items: []string{
			"Monday", "Tuesday", "Wednesday", "Thursday", "Friday",
			"Saturday", "Sunday",
		},
	}

	_, result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("You choose %q\n", result)
}
