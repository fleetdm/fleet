//go:build darwin

package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/constant"
	"github.com/rs/zerolog/log"
)

// isLaunchDaemonProcess reports whether this process is the one fleetd's LaunchDaemon started, rather than a command
// run by hand or by fleetctl preview.
func isLaunchDaemonProcess() bool {
	return os.Getenv("XPC_SERVICE_NAME") == macOSLaunchDaemonLabel
}

// UsesFleetPath reports whether fleetd's LaunchDaemon starts bin/orbit/Fleet under rootDir, in which case the orbit
// executable is stored as Fleet.
func UsesFleetPath(rootDir string) bool {
	data, err := os.ReadFile(macOSLaunchDaemonPath)
	if err != nil {
		return false
	}
	program, err := launchDaemonProgram(data)
	return err == nil && program == FleetEntryPointPath(rootDir)
}

// ReexecAsFleet replaces this process with the same executable started through bin/orbit/Fleet, so ps and pgrep show
// Fleet before launchd loads the changed LaunchDaemon at the next boot. It keeps the pid, so launchd doesn't notice. It
// returns when it doesn't exec, and must run before orbit starts anything.
func ReexecAsFleet(rootDir string) {
	entryPoint := FleetEntryPointPath(rootDir)
	if !isLaunchDaemonProcess() || os.Args[0] == entryPoint || !UsesFleetPath(rootDir) {
		return
	}
	exe, err := resolvedExecutable()
	if err != nil {
		log.Error().Err(err).Msg("find the orbit executable, not restarting as Fleet")
		return
	}
	// The entry point links to bin/orbit/orbit; right after an update that link may point at another file.
	if target, err := filepath.EvalSymlinks(entryPoint); err != nil || target != exe {
		log.Info().Str("entry_point", entryPoint).Msg("entry point doesn't start this executable, not restarting as Fleet")
		return
	}
	log.Info().Str("path", entryPoint).Msg("restarting as Fleet")
	err = syscall.Exec(entryPoint, append([]string{entryPoint}, os.Args[1:]...), os.Environ())
	log.Error().Err(err).Msg("restart as Fleet, continuing as orbit")
}

func resolvedExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// ensureAgentPath moves orbit to bin/orbit/macos/<channel>/Fleet, started through bin/orbit/Fleet, when Full Disk Access
// allows that path (see chooseFleetPath), and back otherwise. It copies the running executable to the other name and
// changes the path the LaunchDaemon starts; after the restart, the update runner points bin/orbit/orbit at the copy.
// It returns whether orbit must restart.
func ensureAgentPath(rootDir string) (bool, error) {
	if !isLaunchDaemonProcess() {
		return false, nil
	}
	exe, err := resolvedExecutable()
	if err != nil {
		return false, fmt.Errorf("find the orbit executable: %w", err)
	}
	return moveAgentPath(agentPathEnv{
		rootDir:           rootDir,
		exe:               exe,
		launchDaemonPath:  macOSLaunchDaemonPath,
		profiles:          profilesShowXML,
		hasFullDiskAccess: hasFullDiskAccess,
		checkExecutable:   checkExecutable,
	})
}

// agentPathEnv is what moveAgentPath reads from the host, so tests can provide it.
type agentPathEnv struct {
	rootDir           string
	exe               string // the running executable, symlinks resolved
	launchDaemonPath  string
	profiles          func() ([]byte, error)
	hasFullDiskAccess func() bool
	checkExecutable   func(path string) error
}

