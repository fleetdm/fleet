package nanomdm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	mock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/micromdm/nanolib/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandAndReportResultsPrimaryDBUse(t *testing.T) {
	ds := new(mock.MDMAppleStore)

	enrollID := &mdm.EnrollID{
		ID:   "1",
		Type: mdm.Device,
	}
	s := Service{
		logger: log.NopLogger,
		store:  ds,
		normalizer: func(e *mdm.Enrollment) *mdm.EnrollID {
			return enrollID
		},
	}
	ds.StoreCommandReportFunc = func(r *mdm.Request, report *mdm.CommandResults) error {
		return nil
	}
	var primaryRequired bool
	ds.RetrieveNextCommandFunc = func(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
		assert.Equal(t, primaryRequired, ctxdb.IsPrimaryRequired(r.Context))
		return nil, nil
	}

	mdmRequest := &mdm.Request{
		Context: context.Background(),
	}
	mdmCommandResults := &mdm.CommandResults{
		Status: "Idle",
	}
	// We don't use primary DB with "Idle" status because we don't update the status of existing commands
	cmd, err := s.CommandAndReportResults(mdmRequest, mdmCommandResults)
	require.NoError(t, err)
	require.Nil(t, cmd)

	// We use primary DB with non-"Idle" status
	mdmCommandResults = &mdm.CommandResults{
		Status: "Acknowledge",
	}
	primaryRequired = true
	cmd, err = s.CommandAndReportResults(mdmRequest, mdmCommandResults)
	require.NoError(t, err)
	require.Nil(t, cmd)

}

func TestCommandAndReportResultsRotateFileVaultKeyExpansion(t *testing.T) {
	const hostUUID = "host-uuid-1"
	placeholder := fleet.HostSecretPlaceholder(fleet.HostSecretFileVaultKey)
	raw := `<plist><dict><key>Password</key><string>` + placeholder + `</string></dict></plist>`

	newService := func(ds *mock.MDMAppleStore) *Service {
		ds.StoreCommandReportFunc = func(r *mdm.Request, report *mdm.CommandResults) error { return nil }
		ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) { return document, nil }
		ds.RetrieveNextCommandFunc = func(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
			cmd := &mdm.CommandWithSubtype{CommandUUID: "cmd-1", Raw: []byte(raw)}
			cmd.Command.Command.RequestType = fleet.RotateFileVaultKeyCmdName
			return cmd, nil
		}
		enrollID := &mdm.EnrollID{ID: hostUUID, Type: mdm.Device}
		return &Service{
			logger:     log.NopLogger,
			store:      ds,
			normalizer: func(e *mdm.Enrollment) *mdm.EnrollID { return enrollID },
		}
	}
	results := func() *mdm.CommandResults {
		return &mdm.CommandResults{UDID: hostUUID, Status: "Idle"}
	}

	t.Run("expands the current key", func(t *testing.T) {
		ds := new(mock.MDMAppleStore)
		ds.ExpandHostSecretsFunc = func(ctx context.Context, document string, enrollmentID string) (string, error) {
			require.Equal(t, hostUUID, enrollmentID)
			return strings.ReplaceAll(document, placeholder, "ABCD-EFGH"), nil
		}
		cmd, err := newService(ds).CommandAndReportResults(&mdm.Request{Context: t.Context()}, results())
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Contains(t, string(cmd.Raw), "ABCD-EFGH")
		require.NotContains(t, string(cmd.Raw), placeholder)
		require.False(t, ds.SetDiskEncryptionKeyRotationFailedFuncInvoked)
	})

	t.Run("expansion failure fails the rotation", func(t *testing.T) {
		ds := new(mock.MDMAppleStore)
		newSvc := newService(ds)
		ds.ExpandHostSecretsFunc = func(ctx context.Context, document string, enrollmentID string) (string, error) {
			return "", errors.New("decrypting disk encryption key")
		}
		var stored []*mdm.CommandResults
		ds.StoreCommandReportFunc = func(r *mdm.Request, report *mdm.CommandResults) error {
			stored = append(stored, report)
			return nil
		}
		ds.SetDiskEncryptionKeyRotationFailedFunc = func(ctx context.Context, gotHostUUID, commandUUID, errorMsg string) error {
			require.Equal(t, hostUUID, gotHostUUID)
			require.Equal(t, "cmd-1", commandUUID)
			require.Contains(t, errorMsg, "decrypting disk encryption key")
			return nil
		}
		cmd, err := newSvc.CommandAndReportResults(&mdm.Request{Context: t.Context()}, results())
		require.NoError(t, err)
		require.Nil(t, cmd)
		require.True(t, ds.SetDiskEncryptionKeyRotationFailedFuncInvoked)
		require.Len(t, stored, 2)
		require.Equal(t, "cmd-1", stored[1].CommandUUID)
		require.Equal(t, "Error", stored[1].Status)
		// nano_command_results.result is NOT NULL; an empty body fails the insert
		// and leaves the command at the head of the queue.
		decoded, err := mdm.DecodeCommandResults(stored[1].Raw)
		require.NoError(t, err)
		require.Equal(t, "cmd-1", decoded.CommandUUID)
		require.Equal(t, "Error", decoded.Status)
		require.Len(t, decoded.ErrorChain, 1)
		require.Contains(t, decoded.ErrorChain[0].LocalizedDescription, "decrypting disk encryption key")
	})
}

