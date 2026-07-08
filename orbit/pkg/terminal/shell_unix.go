//go:build !windows

package terminal

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"syscall"

	"github.com/creack/pty"
)

// shell wraps a running shell process backed by a real PTY.
// The shell is started as the root user with a clean login environment,
// mirroring what an SSH session as root provides.
type shell struct {
	cmd *exec.Cmd
	ptm *os.File // master end of the PTY (read output / write input)
}

func startShell(ctx context.Context) (*shell, error) {
	// Prefer bash; fall back to zsh (macOS default since Catalina) then sh.
	bin := "/bin/sh"
	for _, candidate := range []string{"bash", "zsh"} {
		if p, err := exec.LookPath(candidate); err == nil {
			bin = p
			break
		}
	}

	// Resolve root's actual home directory.
	// Linux: /root   macOS: /var/root
	rootHome := "/root"
	if u, err := user.Lookup("root"); err == nil && u.HomeDir != "" {
		rootHome = u.HomeDir
	}

	cmd := exec.CommandContext(ctx, bin, "--login")
	cmd.Dir = rootHome // start in root's home, like SSH

	// Clean root login environment, matching what SSH provides.
	cmd.Env = []string{
		"HOME=" + rootHome,
		"USER=root",
		"LOGNAME=root",
		"SHELL=" + bin,
		"TERM=xterm-256color",
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}

	// pty.StartWithAttrs creates a PTY pair, wires stdin/stdout/stderr to the
	// slave end, and starts the process.  We supply our own SysProcAttr so we
	// can set the UID/GID to root (0/0) explicitly — orbit already runs as
	// root, but being explicit mirrors the SSH model.
	ptm, err := pty.StartWithAttrs(cmd, nil, &syscall.SysProcAttr{
		Setsid: true, // new session so PTY becomes the controlling terminal
		Credential: &syscall.Credential{
			Uid: 0,
			Gid: 0,
		},
	})
	if err != nil {
		return nil, err
	}

	return &shell{cmd: cmd, ptm: ptm}, nil
}

func (s *shell) read(p []byte) (int, error) {
	return s.ptm.Read(p)
}

func (s *shell) write(p []byte) (int, error) {
	return s.ptm.Write(p)
}

// resize updates the PTY window size so the shell prompt and ncurses apps
// (vim, htop, etc.) adapt to the current browser terminal dimensions.
func (s *shell) resize(cols, rows uint16) error {
	return pty.Setsize(s.ptm, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
}

func (s *shell) close() {
	s.ptm.Close()
	if s.cmd.Process != nil {
		// Kill the entire process group so any child processes also die.
		syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL) //nolint:errcheck
	}
	s.cmd.Wait() //nolint:errcheck
}
