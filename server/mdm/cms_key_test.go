package mdm

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

	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

// selfSignedCert issues a certificate for key with the given serial, so two certificates can share one key.
func selfSignedCert(t *testing.T, key *rsa.PrivateKey, serial int64) *x509.Certificate {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: "Fleet WSTEP test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func TestDecryptBase64CMSWithKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	original := selfSignedCert(t, key, 1)
	renewed := selfSignedCert(t, key, 2)

	const recoveryKey = "123456-123456-123456-123456-123456-123456-123456-123456"
	encrypted, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{original})
	require.NoError(t, err)
	b64 := base64.StdEncoding.EncodeToString(encrypted)

	// The certificate-matched decrypt is what breaks on renewal: same key, new serial, no recipient match.
	_, err = DecryptBase64CMS(b64, renewed, key)
	require.ErrorContains(t, err, "no enveloped recipient")

	plain, err := DecryptBase64CMSWithKey(b64, key)
	require.NoError(t, err)
	require.Equal(t, recoveryKey, string(plain))

	_, err = DecryptBase64CMSWithKey(b64, otherKey)
	require.Error(t, err)

	_, err = DecryptBase64CMSWithKey("not base64", key)
	require.Error(t, err)

	_, err = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(append(encrypted, 0x00)), key)
	require.ErrorContains(t, err, "trailing data")

	_, err = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(encrypted[:len(encrypted)/2]), key)
	require.Error(t, err)

	notEnveloped, err := asn1.Marshal(cmsContentInfo{ContentType: pkcs7.OIDData})
	require.NoError(t, err)
	_, err = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(notEnveloped), key)
	require.ErrorIs(t, err, pkcs7.ErrNotEncryptedContent)

	// Trailing elements of the enveloped data are not parsed here, and a truncated one makes the pkcs7 package panic
	// instead of failing. The helper turns that into an error.
	type craftedEnvelopedData struct {
		Version        int
		RecipientInfos []cmsRecipientInfo `asn1:"set"`
		Extra          asn1.RawValue
	}
	issuer, err := asn1.Marshal(asn1.RawValue{Class: 0, Tag: 16, IsCompound: true, Bytes: make([]byte, 300)})
	require.NoError(t, err)
	crafted, err := asn1.Marshal(craftedEnvelopedData{
		RecipientInfos: []cmsRecipientInfo{{IssuerAndSerialNumber: cmsIssuerAndSerial{IssuerName: asn1.RawValue{FullBytes: issuer}, SerialNumber: big.NewInt(1)}}},
		Extra:          asn1.RawValue{FullBytes: []byte{0x3F, 0x81}},
	})
	require.NoError(t, err)
	craftedMsg, err := asn1.Marshal(cmsContentInfo{ContentType: pkcs7.OIDEnvelopedData, Content: asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: crafted}})
	require.NoError(t, err)
	_, err = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(craftedMsg), key)
	require.ErrorContains(t, err, "parse CMS message")

	// A message with two recipients does not identify which one the key belongs to.
	two, err := pkcs7.Encrypt([]byte(recoveryKey), []*x509.Certificate{original, selfSignedCert(t, otherKey, 3)})
	require.NoError(t, err)
	_, err = DecryptBase64CMSWithKey(base64.StdEncoding.EncodeToString(two), key)
	require.ErrorContains(t, err, "expected 1")
}

func TestDecodeAndDecryptRejectsShortInput(t *testing.T) {
	const serverKey = "0123456789abcdef0123456789abcdef"
	_, err := DecodeAndDecrypt(base64.StdEncoding.EncodeToString([]byte{0x30, 0x82, 0x00}), serverKey)
	require.ErrorContains(t, err, "shorter than the nonce")
}
