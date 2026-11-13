package cryptoutil

import (
	"bytes"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

func FuzzValidateBERDepth(f *testing.F) {
	for _, raw := range [][]byte{nil, {0x30, 0x80, 0, 0}, {0x30, 0x82}, {0x04, 0x80}, {0x3f, 0x80}, {0x30, 0x89}} {
		f.Add(raw, uint8(63), uint8(64), false)
		f.Add(raw, uint8(63), uint8(63), true)
	}
	f.Fuzz(func(t *testing.T, raw []byte, limitByte, depthByte uint8, indefinite bool) {
		if len(raw) > 16*1024 {
			t.Skip()
		}
		limit, depth := int(limitByte)%96+1, int(depthByte)%128+1
		err := ValidateBERDepth(raw, limit)
		require.True(t, err == nil || errors.Is(err, ErrBERTooDeep))
		require.NoError(t, ValidateBERDepth(raw, 0))
		require.NoError(t, ValidateBERDepth(raw, -1))

		// Syntax errors belong to the downstream parser, not this depth guard.
		for _, malformed := range [][]byte{{0x30}, {0x30, 0x82, 1}, {0x04, 0x80}, {0x04, 0x7f}} {
			require.NoError(t, ValidateBERDepth(malformed, limit))
		}
		// An independent generator produces valid BER, including long-form
		// lengths. Primitive contents must never be treated as nested tags.
		value := []byte{4, 2, 0x30, 0x80}
		for i := range depth {
			useIndefinite := indefinite
			if len(raw) != 0 && raw[i%len(raw)]&1 != 0 {
				useIndefinite = !useIndefinite
			}
			if useIndefinite {
				value = append(append([]byte{0x30, 0x80}, value...), 0, 0)
			} else {
				value, err = asn1.Marshal(asn1.RawValue{Tag: asn1.TagSequence, IsCompound: true, Bytes: value})
				require.NoError(t, err)
			}
		}
		require.NoError(t, ValidateBERDepth(value, depth))
		if depth > 1 {
			require.ErrorIs(t, ValidateBERDepth(value, depth-1), ErrBERTooDeep)
		}
		err = ValidateBERDepth(value, limit)
		if depth > limit {
			require.ErrorIs(t, err, ErrBERTooDeep)
		} else {
			require.NoError(t, err)
		}
	})
}

// Generated solely for these tests; never use this public synthetic key outside tests.
const fuzzMDMKey = "MIICXQIBAAKBgQDwBcGy0S4ZozfjP0SZqx1NR3sRop3P+XZjfusIFDecuGA57lFAEeEiVoPSIzbxCm2m6sSuTfzoJYaRrYyP3oI1CNFTyPc4nCVg9NBihsNVKp92psJjk92O/GX5//lv6xo8G/XyDQRu8usQlfglWkqUtVVl4lYTMx8tDtpF9/pGOQIDAQABAoGBAIW5n1ZiJr4P6pZIbI70/+lLVuzXXfc+NFBwqL+ii+1b5/i+pzP8AiOgcecEyZp3xenGSPgX6Tzpk1TcjUCmpoGTaTgTH3E6HUueiDv9WMHROabIwm5HaFdCsvs8J+4IJlsAauCFohe7KSo/KpI8D76oDozAkrySd4IEZ7Hb5t2RAkEA/yNb1rwMkCLlNcjFAnerCR1hKhhvn3sTZctyeOzfIJprf6aletRuk5VUBq57naX4VAtKbPginfBdNnc0rOkpVQJBAPDVU3YGKjhYSdBK15DBmU3bkyjtdqpkIfpUSqmOm6hmrb0LwOzk5RjYUwgkH0vY0XhiB+Ortsm5Z+s4mU5tWVUCQQDYs7+dc7t38SudsxkiL2ImK74LcxXb7z1aiKKB8y7REleb5H9JBIWm9H6Z7jkwfXeearicNWWxoCaLzFQBSUjBAkBsr1kjAlU7aFyna58s7LHKuTUIW2IaM2ZCGpg3POxyayQtFIjUX9gkBjwyiFq3V/nwYg09adjr4IhNVxWYlzChAkBpwA37FCqHjUUkwHlDbdrENt6lo2/t8LYKSwEX/tGCbaqxRpORpmTj6Pp+rkFb0WECRSkK6U/bi1tiqiJju/25"

func fuzzMDMSignature(tb testing.TB, body []byte, forged bool) ([]byte, *x509.Certificate) {
	tb.Helper()
	keyDER, err := base64.StdEncoding.DecodeString(fuzzMDMKey)
	require.NoError(tb, err)
	key, err := x509.ParsePKCS1PrivateKey(keyDER)
	require.NoError(tb, err)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "synthetic-mdm-fuzz"},
		NotBefore: time.Unix(0, 0), NotAfter: time.Unix(2000000000, 0),
		KeyUsage: x509.KeyUsageDigitalSignature,
	}
	publicKey := key.PublicKey
	if forged {
		// The CMS signature will still be made by the original key, but the
		// advertised signer carries a different, fixed synthetic public key.
		publicKey.N = new(big.Int).Add(key.N, big.NewInt(2))
	}
	der, err := x509.CreateCertificate(nil, template, template, &publicKey, key)
	require.NoError(tb, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(tb, err)
	sd, err := pkcs7.NewSignedData(body)
	require.NoError(tb, err)
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	require.NoError(tb, sd.SignWithoutAttr(cert, key, pkcs7.SignerInfoConfig{}))
	sd.Detach()
	blob, err := sd.Finish()
	require.NoError(tb, err)
	return blob, cert
}

func FuzzVerifyMdmSignature(f *testing.F) {
	body := []byte("synthetic MDM request")
	blob, signer := fuzzMDMSignature(f, body, false)
	forged, _ := fuzzMDMSignature(f, body, true)
	repeated, _ := fuzzMDMSignature(f, body, false)
	require.Equal(f, blob, repeated)
	f.Add(base64.StdEncoding.EncodeToString(blob), body)
	f.Add(base64.StdEncoding.EncodeToString(forged), body)
	f.Add("", []byte{})
	f.Add("!!!!", []byte("body"))
	f.Add(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0}, maxMdmSignatureBytes+1)), body)
	f.Fuzz(func(t *testing.T, header string, body []byte) {
		if len(header) > 16*1024 || len(body) > 4096 {
			t.Skip()
		}
		cert, err := VerifyMdmSignature(header, body)
		if err == nil {
			require.NotNil(t, cert)
			require.NotEmpty(t, cert.Raw)
		}
		_, err = VerifyMdmSignature(base64.StdEncoding.EncodeToString(forged), body)
		require.Error(t, err)
		// Keep the signed input fixed and mutate only its body. Acceptance
		// must imply the exact original bytes, not merely a parseable CMS.
		cert, err = VerifyMdmSignature(base64.StdEncoding.EncodeToString(blob), body)
		if bytes.Equal(body, []byte("synthetic MDM request")) {
			require.NoError(t, err)
			require.Equal(t, signer.Raw, cert.Raw)
		} else {
			require.Error(t, err)
			require.Nil(t, cert)
		}
	})
}
