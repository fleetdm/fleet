package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/rs/zerolog/log"
)

// IsExecCorruptionErr reports whether err indicates that an executable on disk
// is corrupt (e.g. truncated or otherwise malformed), as opposed to a transient
// runtime failure.
//
// These are the errors observed when orbit fork/execs a binary that was
// corrupted during a TUF download/extraction (see
// https://github.com/fleetdm/fleet/issues/47552). A corrupt binary fails to
// exec the same way on every restart, so without self-healing orbit crash-loops
// on it forever.
func IsExecCorruptionErr(err error) bool {
	if err == nil {
		return false
	}
	// Linux: "exec format error" (ENOEXEC) for a truncated or wrong-format binary.
	// We check both the error chain and the message, because some callers
	// stringify the error and break the chain.
	if errors.Is(err, syscall.ENOEXEC) {
		return true
	}
	msg := err.Error()
	switch {
	// Linux: ENOEXEC message ("exec format error").
	case strings.Contains(msg, "exec format error"):
		return true
	// macOS: the Go runtime reports a malformed Mach-O binary as "malformed Mach-o file".
	case strings.Contains(msg, "malformed Mach-o"):
		return true
	// Windows: ERROR_BAD_EXE_FORMAT surfaces as "is not a valid Win32 application".
	case strings.Contains(msg, "not a valid Win32 application"):
		return true
	}
	return false
}

// CheckExec verifies that the target's installed executable can run, using the
// same check applied to freshly downloaded targets (the target's CustomCheckExec
// if set, otherwise running it with --help).
//
// It returns nil for targets that can't be verified on the current
// platform/arch (e.g. when cross-building packages). A non-nil error means the
// on-disk executable failed to run; use IsExecCorruptionErr to distinguish a
// corrupt binary from a transient failure.
//
// NOTE: this duplicates the platform/arch guards in checkExec on purpose:
// checkExec verifies a freshly downloaded archive (and is on the critical
// download path), whereas CheckExec verifies the already-installed executable.
func (u *Updater) CheckExec(target string) error {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return fmt.Errorf("load local target %s: %w", target, err)
	}

	platformGOOS, err := goosFromPlatform(localTarget.Info.Platform)
	if err != nil {
		return err
	}
	if platformGOOS != runtime.GOOS {
		// Can't verify a binary for another OS (happens when cross-building packages).
		return nil
	}
	platformGOARCH, err := goarchFromPlatform(localTarget.Info.Platform)
	if err != nil {
		return err
	}
	var containsArch bool
	for _, arch := range platformGOARCH {
		if arch == runtime.GOARCH {
			containsArch = true
		}
	}
	if !containsArch && len(os.Args) > 0 && strings.HasSuffix(os.Args[0], "fleetctl") {
		// Can't reliably execute a cross-architecture binary (happens when cross-building).
		return nil
	}

	if localTarget.Info.CustomCheckExec != nil {
		if err := localTarget.Info.CustomCheckExec(localTarget.ExecPath); err != nil {
			return fmt.Errorf("custom exec check %q: %w", localTarget.ExecPath, err)
		}
		return nil
	}

	// Note: this would fail for any binary that returns nonzero for --help.
	cmd := exec.Command(localTarget.ExecPath, "--help")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("exec check %q: %s: %w", localTarget.ExecPath, string(out), err)
	}
	return nil
}

// RemoveTarget removes the on-disk artifacts for the given target so that the
// next call to Get re-downloads and re-extracts it from the remote TUF
// repository. It removes:
//
//   - the extracted directory (e.g. .../<version>/osquery.app), if any;
//   - the downloaded archive (e.g. .../osqueryd.app.tar.gz); and
//   - the cached archive hash (.sha512).
//
// Removing the archive (not just the extracted directory) forces a fresh
// download from TUF rather than re-extracting a possibly-corrupt archive.
//
// This is used to self-heal from a corrupt component binary that fails to
// fork/exec (see IsExecCorruptionErr).
func (u *Updater) RemoveTarget(target string) error {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return fmt.Errorf("load local target %s: %w", target, err)
	}

	// Remove the extracted directory (e.g. .../<version>/osquery.app), if any.
	if localTarget.DirPath != "" {
		if err := os.RemoveAll(localTarget.DirPath); err != nil {
			return fmt.Errorf("remove extracted dir %q: %w", localTarget.DirPath, err)
		}
	}

	// Remove the downloaded archive and its cached hash so the next Get
	// re-downloads from TUF instead of re-extracting a possibly-corrupt archive.
	if err := os.RemoveAll(localTarget.Path); err != nil {
		return fmt.Errorf("remove archive %q: %w", localTarget.Path, err)
	}
	removeCachedHashes(localTarget.Path)

	log.Info().
		Str("target", target).
		Str("path", localTarget.Path).
		Str("dir", localTarget.DirPath).
		Msg("removed corrupt target for re-download")

	return nil
}
