package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// ClientCertificateInstall/SCEP/<id>/Status values. Anything else (32 while the enrollment is still running, a 404
// before the node exists) means "ask again".
const (
	windowsSCEPStatusSucceeded = "1"
	windowsSCEPStatusFailed    = "16"
)

// windowsSCEPStatusProbeMaxAttempts bounds how many status probes Fleet sends for one delivery. Each probe waits
// fleet.WindowsSCEPStatusProbeDelay and fleetd wakes the device at most once a minute, so this covers roughly 15
// minutes: long enough for the SCEP CSP's own retries against a CA that answers PENDING (3 retries 5 minutes apart by
// default). After that, the osquery certificate scan decides.
const windowsSCEPStatusProbeMaxAttempts = 15

// windowsSCEPVerifyingInstall is a proxied SCEP install the host just ACKed, which now waits in "verifying".
type windowsSCEPVerifyingInstall struct {
	ProfileUUID string
	CommandUUID string
}

// A probe's command UUID is the prefix, the install command UUID and the attempt number, so the probes for a delivery
// need no extra state and a late answer can't be applied to a later delivery.
func windowsSCEPStatusProbeCmdUUID(installCmdUUID string, attempt int) string {
	return fmt.Sprintf("%s%s-%d", fleet.WindowsSCEPStatusProbeCmdUUIDPrefix, installCmdUUID, attempt)
}

func parseWindowsSCEPStatusProbeCmdUUID(probeCmdUUID string) (installCmdUUID string, attempt int, ok bool) {
	rest, ok := strings.CutPrefix(probeCmdUUID, fleet.WindowsSCEPStatusProbeCmdUUIDPrefix)
	if !ok {
		return "", 0, false
	}
	i := strings.LastIndex(rest, "-")
	if i <= 0 {
		return "", 0, false
	}
	attempt, err := strconv.Atoi(rest[i+1:])
	if err != nil {
		return "", 0, false
	}
	return rest[:i], attempt, true
}

func windowsSCEPStatusProbeCommand(installCmdUUID, profileUUID string, userScoped bool, attempt int) *fleet.MDMWindowsCommand {
	scope := "./Device"
	if userScoped {
		scope = "./User"
	}
	// The SCEP node ID is the profile UUID, substituted for $FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID at delivery.
	node := fmt.Sprintf("%s/Vendor/MSFT/ClientCertificateInstall/SCEP/%s", scope, profileUUID)
	cmdUUID := windowsSCEPStatusProbeCmdUUID(installCmdUUID, attempt)
	raw := fmt.Sprintf(`<Get><CmdID>%s</CmdID>`+
		`<Item><Target><LocURI>%s/Status</LocURI></Target></Item>`+
		`<Item><Target><LocURI>%s/ErrorCode</LocURI></Target></Item>`+
		`</Get>`, cmdUUID, node, node)
	return &fleet.MDMWindowsCommand{
		CommandUUID:  cmdUUID,
		RawCommand:   []byte(raw),
		TargetLocURI: node + "/Status",
	}
}

// nextWindowsSCEPStatusProbeCommand builds the probe that follows the given one, or returns nil once the attempts run out.
func nextWindowsSCEPStatusProbeCommand(probe fleet.MDMWindowsCommand) *fleet.MDMWindowsCommand {
	installCmdUUID, attempt, ok := parseWindowsSCEPStatusProbeCmdUUID(probe.CommandUUID)
	if !ok || attempt >= windowsSCEPStatusProbeMaxAttempts {
		return nil
	}
	node, ok := strings.CutSuffix(probe.TargetLocURI, "/Status")
	if !ok {
		return nil
	}
	profileUUID := node[strings.LastIndex(node, "/")+1:]
	return windowsSCEPStatusProbeCommand(installCmdUUID, profileUUID, strings.HasPrefix(node, "./User/"), attempt+1)
}

// isUserScopedWindowsSCEPInstall reports whether an install command enrolls into the user's certificate store. Profile
// validation keeps a profile to a single scope.
func isUserScopedWindowsSCEPInstall(rawCommand []byte) bool {
	return bytes.Contains(rawCommand, []byte("/User/Vendor/MSFT/ClientCertificateInstall/SCEP"))
}

