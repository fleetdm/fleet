package test

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_authz "github.com/fleetdm/fleet/v4/server/platform/authz"
	"github.com/stretchr/testify/require"
)

// ErrForbidden and ErrNotFound are markers for authorization test tables.
var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

// RequireErrKind asserts that got is of the kind want describes. A nil want
// means the call must succeed.
func RequireErrKind(t *testing.T, want, got error) {
	t.Helper()

	switch want {
	case nil:
		require.NoError(t, got)
	case ErrForbidden:
		require.Error(t, got)
		require.True(t, platform_authz.IsForbidden(got), "expected a forbidden error, got %v", got)
	case ErrNotFound:
		require.Error(t, got)
		require.True(t, fleet.IsNotFound(got), "expected a not-found error, got %v", got)
	default:
		t.Fatalf("unknown expected-error marker: %v", want)
	}
}
