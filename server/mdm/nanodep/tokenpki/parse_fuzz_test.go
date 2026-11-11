package tokenpki

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

func FuzzUnwrapToken(f *testing.F) {
	f.Add([]byte{}, false)
	f.Add([]byte("Content-Type: application/pkcs7-mime\r\n\r\nMAA=\r\n"), false)
	f.Add([]byte("Content-Type: text/plain\r\n\r\n-----BEGIN MESSAGE-----\r\n{}\r\n-----END MESSAGE-----\r\n"), false)
	f.Add([]byte("token"), true)
	f.Add([]byte{}, true)

	f.Fuzz(func(t *testing.T, data []byte, wrap bool) {
		if len(data) > 8192 {
			t.Skip()
		}
		if !wrap {
			_, _ = UnwrapSMIME(data)
			_, _ = UnwrapTokenJSON(data)
			return
		}
		smime := []byte("Content-Type: application/pkcs7-mime\r\n\r\n" + base64.StdEncoding.EncodeToString(data) + "\r\n.\r\n")
		decoded, err := UnwrapSMIME(smime)
		require.NoError(t, err)
		require.True(t, bytes.Equal(data, decoded))

		payload, err := json.Marshal(map[string]string{"token": string(data)})
		require.NoError(t, err)
		wrapped := append([]byte("Content-Type: text/plain\r\n\r\n-----BEGIN MESSAGE-----\r\n"), payload...)
		wrapped = append(wrapped, []byte("\r\n-----END MESSAGE-----\r\n")...)
		decoded, err = UnwrapTokenJSON(wrapped)
		require.NoError(t, err)
		require.Equal(t, payload, decoded)
	})
}

func FuzzDecryptTokenJSON(f *testing.F) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(f, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ABM fuzz recipient"},
		NotBefore:    time.Unix(0, 0),
		NotAfter:     time.Unix(4102444800, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(f, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(f, err)
	f.Add([]byte("synthetic-token"))
	f.Add([]byte{})
	f.Add([]byte{0x30, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			t.Skip()
		}
		_, _ = DecryptTokenJSON(data, cert, key)
		rawSMIME := []byte("Content-Type: application/pkcs7-mime\r\n\r\n" + base64.StdEncoding.EncodeToString(data) + "\r\n.\r\n")
		_, _ = DecryptTokenJSON(rawSMIME, cert, key)

		payload, err := json.Marshal(map[string]string{"token": string(data)})
		require.NoError(t, err)
		wrapped := append([]byte("Content-Type: text/plain\r\n\r\n-----BEGIN MESSAGE-----\r\n"), payload...)
		wrapped = append(wrapped, []byte("\r\n-----END MESSAGE-----\r\n")...)
		cms, err := pkcs7.Encrypt(wrapped, []*x509.Certificate{cert})
		require.NoError(t, err)
		smime := []byte("Content-Type: application/pkcs7-mime\r\n\r\n" + base64.StdEncoding.EncodeToString(cms) + "\r\n.\r\n")
		decoded, err := DecryptTokenJSON(smime, cert, key)
		require.NoError(t, err)
		require.Equal(t, payload, decoded)
		renewed := *cert
		renewed.SerialNumber = big.NewInt(2)
		_, err = DecryptTokenJSON(smime, &renewed, key)
		require.Error(t, err)
	})
}
