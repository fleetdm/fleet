package apple_mdm

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/micromdm/plist"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

// Public synthetic test keys, generated once so worker seed corpora are identical.
const fuzzDeviceCAKey = "MIICXQIBAAKBgQDwBcGy0S4ZozfjP0SZqx1NR3sRop3P+XZjfusIFDecuGA57lFAEeEiVoPSIzbxCm2m6sSuTfzoJYaRrYyP3oI1CNFTyPc4nCVg9NBihsNVKp92psJjk92O/GX5//lv6xo8G/XyDQRu8usQlfglWkqUtVVl4lYTMx8tDtpF9/pGOQIDAQABAoGBAIW5n1ZiJr4P6pZIbI70/+lLVuzXXfc+NFBwqL+ii+1b5/i+pzP8AiOgcecEyZp3xenGSPgX6Tzpk1TcjUCmpoGTaTgTH3E6HUueiDv9WMHROabIwm5HaFdCsvs8J+4IJlsAauCFohe7KSo/KpI8D76oDozAkrySd4IEZ7Hb5t2RAkEA/yNb1rwMkCLlNcjFAnerCR1hKhhvn3sTZctyeOzfIJprf6aletRuk5VUBq57naX4VAtKbPginfBdNnc0rOkpVQJBAPDVU3YGKjhYSdBK15DBmU3bkyjtdqpkIfpUSqmOm6hmrb0LwOzk5RjYUwgkH0vY0XhiB+Ortsm5Z+s4mU5tWVUCQQDYs7+dc7t38SudsxkiL2ImK74LcxXb7z1aiKKB8y7REleb5H9JBIWm9H6Z7jkwfXeearicNWWxoCaLzFQBSUjBAkBsr1kjAlU7aFyna58s7LHKuTUIW2IaM2ZCGpg3POxyayQtFIjUX9gkBjwyiFq3V/nwYg09adjr4IhNVxWYlzChAkBpwA37FCqHjUUkwHlDbdrENt6lo2/t8LYKSwEX/tGCbaqxRpORpmTj6Pp+rkFb0WECRSkK6U/bi1tiqiJju/25"
const fuzzDeviceLeafKey = "MIICWwIBAAKBgQC01sFNQmBiztrFsTP+qdIoPUOceNWPqDoDm82yuXlM9UZ9ZYpoDAOggrzJ7wJzariXoP/kLXxfr4irdEoWpOyYbLUMN2CG1C1yUfJ9/kAR6L7s6y0iQ2LoY6fqubAdEd/6g9/BpFHT62FN9kVCtFLIm+G5bw6iZrKc/Bs9bC8XqwIDAQABAoGAHBcPx7R7zF1dVkAEV5aBcoaxreiHI915xX26Ifz1fqLK5EeVjDvxhElKkgNyWgNF0SfNF+CI5UQa8qcJqyLLLU8vzLKXztU0RMHGexB+aZ5ORA7kkxnWuy3Y0q7g8a/JucjVdbAKi/HcEg0fiQGXBNMpl6q7mB1C66513xUb4EECQQDf6JRu6Cc+zaUs8Gr1XkEMHJu3sMtY5LLKijMbtAVRJ+7qyCMHYevE9g7tSbfpnvgw++55Rmr4TjuD54faGNFTAkEAzsHor09ly2nPm59GQq8Q0ZJ3GBwGe//xr4wmrOGfoXNNnvIJTPZyDEPrA09Hkd5nPy5Y4i+NAuLmAvnuoWcdSQJAPDrW+sJuFCHDRcRiMSCqhLSJWq8u8/mI8pycRN528663QOaXLZw0jKXXvBKxayF9B8gtDOhZS/PXZJL6Yzt7ywJAQeA0lA7XHy2ScUIZL8v5rtxcx7i+NCoDRmyAMhsdMmK7e4Yggn4PLmtBaOeMpSRYfx/0P/Tol9/Va2NYreJdyQJAL1K++zA3L+urggZ+pwxDTMbFx4iJW2+VuoGuJBfA3BhXQX/uUgZ8F3Acoh3y6BaEYie5APyo90xYJh80OCMrCQ=="

type fuzzDeviceIdentity struct {
	cert *x509.Certificate
	key  *rsa.PrivateKey
}

