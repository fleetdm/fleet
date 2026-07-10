package service

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// deleteDuplicateOktaSCEPScript is the macOS shell script that removes
// orphaned duplicate SCEP certificates left in the per-user keychain after
// the Okta conditional access profile is reinstalled or renewed. The
// canonical, customer-facing copy lives at
// docs/solutions/macos/scripts/delete-duplicate-scep-certificates.sh; the
// embed_sync test asserts the two stay byte-identical.
//
//go:embed embedded_scripts/delete-duplicate-scep-certificates.sh
var deleteDuplicateOktaSCEPScript string

// macOS short names are 1-31 chars of ASCII letters, digits, and a small
// set of punctuation. We're conservative here because the value is
// interpolated into a shell command line; single-quote escaping covers the
// rest. Reject anything weird so a malformed nano_users row cannot reach
// the shell.
var validMacOSShortNameRE = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)

// buildOktaCACleanupScript wraps the embedded cleanup script with the
// positional arguments it expects (auto-confirm, target user, certificate
// CN). Returns ok=false when the username fails validation, in which case
// the caller should log and skip rather than dispatch a malformed script.
func buildOktaCACleanupScript(username string) (string, bool) {
	if len(username) == 0 || len(username) > 31 || !validMacOSShortNameRE.MatchString(username) {
		return "", false
	}
	return fmt.Sprintf("#!/bin/bash\nset -- -y -u %s %s\n%s",
		shellSingleQuote(username),
		shellSingleQuote(fleet.ConditionalAccessOktaCertificateCN),
		deleteDuplicateOktaSCEPScript,
	), true
}

// shellSingleQuote returns s wrapped in single quotes, escaping any
// embedded single quotes via the POSIX 'foo'"'"'bar' idiom.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// maybeRunOktaCACleanupScript schedules the Okta conditional access
// keychain-cleanup script on the host after a successful InstallProfile
// ack for the Okta CA profile. It silently no-ops when the command does
// not apply to the Okta CA profile, when no per-user MDM enrollment short
// name is on record, or when the short name fails validation. Errors are
// returned to the caller for logging; the caller must not fail the ack
// path on them.
func (svc *MDMAppleCheckinAndCommandService) maybeRunOktaCACleanupScript(ctx context.Context, hostUUID, commandUUID string) error {
	target, ok, err := svc.ds.OktaCACleanupTargetForInstallCommand(ctx, hostUUID, commandUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "look up Okta CA cleanup target")
	}
	if !ok {
		return nil
	}

	script, ok := buildOktaCACleanupScript(target.UserShortName)
	if !ok {
		svc.logger.DebugContext(ctx, "skip Okta CA keychain cleanup: invalid macOS username",
			"host_uuid", hostUUID, "command_uuid", commandUUID)
		return nil
	}

	if _, err := svc.ds.NewInternalHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         target.HostID,
		ScriptContents: script,
		SyncRequest:    false,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueue Okta CA keychain cleanup script")
	}
	return nil
}
