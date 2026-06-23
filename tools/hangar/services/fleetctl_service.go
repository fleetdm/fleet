package services

import "github.com/fleetdm/fleet/tools/hangar/internal/fleetctl"

// FleetctlService exposes fleetctl binary resolution, config read/write, and
// a capture runner. Mirrors fleetctl.rs.
type FleetctlService struct{}

func (s *FleetctlService) FleetctlResolveBinary(repo, settingsPath string) fleetctl.ResolvedBinary {
	return fleetctl.ResolveBinary(repo, settingsPath)
}
func (s *FleetctlService) FleetctlReadContext() (fleetctl.ContextInfo, error) {
	return fleetctl.ReadContext(fleetctl.DefaultConfigPath())
}
func (s *FleetctlService) FleetctlReadConfigRaw() (fleetctl.RawConfig, error) {
	return fleetctl.ReadConfigRaw(fleetctl.DefaultConfigPath())
}
func (s *FleetctlService) FleetctlSaveConfig(yaml string) error {
	return fleetctl.SaveConfig(fleetctl.DefaultConfigPath(), yaml)
}
func (s *FleetctlService) FleetctlRunCapture(program, cwd string, args []string, env map[string]string, stdinData string, timeoutMS uint64) (fleetctl.CapturedRun, error) {
	return fleetctl.RunCapture(program, cwd, args, env, stdinData, timeoutMS)
}
