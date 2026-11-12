package update

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/micromdm/plist"
	"github.com/rs/zerolog/log"
)

// On macOS, Activity Monitor and top name a process after its executable file, while ps and pgrep use the path launchd
// started. So a host shows Fleet once its executable is bin/orbit/macos/<channel>/Fleet and its LaunchDaemon starts
// bin/orbit/Fleet. Full Disk Access profiles match orbit by path, so the move waits for a profile that allows the new path.
const (
	macOSLaunchDaemonLabel = "com.fleetdm.orbit"
	macOSLaunchDaemonPath  = "/Library/LaunchDaemons/" + macOSLaunchDaemonLabel + ".plist"
	agentPathCheckInterval = 10 * time.Minute
	agentPathMaxRetryWait  = 6 * time.Hour
)

// OrbitLinkPath returns bin/orbit/orbit[.exe], the link to the current orbit executable that every orbit version maintains.
func OrbitLinkPath(rootDir, goos string) string {
	name := constant.OrbitTUFTargetName
	if goos == "windows" {
		name += ".exe"
	}
	return filepath.Join(rootDir, binDir, constant.OrbitTUFTargetName, name)
}

// FleetEntryPointPath returns bin/orbit/Fleet, the path the macOS LaunchDaemon starts once the host moved. It links to
// bin/orbit/orbit, so an older orbit, which only maintains that link, still starts through it after a downgrade.
func FleetEntryPointPath(rootDir string) string {
	return filepath.Join(rootDir, binDir, constant.OrbitTUFTargetName, constant.AgentDisplayName)
}

// AgentPathReceiver moves the macOS executable to the Fleet name once the server has fleet.CapabilityFleetAgentNames,
// and back if Full Disk Access profiles only allow the old path, then restarts orbit.
type AgentPathReceiver struct {
	serverCapabilities func() fleet.CapabilityMap
	triggerRestart     func(reason string)
	ensure             func() (moved bool, err error)

	nextCheck time.Time
	retryWait time.Duration // after a failed check, 0 after a successful one
}

func NewAgentPathReceiver(rootDir string, serverCapabilities func() fleet.CapabilityMap, triggerRestart func(reason string)) *AgentPathReceiver {
	return &AgentPathReceiver{
		serverCapabilities: serverCapabilities,
		triggerRestart:     triggerRestart,
		ensure:             func() (bool, error) { return ensureAgentPath(rootDir) },
	}
}

// Run checks the path at most every agentPathCheckInterval, because profiles rarely change and the check runs a command.
func (r *AgentPathReceiver) Run(*fleet.OrbitConfig) error {
	if !r.serverCapabilities().Has(fleet.CapabilityFleetAgentNames) {
		return nil
	}
	now := time.Now()
	if now.Before(r.nextCheck) {
		return nil
	}
	r.nextCheck = now.Add(agentPathCheckInterval)

	moved, err := r.ensure()
	if err != nil {
		// Back off: each failed attempt may copy the executable and run it. Only the first failure is an error, so a host
		// where it can't succeed doesn't log one on every check.
		if r.retryWait == 0 {
			log.Error().Err(err).Msg("check the agent's executable path")
			r.retryWait = agentPathCheckInterval
		} else {
			log.Debug().Err(err).Msg("check the agent's executable path")
		}
		r.retryWait = min(2*r.retryWait, agentPathMaxRetryWait)
		r.nextCheck = now.Add(r.retryWait)
		return nil
	}
	r.retryWait = 0
	if moved {
		r.triggerRestart("agent executable path changed")
	}
	return nil
}

// fdaGrants is what the installed profiles say about Full Disk Access for each path.
type fdaGrants struct {
	allowed map[string]bool
	denied  map[string]bool
}

// allows reports whether a profile allows path and none denies it: orbit doesn't count on access that profiles disagree on.
func (g fdaGrants) allows(path string) bool {
	return g.allowed[path] && !g.denied[path]
}

