package testing_utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
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

// EncodeDeviceInfo is a helper function to mock the x-aspen-deviceinfo header that is sent
// by the device during the Apple MDM enrollment process.
func EncodeDeviceInfo(t *testing.T, machineInfo fleet.MDMAppleMachineInfo) string {
	body, err := plist.Marshal(machineInfo)
	require.NoError(t, err)

	// body is expected to be a PKCS7 signed message, although we don't currently verify the signature
	signedData, err := pkcs7.NewSignedData(body)
	require.NoError(t, err)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	crtBytes, err := depot.NewCACert().SelfSign(rand.Reader, key.Public(), key)
	require.NoError(t, err)
	crt, err := x509.ParseCertificate(crtBytes)
	require.NoError(t, err)
	require.NoError(t, signedData.AddSigner(crt, key, pkcs7.SignerInfoConfig{}))
	sig, err := signedData.Finish()
	require.NoError(t, err)

	return base64.URLEncoding.EncodeToString(sig)
}
