package installer

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/osquery/osquery-go"
	osquery_gen "github.com/osquery/osquery-go/gen/osquery"
)

type QueryResponse = osquery_gen.ExtensionResponse

// Client defines the methods required for the API requests to the server. The
// fleet.OrbitClient type satisfies this interface.
type Client interface {
	GetHostScript(execID string) (*fleet.HostScriptResult, error)
	GetInstaller(installerID, downloadDir string) (string, error)
	GetInstallerDetails(installId string) (*fleet.SoftwareInstallDetails, error)
	SaveInstallerResult(payload *fleet.HostSoftwareInstallResultPayload) error
}

type QueryClient interface {
	Query(context.Context, string) (*QueryResponse, error)
}

type Runner struct {
	OsqueryClient QueryClient
	OrbitClient   Client

	// tempDirFn is the function to call to get the temporary directory to use,
	// inside of which the script-specific subdirectories will be created. If nil,
	// the user's temp dir will be used (can be set to t.TempDir in tests).
	tempDirFn func() string

	// execCmdFn can be set for tests to mock actual execution of the script. If
	// nil, execCmd will be used, which has a different implementation on Windows
	// and non-Windows platforms.
	execCmdFn func(ctx context.Context, scriptPath string) ([]byte, int, error)

	// can be set for tests to replace os.RemoveAll, which is called to remove
	// the script's temporary directory after execution.
	removeAllFn func(string) error
}

func NewRunner(client Client, socketPath string, timeout time.Duration) (*Runner, error) {
	r := &Runner{
		OrbitClient: client,
	}

	osqueryClient, err := osquery.NewClient(socketPath, timeout)
	if err != nil {
		return nil, fmt.Errorf("creating new osquery client: %w", err)
	}

	r.OsqueryClient = osqueryClient.Client

	return r, nil
}
