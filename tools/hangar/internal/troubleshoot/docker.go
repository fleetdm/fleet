package troubleshoot

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// Container is a running Docker container found by ScanDocker.
type Container struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	// Project and Service are the compose labels; both "" for a container
	// started with plain docker run (e.g. fleetctl package's .msi build).
	Project string `json:"project"`
	Service string `json:"service"`
	Status  string `json:"status"`
	// Ports are the published host ports.
	Ports []uint16 `json:"ports"`
}

const containerFormat = `{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Label "com.docker.compose.project"}}\t{{.Label "com.docker.compose.service"}}\t{{.Status}}\t{{.Ports}}`

// ScanDocker lists running containers. Hangar starts each server's stack with
// `docker compose up -d`, so what outlives a crash or a removed server profile
// is containers, not a process ScanPattern could find.
func ScanDocker() ([]Container, error) {
	// docker ps can hang while Docker Desktop is starting.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := shellpath.CommandContext(ctx, "docker", "ps", "--format", containerFormat).Output()
	if err != nil {
		return nil, dockerError(ctx, err)
	}
	return parseContainers(string(out)), nil
}

var containerID = regexp.MustCompile(`^[0-9a-f]{12,64}$`)

// StopContainers runs docker stop on ids from ScanDocker: each container gets
// its stop grace period, then is killed. Containers and volumes are kept, so
// compose can start the stack again with its data.
func StopContainers(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	for _, id := range ids {
		if !containerID.MatchString(id) {
			return fmt.Errorf("not a container id: %q", id)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if _, err := shellpath.CommandContext(ctx, "docker", append([]string{"stop"}, ids...)...).Output(); err != nil {
		return dockerError(ctx, err)
	}
	return nil
}

func dockerError(ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return errors.New("docker didn't answer in time; is Docker Desktop still starting?")
	}
	if errors.Is(err, exec.ErrNotFound) {
		return errors.New("docker isn't installed (not found on PATH)")
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		msg := strings.TrimSpace(string(ee.Stderr))
		if strings.Contains(msg, "Cannot connect to the Docker daemon") {
			return errors.New("docker isn't running; start Docker Desktop")
		}
		if msg != "" {
			return errors.New(msg)
		}
	}
	return err
}

func parseContainers(raw string) []Container {
	out := []Container{}
	for _, line := range strings.Split(raw, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 7 {
			continue
		}
		out = append(out, Container{
			ID: f[0], Name: f[1], Image: f[2], Project: f[3], Service: f[4], Status: f[5],
			Ports: publishedPorts(f[6]),
		})
	}
	return out
}

var hostPort = regexp.MustCompile(`:(\d+)(?:-(\d+))?->`)

// publishedPorts reads the host side of docker ps's Ports column, e.g.
// "0.0.0.0:3306->3306/tcp, [::]:3306->3306/tcp, 33060/tcp" is [3306].
func publishedPorts(s string) []uint16 {
	ports := []uint16{}
	seen := map[uint64]bool{}
	for _, m := range hostPort.FindAllStringSubmatch(s, -1) {
		lo, err := strconv.ParseUint(m[1], 10, 16)
		if err != nil {
			continue
		}
		hi := lo
		if h, err := strconv.ParseUint(m[2], 10, 16); err == nil && h >= lo && h-lo < 1024 {
			hi = h
		}
		for p := lo; p <= hi; p++ {
			if !seen[p] {
				seen[p] = true
				ports = append(ports, uint16(p))
			}
		}
	}
	return ports
}
