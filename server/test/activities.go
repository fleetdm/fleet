package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// CreateHostScriptUpcomingActivity creates a host script execution request
// for the provided host. It returns the upcoming activity's execution ID.
func CreateHostScriptUpcomingActivity(t *testing.T, ds fleet.Datastore, host *fleet.Host) string {
	ctx := context.Background()
	hsr, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{
		HostID:         host.ID,
		ScriptContents: "echo 'a'",
	})
	require.NoError(t, err)
	return hsr.ExecutionID
}

// SetHostScriptResult sets the result of a host script queued via
// CreateHostScriptUpcomingActivity.
func SetHostScriptResult(t *testing.T, ds fleet.Datastore, host *fleet.Host, execID string, exitCode int) {
	ctx := context.Background()
	_, _, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID: host.ID, ExecutionID: execID, Output: "a", ExitCode: exitCode,
	})
	require.NoError(t, err)
}

// CreateHostSoftwareInstallUpcomingActivity creates a host software install
// execution request for the provided host. It returns the upcoming activity's
// execution ID.
func CreateHostSoftwareInstallUpcomingActivity(t *testing.T, ds fleet.Datastore, host *fleet.Host, user *fleet.User) string {
	ctx := context.Background()
	installer, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install foo",
		InstallerFile:   installer,
		StorageID:       uuid.NewString(),
		Filename:        "foo.pkg",
		Title:           uuid.NewString(),
		Source:          "apps",
		Version:         "0.0.1",
		UserID:          user.ID,
		UninstallScript: "uninstall foo",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	execID, err := ds.InsertSoftwareInstallRequest(ctx, host.ID, installerID, fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	return execID
}

// SetHostSoftwareInstallResult sets the result of a host software install
// queued via CreateHostSoftwareInstallUpcomingActivity.
func SetHostSoftwareInstallResult(t *testing.T, ds fleet.Datastore, host *fleet.Host, execID string, exitCode int) {
	ctx := context.Background()
	_, err := ds.SetHostSoftwareInstallResult(ctx, &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           execID,
		InstallScriptExitCode: &exitCode,
	})
	require.NoError(t, err)
}

// CreateHostSoftwareUninstallUpcomingActivity creates a host software uninstall
// execution request for the provided host. It returns the upcoming activity's
// execution ID.
func CreateHostSoftwareUninstallUpcomingActivity(t *testing.T, ds fleet.Datastore, host *fleet.Host, user *fleet.User) string {
	ctx := context.Background()
	installer, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "install foo",
		InstallerFile:   installer,
		StorageID:       uuid.NewString(),
		Filename:        "foo.pkg",
		Title:           uuid.NewString(),
		Source:          "apps",
		Version:         "0.0.1",
		UserID:          user.ID,
		UninstallScript: "uninstall foo",
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	})
	require.NoError(t, err)

	execID := uuid.NewString()
	err = ds.InsertSoftwareUninstallRequest(ctx, execID, host.ID, installerID, false)
	require.NoError(t, err)
	return execID
}

// SetHostSoftwareUninstallResult sets the result of a host software uninstall
// queued via CreateHostSoftwareUninstallUpcomingActivity.
func SetHostSoftwareUninstallResult(t *testing.T, ds fleet.Datastore, host *fleet.Host, execID string, exitCode int) {
	ctx := context.Background()
	_, _, err := ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host.ID,
		ExecutionID: execID,
		ExitCode:    exitCode,
	})
	require.NoError(t, err)
}

// CreateHostVPPAppInstallUpcomingActivity creates a VPP app install request
// for the provided host. It returns the upcoming activity's execution ID.
// Note that test.CreateInsertGlobalVPPToken(t, ds) should be used to enable
// VPP apps (create a VPP token).
func CreateHostVPPAppInstallUpcomingActivity(t *testing.T, ds fleet.Datastore, host *fleet.Host) (execID, adamID string) {
	ctx := context.Background()
	adamID = uuid.NewString()
	vppApp := &fleet.VPPApp{
		Name: "vpp_1", VPPAppTeam: fleet.VPPAppTeam{VPPAppID: fleet.VPPAppID{AdamID: adamID, Platform: fleet.MacOSPlatform}},
		BundleIdentifier: adamID,
	}
	_, err := ds.InsertVPPAppWithTeam(ctx, vppApp, nil)
	require.NoError(t, err)
	execID = uuid.NewString()
	err = ds.InsertHostVPPSoftwareInstall(ctx, host.ID, vppApp.VPPAppID, execID, "event-id-1", fleet.HostSoftwareInstallOptions{})
	require.NoError(t, err)
	return execID, adamID
}

// SetHostVPPAppInstallResult sets the result of a VPP app install queued via
// CreateHostVPPAppInstallUpcomingActivity.
// The adamID is the one for the VPP app created by that call, and status is
// one of the Apple MDM status string (Acknowledged, Error, CommandFormatError,
// etc).
func SetHostVPPAppInstallResult(t *testing.T, ds fleet.Datastore, nanods storage.CommandAndReportResultsStore, host *fleet.Host, execID, adamID, status string) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, fleet.ActivityWebhookContextKey, true)
	nanoCtx := &mdm.Request{EnrollID: &mdm.EnrollID{ID: host.UUID}, Context: ctx}

	cmdRes := &mdm.CommandResults{
		CommandUUID: execID,
		Status:      status,
		Raw:         []byte(`<?xml version="1.0" encoding="UTF-8"?>`),
	}
	err := nanods.StoreCommandReport(nanoCtx, cmdRes)
	require.NoError(t, err)
	err = ds.NewActivity(ctx, nil, fleet.ActivityInstalledAppStoreApp{
		HostID:      host.ID,
		AppStoreID:  adamID,
		CommandUUID: execID,
	}, []byte(`{}`), time.Now())
	require.NoError(t, err)
}
