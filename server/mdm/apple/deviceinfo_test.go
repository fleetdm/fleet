package apple_mdm

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

// TestVerifyMachineInfoSignatureGenuineCapture verifies a real Apple-signed
// deviceinfo blob captured from a lab iPad against the pinned Apple iPhone
// Device CA. The blob has CMS authenticated attributes, a SHA-1/RSA signature,
// and a chain with intermediates that expired in 2014 and 2022 just like any
// other Apple device.
func TestVerifyMachineInfoSignatureGenuineCapture(t *testing.T) {
	buf, err := os.ReadFile("testdata/deviceinfo/machineinfo-ipados.der")
	require.NoError(t, err)

	info, p7, err := ParseMachineInfoFromPKCS7(buf)
	require.NoError(t, err)
	require.Equal(t, "iPad13,18", info.Product)
	require.Equal(t, "G4NQM9HHK0", info.Serial)
	require.Equal(t, "00008101-001938611EE9001E", info.UDID)

	require.NoError(t, VerifyMachineInfoSignature(p7))

	t.Run("tampered content is rejected", func(t *testing.T) {
		tampered := *p7
		content := make([]byte, len(p7.Content))
		copy(content, p7.Content)
		content[len(content)/2] ^= 0xff
		tampered.Content = content
		err := VerifyMachineInfoSignature(&tampered)
		require.ErrorContains(t, err, "messageDigest")
	})

	t.Run("signerless payload is rejected", func(t *testing.T) {
		signerless := *p7
		signerless.Signers = nil
		err := VerifyMachineInfoSignature(&signerless)
		require.ErrorContains(t, err, "not signed")
	})

	t.Run("multiple signers are rejected", func(t *testing.T) {
		multi := *p7
		multi.Signers = slices.Concat(p7.Signers, p7.Signers[:1])
		err := VerifyMachineInfoSignature(&multi)
		require.ErrorContains(t, err, "exactly one signer")
	})

	t.Run("unsupported digest algorithm is rejected", func(t *testing.T) {
		bad := *p7
		bad.Signers = slices.Clone(p7.Signers)
		bad.Signers[0].DigestAlgorithm.Algorithm = asn1.ObjectIdentifier{1, 2, 840, 113549, 2, 5} // MD5
		err := VerifyMachineInfoSignature(&bad)
		require.ErrorContains(t, err, "unsupported digest algorithm")
	})

	t.Run("missing signer certificate is an error, not a panic", func(t *testing.T) {
		noCerts := *p7
		noCerts.Certificates = nil
		err := VerifyMachineInfoSignature(&noCerts)
		require.ErrorContains(t, err, "signer certificate not included")
	})

	t.Run("wrong device CA is rejected", func(t *testing.T) {
		ca := newTestCA(t, x509.SHA256WithRSA)
		err := verifyAppleSignedPKCS7(p7, []*x509.Certificate{ca.cert})
		require.ErrorContains(t, err, "is not issued by a pinned Apple device CA")
	})

	t.Run("verifies against the pinned Apple iPhone Device CA", func(t *testing.T) {
		// the production anchor set; the genuine capture must verify against it
		require.NoError(t, verifyAppleSignedPKCS7(p7, appleDeviceIdentityCAs()))
	})
}

// TestVerifyMachineInfoSignatureAccountDrivenCapture verifies a real
// Account-Driven User Enrollment blob captured from a lab iPhone. Despite the
// design note that this lane was "not an Apple device signature", the blob is
// signed by the same Apple device identity as ADE/OTA — a per-device GUID leaf
// under the pinned Apple iPhone Device CA — so it verifies. The payload omits
// SERIAL/UDID (BYOD privacy), which the parser leaves empty.
func TestVerifyMachineInfoSignatureAccountDrivenCapture(t *testing.T) {
	buf, err := os.ReadFile("testdata/deviceinfo/machineinfo-account-driven-iphone.der")
	require.NoError(t, err)

	info, p7, err := ParseMachineInfoFromPKCS7(buf)
	require.NoError(t, err)
	require.Equal(t, "iPhone14,5", info.Product)
	require.Empty(t, info.Serial)
	require.Empty(t, info.UDID)

	require.NoError(t, VerifyMachineInfoSignature(p7))
}

