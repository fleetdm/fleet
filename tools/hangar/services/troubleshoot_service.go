package services

import "github.com/fleetdm/fleet/tools/hangar/internal/troubleshoot"

// TroubleshootService exposes port/pattern process scans and kill, plus a
// running-container scan and docker stop. The process half mirrors
// troubleshoot.rs.
type TroubleshootService struct{}

func (s *TroubleshootService) TroubleshootScanPort(port uint16) ([]troubleshoot.DetectedProcess, error) {
	return troubleshoot.ScanPort(port)
}
func (s *TroubleshootService) TroubleshootScanPattern(pattern string) ([]troubleshoot.DetectedProcess, error) {
	return troubleshoot.ScanPattern(pattern)
}
func (s *TroubleshootService) TroubleshootKillPid(pid uint32) troubleshoot.KillOutcome {
	return troubleshoot.KillPID(pid)
}
func (s *TroubleshootService) TroubleshootScanDocker() ([]troubleshoot.Container, error) {
	return troubleshoot.ScanDocker()
}
func (s *TroubleshootService) TroubleshootStopContainers(ids []string) error {
	return troubleshoot.StopContainers(ids)
}