func TestCommandAndReportResultsHostSecretExpansionFailure(t *testing.T) {
	const hostUUID = "host-uuid-1"
	ds := new(mock.MDMAppleStore)
	var stored []*mdm.CommandResults
	ds.StoreCommandReportFunc = func(r *mdm.Request, report *mdm.CommandResults) error {
		stored = append(stored, report)
		return nil
	}
	ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) { return document, nil }
	ds.RetrieveNextCommandFunc = func(r *mdm.Request, skipNotNow bool) (*mdm.CommandWithSubtype, error) {
		cmd := &mdm.CommandWithSubtype{CommandUUID: "cmd-1", Raw: []byte("<plist/>")}
		cmd.Command.Command.RequestType = fleet.SetRecoveryLockCmdName
		return cmd, nil
	}
	ds.ExpandHostSecretsFunc = func(ctx context.Context, document string, enrollmentID string) (string, error) {
		return "", errors.New("pending recovery lock password not found")
	}
	ds.SetRecoveryLockFailedFunc = func(ctx context.Context, gotHostUUID, commandUUID, errorMsg string) error { return nil }
	enrollID := &mdm.EnrollID{ID: hostUUID, Type: mdm.Device}
	s := &Service{
		logger:     log.NopLogger,
		store:      ds,
		normalizer: func(e *mdm.Enrollment) *mdm.EnrollID { return enrollID },
	}

	cmd, err := s.CommandAndReportResults(&mdm.Request{Context: t.Context()}, &mdm.CommandResults{
		UDID:   hostUUID,
		Status: "Idle",
	})
	require.NoError(t, err)
	require.Nil(t, cmd)
	require.True(t, ds.SetRecoveryLockFailedFuncInvoked)

	// The first report is the device's Idle; the second is the failure Fleet records.
	require.Len(t, stored, 2)
	// nano_command_results.result is NOT NULL, so an empty body fails the insert
	// and the command is served again on every check-in.
	decoded, err := mdm.DecodeCommandResults(stored[1].Raw)
	require.NoError(t, err)
	require.Equal(t, "cmd-1", decoded.CommandUUID)
	require.Equal(t, "Error", decoded.Status)
	require.Len(t, decoded.ErrorChain, 1)
	require.Contains(t, decoded.ErrorChain[0].LocalizedDescription, "pending recovery lock password not found")
}
