package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Ensure there's a make target specified
	if len(os.Args) < 2 {
		fmt.Println("Usage: fdm <command> [--option=value ...] -- [make-options]")
		os.Exit(1)
	}

	// Determine the path to the top-level directory (where the Makefile resides)
	repoRoot, err := getRepoRoot()
	if err != nil {
		fmt.Printf("Error determining repo root: %v\n", err)
		os.Exit(1)
	}

	// Change the working directory to the repo root
	if err := os.Chdir(repoRoot); err != nil {
		fmt.Printf("Error changing directory to repo root: %v\n", err)
		os.Exit(1)
	}

	// Extract the make target
	makeTarget := os.Args[1]

	// Split arguments into options and make arguments
	options, makeArgs := splitArgs(os.Args[2:])

	// Special logic for the help command
	if makeTarget == "help" {
		if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "--") {
			options["SPECIFIC_CMD"] = os.Args[2]
			options["REFORMAT_OPTIONS"] = "true"
		} else {
			fmt.Println("fdm - developer tools for fleet device management")
			fmt.Println()
			fmt.Println("USAGE:")
			fmt.Println("  fdm <command> [--option=value ...] -- [make-options]")
			fmt.Println()
			fmt.Println("COMMANDS:")
			options["HELP_CMD_PREFIX"] = "fdm"
		}
	}

	// Transform options into Makefile-compatible environment variables
	makeVars := transformToMakeVars(options)
	makeArgs = append(makeVars, "TOOL_CMD=fdm")

	// Call the Makefile with the specified target, environment variables, and additional arguments
	err = callMake(makeTarget, makeVars, makeArgs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// splitArgs splits the arguments into options and make arguments based on the `--` delimiter
func splitArgs(args []string) (map[string]string, []string) {
	options := make(map[string]string)
	var makeArgs []string
	isMakeArgs := false
	skipNext := false

	for idx, arg := range args {
		if skipNext == true {
			skipNext = false
			continue
		}

		if arg == "--" {
			isMakeArgs = true
			continue
		}

		if isMakeArgs {
			makeArgs = append(makeArgs, arg)
		} else if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2) // Remove "--" and split by "="
			if len(parts) == 2 {
				options[parts[0]] = parts[1]
			} else if idx+1 < len(args) && !strings.HasPrefix(args[idx+1], "--") {
				options[arg[2:]] = args[idx+1]
				skipNext = true
			} else {
				// Flags without values default to "true"
				options[parts[0]] = "true"
			}
		}
	}

	return options, makeArgs
}

// transformToMakeVars converts kebab-cased options to snake-cased Makefile env variables
func transformToMakeVars(options map[string]string) []string {
	var makeVars []string

	for key, value := range options {
		// Convert kebab-case to snake_case and uppercase
		envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		makeVars = append(makeVars, fmt.Sprintf("%s=%s", envKey, value))
	}

	return makeVars
}

// callMake invokes the `make` command with the given target, environment variables, and additional arguments
func callMake(target string, makeVars []string, makeArgs []string) error {
	// Construct the command with target and makeArgs
	finalArgs := []string{target}
	finalArgs = append(finalArgs, makeVars...)
	finalArgs = append(finalArgs, makeArgs...)
	cmd := exec.Command("make", finalArgs...)

	// Use the same stdin, stdout, and stderr as the parent process
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command
	return cmd.Run()
}

// getRepoRoot determines the repo root (top-level directory) relative to this binary
func getRepoRoot() (string, error) {
	// Get the path of the currently executing binary
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Get the directory of the binary
	execDir, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	execDir = filepath.Dir(execDir)
	fmt.Println(execDir)

	// Compute the repo root relative to the binary's location
	repoRoot := filepath.Join(execDir, "../") // Adjust based on your repo structure

	return filepath.Abs(repoRoot) // Return the absolute path
}
