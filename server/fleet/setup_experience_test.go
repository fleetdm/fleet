package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSetupExperienceStatusResultIsValid(t *testing.T) {
	id := ptr.Uint(1)
	str := ptr.String("x")
	for _, tc := range []struct {
		Name  string
		Case  SetupExperienceStatusResult
		Valid bool
	}{
		{
			Case: SetupExperienceStatusResult{
				SoftwareInstallerID: id,
			},
			Valid: true,
			Name:  "just software installer",
		},
		{
			Case: SetupExperienceStatusResult{
				SoftwareInstallerID:             id,
				HostSoftwareInstallsExecutionID: str,
			},
			Valid: true,
			Name:  "software and result",
		},
		{
			Case: SetupExperienceStatusResult{
				SoftwareInstallerID: id,
				NanoCommandUUID:     str,
			},
			Valid: false,
			Name:  "installer and vpp secondary",
		},
		{
			Case: SetupExperienceStatusResult{
				SoftwareInstallerID: id,
				ScriptExecutionID:   str,
			},
			Valid: false,
			Name:  "installer and script secondary",
		},

		{
			Case: SetupExperienceStatusResult{
				VPPAppTeamID: id,
			},
			Valid: true,
			Name:  "just vpp app team",
		},
		{
			Case: SetupExperienceStatusResult{
				VPPAppTeamID:    id,
				NanoCommandUUID: str,
			},
			Valid: true,
			Name:  "vpp app and result",
		},
		{
			Case: SetupExperienceStatusResult{
				VPPAppTeamID:                    id,
				HostSoftwareInstallsExecutionID: str,
			},
			Valid: false,
			Name:  "vpp and installer secondary",
		},
		{
			Case: SetupExperienceStatusResult{
				VPPAppTeamID:      id,
				ScriptExecutionID: str,
			},
			Valid: false,
			Name:  "vpp and script secondary",
		},
		{
			Case: SetupExperienceStatusResult{
				SetupExperienceScriptID: id,
			},
			Valid: true,
			Name:  "just script id",
		},
		{
			Case: SetupExperienceStatusResult{
				SetupExperienceScriptID: id,
				ScriptExecutionID:       str,
			},
			Valid: true,
			Name:  "script and result",
		},
		{
			Case: SetupExperienceStatusResult{
				SetupExperienceScriptID:         id,
				HostSoftwareInstallsExecutionID: str,
			},
			Valid: false,
			Name:  "script and installer secondary",
		},
		{
			Case: SetupExperienceStatusResult{
				SetupExperienceScriptID: id,
				NanoCommandUUID:         str,
			},
			Valid: false,
			Name:  "script and vpp secondary",
		},
		{
			Case: SetupExperienceStatusResult{
				SoftwareInstallerID: id,
				VPPAppTeamID:        id,
			},
			Valid: false,
			Name:  "installer and vpp",
		},
		{
			Case: SetupExperienceStatusResult{
				SoftwareInstallerID:     id,
				SetupExperienceScriptID: id,
			},
			Valid: false,
			Name:  "installer and script",
		},
		{
			Case: SetupExperienceStatusResult{
				VPPAppTeamID:            id,
				SetupExperienceScriptID: id,
			},
			Valid: false,
			Name:  "vpp and script",
		},
	} {
		err := tc.Case.IsValid()
		if tc.Valid {
			require.NoError(t, err, tc.Name)
		} else {
			require.Error(t, err, tc.Name)
		}
	}
}
