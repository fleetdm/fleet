// Package slots runs the agent API's slot lifecycle actions (build, Docker
// up, prepare DB, serve, stop, backup) as sequences of managed processes.
//
// The command builders port frontend/src/lib/servers.ts and
// orchestration.ts for offset stacks only: the API never runs s1's default
// stack, so the default-port branches aren't carried over.
package slots

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

// dbClientImage matches tools/backup_db/*.sh (${FLEET_MYSQL_IMAGE:-mysql:8.0.44}).
const dbClientImage = "mysql:8.0.44"

// ProcID namespaces a per-slot process id, e.g. "s2:fleet-serve".
func ProcID(slotID, base string) string { return slotID + ":" + base }

// ServeChannel is the slot's fleet serve log channel, shared with the GUI.
func ServeChannel(slotID string) string { return "fleet-serve-" + slotID }

// AgentChannel is the log channel for every non-serve agent step on a slot.
func AgentChannel(slotID string) string { return "agent-" + slotID }

// Step is one build-chain process.
type Step struct {
	ID      string
	Label   string
	Program string
	Args    []string
}

// BuildChain is make deps, generate, and build, namespaced to the slot.
func BuildChain(slot settings.ServerProfile) []Step {
	var out []Step
	for _, target := range []string{"deps", "generate", "build"} {
		out = append(out, Step{
			ID:      ProcID(slot.ID, "make-"+target),
			Label:   "make " + target,
			Program: "make",
			Args:    []string{target},
		})
	}
	return out
}

// DockerEnv maps the compose file's ${FLEET_*_PORT} substitutions to the slot.
func DockerEnv(slot settings.ServerProfile) []processes.EnvPair {
	p := slot.Ports
	return []processes.EnvPair{
		{Key: "FLEET_MYSQL_PORT", Value: fmt.Sprint(p.MySQL)},
		{Key: "FLEET_REDIS_PORT", Value: fmt.Sprint(p.Redis)},
		{Key: "FLEET_S3_PORT", Value: fmt.Sprint(p.S3)},
		{Key: "FLEET_S3_CONSOLE_PORT", Value: fmt.Sprint(p.S3Console)},
	}
}

// DockerUpArgs brings up only the services a parallel server needs, so the
// optional ones don't collide on host ports between projects.
func DockerUpArgs(slot settings.ServerProfile) []string {
	return []string{"compose", "-p", slot.ComposeProject, "up", "-d", "mysql", "redis", "s3"}
}

// ServeArgs is `fleet serve --dev` with the slot's addresses.
func ServeArgs(slot settings.ServerProfile) []string {
	cfg := slot.FleetServe
	args := []string{"serve", "--dev"}
	if cfg.ConfigPath != nil {
		if c := strings.TrimSpace(*cfg.ConfigPath); c != "" {
			args = append(args, "--config", c)
		}
	}
	if cfg.Premium {
		args = append(args, "--dev_license")
	}
	if cfg.Debug {
		args = append(args, "--debug")
	}
	if cfg.Logging {
		args = append(args, "--logging_debug")
	}
	p := slot.Ports
	return append(args,
		"--server_address", fmt.Sprintf("localhost:%d", p.Server),
		"--mysql_address", fmt.Sprintf("localhost:%d", p.MySQL),
		"--redis_address", fmt.Sprintf("localhost:%d", p.Redis),
	)
}

// ServeEnv is the slot's enabled env rows, then its own S3 endpoints.
func ServeEnv(slot settings.ServerProfile) []processes.EnvPair {
	var out []processes.EnvPair
	for _, e := range slot.FleetServe.Env {
		if k := strings.TrimSpace(e.Key); e.Enabled && k != "" {
			out = append(out, processes.EnvPair{Key: k, Value: e.Value})
		}
	}
	s3 := fmt.Sprintf("http://localhost:%d", slot.Ports.S3)
	return append(out,
		processes.EnvPair{Key: "FLEET_S3_SOFTWARE_INSTALLERS_ENDPOINT_URL", Value: s3},
		processes.EnvPair{Key: "FLEET_S3_CARVES_ENDPOINT_URL", Value: s3},
	)
}

// ServeLabel matches the GUI's fleet-serve row label.
func ServeLabel(slot settings.ServerProfile) string {
	if slot.FleetServe.Premium {
		return "fleet serve --dev (premium)"
	}
	return "fleet serve --dev (free)"
}

// PrepareDBArgs is `fleet prepare db --dev` against the slot's MySQL.
func PrepareDBArgs(slot settings.ServerProfile) []string {
	return []string{
		"prepare", "db", "--dev",
		fmt.Sprintf("--mysql_address=localhost:%d", slot.Ports.MySQL),
		"--mysql_username=root", "--mysql_password=toor", "--mysql_database=fleet",
	}
}

// DBBackupCommand is the `bash -c` payload that dumps the slot's fleet DB
// through its compose network to absPath. pipefail makes a failed mysqldump
// fail the command instead of writing an empty archive.
func DBBackupCommand(slot settings.ServerProfile, absPath string) string {
	return "docker run --rm --network " + shellQuote(slot.ComposeProject+"_default") + " " + dbClientImage + " " +
		"bash -c 'set -o pipefail; mysqldump -hmysql -uroot -ptoor " +
		"--default-character-set=utf8mb4 --add-drop-database --databases fleet | gzip -' " +
		"> " + shellQuote(absPath)
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }
