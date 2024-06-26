package cryptoutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"testing"
)

func TestGenerateSubjectKeyID(t *testing.T) {
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		testName string
		pub      crypto.PublicKey
	}{
		{"RSA", &rsa.PublicKey{N: big.NewInt(123), E: 65537}},
		{"ECDSA", &ecdsa.PublicKey{X: ecdsaKey.X, Y: ecdsaKey.Y, Curve: elliptic.P224()}},
	} {
		test := test
		t.Run(test.testName, func(t *testing.T) {
			t.Parallel()
			ski, err := GenerateSubjectKeyID(test.pub)
			if err != nil {
				t.Fatal(err)
			}
			if len(ski) != 20 {
				t.Fatalf("unexpected subject public key identifier length: %d", len(ski))
			}
			ski2, err := GenerateSubjectKeyID(test.pub)
			if err != nil {
				t.Fatal(err)
			}
			if !testSKIEq(ski, ski2) {
				t.Fatal("subject key identifier generation is not deterministic")
			}
		})
	}
}

func testSKIEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
