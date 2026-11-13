package cryptoinfo

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	p12 "software.sslmate.com/src/go-pkcs12"
)

func FuzzTryP12(f *testing.F) {
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "PKCS12 fuzz identity"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(4102444800, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(f, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(f, err)
	f.Add([]byte{}, "", true)
	f.Add([]byte{0x30, 0x00}, "test123", false)
	f.Add(der, "password", false)

	f.Fuzz(func(t *testing.T, certificateDER []byte, password string, valid bool) {
		if len(certificateDER) > 8192 || len(password) > 128 {
			t.Skip()
		}
		for _, r := range password {
			if r > 0xffff {
				t.Skip()
			}
		}
		leaf := *cert
		if !valid {
			leaf.Raw = certificateDER
		}
		// Keep KDF parameters out of the mutation space to prevent unbounded work.
		bundle, err := p12.Modern.WithIterations(1).Encode(key, &leaf, []*x509.Certificate{cert}, password)
		require.NoError(t, err)
		infos, err := tryP12(bundle, password)
		parsed, parseErr := x509.ParseCertificate(leaf.Raw)
		if parseErr != nil {
			require.Error(t, err)
			return
		}
		require.NoError(t, err)
		require.Len(t, infos, 3)
		require.Equal(t, kiKEY, infos[0].Type)
		require.Nil(t, infos[0].Data)
		require.Equal(t, kiCERTIFICATE, infos[1].Type)
		require.Equal(t, kiCACERTIFICATE, infos[2].Type)
		for _, info := range infos {
			require.Equal(t, kiP12, info.Encoding)
			require.NoError(t, info.Error)
			_, err := json.Marshal(info)
			require.NoError(t, err)
		}
		extracted, ok := infos[1].Data.(*certExtract)
		require.True(t, ok)
		require.Equal(t, parsed.SerialNumber.String(), extracted.SerialNumber)
		require.Equal(t, parsed.Subject.String(), extracted.SubjectParsed)
		_, err = tryP12(bundle, password+"x")
		require.Error(t, err)
	})
}
