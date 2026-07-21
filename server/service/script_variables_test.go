package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestMaybeExpandScriptFleetVariables(t *testing.T) {
	newSvcAndCtx := func(tier string) (*Service, context.Context, *mock.Store) {
		ds := new(mock.Store)
		svc := &Service{ds: ds}
		ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: tier})
		return svc, ctx, ds
	}

	host := &fleet.Host{
		ID:             42,
		UUID:           "ABC-123",
		HardwareSerial: "SERIAL-1",
		Platform:       "darwin",
	}

	scimUser := &fleet.ScimUser{
		UserName:   "user@example.com",
		GivenName:  new("Ada"),
		FamilyName: new("Lovelace"),
		Department: new("Engineering"),
		Groups:     []fleet.ScimUserGroup{{DisplayName: "g1"}, {DisplayName: "g2"}},
	}
	mockScimUser := func(ds *mock.Store, user *fleet.ScimUser) {
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			if user == nil {
				return nil, newNotFoundError()
			}
			return user, nil
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
	}

	t.Run("no variables is byte-for-byte unchanged", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		for _, contents := range []string{
			"#!/bin/sh\necho hello\n",
			"echo $FLEET_SECRET_FOO and $FLEET_HOST_VITAL_computer_name",
			"echo $OTHER_VAR",
			"",
		} {
			expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, contents)
			require.NoError(t, err)
			require.Empty(t, failMsg)
			require.Equal(t, contents, expanded)
		}
	})

	t.Run("host variables expand", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host,
			"echo $FLEET_VAR_HOST_UUID $FLEET_VAR_HOST_HARDWARE_SERIAL ${FLEET_VAR_HOST_PLATFORM}")
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, "echo ABC-123 SERIAL-1 macos", expanded)
	})

	t.Run("platform passes through for linux and windows", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		for platform, want := range map[string]string{"ubuntu": "ubuntu", "rhel": "rhel", "windows": "windows"} {
			h := *host
			h.Platform = platform
			expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h, "echo $FLEET_VAR_HOST_PLATFORM")
			require.NoError(t, err)
			require.Empty(t, failMsg)
			require.Equal(t, "echo "+want, expanded)
		}
	})

	t.Run("IdP variables expand", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(fleet.TierPremium)
		mockScimUser(ds, scimUser)
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host,
			"user: $FLEET_VAR_HOST_END_USER_IDP_USERNAME\n"+
				"email: user_${FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART}@corp.example.com\n"+
				"name: $FLEET_VAR_HOST_END_USER_IDP_FULL_NAME\n"+
				"groups: $FLEET_VAR_HOST_END_USER_IDP_GROUPS\n"+
				"dept: $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT\n")
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, "user: user@example.com\n"+
			"email: user_user@corp.example.com\n"+
			"name: Ada Lovelace\n"+
			"groups: g1,g2\n"+
			"dept: Engineering\n", expanded)
	})

	t.Run("missing IdP user is a resolution failure", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(fleet.TierPremium)
		mockScimUser(ds, nil)
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host,
			"echo $FLEET_VAR_HOST_END_USER_IDP_USERNAME")
		require.NoError(t, err)
		require.Empty(t, expanded)
		require.Contains(t, failMsg, "There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_USERNAME.")
	})

	t.Run("multiple failures accumulate", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(fleet.TierPremium)
		mockScimUser(ds, nil)
		h := *host
		h.HardwareSerial = ""
		_, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h,
			"echo $FLEET_VAR_HOST_HARDWARE_SERIAL $FLEET_VAR_HOST_END_USER_IDP_USERNAME")
		require.NoError(t, err)
		require.Contains(t, failMsg, "There is no hardware serial for this host.")
		require.Contains(t, failMsg, "There is no IdP username for this host.")
		require.Len(t, splitLines(failMsg), 2)
	})

	t.Run("unsupported variable names are left untouched", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		contents := "echo $FLEET_VAR_SOMETHING_ELSE and $FLEET_VAR_HOST_UUID"
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, contents)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, "echo $FLEET_VAR_SOMETHING_ELSE and ABC-123", expanded)
	})

	t.Run("variables on free license fail instead of expanding", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierFree)
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, "echo $FLEET_VAR_HOST_UUID")
		require.NoError(t, err)
		require.Empty(t, expanded)
		require.Contains(t, failMsg, "Fleet Premium license")

		// variable-free content is unaffected on free
		expanded, failMsg, err = svc.maybeExpandScriptFleetVariables(ctx, host, "echo hello")
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, "echo hello", expanded)
	})
}

