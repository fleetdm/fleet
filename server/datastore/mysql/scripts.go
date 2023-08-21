package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (ds *Datastore) GetScriptResult(ctx context.Context, scriptID uint) (*fleet.ScriptResult, error) {
	// TODO: implement when we have results data setup
	return &fleet.ScriptResult{
		ScriptContents: "echo 'hello world'",
		ExitCode:       0,
		Output:         "hello world",
		Message:        "",
		Runtime:        0,
	}, nil
}