func moveAgentPath(env agentPathEnv) (bool, error) {
	rootDir, exe := env.rootDir, env.exe
	channelDir := filepath.Dir(exe)
	if filepath.Dir(filepath.Dir(channelDir)) != filepath.Join(rootDir, binDir, constant.OrbitTUFTargetName) {
		return false, nil // not an orbit managed by the updater, for example a development build
	}
	data, err := os.ReadFile(env.launchDaemonPath)
	if err != nil {
		return false, fmt.Errorf("read LaunchDaemon: %w", err)
	}
	program, err := launchDaemonProgram(data)
	if err != nil {
		return false, err
	}
	entryPoint, orbitLink := FleetEntryPointPath(rootDir), OrbitLinkPath(rootDir, "darwin")
	if program != entryPoint && program != orbitLink {
		return false, fmt.Errorf("LaunchDaemon starts %s, leaving it alone", program)
	}
	current := program == entryPoint

	out, err := env.profiles()
	if err != nil {
		return false, err
	}
	grants, err := parseFDAGrants(out)
	if err != nil {
		return false, err
	}
	fleetPath := filepath.Join(channelDir, constant.AgentDisplayName)
	orbitPath := filepath.Join(channelDir, constant.OrbitTUFTargetName)
	want := chooseFleetPath(current, grants, fleetPath, orbitPath, env.hasFullDiskAccess)
	if want == current {
		return false, nil
	}

	target, newProgram := orbitPath, orbitLink
	if want {
		target, newProgram = fleetPath, entryPoint
	}
	if target != exe {
		if err := copyExecutable(exe, target); err != nil {
			return false, err
		}
	}
	if err := env.checkExecutable(target); err != nil {
		return false, err
	}
	if err := ensureSymlink(entryPoint, orbitLink); err != nil {
		return false, err
	}
	// Last, because launchd reads it at the next boot: everything it starts must already be in place.
	newData, err := setLaunchDaemonProgram(data, newProgram)
	if err != nil {
		return false, err
	}
	if err := writeFileAtomically(env.launchDaemonPath, newData, 0o644); err != nil {
		return false, fmt.Errorf("write LaunchDaemon: %w", err)
	}
	log.Info().Str("executable", target).Str("launch_daemon_program", newProgram).Msg("moved the agent executable")
	return true, nil
}

// profilesShowXML lists the installed configuration profiles, Full Disk Access ones included. It doesn't need Full Disk
// Access, unlike reading the TCC database that profiles feed.
func profilesShowXML() ([]byte, error) {
	// Short: it runs before orbit starts its subsystems.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "/usr/bin/profiles", "show", "-output", "stdout-xml").Output()
	if err != nil {
		return nil, fmt.Errorf("list configuration profiles: %w", err)
	}
	// Without profiles it prints an empty dict, so anything other than a plist is unexpected.
	if !bytes.Contains(out, []byte("<plist")) {
		return nil, fmt.Errorf("list configuration profiles: unexpected output %q", out)
	}
	return out, nil
}

// hasFullDiskAccess reports whether orbit can read the TCC directory, which needs Full Disk Access. An unexpected error
// counts as access, so orbit doesn't move without a profile allowing the new path.
func hasFullDiskAccess() bool {
	_, err := os.ReadDir("/Library/Application Support/com.apple.TCC")
	return !errors.Is(err, fs.ErrPermission)
}

func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".tmp")
	if err != nil {
		return fmt.Errorf("create copy of %s: %w", src, err)
	}
	defer os.Remove(tmp.Name()) // fails harmlessly after the rename
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod copy of %s: %w", src, err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync copy of %s: %w", src, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close copy of %s: %w", src, err)
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return fmt.Errorf("rename copy to %s: %w", dst, err)
	}
	return nil
}

// checkExecutable runs the executable before the LaunchDaemon depends on it.
func checkExecutable(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return fmt.Errorf("run %s --version: %w", path, err)
	}
	if !strings.HasPrefix(string(out), "orbit ") {
		return fmt.Errorf("run %s --version: unexpected output %q", path, out)
	}
	return nil
}

// ensureSymlink makes linkPath a symlink to target, replacing whatever is there in one rename.
func ensureSymlink(linkPath, target string) error {
	if existing, err := os.Readlink(linkPath); err == nil && existing == target {
		return nil
	}
	tmp := linkPath + ".tmp"
	_ = os.Remove(tmp)
	if err := os.Symlink(target, tmp); err != nil {
		return fmt.Errorf("create link %s: %w", linkPath, err)
	}
	if err := os.Rename(tmp, linkPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace link %s: %w", linkPath, err)
	}
	return nil
}

// writeFileAtomically replaces path in one rename after flushing the data, so a crash or power loss leaves the old or the
// new file, never a partial one. The temporary name doesn't end in .plist, so launchd never picks it up.
func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // fails harmlessly after the rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