// TestOTAPhase2FleetSignatureCapture verifies a real OTA phase-2 request body
// captured from a lab iPhone. Phase 2 is signed by the Fleet SCEP identity the
// device obtained in phase 1 (not the Apple device identity), so it's verified
// with smallstep's p7.Verify() — a standard CMS signature check — rather than
// the Apple device-CA verifier. This pins the behavior the OTA decoder relies
// on: a genuine Fleet-signed body verifies, and tampering the content breaks it.
func TestOTAPhase2FleetSignatureCapture(t *testing.T) {
	buf, err := os.ReadFile("testdata/deviceinfo/ota-phase2-iphone.der")
	require.NoError(t, err)

	p7, err := pkcs7.Parse(buf)
	require.NoError(t, err)

	signer := p7.GetOnlySigner()
	require.NotNil(t, signer)
	require.Equal(t, "Fleet Identity", signer.Subject.CommonName)

	// genuine Fleet-signed phase-2 body verifies (signature over the
	// authenticated attributes, plus messageDigest == hash(content))
	require.NoError(t, p7.Verify())

	// tampering the content breaks the signature
	tampered := *p7
	content := make([]byte, len(p7.Content))
	copy(content, p7.Content)
	content[len(content)/2] ^= 0xff
	tampered.Content = content
	require.Error(t, tampered.Verify())
}

func TestVerifyMachineInfoSignatureSyntheticChains(t *testing.T) {
	content := []byte("<plist><dict></dict></plist>")

	t.Run("leaf issued by an expired device CA verifies", func(t *testing.T) {
		// mirrors Apple's real chain: the device CA is long expired but still
		// issues current leaves, so validity windows must be ignored
		root := newTestCA(t, x509.SHA256WithRSA)
		deviceCA := newTestCert(t, root, x509.SHA256WithRSA, true, time.Now().Add(-24*time.Hour))
		leaf := newTestCert(t, deviceCA, x509.SHA256WithRSA, false, time.Now().Add(24*time.Hour))

		blob := signPKCS7(t, content, leaf, []*x509.Certificate{deviceCA.cert, root.cert}, true)
		p7 := parsePKCS7(t, blob)

		require.NoError(t, verifyAppleSignedPKCS7(p7, []*x509.Certificate{deviceCA.cert}))

		// the same blob must not verify against the real pinned device CA
		require.ErrorContains(t, VerifyMachineInfoSignature(p7), "is not issued by a pinned Apple device CA")
	})

	t.Run("developer-cert is rejected", func(t *testing.T) {
		// A cert which chains up to the root CA but not through the expected sub-CA must be rejected,
		// even though everything chains to the trusted root, because the path never passes through
		// a pinned device CA.
		root := newTestCA(t, x509.SHA256WithRSA)
		deviceCA := newTestCert(t, root, x509.SHA256WithRSA, true, time.Now().Add(-24*time.Hour))
		developerCA := newTestCert(t, root, x509.SHA256WithRSA, true, time.Now().Add(24*time.Hour))
		attackerLeaf := newTestCert(t, developerCA, x509.SHA256WithRSA, false, time.Now().Add(24*time.Hour))

		blob := signPKCS7(t, content, attackerLeaf, []*x509.Certificate{developerCA.cert, root.cert}, true)
		p7 := parsePKCS7(t, blob)

		// pin only the device CA
		err := verifyAppleSignedPKCS7(p7, []*x509.Certificate{deviceCA.cert})
		require.ErrorContains(t, err, "is not issued by a pinned Apple device CA")
	})

	t.Run("ECDSA chain verifies without panicking", func(t *testing.T) {
		deviceCA := newTestCA(t, x509.ECDSAWithSHA256)
		leaf := newTestCert(t, deviceCA, x509.ECDSAWithSHA256, false, time.Now().Add(24*time.Hour))

		blob := signPKCS7(t, content, leaf, nil, true)
		p7 := parsePKCS7(t, blob)

		require.NoError(t, verifyAppleSignedPKCS7(p7, []*x509.Certificate{deviceCA.cert}))
	})

	t.Run("signature without authenticated attributes verifies", func(t *testing.T) {
		deviceCA := newTestCA(t, x509.SHA256WithRSA)
		leaf := newTestCert(t, deviceCA, x509.SHA256WithRSA, false, time.Now().Add(24*time.Hour))

		blob := signPKCS7(t, content, leaf, nil, false)
		p7 := parsePKCS7(t, blob)

		require.NoError(t, verifyAppleSignedPKCS7(p7, []*x509.Certificate{deviceCA.cert}))
	})

	t.Run("self-signed signer is rejected", func(t *testing.T) {
		selfSigned := newTestCA(t, x509.SHA256WithRSA)

		blob := signPKCS7(t, content, selfSigned, nil, true)
		p7 := parsePKCS7(t, blob)

		err := verifyAppleSignedPKCS7(p7, []*x509.Certificate{newTestCA(t, x509.SHA256WithRSA).cert})
		require.ErrorContains(t, err, "is not issued by a pinned Apple device CA")
	})

	t.Run("non-CA issuer in chain is rejected", func(t *testing.T) {
		// a leaf (IsCA=false) acting as an issuer, e.g. an Apple-issued
		// developer certificate. Go's CreateCertificate refuses a non-CA parent,
		// so the leaf is issued via a CA twin sharing the non-CA cert's subject and
		// key; only the non-CA variant goes in the bag.
		deviceCA := newTestCA(t, x509.SHA256WithRSA)
		notCA := newTestCert(t, deviceCA, x509.SHA256WithRSA, false, time.Now().Add(24*time.Hour))
		caTwin := *notCA.cert
		caTwin.IsCA = true
		forgedLeaf := newTestCert(t, &testIdentity{cert: &caTwin, key: notCA.key}, x509.SHA256WithRSA, false, time.Now().Add(24*time.Hour))

		blob := signPKCS7(t, content, forgedLeaf, []*x509.Certificate{notCA.cert}, true)
		p7 := parsePKCS7(t, blob)

		err := verifyAppleSignedPKCS7(p7, []*x509.Certificate{deviceCA.cert})
		require.ErrorContains(t, err, "not a CA certificate")
	})

	t.Run("signature by a different key is rejected", func(t *testing.T) {
		deviceCA := newTestCA(t, x509.SHA256WithRSA)
		leaf := newTestCert(t, deviceCA, x509.SHA256WithRSA, false, time.Now().Add(24*time.Hour))
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		blob := signPKCS7(t, content, &testIdentity{cert: leaf.cert, key: otherKey}, nil, true)
		p7 := parsePKCS7(t, blob)

		err = verifyAppleSignedPKCS7(p7, []*x509.Certificate{deviceCA.cert})
		require.ErrorContains(t, err, "signature verification failed")
	})
}

