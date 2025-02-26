package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	// Ensure there's a make target specified.
	if len(os.Args) < 2 {
		fmt.Println("Usage: fdm <command> [--option=value ...] -- [make-options]")
		os.Exit(1)
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

	// Extract the make target.
	makeTarget := os.Args[1]

	// Split arguments into options and make arguments.
	options, makeArgs := splitArgs(os.Args[2:])

	// Special logic for the help command.
	if makeTarget == "help" {
		if len(os.Args) > 2 && !strings.HasPrefix(os.Args[2], "--") {
			options["REFORMAT_OPTIONS"] = "true"
		} else {
			fmt.Println("\033[1mNAME\033[0m")
			fmt.Println("  fdm - developer tools for fleet device management")
			fmt.Println()
			fmt.Println("\033[1mUSAGE:\033[0m")
			fmt.Println("  fdm <command> [--option=value ...] -- [make-options]")
			fmt.Println()
			fmt.Println("\033[1mCOMMANDS:\033[0m")
			options["HELP_CMD_PREFIX"] = "fdm"
		}
	}

	// Transform options into Makefile-compatible variables.
	makeVars := transformToMakeVars(options)
	makeVars = append(makeVars, "TOOL_CMD=fdm")

	// Call the Makefile with the specified target, Make variables, and additional arguments.
	err = callMake(makeTarget, makeVars, makeArgs)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

// splitArgs splits the arguments into options and make arguments based on the `--` delimiter.
func splitArgs(args []string) (map[string]string, []string) {
	options := make(map[string]string)
	var makeArgs []string
	positionalArgsIndex := 1
	isMakeArgs := false
	skipNext := false

	for idx, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}

		if arg == "--" {
			isMakeArgs = true
			continue
		}

		switch {
		// If we're processing make args (anything after a bare -- )
		// then add the current arg to the list.
		case isMakeArgs:
			makeArgs = append(makeArgs, arg)
		// Otherwise if the arg has a -- prefix, treat it like an option
		// for the command.
		case strings.HasPrefix(arg, "--"):
			// Remove "--" and split by "=".
			parts := strings.SplitN(arg[2:], "=", 2)
			switch {
			// Handle options like --name=foo
			case len(parts) == 2:
				options[parts[0]] = parts[1]
			// Handle options like --name foo
			case idx+1 < len(args) && !strings.HasPrefix(args[idx+1], "--"):
				options[arg[2:]] = args[idx+1]
				skipNext = true
			// Handle options like --useturbocharge by assuming they're booleans.
			default:
				options[parts[0]] = "true"
			}
		// Otherwise assume we're dealing with a positional argument.
		default:
			options["arg"+strconv.Itoa(positionalArgsIndex)] = arg
			positionalArgsIndex++
		}
	}

	return options, makeArgs
}

// transformToMakeVars converts kebab-cased options to snake-cased Makefile variables.
func transformToMakeVars(options map[string]string) []string {
	var makeVars []string

	for key, value := range options {
		// Convert kebab-case to snake_case and uppercase.
		varName := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		makeVars = append(makeVars, fmt.Sprintf("%s=%s", varName, value))
	}

	return makeVars
}

// callMake invokes the `make` command with the given target, variables, and additional arguments.
func callMake(target string, makeVars []string, makeArgs []string) error {
	// Construct the command with target and makeArgs.
	finalArgs := []string{target}
	finalArgs = append(finalArgs, makeVars...)
	finalArgs = append(finalArgs, makeArgs...)
	cmd := exec.Command("make", finalArgs...)

	// Use the same stdin, stdout, and stderr as the parent process.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command.
	return cmd.Run()
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
