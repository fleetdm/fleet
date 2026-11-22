package x509util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func FuzzParseChallengePassword(f *testing.F) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(f, err)
	f.Add([]byte{}, "challenge")
	f.Add([]byte{0x30, 0x00}, "")
	f.Add([]byte("not DER"), "with \x00 and \u00e9")

	f.Fuzz(func(t *testing.T, der []byte, password string) {
		if len(der) > 8192 || len(password) > 512 {
			t.Skip()
		}
		_, _ = ParseChallengePassword(der)
		if !utf8.ValidString(password) {
			return
		}

		template := &CertificateRequest{ChallengePassword: password}
		template.Subject.CommonName = "SCEP fuzz device"
		csr, err := CreateCertificateRequest(rand.Reader, template, key)
		require.NoError(t, err)
		got, err := ParseChallengePassword(csr)
		require.NoError(t, err)
		require.Equal(t, password, got)
		parsed, err := x509.ParseCertificateRequest(csr)
		require.NoError(t, err)
		require.NoError(t, parsed.CheckSignature())
		_, err = ParseChallengePassword(append(csr, 0))
		require.Error(t, err)
	})
}
