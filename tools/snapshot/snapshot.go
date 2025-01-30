package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/manifoldco/promptui"
)

// Represents a snapshot.
// Snapshots are stored in folders named after the snapshot name.
// Each snapshot folder contains a db.sql.gz file.
type Snapshot struct {
	Name string
	Date string
	Path string // The directory containing the snapshot.
}

// Which command to run.
type Command int

const (
	CMD_SNAPSHOT Command = iota
	CMD_RESTORE
)

func main() {
	// Ensure there's a command specified.
	// TODO - as we add more commands, we should probably use a library like spf13/cobra.
	if len(os.Args) < 2 {
		fmt.Println("Please specify whether to (b)ackup or (r)estore.")
		os.Exit(1)
	}

	// Determine the command.
	var command Command
	switch os.Args[1] {
	case "s", "snap", "snapshot":
		command = CMD_SNAPSHOT
	case "r", "restore":
		command = CMD_RESTORE
	default:
		fmt.Println("Please specify whether to (s)snapshot or (r)estore.")
	}

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

	// Get the home directory so we can get the snapshots dir.
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Could not determine home directory: %v\n", err)
		return
	}

	// Run the command.
	switch command {
	case CMD_SNAPSHOT:
		snapshot(homedir)
	case CMD_RESTORE:
		restore(homedir)
	}
}

// Restore a snapshot.
func restore(homedir string) error {
	snapshotsDir := filepath.Join(homedir, ".fleet", "snapshots")
	_, err := os.Lstat(snapshotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("You don't currently have any snapshots.\n")
		} else {
			// Handle other PathError-specific cases
			fmt.Printf("Error reading snapshots directory (%s): %v\n", snapshotsDir, err)
		}
		return err
	}

	// Walk the ~/.fleet/snapshots directory if it exists.
	dirEntries, err := os.ReadDir(snapshotsDir)
	var snapshots []Snapshot
	for _, entry := range dirEntries {
		if entry.IsDir() {
			// Ensure there's a db backup file.
			dbBackupFile := filepath.Join(snapshotsDir, entry.Name(), "db.sql.gz")
			dbBackupFileInfo, err := os.Lstat(dbBackupFile)
			if err != nil {
				continue
			}
			snapshot := Snapshot{
				Name: entry.Name(),
				Date: dbBackupFileInfo.ModTime().Format("Jan 02, 2006 03:04:05 PM"),
				Path: dbBackupFile,
			}
			snapshots = append(snapshots, snapshot)
		}
	}

	// Set up and run the "Select snapshot" UI.
	templates := &promptui.SelectTemplates{
		Label:    "{{ .Name }}",
		Active:   "> {{ .Name }} ({{ .Date }})",
		Inactive: "{{ .Name }} ({{ .Date }})",
		Selected: "{{ .Name }} ({{ .Date }})",
	}
	prompt := promptui.Select{
		Label:     "Select snapshot to restore",
		Items:     snapshots,
		Templates: templates,
	}
	index, _, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return err
	}

	// Prepare the restore script with the selected snapshot.
	cmd := exec.Command("/Users/scott/Development/fleet/tools/backup_db/restore.sh", snapshots[index].Path)

	// Use the same stdin, stdout, and stderr as the parent process.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command.
	err = cmd.Run()
	output, _ := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}

	return nil
}

// Create a snapshot.
func snapshot(homedir string) error {
	snapshotsDir := filepath.Join(homedir, ".fleet", "snapshots")

	// Ensure the snapshots directory exists.
	_, err := os.Lstat(snapshotsDir)
	if err != nil {
		// If the directory doesn't exist, create it.
		if os.IsNotExist(err) {
			err = os.Mkdir(snapshotsDir, 0o755)
			if err != nil {
				fmt.Printf("Error creating snapshots directory (%s): %v\n", snapshotsDir, err)
			}
		} else {
			fmt.Printf("Error reading snapshots directory (%s): %v\n", snapshotsDir, err)
		}
		return err
	}

	// Prompt the user for a name for the snapshot.
	prompt := promptui.Prompt{
		Label: "Enter a name for the snapshot",
	}
	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return err
	}
	snapshotPath := filepath.Join(snapshotsDir, result)

	// Check if the snapshot already exists.
	_, err = os.Lstat(snapshotPath)
	// If the file exists, prompt the user to overwrite it.
	if err == nil {
		prompt := promptui.Prompt{
			Label: "This snapshot already exists. Overwrite? (Y/n)",
		}
		result, err := prompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed %v\n", err)
			return err
		}
		switch result {
		case "Y", "y", "":
			err = os.RemoveAll(snapshotPath)
			if err != nil {
				fmt.Printf("Error removing existing snapshot (%s): %v\n", result, err)
				return err
			}
		default:
			return nil
		}
	} else if !os.IsNotExist(err) {
		fmt.Printf("Error checking for existing snapshot (%s): %v\n", result, err)
		return err
	}

	// Create the snapshot directory
	err = os.Mkdir(snapshotPath, 0o755)
	if err != nil {
		fmt.Printf("Error creating snapshot directory (%s): %v\n", snapshotPath, err)
	}

	// Prepare the backup script with the snapshot path.
	cmd := exec.Command("/Users/scott/Development/fleet/tools/backup_db/backup.sh", filepath.Join(snapshotPath, "db.sql.gz"))

	// Use the same stdin, stdout, and stderr as the parent process.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command.
	err = cmd.Run()
	output, _ := cmd.CombinedOutput()
	fmt.Println(string(output))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return err
	}

	return nil
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