func TestMachineInfoVerification(t *testing.T) {
	require.True(t, MachineInfoVerificationEnabled(), "verification is enforced by default")

	SetMachineInfoVerification(false)
	require.False(t, MachineInfoVerificationEnabled())
	SetMachineInfoVerification(true)
	require.True(t, MachineInfoVerificationEnabled())

	// SetMachineInfoVerificationForTest restores the prior value on cleanup.
	SetMachineInfoVerificationForTest(t, false)
	require.False(t, MachineInfoVerificationEnabled())
}

type testIdentity struct {
	cert *x509.Certificate
	key  crypto.PrivateKey
}

func newTestCA(t *testing.T, sigAlg x509.SignatureAlgorithm) *testIdentity {
	return newTestCert(t, nil, sigAlg, true, time.Now().Add(365*24*time.Hour))
}

func newTestCert(t *testing.T, parent *testIdentity, sigAlg x509.SignatureAlgorithm, isCA bool, notAfter time.Time) *testIdentity {
	t.Helper()

	var key crypto.PrivateKey
	var pub any
	switch sigAlg {
	case x509.ECDSAWithSHA256:
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		key, pub = k, &k.PublicKey
	default:
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		key, pub = k, &k.PublicKey
	}

	serial, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "test-cert-" + serial.String()},
		NotBefore:             time.Now().Add(-48 * time.Hour),
		NotAfter:              notAfter,
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	parentCert := tmpl
	signingKey := key
	if parent != nil {
		parentCert = parent.cert
		signingKey = parent.key
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, parentCert, pub, signingKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &testIdentity{cert: cert, key: key}
}

// signPKCS7 signs content as the given identity, including the provided chain
// certificates in the bag, with or without CMS authenticated attributes —
// Apple's deviceinfo blobs include them.
func signPKCS7(t *testing.T, content []byte, signer *testIdentity, chain []*x509.Certificate, withAttrs bool) []byte {
	t.Helper()

	signedData, err := pkcs7.NewSignedData(content)
	require.NoError(t, err)
	signedData.SetDigestAlgorithm(pkcs7.OIDDigestAlgorithmSHA256)

	if withAttrs {
		require.NoError(t, signedData.AddSigner(signer.cert, signer.key, pkcs7.SignerInfoConfig{}))
	} else {
		require.NoError(t, signedData.SignWithoutAttr(signer.cert, signer.key, pkcs7.SignerInfoConfig{}))
	}
	for _, c := range chain {
		signedData.AddCertificate(c)
	}

	blob, err := signedData.Finish()
	require.NoError(t, err)
	return blob
}

func parsePKCS7(t *testing.T, blob []byte) *pkcs7.PKCS7 {
	t.Helper()
	p7, err := pkcs7.Parse(blob)
	require.NoError(t, err)
	return p7
}
