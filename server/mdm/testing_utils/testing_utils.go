package testing_utils

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/stretchr/testify/require"
)

func NewTestMDMAppleCertTemplate() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1},
					Value: "com.apple.mgmt.Example",
				},
			},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
}

// StartNewAppleGDMFTestServer creates a new test server that serves the GDMF data from the testdata
// file. It also sets the necessary dev mode overrides to point to the test server and disable
// caching. It closes the server and clears the underlying overrides when the test finishes.
func StartNewAppleGDMFTestServer(t *testing.T) {
	appleGDMFSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// load the test data from the file
		b, err := os.ReadFile("../../../server/mdm/apple/gdmf/testdata/gdmf.json")
		require.NoError(t, err)
		_, err = w.Write(b)
		require.NoError(t, err)
	}))
	t.Cleanup(appleGDMFSrv.Close)

	dev_mode.SetOverride("FLEET_DEV_GDMF_URL", appleGDMFSrv.URL, t)
	dev_mode.SetOverride("FLEET_DEV_GDMF_CACHE_DURATION", "0", t) // disable cache to ensure we're getting the data from the test server
}