func splitLines(s string) []string {
	var lines []string
	for line := range strings.SplitSeq(s, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestGetHostScriptFleetVariables(t *testing.T) {
	newSvcAndCtx := func(t *testing.T, host *fleet.Host, storedContents string, storedExitCode *int64) (fleet.Service, context.Context, *mock.Store) {
		ds := new(mock.Store)
		lic := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: lic, SkipCreateTestUsers: true})
		ctx = test.HostContext(ctx, host)

		ds.GetHostScriptExecutionResultFunc = func(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
			return &fleet.HostScriptResult{
				HostID:         host.ID,
				ExecutionID:    execID,
				ScriptContents: storedContents,
				ExitCode:       storedExitCode,
			}, nil
		}
		ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
			return document, nil
		}
		ds.ExpandCustomHostVitalsFunc = func(ctx context.Context, hostID uint, document string) (string, error) {
			return document, nil
		}
		return svc, ctx, ds
	}

	host := &fleet.Host{
		ID:             42,
		UUID:           "ABC-123",
		HardwareSerial: "SERIAL-1",
		Platform:       "ubuntu",
	}

	t.Run("variables expand for the fetching host", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(t, host, "echo $FLEET_VAR_HOST_UUID on $FLEET_VAR_HOST_PLATFORM", nil)

		// pin the ordering: secrets expansion runs before fleet variables, so
		// its input must still contain the unexpanded variable references
		ds.ExpandEmbeddedSecretsFunc = func(ctx context.Context, document string) (string, error) {
			require.Contains(t, document, "$FLEET_VAR_HOST_UUID")
			return document, nil
		}

		script, err := svc.GetHostScript(ctx, "exec-1")
		require.NoError(t, err)
		require.Equal(t, "echo ABC-123 on ubuntu", script.ScriptContents)
		require.Nil(t, script.ExitCode)
		require.True(t, ds.ExpandEmbeddedSecretsFuncInvoked)
	})

	t.Run("unresolvable variable records failed result and returns marked script", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(t, host, "echo $FLEET_VAR_HOST_END_USER_IDP_USERNAME", nil)
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return nil, newNotFoundError()
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		var savedResult *fleet.HostScriptResultPayload
		ds.SetHostScriptExecutionResultFunc = func(ctx context.Context, result *fleet.HostScriptResultPayload, attemptNumber *int) (*fleet.HostScriptResult, string, error) {
			savedResult = result
			exitCode := int64(result.ExitCode)
			return &fleet.HostScriptResult{
				HostID:      result.HostID,
				ExecutionID: result.ExecutionID,
				Output:      result.Output,
				ExitCode:    &exitCode,
			}, "", nil
		}
		ds.MaybeUpdateSetupExperienceScriptStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			return false, nil
		}

		script, err := svc.GetHostScript(ctx, "exec-1")
		require.NoError(t, err)

		// the failure was recorded through the normal result-saving path
		require.NotNil(t, savedResult)
		require.Equal(t, fleet.ScriptFleetVarResolutionFailedExitCode, savedResult.ExitCode)
		require.Contains(t, savedResult.Output, "There is no IdP username for this host.")
		require.Equal(t, host.ID, savedResult.HostID)

		// the returned script carries the exit code so fleetd skips it and
		// keeps processing its queue
		require.NotNil(t, script.ExitCode)
		require.EqualValues(t, fleet.ScriptFleetVarResolutionFailedExitCode, *script.ExitCode)
	})

	t.Run("already-completed execution is not re-recorded", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(t, host, "echo $FLEET_VAR_HOST_END_USER_IDP_USERNAME",
			new(int64(fleet.ScriptFleetVarResolutionFailedExitCode)))

		script, err := svc.GetHostScript(ctx, "exec-1")
		require.NoError(t, err)
		require.EqualValues(t, fleet.ScriptFleetVarResolutionFailedExitCode, *script.ExitCode)
		require.False(t, ds.SetHostScriptExecutionResultFuncInvoked)
	})

	t.Run("internal scripts without variables are unchanged", func(t *testing.T) {
		const lockScript = "#!/bin/sh\npmset displaysleepnow && shutdown -h now\n"
		svc, ctx, _ := newSvcAndCtx(t, host, lockScript, nil)
		script, err := svc.GetHostScript(ctx, "exec-1")
		require.NoError(t, err)
		require.Equal(t, lockScript, script.ScriptContents)
	})
}