func fuzzDeviceKey(tb testing.TB, encoded string) *rsa.PrivateKey {
	tb.Helper()
	der, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(tb, err)
	key, err := x509.ParsePKCS1PrivateKey(der)
	require.NoError(tb, err)
	return key
}

func fuzzDeviceCert(tb testing.TB, serial int64, subject string, key *rsa.PrivateKey, parent *fuzzDeviceIdentity, ca bool) *fuzzDeviceIdentity {
	tb.Helper()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: subject},
		NotBefore: time.Unix(0, 0), NotAfter: time.Unix(1, 0),
		IsCA: ca, BasicConstraintsValid: true,
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	issuer, signingKey := template, key
	if parent != nil {
		issuer, signingKey = parent.cert, parent.key
	}
	der, err := x509.CreateCertificate(nil, template, issuer, &key.PublicKey, signingKey)
	require.NoError(tb, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(tb, err)
	return &fuzzDeviceIdentity{cert: cert, key: key}
}

func fuzzDeviceChain(tb testing.TB) (ca, intermediate, leaf *fuzzDeviceIdentity) {
	tb.Helper()
	caKey, leafKey := fuzzDeviceKey(tb, fuzzDeviceCAKey), fuzzDeviceKey(tb, fuzzDeviceLeafKey)
	ca = fuzzDeviceCert(tb, 1, "synthetic-device-ca", caKey, nil, true)
	intermediate = fuzzDeviceCert(tb, 2, "synthetic-intermediate", caKey, ca, true)
	leaf = fuzzDeviceCert(tb, 3, "synthetic-device", leafKey, intermediate, false)
	return
}

func fuzzSignedDeviceinfo(tb testing.TB, body []byte, leaf *fuzzDeviceIdentity, bag []*x509.Certificate, attrs bool) []byte {
	tb.Helper()
	sd, err := pkcs7.NewSignedData(body)
	require.NoError(tb, err)
	sd.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)
	if attrs {
		require.NoError(tb, sd.AddSigner(leaf.cert, leaf.key, pkcs7.SignerInfoConfig{}))
		signer := &sd.GetSignedData().SignerInfos[0]
		// AddSigner inserts time.Now(). Replace it and re-sign to make the
		// complete seed, not just its key and content, stable across workers.
		for i := range signer.AuthenticatedAttributes {
			if signer.AuthenticatedAttributes[i].Type.Equal(pkcs7.OIDAttributeSigningTime) {
				fixed, err := asn1.Marshal(time.Unix(1700000000, 0).UTC())
				require.NoError(tb, err)
				signer.AuthenticatedAttributes[i].Value = asn1.RawValue{Tag: asn1.TagSet, IsCompound: true, Bytes: fixed}
			}
		}
		attributes := make([]authenticatedAttribute, 0, len(signer.AuthenticatedAttributes))
		for _, attr := range signer.AuthenticatedAttributes {
			attributes = append(attributes, authenticatedAttribute{Type: attr.Type, Value: attr.Value})
		}
		encoded, err := marshalAuthenticatedAttributes(attributes)
		require.NoError(tb, err)
		digest := sha256.Sum256(encoded)
		signer.EncryptedDigest, err = rsa.SignPKCS1v15(nil, leaf.key, crypto.SHA256, digest[:])
		require.NoError(tb, err)
	} else {
		require.NoError(tb, sd.SignWithoutAttr(leaf.cert, leaf.key, pkcs7.SignerInfoConfig{}))
	}
	for _, cert := range bag {
		sd.AddCertificate(cert)
	}
	blob, err := sd.Finish()
	require.NoError(tb, err)
	return blob
}

