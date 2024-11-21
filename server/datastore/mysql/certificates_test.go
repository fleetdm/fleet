package mysql

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCertificates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SavePKICertificate", testSavePKICertificate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testSavePKICertificate(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Failure case
	err := ds.SavePKICertificate(ctx, &fleet.PKICertificate{})
	require.Error(t, err)

	certs, err := ds.ListPKICertificates(ctx)
	require.NoError(t, err)
	assert.Empty(t, certs)

	// Save name and key only
	pkiCert := &fleet.PKICertificate{
		Name: "test",
		Key:  []byte("key"),
	}

	// Save name and key only
	err = ds.SavePKICertificate(ctx, pkiCert)
	require.NoError(t, err)
	retrieved, err := ds.GetPKICertificate(ctx, pkiCert.Name)
	require.NoError(t, err)
	assert.Equal(t, pkiCert.Name, retrieved.Name)
	assert.Equal(t, pkiCert.Key, retrieved.Key)

	// Update cert with other values
	pkiCert.Cert = []byte("cert")
	now := time.Now().Truncate(time.Second).UTC()
	pkiCert.NotValidAfter = &now
	h := sha256.New()
	_, _ = io.Copy(h, bytes.NewReader(pkiCert.Cert)) // writes to a Hash can never fail
	sha256Hash := hex.EncodeToString(h.Sum(nil))
	pkiCert.Sha256 = &sha256Hash
	err = ds.SavePKICertificate(ctx, pkiCert)
	require.NoError(t, err)

	retrieved, err = ds.GetPKICertificate(ctx, pkiCert.Name)
	require.NoError(t, err)
	assert.Equal(t, pkiCert.Name, retrieved.Name)
	assert.Equal(t, pkiCert.Key, retrieved.Key)
	assert.Equal(t, pkiCert.Cert, retrieved.Cert)
	assert.Equal(t, pkiCert.NotValidAfter, retrieved.NotValidAfter)
	assert.Equal(t, pkiCert.Sha256, retrieved.Sha256)
	pkiCerts := []*fleet.PKICertificate{pkiCert}

	// Now save all values at once
	later := time.Now().Truncate(time.Second).UTC().Add(time.Hour)
	fakeSha256 := strings.Repeat("b", 64)
	pkiCert = &fleet.PKICertificate{
		Name:          "test2",
		Key:           []byte("key2"),
		Cert:          []byte("cert2"),
		NotValidAfter: &later,
		Sha256:        &fakeSha256,
	}
	err = ds.SavePKICertificate(ctx, pkiCert)
	require.NoError(t, err)

	retrieved, err = ds.GetPKICertificate(ctx, pkiCert.Name)
	require.NoError(t, err)
	assert.Equal(t, pkiCert.Name, retrieved.Name)
	assert.Equal(t, pkiCert.Key, retrieved.Key)
	assert.Equal(t, pkiCert.Cert, retrieved.Cert)
	assert.Equal(t, pkiCert.NotValidAfter, retrieved.NotValidAfter)
	assert.Equal(t, pkiCert.Sha256, retrieved.Sha256)
	pkiCerts = append(pkiCerts, pkiCert)

	certs, err = ds.ListPKICertificates(ctx)
	require.NoError(t, err)
	require.Len(t, certs, 2)
	for i, cert := range certs {
		assert.Equal(t, pkiCerts[i].Name, cert.Name)
		assert.Empty(t, cert.Cert)
		assert.Empty(t, cert.Key)
		assert.Equal(t, pkiCerts[i].NotValidAfter, cert.NotValidAfter)
		assert.Equal(t, pkiCerts[i].Sha256, cert.Sha256)
	}

	// Delete certs
	require.NoError(t, ds.DeletePKICertificate(ctx, pkiCerts[0].Name))
	certs, err = ds.ListPKICertificates(ctx)
	require.NoError(t, err)
	assert.Len(t, certs, 1)
	require.NoError(t, ds.DeletePKICertificate(ctx, pkiCerts[1].Name))
	certs, err = ds.ListPKICertificates(ctx)
	require.NoError(t, err)
	assert.Empty(t, certs)

}