func TestGetSoftwareInstallDetailsFleetVariables(t *testing.T) {
	host := &fleet.Host{
		ID:             42,
		UUID:           "ABC-123",
		HardwareSerial: "SERIAL-1",
		Platform:       "ubuntu",
		OsqueryHostID:  new("osquery-42"),
	}

	newSvcAndCtx := func(t *testing.T, details *fleet.SoftwareInstallDetails) (fleet.Service, context.Context, *mock.Store) {
		ds := new(mock.Store)
		lic := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: lic, SkipCreateTestUsers: true})
		ctx = test.HostContext(ctx, host)
		ds.GetSoftwareInstallDetailsFunc = func(ctx context.Context, executionID string) (*fleet.SoftwareInstallDetails, error) {
			return details, nil
		}
		return svc, ctx, ds
	}

	t.Run("variables expand in all three scripts", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(t, &fleet.SoftwareInstallDetails{
			HostID:            host.ID,
			ExecutionID:       "install-1",
			InstallScript:     "install $FLEET_VAR_HOST_HARDWARE_SERIAL",
			PostInstallScript: "post ${FLEET_VAR_HOST_UUID}",
			UninstallScript:   "uninstall $FLEET_VAR_HOST_PLATFORM",
		})

		details, err := svc.GetSoftwareInstallDetails(ctx, "install-1")
		require.NoError(t, err)
		require.Equal(t, "install SERIAL-1", details.InstallScript)
		require.Equal(t, "post ABC-123", details.PostInstallScript)
		require.Equal(t, "uninstall ubuntu", details.UninstallScript)
	})

	t.Run("scripts without variables are unchanged", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(t, &fleet.SoftwareInstallDetails{
			HostID:        host.ID,
			ExecutionID:   "install-1",
			InstallScript: "install --flag",
		})

		details, err := svc.GetSoftwareInstallDetails(ctx, "install-1")
		require.NoError(t, err)
		require.Equal(t, "install --flag", details.InstallScript)
		require.Empty(t, details.PostInstallScript)
	})

	t.Run("unresolvable variable records failed install and returns not found", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(t, &fleet.SoftwareInstallDetails{
			HostID:          host.ID,
			ExecutionID:     "install-1",
			InstallScript:   "install $FLEET_VAR_HOST_END_USER_IDP_USERNAME",
			UninstallScript: "uninstall $FLEET_VAR_HOST_END_USER_IDP_USERNAME",
		})
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return nil, newNotFoundError()
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		hsi := &fleet.HostSoftwareInstallerResult{
			InstallUUID: "install-1",
			HostID:      host.ID,
			Status:      fleet.SoftwareInstallPending,
		}
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return hsi, nil
		}
		var savedResult *fleet.HostSoftwareInstallResultPayload
		ds.SetHostSoftwareInstallResultFunc = func(ctx context.Context, result *fleet.HostSoftwareInstallResultPayload, attemptNumber *int) (bool, error) {
			savedResult = result
			return false, nil
		}
		ds.MaybeUpdateSetupExperienceSoftwareInstallStatusFunc = func(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
			return false, nil
		}

		_, err := svc.GetSoftwareInstallDetails(ctx, "install-1")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err), "expected not-found, got: %v", err)

		// the failure was recorded through the normal result-saving path, with
		// the identical failure reported once even though two scripts hit it
		require.NotNil(t, savedResult)
		require.NotNil(t, savedResult.InstallScriptExitCode)
		require.Equal(t, fleet.ScriptFleetVarResolutionFailedExitCode, *savedResult.InstallScriptExitCode)
		require.NotNil(t, savedResult.InstallScriptOutput)
		require.Equal(t, "There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_HOST_END_USER_IDP_USERNAME.", *savedResult.InstallScriptOutput)
	})

	t.Run("already-recorded install failure is not re-recorded", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(t, &fleet.SoftwareInstallDetails{
			HostID:        host.ID,
			ExecutionID:   "install-1",
			InstallScript: "install $FLEET_VAR_HOST_END_USER_IDP_USERNAME",
		})
		ds.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
			return nil, newNotFoundError()
		}
		ds.ListHostDeviceMappingFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostDeviceMapping, error) {
			return nil, nil
		}
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return &fleet.HostSoftwareInstallerResult{
				InstallUUID: "install-1",
				HostID:      host.ID,
				Status:      fleet.SoftwareInstallFailed,
			}, nil
		}

		_, err := svc.GetSoftwareInstallDetails(ctx, "install-1")
		require.Error(t, err)
		require.True(t, fleet.IsNotFound(err), "expected not-found, got: %v", err)
		require.False(t, ds.SetHostSoftwareInstallResultFuncInvoked)
	})
}
