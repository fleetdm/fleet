package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// proc is a long-running supervised process. Output is teed to a log file
// (so coding agents can `tail` it) and forwarded to a sink channel as
// individual log lines.
//
// proc is intentionally minimal — for one-shot commands (make deps,
// prepare db, etc.) we use runOneShot() instead.
type proc struct {
	name    string
	cmd     *exec.Cmd
	logFile *os.File
	wg      sync.WaitGroup
}

// logLine is what proc and runOneShot emit on the sink channel. The TUI
// turns these into log pane lines and tea.Msg tokens.
type logLine struct {
	Source string // "fleet", "webpack", "ngrok", "start"
	Line   string
}

// startProc launches a command in dir with env, captures stdout+stderr to
// logPath, and forwards every line to sink. Returns immediately once the
// process is spawned — the caller is responsible for stopping it.
func startProc(name, dir string, env []string, logPath string, sink chan<- logLine, command string, args ...string) (*proc, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir for %s log: %w", name, err)
	}
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open %s log: %w", name, err)
	}

	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	// Run in its own process group so we can kill children too (webpack
	// spawns sub-processes via make generate-dev).
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		// errors.Join silently drops nil errors, so a clean Close
		// leaves the returned error untouched; a Close failure gets
		// surfaced alongside the original error rather than swallowed.
		return nil, errors.Join(fmt.Errorf("stdout pipe: %w", err), logFile.Close())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, errors.Join(fmt.Errorf("stderr pipe: %w", err), logFile.Close())
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.Join(fmt.Errorf("start %s: %w", name, err), logFile.Close())
	}

	p := &proc{name: name, cmd: cmd, logFile: logFile}
	p.wg.Add(2)
	go p.scan(stdout, sink)
	go p.scan(stderr, sink)

	return p, nil
}

func (p *proc) scan(r io.Reader, sink chan<- logLine) {
	defer p.wg.Done()
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		// Tee to disk; ignore errors here — we don't want a full disk to
		// crash the whole TUI.
		fmt.Fprintln(p.logFile, line)
		// Non-blocking send to the sink — if the TUI hasn't drained yet
		// we drop the line rather than stalling the reader.
		select {
		case sink <- logLine{Source: p.name, Line: line}:
		default:
		}
	}
}

// Stop sends SIGTERM, waits up to grace, then SIGKILLs the whole process
// group. Always closes the log file. A failure to close the log file
// (rare — usually buffered-write fsync failures) is joined into the
// returned error rather than silently dropped.
func (p *proc) Stop(ctx context.Context, grace time.Duration) (retErr error) {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	defer func() {
		if cerr := p.logFile.Close(); cerr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("close %s log: %w", p.name, cerr))
		}
	}()

	pgid, err := syscall.Getpgid(p.cmd.Process.Pid)
	if err != nil {
		// Fall back to single-process kill.
		_ = p.cmd.Process.Signal(syscall.SIGTERM)
	} else {
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
	}

	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()

	select {
	case <-time.After(grace):
		if pgid > 0 {
			_ = syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			_ = p.cmd.Process.Kill()
		}
		<-done
		p.wg.Wait()
		return errors.New(p.name + ": forced kill after timeout")
	case err := <-done:
		p.wg.Wait()
		// Exit-on-signal isn't a "real" error for our purposes.
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
					return nil
				}
			}
		}
		return err
	}
}

// runOneShot executes a command synchronously, streams output, returns the
// exit error. Used for "make deps", "fleet prepare db", etc. — anything
// expected to terminate quickly.
func runOneShot(ctx context.Context, name, dir string, env []string, sink chan<- logLine, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	scan := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			select {
			case sink <- logLine{Source: name, Line: sc.Text()}:
			default:
			}
		}
	}
	go scan(stdout)
	go scan(stderr)

	err = cmd.Wait()
	wg.Wait()
	return err
}
