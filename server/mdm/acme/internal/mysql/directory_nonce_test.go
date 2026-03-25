package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestDirectoryNonce(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "acme_directory_nonce")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"GetACMEEnrollment", testGetACMEEnrollment},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

func testGetACMEEnrollment(t *testing.T, env *testEnv) {
	// non-existing
	enrollment, err := env.ds.GetACMEEnrollment(t.Context(), "non-existing")
	require.Nil(t, enrollment)
	var acmeErr *types.ACMEError
	require.ErrorAs(t, err, &acmeErr)
	require.Contains(t, acmeErr.Type, "error/enrollmentNotFound") // nolint:nilaway // cannot be nil due to previous require

	// existing and valid
	enrollValid := &types.Enrollment{}
	env.InsertACMEEnrollment(t, enrollValid)
	enrollment, err = env.ds.GetACMEEnrollment(t.Context(), enrollValid.PathIdentifier)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	require.True(t, enrollment.IsValid())

	// existing and revoked
	enrollRevoked := &types.Enrollment{Revoked: true}
	env.InsertACMEEnrollment(t, enrollRevoked)
	enrollment, err = env.ds.GetACMEEnrollment(t.Context(), enrollRevoked.PathIdentifier)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	require.False(t, enrollment.IsValid())

	// existing and not-valid-after in the future
	enrollFuture := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(24 * time.Hour))}
	env.InsertACMEEnrollment(t, enrollFuture)
	enrollment, err = env.ds.GetACMEEnrollment(t.Context(), enrollFuture.PathIdentifier)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	require.True(t, enrollment.IsValid())

	// existing and not-valid-after in the past
	enrollPast := &types.Enrollment{NotValidAfter: ptr.T(time.Now().Add(-24 * time.Hour))}
	env.InsertACMEEnrollment(t, enrollPast)
	enrollment, err = env.ds.GetACMEEnrollment(t.Context(), enrollPast.PathIdentifier)
	require.NoError(t, err)
	require.NotNil(t, enrollment)
	require.False(t, enrollment.IsValid())
}
