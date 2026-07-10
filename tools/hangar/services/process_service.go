package services

import "github.com/fleetdm/fleet/tools/hangar/internal/processes"

// ProcessService exposes the process/log/docker engine to the frontend.
// Mirrors the process commands from processes.rs.
type ProcessService struct {
	pm     *processes.Manager
	onQuit func()
}

// NewProcessService wires the service to the manager and a quit callback
// (which performs the app teardown + exit after ShutdownNow).
func NewProcessService(pm *processes.Manager, onQuit func()) *ProcessService {
	return &ProcessService{pm: pm, onQuit: onQuit}
}

func (s *ProcessService) ListProcesses() []processes.ProcInfo { return s.pm.ListProcesses() }

func (s *ProcessService) StartProcess(id, label, cwd, program string, args []string, logChannel string, env []processes.EnvPair) error {
	return s.pm.Start(id, processes.StartArgs{
		Label: label, Cwd: cwd, Program: program, Args: args, LogChannel: logChannel, Env: env,
	})
}

func (s *ProcessService) StopProcess(id string) error    { return s.pm.Stop(id) }
func (s *ProcessService) RestartProcess(id string) error { return s.pm.Restart(id) }
func (s *ProcessService) ForgetProcess(id string) error  { return s.pm.Forget(id) }

// ShutdownNow stops all managed processes + docker, then triggers app exit.
func (s *ProcessService) ShutdownNow(repoPath string) error {
	s.pm.ShutdownNow(repoPath)
	if s.onQuit != nil {
		s.onQuit()
	}
	return nil
}

func (s *ProcessService) DockerComposeStatus(cwd string) (processes.DockerStatus, error) {
	return processes.DockerComposeStatus(cwd)
}
func (s *ProcessService) DockerComposeDown(cwd string) (string, error) {
	return s.pm.DockerComposeDown(cwd)
}
func (s *ProcessService) DockerComposeRestart(cwd string) (string, error) {
	return processes.DockerComposeRestart(cwd)
}

func (s *ProcessService) ServeTCPCheck(host string, port uint16) bool {
	return processes.ServeTCPCheck(host, port)
}

func (s *ProcessService) ReadLogWindow(source string, sinceMS uint64, levels []string, search *string, maxLines *int) processes.LogWindow {
	return s.pm.ReadLogWindow(source, sinceMS, levels, search, maxLines)
}
func (s *ProcessService) ClearLogChannel(channel string) error { return s.pm.ClearLogChannel(channel) }
func (s *ProcessService) SaveLogSnapshot(filename, contents string) (string, error) {
	return s.pm.SaveLogSnapshot(filename, contents)
}
func (s *ProcessService) LogsDirPath() string { return s.pm.LogsDirPath() }
