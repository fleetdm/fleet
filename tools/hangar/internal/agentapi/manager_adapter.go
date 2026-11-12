package agentapi

import (
	"context"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
)

// managerAdapter maps the API's Manager onto *processes.Manager. Hold,
// Release, HeldBy, Wait, ListProcesses, and ReadLogWindowScrubbed pass
// through.
type managerAdapter struct{ *processes.Manager }

// NewManager adapts the process manager the GUI uses.
func NewManager(m *processes.Manager) Manager { return managerAdapter{m} }

func (a managerAdapter) Start(id string, args processes.StartArgs) (processes.Invocation, error) {
	return a.Manager.StartInvocation(id, args)
}

func (a managerAdapter) Stop(id, owner string) error { return a.Manager.StopAs(id, owner) }

func (a managerAdapter) ComposeDown(id, cwd, project, owner string) error {
	_, err := a.Manager.DockerComposeDownAs(id, cwd, project, owner)
	return err
}

// ComposeDownContext is ComposeDown that ends, killing the compose CLI, when
// ctx does.
func (a managerAdapter) ComposeDownContext(ctx context.Context, id, cwd, project, owner string) error {
	_, err := a.Manager.DockerComposeDownContext(ctx, id, cwd, project, owner)
	return err
}
