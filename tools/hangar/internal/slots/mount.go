package slots

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const overrideFile = "docker-compose.override.yml"

type override struct {
	Volumes map[string]struct {
		Name string `yaml:"name"`
	} `yaml:"volumes"`
	Workflow struct {
		Generation string `yaml:"generation"`
	} `yaml:"x-fleet-workflow"`
}

type inspected struct {
	Args   []string
	Mounts []struct {
		Type        string
		Name        string
		Destination string
	}
}

// CheckMount refuses unless the slot's MySQL keeps its data in a volume the
// worktree's override names or, without an override, in the compose
// project's own volume, as the GUI uses it. The data volume is the one
// mounted at or above mysqld's --datadir (/tmp on older compose files,
// /var/lib/mysql on newer ones). When the override records a generation, the
// volume must belong to it, which catches a fresh-data override whose /tmp
// volume still holds the previous DB.
func CheckMount(ctx context.Context, p Probe, worktree, project string) error {
	b, err := os.ReadFile(filepath.Join(worktree, overrideFile))
	hasOverride := !errors.Is(err, fs.ErrNotExist)
	if hasOverride && err != nil {
		return fmt.Errorf("mount check: reading %s: %w", overrideFile, err)
	}
	var ov override
	named := map[string]bool{}
	if hasOverride {
		if err := yaml.Unmarshal(b, &ov); err != nil {
			return fmt.Errorf("mount check: parsing %s: %w", overrideFile, err)
		}
		for _, v := range ov.Volumes {
			if v.Name != "" {
				named[v.Name] = true
			}
		}
		if len(named) == 0 {
			return fmt.Errorf("mount check: %s names no volumes", overrideFile)
		}
	}

	container := MySQLContainer(project)
	raw, err := p.DockerInspect(ctx, container)
	if !hasOverride && errors.Is(err, ErrNoContainer) {
		return nil // nothing holds data yet; docker-up creates the project's volume
	}
	if err != nil {
		return fmt.Errorf("mount check: %w", err)
	}
	var list []inspected
	if err := json.Unmarshal(raw, &list); err != nil {
		return fmt.Errorf("mount check: parsing docker inspect: %w", err)
	}
	if len(list) != 1 {
		return fmt.Errorf("mount check: docker inspect returned %d containers for %s", len(list), container)
	}
	c := list[0]

	datadir := datadirFromArgs(c.Args)
	best := -1
	for i, m := range c.Mounts {
		d := filepath.Clean(m.Destination)
		if datadir != d && !strings.HasPrefix(datadir, d+"/") {
			continue
		}
		if best < 0 || len(d) > len(filepath.Clean(c.Mounts[best].Destination)) {
			best = i
		}
	}
	if best < 0 {
		return fmt.Errorf("mount check: no mount holds %s datadir %s", container, datadir)
	}
	m := c.Mounts[best]
	if !hasOverride {
		// Compose names an undeclared-name volume <project>_<key>; the key is
		// docker-compose.yml's mysql-persistent-volume.
		if want := project + "_mysql-persistent-volume"; m.Type != "volume" || m.Name != want {
			return fmt.Errorf("mount check: %s datadir %s is on %s %q, not the project's %q", container, datadir, m.Type, m.Name, want)
		}
		return nil
	}
	if m.Type != "volume" || !named[m.Name] {
		return fmt.Errorf("mount check: %s datadir %s is on %s %q, which %s doesn't name", container, datadir, m.Type, m.Name, overrideFile)
	}
	if g := ov.Workflow.Generation; g != "" && !strings.HasSuffix(m.Name, "-"+g) {
		return fmt.Errorf("mount check: %s datadir volume %q isn't from override generation %s", container, m.Name, g)
	}
	return nil
}

func datadirFromArgs(args []string) string {
	for i, a := range args {
		if v, ok := strings.CutPrefix(a, "--datadir="); ok {
			return filepath.Clean(v)
		}
		if a == "--datadir" && i+1 < len(args) {
			return filepath.Clean(args[i+1])
		}
	}
	return "/var/lib/mysql"
}
