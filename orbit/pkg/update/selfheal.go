package update

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/rs/zerolog/log"
)

// CheckExec verifies that the target's installed executable can run, using the
// same check applied to freshly downloaded targets (the target's CustomCheckExec
// if set, otherwise running it with --help).
//
// A non-nil error means the on-disk executable failed to run (corrupt/truncated
// download, crash on startup, etc.) and the caller should self-heal by
// re-downloading it.
//
// Unlike the download-path checkExec, this needs no platform/arch guards: it
// only runs in orbit, which loads targets matching the host OS/arch.
func (u *Updater) CheckExec(target string) error {
	localTarget, err := u.localTarget(target)
	if err != nil {
		return fmt.Errorf("load local target %s: %w", target, err)
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
// This is used to self-heal from a component binary that fails its exec check
// (a corrupt/truncated download that won't fork/exec or crashes on startup).
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
