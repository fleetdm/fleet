package file

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasCNRevokeRewritesIndex(t *testing.T) {
	dir := t.TempDir()
	d, err := NewFileDepot(dir)
	require.NoError(t, err)

	valid := "V\t300101000000Z\t\t01\tdevice.1.pem\t/CN=device\n"
	revoked := "R\t300101000000Z\t250101000000Z\t01\tdevice.1.pem\t/CN=device\n"
	other := "V\t300101000000Z\t\t02\tother.2.pem\t/CN=other\n"
	indexPath := filepath.Join(dir, "index.txt")
	require.NoError(t, os.WriteFile(indexPath, []byte(valid+revoked+other), dbPerm))

	cert := &x509.Certificate{Subject: pkix.Name{CommonName: "device"}}
	_, err = d.HasCN("device", 0, cert, true)
	require.NoError(t, err)

	got, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	require.Equal(t, revoked+other, string(got))
}
