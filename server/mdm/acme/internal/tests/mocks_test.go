package tests

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/mdm/acme"
)

// Mock implementations for dependencies outside the bounded context

// mockDataProviders implements acme.DataProviders for testing.
type mockDataProviders struct {
	serverURL string
	assets    map[string][]byte // asset name → PEM bytes
	signer    acme.CSRSigner
}

func newMockDataProviders(serverURL string, signer acme.CSRSigner, caCertPEM []byte) *mockDataProviders {
	return &mockDataProviders{
		serverURL: serverURL,
		signer:    signer,
		assets:    map[string][]byte{"ca_cert": caCertPEM},
	}
}

func (m *mockDataProviders) ServerURL(_ context.Context) (string, error) {
	return m.serverURL, nil
}

func (m *mockDataProviders) GetCACertificatePEM(_ context.Context) ([]byte, error) {
	if pem, ok := m.assets["ca_cert"]; ok {
		return pem, nil
	}
	return nil, errors.New("ca_cert not found")
}

func (m *mockDataProviders) CSRSigner(_ context.Context) (acme.CSRSigner, error) {
	return m.signer, nil
}

// Returns true for "valid-serial", error for "error-serial", false otherwise
func (m *mockDataProviders) IsDEPEnrolled(_ context.Context, serial string) (bool, error) {
	if serial == "valid-serial" {
		return true, nil
	} else if serial == "error-serial" {
		return false, errors.New("Mocked error for IsDEPEnrolled")
	}
	return false, nil
}