// parseFDAGrants reads the output of `profiles show -output stdout-xml`. Only device profiles grant Full Disk Access, and
// only path entries are known to match orbit, which isn't an app bundle. Payload contents vary by type (some are data), so
// it decodes them generically.
func parseFDAGrants(out []byte) (fdaGrants, error) {
	var byScope map[string][]struct {
		ProfileItems []struct {
			PayloadType    string `plist:"PayloadType"`
			PayloadContent any    `plist:"PayloadContent"`
		} `plist:"ProfileItems"`
	}
	if err := plist.Unmarshal(out, &byScope); err != nil {
		return fdaGrants{}, fmt.Errorf("parse profiles: %w", err)
	}
	grants := fdaGrants{allowed: map[string]bool{}, denied: map[string]bool{}}
	for _, p := range byScope["_computerlevel"] {
		for _, item := range p.ProfileItems {
			if item.PayloadType != "com.apple.TCC.configuration-profile-policy" {
				continue
			}
			content, _ := item.PayloadContent.(map[string]any)
			services, _ := content["Services"].(map[string]any)
			entries, _ := services["SystemPolicyAllFiles"].([]any)
			for _, e := range entries {
				entry, _ := e.(map[string]any)
				identifier, _ := entry["Identifier"].(string)
				if identifierType, _ := entry["IdentifierType"].(string); identifierType != "path" {
					continue
				}
				if pppcEntryAllows(entry) {
					grants.allowed[identifier] = true
				} else {
					grants.denied[identifier] = true
				}
			}
		}
	}
	return grants, nil
}

// pppcEntryAllows reports whether a PPPC entry grants access. Apple replaced the Allowed key with Authorization.
func pppcEntryAllows(entry map[string]any) bool {
	if auth, ok := entry["Authorization"].(string); ok {
		return auth == "Allow"
	}
	switch allowed := entry["Allowed"].(type) {
	case bool:
		return allowed
	case uint64:
		return allowed == 1
	case int64:
		return allowed == 1
	}
	return false
}

// chooseFleetPath returns whether orbit should run from its Fleet path. Without a profile for either path it moves only
// if it has no Full Disk Access to lose, and otherwise stays where it is: access granted by hand in System Settings isn't
// in any profile.
func chooseFleetPath(current bool, grants fdaGrants, fleetPath, orbitPath string, hasFullDiskAccess func() bool) bool {
	switch {
	case grants.allows(fleetPath):
		return true
	case grants.allows(orbitPath):
		return false
	case !hasFullDiskAccess():
		return true
	default:
		return current
	}
}

// launchDaemonProgram returns the path fleetd's LaunchDaemon starts. It fails for a LaunchDaemon that doesn't start
// exactly one path with no arguments, as fleetd's packages write it, so a customized one is left alone.
func launchDaemonProgram(data []byte) (string, error) {
	var daemon map[string]any
	if err := plist.Unmarshal(data, &daemon); err != nil {
		return "", fmt.Errorf("parse LaunchDaemon: %w", err)
	}
	if _, ok := daemon["Program"]; ok {
		return "", errors.New("LaunchDaemon has a Program key")
	}
	args, _ := daemon["ProgramArguments"].([]any)
	if len(args) != 1 {
		return "", fmt.Errorf("LaunchDaemon has %d program arguments, want 1", len(args))
	}
	program, ok := args[0].(string)
	if !ok || program == "" {
		return "", errors.New("LaunchDaemon program isn't a path")
	}
	return program, nil
}

// setLaunchDaemonProgram returns the LaunchDaemon with program as the path it starts, everything else unchanged.
func setLaunchDaemonProgram(data []byte, program string) ([]byte, error) {
	if _, err := launchDaemonProgram(data); err != nil {
		return nil, err
	}
	daemon := map[string]any{}
	if err := plist.Unmarshal(data, &daemon); err != nil {
		return nil, fmt.Errorf("parse LaunchDaemon: %w", err)
	}
	daemon["ProgramArguments"] = []any{program}
	out, err := plist.MarshalIndent(daemon, "\t")
	if err != nil {
		return nil, fmt.Errorf("encode LaunchDaemon: %w", err)
	}
	// Check the result before it replaces the file launchd reads at boot.
	if got, err := launchDaemonProgram(out); err != nil || got != program {
		return nil, fmt.Errorf("encoded LaunchDaemon doesn't start %s: %v", program, err)
	}
	return out, nil
}
