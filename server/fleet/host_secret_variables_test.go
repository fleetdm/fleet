package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateNoHostSecretVariables(t *testing.T) {
	require.NoError(t, ValidateNoHostSecretVariables(""))
	require.NoError(t, ValidateNoHostSecretVariables("$FLEET_SECRET_X $FLEET_VAR_HOST_UUID FLEET_HOST_SECRET_ENROLL_SECRET without a dollar sign"))

	err := ValidateNoHostSecretVariables("<string>" + HostSecretPlaceholder(HostSecretEnrollSecret) + "</string>")
	require.Error(t, err)
	require.Contains(t, err.Error(), "$FLEET_HOST_SECRET_ENROLL_SECRET is reserved")

	err = ValidateNoHostSecretVariables("${FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD}")
	require.Error(t, err)
	require.Contains(t, err.Error(), "$FLEET_HOST_SECRET_RECOVERY_LOCK_PASSWORD is reserved")
}
