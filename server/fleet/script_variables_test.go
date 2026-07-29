package fleet

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFleetVariablesInScript(t *testing.T) {
	t.Parallel()

	t.Run("no variables", func(t *testing.T) {
		for _, contents := range []string{
			"",
			"#!/bin/sh\necho hello",
			"echo $FLEET_SECRET_MY_SECRET",
			"echo $FLEET_HOST_VITAL_computer_name",
			"echo $SOME_OTHER_VAR and ${ANOTHER}",
		} {
			require.NoError(t, ValidateFleetVariablesInScript(contents, false), contents)
			require.NoError(t, ValidateFleetVariablesInScript(contents, true), contents)
		}
	})

	t.Run("supported variables on premium", func(t *testing.T) {
		for _, v := range FleetVarsSupportedInScripts {
			require.NoError(t, ValidateFleetVariablesInScript("echo "+v.WithPrefix(), true), v)
			require.NoError(t, ValidateFleetVariablesInScript("echo "+v.WithBraces(), true), v)
			require.NoError(t, ValidateFleetVariablesInScript(
				fmt.Sprintf("echo user_%s@example.com", v.WithBraces()), true), v)
		}
		require.NoError(t, ValidateFleetVariablesInScript(
			"#!/bin/sh\necho $FLEET_VAR_HOST_UUID on ${FLEET_VAR_HOST_PLATFORM}\n", true))
	})

	t.Run("unsupported variables on premium", func(t *testing.T) {
		unsupported := []FleetVarName{
			"NONEXISTENT",
			FleetVarHostEndUserEmailIDP,
			FleetVarNDESSCEPChallenge,
			FleetVarPSSODeviceRegistrationToken,
			"CUSTOM_SCEP_CHALLENGE_FOO",
			"DIGICERT_DATA_FOO",
			"SMALLSTEP_SCEP_CHALLENGE_FOO",
		}
		for _, v := range unsupported {
			for _, contents := range []string{"echo " + v.WithPrefix(), "echo " + v.WithBraces()} {
				err := ValidateFleetVariablesInScript(contents, true)
				require.Error(t, err, contents)
				var iae *InvalidArgumentError
				require.ErrorAs(t, err, &iae, contents)
				require.ErrorContains(t, err,
					fmt.Sprintf("Fleet variable $FLEET_VAR_%s is not supported in scripts.", v))
			}
		}
	})

	t.Run("mixed supported and unsupported", func(t *testing.T) {
		err := ValidateFleetVariablesInScript(
			"echo $FLEET_VAR_HOST_UUID\necho $FLEET_VAR_NONEXISTENT", true)
		require.ErrorContains(t, err, "$FLEET_VAR_NONEXISTENT is not supported in scripts")
	})

	t.Run("any variable on free returns license error", func(t *testing.T) {
		for _, contents := range []string{
			"echo $FLEET_VAR_HOST_UUID",
			"echo ${FLEET_VAR_HOST_END_USER_IDP_USERNAME}",
			"echo $FLEET_VAR_NONEXISTENT",
		} {
			err := ValidateFleetVariablesInScript(contents, false)
			require.ErrorIs(t, err, ErrMissingLicense, contents)
		}
	})
}
