package service

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/variables"
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

	t.Run("host variables are defined, not substituted", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		const body = "echo $FLEET_VAR_HOST_UUID $FLEET_VAR_HOST_HARDWARE_SERIAL ${FLEET_VAR_HOST_PLATFORM}"
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, body)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		requireVarsDelivered(t, expanded, body, map[string]string{
			"HOST_UUID": "ABC-123", "HOST_HARDWARE_SERIAL": "SERIAL-1", "HOST_PLATFORM": "macos",
		})
	})

	t.Run("platform passes through for linux and windows", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		for platform, want := range map[string]string{"ubuntu": "ubuntu", "rhel": "rhel"} {
			h := *host
			h.Platform = platform
			expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h, "echo $FLEET_VAR_HOST_PLATFORM")
			require.NoError(t, err)
			require.Empty(t, failMsg)
			requireVarsDelivered(t, expanded, "echo $FLEET_VAR_HOST_PLATFORM", map[string]string{"HOST_PLATFORM": want})
		}
	})

	t.Run("IdP variables are defined, not substituted", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(fleet.TierPremium)
		mockScimUser(ds, scimUser)
		const body = "user: $FLEET_VAR_HOST_END_USER_IDP_USERNAME\n" +
			"email: user_${FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART}@corp.example.com\n" +
			"name: $FLEET_VAR_HOST_END_USER_IDP_FULL_NAME\n" +
			"groups: $FLEET_VAR_HOST_END_USER_IDP_GROUPS\n" +
			"dept: $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT\n"
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, body)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		requireVarsDelivered(t, expanded, body, map[string]string{
			"HOST_END_USER_IDP_USERNAME":            "user@example.com",
			"HOST_END_USER_IDP_USERNAME_LOCAL_PART": "user",
			"HOST_END_USER_IDP_FULL_NAME":           "Ada Lovelace",
			"HOST_END_USER_IDP_GROUPS":              "g1,g2",
			"HOST_END_USER_IDP_DEPARTMENT":          "Engineering",
		})
	})

	// SCIM attributes and the host's own osquery-reported vitals both reach this
	// resolver, and neither is validated at ingestion.
	t.Run("interpreter metacharacters never reach the script body", func(t *testing.T) {
		payloads := map[string]string{
			"backtick":    "Engineering`touch /tmp/pwned`",
			"cmd-subst":   "Engineering$(touch /tmp/pwned)",
			"semicolon":   "Engineering; touch /tmp/pwned",
			"embedded-sq": "Engineering'; touch /tmp/pwned; echo '",
			"newline":     "Engineering\ntouch /tmp/pwned",
			"python":      "X\")\nimport os\nos.system(\"touch /tmp/pwned\")\nprint(\"",
		}
		for name, payload := range payloads {
			t.Run("department/"+name, func(t *testing.T) {
				svc, ctx, ds := newSvcAndCtx(fleet.TierPremium)
				u := *scimUser
				u.Department = &payload
				mockScimUser(ds, &u)
				const body = "echo \"dept: $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT\""
				expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, body)
				require.NoError(t, err)
				require.Empty(t, failMsg)
				requireVarsDelivered(t, expanded, body, map[string]string{"HOST_END_USER_IDP_DEPARTMENT": payload})
			})

			t.Run("hardware-serial/"+name, func(t *testing.T) {
				svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
				h := *host
				h.HardwareSerial = payload
				const body = "echo \"serial: $FLEET_VAR_HOST_HARDWARE_SERIAL\""
				expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h, body)
				require.NoError(t, err)
				require.Empty(t, failMsg)
				requireVarsDelivered(t, expanded, body, map[string]string{"HOST_HARDWARE_SERIAL": payload})
			})

			t.Run("uuid/"+name, func(t *testing.T) {
				svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
				h := *host
				h.UUID = payload
				const body = "echo \"uuid: $FLEET_VAR_HOST_UUID\""
				expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h, body)
				require.NoError(t, err)
				require.Empty(t, failMsg)
				requireVarsDelivered(t, expanded, body, map[string]string{"HOST_UUID": payload})
			})
		}
	})

	t.Run("windows hosts get PowerShell assignments and keep the tokens", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		h := *host
		h.Platform = "windows"
		const body = "Write-Output \"$FLEET_VAR_HOST_UUID ${FLEET_VAR_HOST_UUID}\""
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h, body)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, "$FLEET_VAR_HOST_UUID = "+variables.PowerShellCharArray("ABC-123")+"\r\n"+body, expanded)
		// the documented token syntax is unchanged on Windows
		require.Contains(t, expanded, "$FLEET_VAR_HOST_UUID ${FLEET_VAR_HOST_UUID}")
		require.NotContains(t, body, "ABC-123")
	})

	t.Run("PowerShell param block fails instead of breaking the script", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		h := *host
		h.Platform = "windows"
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h,
			"param($Foo = \"bar\")\r\nWrite-Output $FLEET_VAR_HOST_UUID\r\n")
		require.NoError(t, err)
		require.Empty(t, expanded)
		require.Equal(t, powerShellParamBlockMsg, failMsg)
	})

	t.Run("python scripts fail instead of running unexpanded", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		for _, shebang := range []string{
			"#!/usr/bin/env python3", "#!/usr/bin/python3", "#!/opt/homebrew/bin/python3.12",
		} {
			expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host,
				shebang+"\nprint(\"uuid: $FLEET_VAR_HOST_UUID\")\n")
			require.NoError(t, err)
			require.Empty(t, expanded)
			require.Equal(t, pythonFleetVarsMsg, failMsg)
		}
	})

	t.Run("python scripts without variables are unchanged", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		const contents = "#!/usr/bin/env python3\nprint(\"hello\")\n"
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, contents)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, contents, expanded)
	})

	t.Run("unknown platform fails instead of guessing an interpreter", func(t *testing.T) {
		svc, ctx, _ := newSvcAndCtx(fleet.TierPremium)
		h := *host
		h.Platform = ""
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, &h, "echo $FLEET_VAR_HOST_UUID")
		require.NoError(t, err)
		require.Empty(t, expanded)
		require.Equal(t, noPlatformMsg, failMsg)
	})

	t.Run("NUL in a value is a resolution failure", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(fleet.TierPremium)
		u := *scimUser
		u.Department = new("Eng\x00ineering")
		mockScimUser(ds, &u)
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host,
			"echo $FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT")
		require.NoError(t, err)
		require.Empty(t, expanded)
		require.Contains(t, failMsg, "contains an invalid character")
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
		const body = "echo $FLEET_VAR_SOMETHING_ELSE and $FLEET_VAR_HOST_UUID"
		expanded, failMsg, err := svc.maybeExpandScriptFleetVariables(ctx, host, body)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		requireVarsDelivered(t, expanded, body, map[string]string{"HOST_UUID": "ABC-123"})

		// only unsupported names means no preamble at all, on any interpreter
		const onlyUnsupported = "#!/usr/bin/env python3\nprint(\"$FLEET_VAR_SOMETHING_ELSE\")\n"
		expanded, failMsg, err = svc.maybeExpandScriptFleetVariables(ctx, host, onlyUnsupported)
		require.NoError(t, err)
		require.Empty(t, failMsg)
		require.Equal(t, onlyUnsupported, expanded)
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

