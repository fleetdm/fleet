package slots

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// ErrNoContainer means docker inspect found no such container.
var ErrNoContainer = errors.New("no such container")

// Probe reads Docker and port state. Every guard treats a probe error as a
// refusal.
type Probe interface {
	// DockerInspect returns `docker inspect <container>` JSON, or
	// ErrNoContainer when the container doesn't exist.
	DockerInspect(ctx context.Context, container string) ([]byte, error)
	// FleetTableCount counts tables in the container's fleet schema.
	FleetTableCount(ctx context.Context, container string) (int, error)
	// ProjectRunning reports whether any container labeled with the compose
	// project is running.
	ProjectRunning(ctx context.Context, project string) (bool, error)
	// Listening reports whether something accepts TCP connections on the
	// local port.
	Listening(port uint16) bool
}

// MySQLContainer is the compose v2 name of a project's mysql container.
func MySQLContainer(project string) string { return project + "-mysql-1" }

// DockerProbe is the production Probe, backed by the docker CLI.
type DockerProbe struct{}

func (DockerProbe) run(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	cmd := shellpath.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (p DockerProbe) DockerInspect(ctx context.Context, container string) ([]byte, error) {
	out, err := p.run(ctx, "inspect", "--type", "container", container)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no such container") {
		return nil, fmt.Errorf("%w: %s", ErrNoContainer, container)
	}
	return out, err
}

func (p DockerProbe) FleetTableCount(ctx context.Context, container string) (int, error) {
	out, err := p.run(ctx, "exec", "-e", "MYSQL_PWD=toor", container,
		"mysql", "-uroot", "--batch", "--skip-column-names", "--execute",
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'fleet'")
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0, fmt.Errorf("parsing fleet table count %q: %w", out, err)
	}
	return n, nil
}

func (p DockerProbe) ProjectRunning(ctx context.Context, project string) (bool, error) {
	if project == "" {
		return false, errors.New("empty compose project")
	}
	out, err := p.run(ctx, "ps", "--quiet", "--filter", "label=com.docker.compose.project="+project)
	if err != nil {
		return false, err
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

func (DockerProbe) Listening(port uint16) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port))), time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
