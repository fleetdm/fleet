package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/manifoldco/promptui"
)

func main() {
	// Determine the path to the top-level directory (where the Makefile resides).
	repoRoot, err := getRepoRoot()
	if err != nil {
		fmt.Printf("Error determining repo root: %v\n", err)
		os.Exit(1)
	}

	// Change the working directory to the repo root.
	if err := os.Chdir(repoRoot); err != nil {
		fmt.Printf("Error changing directory to repo root: %v\n", err)
		os.Exit(1)
	}

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
	files, err := os.ReadDir(backupsDir)
	fileNames := make([]string, len(files))
	for i, file := range files {
		fileNames[i] = file.Name()
		fileInfo, err := file.Info()
		if err == nil {
			fileNames[i] += " (" + fileInfo.ModTime().String() + ")"
		}
	}

	prompt := promptui.Select{
		Label: "Select snapshot to restore",
		Items: fileNames,
	}

	_, result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	cmd := exec.Command("/Users/scott/Development/fleet/tools/backup_db/restore.sh", result)

	// Use the same stdin, stdout, and stderr as the parent process.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command.
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// getRepoRoot determines the repo root (top-level directory) relative to this binary.
func getRepoRoot() (string, error) {
	// Get the path of the currently executing binary
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Get the path of the binary, following symlinks.
	execDir, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	// Get the directory.
	execDir = filepath.Dir(execDir)

	// Compute the repo root relative to the binary's location.
	repoRoot := filepath.Join(execDir, "../")
	return filepath.Abs(repoRoot)
}