func FuzzParseDeviceinfo(f *testing.F) {
	_, intermediate, leaf := fuzzDeviceChain(f)
	body := []byte(`<plist><dict><key>SERIAL</key><string>synthetic-serial</string></dict></plist>`)
	signed := fuzzSignedDeviceinfo(f, body, leaf, []*x509.Certificate{intermediate.cert}, true)
	oversizedContent, err := plist.Marshal(fleet.MDMAppleMachineInfo{Serial: string(bytes.Repeat([]byte{'x'}, maxDeviceinfoSize))})
	require.NoError(f, err)
	oversized := fuzzSignedDeviceinfo(f, oversizedContent, leaf, []*x509.Certificate{intermediate.cert}, false)
	unsigned, err := pkcs7.DegenerateCertificate(leaf.cert.Raw)
	require.NoError(f, err)
	for _, raw := range [][]byte{nil, {0x30, 0x80, 0, 0}, signed, unsigned, oversized, bytes.Repeat([]byte{0}, maxDeviceinfoSize+1)} {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 16*1024 {
			t.Skip()
		}
		direct, p7, directErr := ParseMachineInfoFromPKCS7(raw)
		// Arbitrary strings exercise both base64 decoders, not only encodings
		// produced by the harness.
		_, _, _ = ParseDeviceinfo(string(raw))
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding} {
			info, wrapped, err := ParseDeviceinfo(encoding.EncodeToString(raw))
			if len(raw) > maxDeviceinfoSize {
				require.Error(t, err)
			} else {
				require.Equal(t, directErr == nil, err == nil)
				if err == nil {
					require.Equal(t, direct, info)
					require.Equal(t, p7.Content, wrapped.Content)
				}
			}
		}
		if directErr == nil {
			require.NotNil(t, direct)
			require.NotNil(t, p7)
			// Parsing is intentionally not an Apple trust decision.
			_ = VerifyMachineInfoSignature(p7)
		}
	})
}

