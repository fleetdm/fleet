package sso

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// A missing session has to answer to two different callers: the authz
// middleware matches on the AuthRequiredError type, and the SSO callbacks match
// on ErrSessionNotFound so they can tell the end user their sign-in timed out.
func TestSessionNotFoundErrorSatisfiesBothCallers(t *testing.T) {
	authRequired := fleet.NewAuthRequiredError("session not found")
	err := &sessionNotFoundError{authRequired: authRequired}

	var asAuthRequired *fleet.AuthRequiredError
	require.ErrorAs(t, err, &asAuthRequired)
	require.ErrorIs(t, err, ErrSessionNotFound)

	// Callers that surface the message must not see it change.
	require.Equal(t, authRequired.Error(), err.Error())
}
