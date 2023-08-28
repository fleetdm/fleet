// Command run-scripts is a tool for testing script execution on hosts exactly
// as Orbit would do following reception of a Fleet server notification of
// pending script(s) to execute.
//
// It allows to run such scripts without having to build and deploy orbit on
// the target host and without having to enroll that host in fleet and have the
// fleet server send script execution requests to it.
//
// The results of script execution, as reported by the host, are printed to the
// standard output.
//
// Usage on the host:
//
//	run-scripts
//	run-scripts -exec-id my-specific-id -content 'echo "Hello, world!"'
//	run-scripts -scripts-disabled -content 'echo "Hello, world!"'
//	run-scripts -scripts-count 10
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/fleetdm/fleet/v4/orbit/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
)

func main() {
	execIDFlag := flag.String("exec-id", "", "Execution ID of the script, will be auto-generated if empty.")
	contentFlag := flag.String("content", "echo \"Hello\"", "Content of the script to execute.")
	scriptsDisabledFlag := flag.Bool("scripts-disabled", false, "Disable execution of scripts on the host.")
	scriptsCountFlag := flag.Int("scripts-count", 1, "Number of scripts to execute. If > 1, the content will all be the same and exec-id will be auto-generated.")

	flag.Parse()

	if *scriptsCountFlag < 1 {
		log.Fatal("scripts-count must be >= 1")
	}

	cli := mockClient{content: *contentFlag}
	runner := &scripts.Runner{
		ScriptExecutionEnabled: !*scriptsDisabledFlag,
		Client:                 cli,
	}

	execIDs := make([]string, *scriptsCountFlag)
	for i := range execIDs {
		if *execIDFlag != "" && len(execIDs) == 1 {
			execIDs[i] = *execIDFlag
			break
		}
		execIDs[i] = uuid.New().String()
	}

	if err := runner.Run(execIDs); err != nil {
		log.Fatal(err)
	}
}

type mockClient struct {
	content string
}

func (m mockClient) GetHostScript(execID string) (*fleet.HostScriptResult, error) {
	return &fleet.HostScriptResult{
		HostID:         1,
		ExecutionID:    execID,
		ScriptContents: m.content,
	}, nil
}

func (m mockClient) SaveHostScriptResult(result *fleet.HostScriptResultPayload) error {
	fmt.Printf(`
Script result for %q:
  Exit code: %d
  Runtime:   %d second(s)
  Output:
%s
---
`, result.ExecutionID, result.ExitCode, result.Runtime, result.Output)
	return nil
}
