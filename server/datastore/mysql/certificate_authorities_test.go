package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthority(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Delete", testDeleteCertificateAuthority},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testDeleteCertificateAuthority(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	var count int
	// TODO: Replace with get single CA once implemented
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, "SELECT COUNT(*) FROM certificate_authorities WHERE id = ?", 1)
	require.NoError(t, err)
	require.EqualValues(t, 0, count)
	// TODO: ^END

	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type: "hydrant",
	})
	require.NoError(t, err)

	// TODO: Replace with get single CA once implemented
	err = sqlx.GetContext(ctx, ds.reader(ctx), &count, "SELECT COUNT(*) FROM certificate_authorities WHERE id = ?", ca.ID)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	// TODO: ^END

	_, err = ds.DeleteCertificateAuthority(ctx, ca.ID)
	require.NoError(t, err)

	_, err = ds.DeleteCertificateAuthority(ctx, ca.ID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}
