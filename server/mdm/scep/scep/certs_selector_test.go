package scep

import (
	"crypto"
	_ "crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"testing"
)

func TestFingerprintCertsSelector(t *testing.T) {
	for _, test := range []struct {
		testName      string
		hashType      crypto.Hash
		hash          string
		certRaw       []byte
		expectedCount int
	}{
		{
			"null SHA-256 hash",
			crypto.SHA256,
			"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			nil,
			1,
		},
		{
			"3 byte SHA-256 hash",
			crypto.SHA256,
			"039058c6f2c0cb492c533b0a4d14ef77cc0f78abccced5287d84a1a2011cfb81",
			[]byte{1, 2, 3},
			1,
		},
		{
			"mismatched hash",
			crypto.SHA256,
			"8db07061ebb4cd0b0cd00825b363e5fb7f8131d8ff2c1fd70d03fa4fd6dc3785",
			[]byte{4, 5, 6},
			0,
		},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()

			fakeCerts := []*x509.Certificate{{Raw: test.certRaw}}

			hash, err := hex.DecodeString(test.hash)
			if err != nil {
				t.Fatal(err)
			}
			if want, have := test.hashType.Size(), len(hash); want != have {
				t.Errorf("invalid input hash length, want: %d have: %d", want, have)
			}

			selected := FingerprintCertsSelector(test.hashType, hash).SelectCerts(fakeCerts)

			if want, have := test.expectedCount, len(selected); want != have {
				t.Errorf("wrong selected certs count, want: %d have: %d", want, have)
			}
		})
	}
}

func TestEnciphermentCertsSelector(t *testing.T) {
	for _, test := range []struct {
		testName              string
		certs                 []*x509.Certificate
		expectedSelectedCerts []*x509.Certificate
	}{
		{
			"empty certificates list",
			[]*x509.Certificate{},
			[]*x509.Certificate{},
		},
		{
			"non-empty certificates list",
			[]*x509.Certificate{
				{KeyUsage: x509.KeyUsageKeyEncipherment},
				{KeyUsage: x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageDigitalSignature},
				{},
			},
			[]*x509.Certificate{
				{KeyUsage: x509.KeyUsageKeyEncipherment},
				{KeyUsage: x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment},
			},
		},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()

			selected := EnciphermentCertsSelector().SelectCerts(test.certs)
			if !certsKeyUsagesEq(selected, test.expectedSelectedCerts) {
				t.Fatal("selected and expected certificates did not match")
			}
		})
	}
}

func TestNopCertsSelector(t *testing.T) {
	for _, test := range []struct {
		testName              string
		certs                 []*x509.Certificate
		expectedSelectedCerts []*x509.Certificate
	}{
		{
			"empty certificates list",
			[]*x509.Certificate{},
			[]*x509.Certificate{},
		},
		{
			"non-empty certificates list",
			[]*x509.Certificate{
				{KeyUsage: x509.KeyUsageKeyEncipherment},
				{KeyUsage: x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageDigitalSignature},
				{},
			},
			[]*x509.Certificate{
				{KeyUsage: x509.KeyUsageKeyEncipherment},
				{KeyUsage: x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment},
				{KeyUsage: x509.KeyUsageDigitalSignature},
				{},
			},
		},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()

			selected := NopCertsSelector().SelectCerts(test.certs)
			if !certsKeyUsagesEq(selected, test.expectedSelectedCerts) {
				t.Fatal("selected and expected certificates did not match")
			}
		})
	}
}

// certsKeyUsagesEq returns true if certs in a have the same key usages
// of certs in b and in the same order.
func certsKeyUsagesEq(a []*x509.Certificate, b []*x509.Certificate) bool {
	if len(a) != len(b) {
		return false
	}
	for i, cert := range a {
		if cert.KeyUsage != b[i].KeyUsage {
			return false
		}
	}
	return true
}
