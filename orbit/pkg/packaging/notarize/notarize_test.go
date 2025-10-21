package notarize

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
)

func TestMain(m *testing.M) {
	// Set our default logger
	logger := hclog.L()
	logger.SetLevel(hclog.Trace)
	hclog.SetDefault(logger)

	// If we got a subcommand, run that
	if v := os.Getenv(childEnv); v != "" && childCommands[v] != nil {
		os.Exit(childCommands[v]())
	}

	os.Exit(m.Run())
}

// childEnv is the env var that must be set to trigger a child command.
const childEnv = "GON_TEST_CHILD"

// childCommands is the list of commands we support
var childCommands = map[string]func() int{}

// childCmd is used to create a command that executes a command in the
// childCommands map in a new process.
func childCmd(t *testing.T, name string, args ...string) *exec.Cmd {
	t.Helper()

	// Get the path to our executable
	selfPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatalf("error creating child command: %s", err)
		return nil
	}

	cmd := exec.Command(selfPath, args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, childEnv+"="+name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}
