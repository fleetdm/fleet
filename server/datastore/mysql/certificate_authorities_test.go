package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestCertificateAuthority(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetCertificateAuthorityByID", testGetCertificateAuthorityByID},
		{"ListCertificateAuthorities", testListCertificateAuthorities},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testGetCertificateAuthorityByID(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// get unknown CA
	id := uint(9999)
	_, err := ds.GetCertificateAuthorityByID(ctx, id, true)
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// TODO: test for a known CA. need ds create method for this.
}

func testListCertificateAuthorities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// list is empty
	cas, err := ds.ListCertificateAuthorities(ctx)
	require.NoError(t, err)
	require.Empty(t, cas)

	// TODO: test for known CAs. need ds create method for this.
}
