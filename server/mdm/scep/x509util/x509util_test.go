package x509util

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"testing"
)

func TestCreateCertificateRequest(t *testing.T) {
	r := rand.Reader
	priv, err := rsa.GenerateKey(r, 1024) // nolint:gosec
	if err != nil {
		t.Fatal(err)
	}

	template := CertificateRequest{
		CertificateRequest: x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: "test.acme.co",
				Country:    []string{"US"},
			},
		},
		ChallengePassword: "foobar",
	}

	derBytes, err := CreateCertificateRequest(r, &template, priv)
	if err != nil {
		t.Fatal(err)
	}

	out, err := x509.ParseCertificateRequest(derBytes)
	if err != nil {
		t.Fatalf("failed to create certificate request: %s", err)
	}

	if err := out.CheckSignature(); err != nil {
		t.Errorf("failed to check certificate request signature: %s", err)
	}

	challenge, err := ParseChallengePassword(derBytes)
	if err != nil {
		t.Fatalf("failed to parse challengePassword attribute: %s", err)
	}

	if have, want := challenge, template.ChallengePassword; have != want {
		t.Errorf("have %s, want %s", have, want)
	}
}
