package mdm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

func FuzzDecryptBase64CMS(f *testing.F) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(f, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "CMS fuzz recipient"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(4102444800, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(f, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(f, err)
	renewed := *cert
	renewed.SerialNumber = big.NewInt(2)

	f.Add([]byte("recovery-key"), true)
	f.Add([]byte{}, true)
	f.Add([]byte{0x30, 0x00}, false)
	f.Add([]byte("not base64!"), false)

	f.Fuzz(func(t *testing.T, data []byte, envelope bool) {
		if len(data) > 4096 {
			t.Skip()
		}
		if !envelope {
			_, _ = DecryptBase64CMSWithKey(string(data), key)
			_, _ = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(data), key)
			return
		}

		cms, err := pkcs7.Encrypt(data, []*x509.Certificate{cert})
		require.NoError(t, err)
		encoded := base64.StdEncoding.EncodeToString(cms)
		plain, err := DecryptBase64CMSWithKey(encoded, key)
		require.NoError(t, err)
		require.Equal(t, data, plain)
		plain, err = DecryptBase64CMS(encoded, cert, key)
		require.NoError(t, err)
		require.Equal(t, data, plain)
		plain, err = DecryptBase64CMSWithCerts(encoded, key, []*x509.Certificate{&renewed, cert})
		require.NoError(t, err)
		require.Equal(t, data, plain)
		_, err = DecryptBase64CMS(encoded, &renewed, key)
		require.Error(t, err)
		_, err = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(append(cms, 0)), key)
		require.Error(t, err)
	})
}