func FuzzVerifyMachineInfoSignature(f *testing.F) {
	ca, intermediate, leaf := fuzzDeviceChain(f)
	bag := []*x509.Certificate{intermediate.cert, ca.cert}
	body := []byte(`<plist><dict><key>UDID</key><string>synthetic-udid</string></dict></plist>`)
	signed := fuzzSignedDeviceinfo(f, body, leaf, bag, true)
	require.Equal(f, signed, fuzzSignedDeviceinfo(f, body, leaf, bag, true))
	// A genuine signature-only cycle: each CA signs the other, but neither
	// reaches our pinned device CA.
	cycleA := fuzzDeviceCert(f, 10, "synthetic-cycle-a", ca.key, nil, true)
	cycleB := fuzzDeviceCert(f, 11, "synthetic-cycle-b", ca.key, cycleA, true)
	cycleA = fuzzDeviceCert(f, 10, "synthetic-cycle-a", ca.key, cycleB, true)
	cycleLeaf := fuzzDeviceCert(f, 12, "synthetic-cycle-leaf", leaf.key, cycleA, false)
	cycleBlob := fuzzSignedDeviceinfo(f, body, cycleLeaf, []*x509.Certificate{cycleA.cert, cycleB.cert}, true)
	cycleP7, err := pkcs7.Parse(cycleBlob)
	require.NoError(f, err)
	require.ErrorContains(f, verifyAppleSignedPKCS7(cycleP7, []*x509.Certificate{ca.cert}), "cycle")
	for mode := range uint8(22) {
		f.Add(signed, mode, uint16(0))
	}
	f.Add(signed, uint8(21), uint16(1))
	f.Add(signed, uint8(21), uint16(2))
	f.Add([]byte{}, uint8(0), uint16(0))
	f.Fuzz(func(t *testing.T, raw []byte, mode uint8, offset uint16) {
		if len(raw) > 12*1024 {
			t.Skip()
		}
		if cryptoutil.ValidateBERDepth(raw, cryptoutil.MaxBERDepth) == nil {
			if p7, err := pkcs7.Parse(raw); err == nil {
				_ = verifyAppleSignedPKCS7(p7, []*x509.Certificate{ca.cert})
				_ = VerifyMachineInfoSignature(p7)
			}
		}
		// Derive a valid plist and signature from every input so mutations
		// continue reaching verification after raw CMS syntax stops parsing.
		payload := raw
		if len(payload) > 512 {
			payload = payload[:512]
		}
		expected := fleet.MDMAppleMachineInfo{Serial: base64.StdEncoding.EncodeToString(payload), UDID: "synthetic-udid"}
		content, err := plist.Marshal(expected)
		require.NoError(t, err)
		blob := fuzzSignedDeviceinfo(t, content, leaf, bag, mode%22 != 19)
		info, p7, err := ParseMachineInfoFromPKCS7(blob)
		require.NoError(t, err)
		require.Equal(t, &expected, info)
		require.NoError(t, verifyAppleSignedPKCS7(p7, []*x509.Certificate{ca.cert}))
		require.Error(t, VerifyMachineInfoSignature(p7))
		wantValid := false
		signer := &p7.Signers[0]
		switch mode % 22 {
		case 0, 19:
			wantValid = true
		case 1:
			p7.Content[int(offset)%len(p7.Content)] ^= 1
		case 2:
			signer.EncryptedDigest[int(offset)%len(signer.EncryptedDigest)] ^= 1
		case 3:
			signer.DigestAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3, int(offset)}
		case 4, 5:
			for i := range signer.AuthenticatedAttributes {
				attr := &signer.AuthenticatedAttributes[i]
				if (mode%22 == 4 && attr.Type.Equal(pkcs7.OIDAttributeMessageDigest)) ||
					(mode%22 == 5 && attr.Type.Equal(pkcs7.OIDAttributeContentType)) {
					attr.Value.Bytes = []byte{5, 0}
				}
			}
		case 6:
			for i := range p7.Certificates {
				j := (i + int(offset)) % len(p7.Certificates)
				p7.Certificates[i], p7.Certificates[j] = p7.Certificates[j], p7.Certificates[i]
			}
			wantValid = true
		case 7:
			p7.Certificates = bag
		case 8, 9:
			count := maxDeviceinfoCerts
			if mode%22 == 8 {
				count++
			}
			for len(p7.Certificates) < count {
				p7.Certificates = append(p7.Certificates, intermediate.cert)
			}
			wantValid = count == maxDeviceinfoCerts
		case 10:
			p7, err = pkcs7.Parse(cycleBlob)
			require.NoError(t, err)
		case 11:
			attributes := make([]authenticatedAttribute, 0, len(signer.AuthenticatedAttributes))
			for _, attr := range signer.AuthenticatedAttributes {
				attributes = append(attributes, authenticatedAttribute{Type: attr.Type, Value: attr.Value})
			}
			encoded, err := marshalAuthenticatedAttributes(attributes)
			require.NoError(t, err)
			digest := sha256.Sum256(encoded)
			signer.EncryptedDigest, err = rsa.SignPKCS1v15(nil, ca.key, crypto.SHA256, digest[:])
			require.NoError(t, err)
		case 12:
			p7.Signers = nil
		case 13:
			p7.Signers = append(p7.Signers, p7.Signers[0])
		case 14:
			signer.DigestEncryptionAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 3, int(offset)}
		case 15:
			signer.AuthenticatedAttributes = nil
		case 16, 17:
			for i, cert := range p7.Certificates {
				if bytes.Equal(cert.Raw, intermediate.cert.Raw) {
					clone := *cert
					if mode%22 == 16 {
						clone.Signature = bytes.Clone(cert.Signature)
						clone.Signature[int(offset)%len(clone.Signature)] ^= 1
					} else {
						clone.IsCA = false
					}
					p7.Certificates[i] = &clone
				}
			}
		case 18:
			// Matching issuer names are not trust: the key must match the pin.
			fakeCA := *ca.cert
			fakeCA.PublicKey = &leaf.key.PublicKey
			require.Error(t, verifyAppleSignedPKCS7(p7, []*x509.Certificate{&fakeCA}))
			wantValid = true
		case 20:
			p7.Certificates = []*x509.Certificate{leaf.cert}
		case 21:
			// Exercise the internal signature-work budget independently of the
			// public certificate-count cap, at and just beyond the boundary.
			n := maxChainSignatureChecks - 2 + int(offset%3)
			decoys := make([]*x509.Certificate, 0, n+1)
			for i := range n {
				decoy := *intermediate.cert
				decoy.Raw = []byte{byte(i), byte(i >> 8)}
				decoy.PublicKey = &leaf.key.PublicKey
				decoys = append(decoys, &decoy)
			}
			decoys = append(decoys, intermediate.cert)
			err := verifyChainToDeviceCA(leaf.cert, decoys, []*x509.Certificate{ca.cert})
			if n+2 > maxChainSignatureChecks {
				require.ErrorIs(t, err, errChainSignatureBudget)
			} else {
				require.NoError(t, err)
			}
			wantValid = true
		}
		err = verifyAppleSignedPKCS7(p7, []*x509.Certificate{ca.cert})
		if wantValid {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
		}
		require.Error(t, VerifyMachineInfoSignature(p7))
	})
}
