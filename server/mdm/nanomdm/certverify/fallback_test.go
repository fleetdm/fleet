package certverify

import (
	"context"
	"crypto/x509"
	"errors"
	"testing"
)

type errVerifier struct{ err error }

func (v *errVerifier) Verify(_ context.Context, _ *x509.Certificate) error {
	return v.err
}

var nilErroringVerifier = &errVerifier{}
var errErroringVerifier = &errVerifier{err: errors.New("verifier error")}

func TestFallbackVerifier(t *testing.T) {
	v := NewFallbackVerifier(nilErroringVerifier)
	err := v.Verify(context.Background(), nil)
	if err != nil {
		t.Errorf("should not have errored: %v", err)
	}

	v = NewFallbackVerifier(nilErroringVerifier, nilErroringVerifier)
	if err = v.Verify(context.Background(), nil); err != nil {
		t.Errorf("should not have errored: %v", err)
	}

	v = NewFallbackVerifier(errErroringVerifier)
	if err = v.Verify(context.Background(), nil); err == nil {
		t.Error("should have errored")
	}

	v = NewFallbackVerifier(errErroringVerifier, nilErroringVerifier)
	if err = v.Verify(context.Background(), nil); err != nil {
		t.Errorf("should not have errored: %v", err)
	}

	v = NewFallbackVerifier(nilErroringVerifier, errErroringVerifier)
	if err = v.Verify(context.Background(), nil); err != nil {
		t.Errorf("should not have errored: %v", err)
	}

	v = NewFallbackVerifier(errErroringVerifier, errErroringVerifier)
	if err = v.Verify(context.Background(), nil); err == nil {
		t.Error("should have errored")
	}
}