// enqueueWindowsSCEPStatusProbeDB queues a probe for the enrollment. The service holds it back for
// fleet.WindowsSCEPStatusProbeDelay, and the queued probe keeps the enrollment's has_pending_commands set so fleetd wakes
// the device to receive it. A probe that already exists, from a duplicate answer, is left alone.
func (ds *Datastore) enqueueWindowsSCEPStatusProbeDB(ctx context.Context, tx sqlx.ExtContext, enrollmentID uint, cmd *fleet.MDMWindowsCommand) error {
	var exists bool
	if err := sqlx.GetContext(ctx, tx, &exists,
		`SELECT EXISTS (SELECT 1 FROM windows_mdm_commands WHERE command_uuid = ?)`, cmd.CommandUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "check for existing windows scep status probe")
	}
	if exists {
		return nil
	}
	if err := ds.mdmWindowsInsertCommandForEnrollmentIDsDB(ctx, tx, []uint{enrollmentID}, cmd); err != nil {
		return ctxerr.Wrap(ctx, err, "enqueue windows scep status probe")
	}
	return nil
}

// applyWindowsSCEPStatusProbeResultDB applies the device's answer to a SCEP status probe to the delivery it was sent for,
// and only while that delivery is still verifying. Status 1 means the certificate is in the store. Status 16 means the
// enrollment failed, or that the certificate was later removed, and is charged like any other delivery failure. It
// reports whether a profile row changed and whether the delivery is still waiting for its certificate. results is nil
// when the device answered the Get without values.
func applyWindowsSCEPStatusProbeResultDB(ctx context.Context, tx sqlx.ExtContext, hostUUID, installCmdUUID string, results *fleet.SyncMLCmd,
) (changed bool, waiting bool, err error) {
	var status, errorCode string
	if results != nil {
		for _, item := range results.Items {
			if item.Source == nil || item.Data == nil {
				continue
			}
			switch {
			case strings.HasSuffix(*item.Source, "/Status"):
				status = strings.TrimSpace(item.Data.Content)
			case strings.HasSuffix(*item.Source, "/ErrorCode"):
				errorCode = strings.TrimSpace(item.Data.Content)
			}
		}
	}

	switch status {
	case windowsSCEPStatusSucceeded:
		res, err := tx.ExecContext(ctx, `
			UPDATE host_mdm_windows_profiles
			SET status = ?, detail = ''
			WHERE host_uuid = ? AND command_uuid = ? AND operation_type = ? AND status = ?`,
			fleet.MDMDeliveryVerified, hostUUID, installCmdUUID, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying)
		if err != nil {
			return false, false, ctxerr.Wrap(ctx, err, "verify windows scep profile from status probe")
		}
		rows, _ := res.RowsAffected()
		return rows > 0, false, nil

	case windowsSCEPStatusFailed:
		var profileUUID string
		if err := sqlx.GetContext(ctx, tx, &profileUUID,
			`SELECT profile_uuid FROM host_mdm_windows_profiles WHERE host_uuid = ? AND command_uuid = ? AND operation_type = ?`,
			hostUUID, installCmdUUID, fleet.MDMOperationTypeInstall); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, false, nil
			}
			return false, false, ctxerr.Wrap(ctx, err, "load windows scep profile for status probe")
		}
		changed, _, err := failOrRetryWindowsHostProfileDB(ctx, tx, hostUUID, profileUUID, installCmdUUID,
			windowsSCEPStatusFailedDetail(errorCode))
		return changed, false, err
	}

	if err := sqlx.GetContext(ctx, tx, &waiting, `
		SELECT EXISTS (
			SELECT 1 FROM host_mdm_windows_profiles
			WHERE host_uuid = ? AND command_uuid = ? AND operation_type = ? AND status = ?
		)`, hostUUID, installCmdUUID, fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerifying); err != nil {
		return false, false, ctxerr.Wrap(ctx, err, "check windows scep profile still verifying")
	}
	return false, waiting, nil
}

func windowsSCEPStatusFailedDetail(errorCode string) string {
	// ErrorCode is an HRESULT that the device reports as a signed 32-bit integer.
	if code, err := strconv.ParseInt(errorCode, 10, 64); err == nil && code != 0 {
		return fmt.Sprintf("The host reported that the SCEP certificate isn't installed (error code 0x%08X).", uint32(code)) //nolint:gosec // HRESULT bit pattern
	}
	return "The host reported that the SCEP certificate isn't installed."
}