// requireVarsDelivered asserts each variable is defined in the preamble, that
// the body still carries its tokens, and that no value leaked into the body.
func requireVarsDelivered(t *testing.T, expanded, wantBody string, vars map[string]string) {
	t.Helper()
	for name, value := range vars {
		require.Contains(t, expanded, "FLEET_VAR_"+name+"="+variables.PosixQuote(value))
	}
	body := stripPreamble(t, expanded)
	require.Equal(t, wantBody, body)
	for name, value := range vars {
		// a value the admin already wrote into the body proves nothing
		if value != "" && !strings.Contains(wantBody, value) {
			require.NotContains(t, body, value, "value for %s reached the script body", name)
		}
	}
}

// stripPreamble removes the preamble, which spans more than three lines when a
// value contains newlines.
func stripPreamble(t *testing.T, contents string) string {
	t.Helper()
	lines := strings.Split(contents, "\n")
	start := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "__fleet_lc=") })
	require.GreaterOrEqual(t, start, 0, "no preamble found in %q", contents)
	end := slices.IndexFunc(lines[start:], func(l string) bool { return strings.HasPrefix(l, "LC_ALL=${__fleet_lc}") })
	require.GreaterOrEqual(t, end, 0, "unterminated preamble in %q", contents)
	return strings.Join(slices.Concat(lines[:start], lines[start+end+1:]), "\n")
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
		requireVarsDelivered(t, script.ScriptContents, "echo $FLEET_VAR_HOST_UUID on $FLEET_VAR_HOST_PLATFORM",
			map[string]string{"HOST_UUID": "ABC-123", "HOST_PLATFORM": "ubuntu"})
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
		require.Equal(t, fleet.ExitCodeFleetVarResolutionFailed, savedResult.ExitCode)
		require.Contains(t, savedResult.Output, "There is no IdP username for this host.")
		require.Equal(t, host.ID, savedResult.HostID)

		// the returned script carries the exit code so fleetd skips it and
		// keeps processing its queue
		require.NotNil(t, script.ExitCode)
		require.EqualValues(t, fleet.ExitCodeFleetVarResolutionFailed, *script.ExitCode)
	})

	t.Run("already-completed execution is not re-recorded", func(t *testing.T) {
		svc, ctx, ds := newSvcAndCtx(t, host, "echo $FLEET_VAR_HOST_END_USER_IDP_USERNAME",
			new(int64(fleet.ExitCodeFleetVarResolutionFailed)))

		script, err := svc.GetHostScript(ctx, "exec-1")
		require.NoError(t, err)
		require.EqualValues(t, fleet.ExitCodeFleetVarResolutionFailed, *script.ExitCode)
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
		requireVarsDelivered(t, details.InstallScript, "install $FLEET_VAR_HOST_HARDWARE_SERIAL",
			map[string]string{"HOST_HARDWARE_SERIAL": "SERIAL-1"})
		requireVarsDelivered(t, details.PostInstallScript, "post ${FLEET_VAR_HOST_UUID}",
			map[string]string{"HOST_UUID": "ABC-123"})
		requireVarsDelivered(t, details.UninstallScript, "uninstall $FLEET_VAR_HOST_PLATFORM",
			map[string]string{"HOST_PLATFORM": "ubuntu"})
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
		require.Equal(t, fleet.ExitCodeFleetVarResolutionFailed, *savedResult.InstallScriptExitCode)
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
