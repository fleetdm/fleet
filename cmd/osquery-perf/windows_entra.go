package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/MicahParks/jwkset"
)

// windowsEntraKeyID is the "kid" of the embedded signing key, in both the tokens and the published JWK set.
const windowsEntraKeyID = "osquery-perf-loadtest"

// windowsDevDetailSerialURI is the LocURI Fleet reads to link a Windows MDM enrollment to its host.
const windowsDevDetailSerialURI = "./DevDetail/Ext/Microsoft/SMBIOSSerialNumber"

// windowsEntraParams configures user-driven (Entra) Windows MDM enrollments.
type windowsEntraParams struct {
	// userDrivenProb is the fraction of Windows MDM agents that enroll user-driven [0, 1].
	userDrivenProb float64
	// tenantID is the token's tenant; Fleet only accepts tenants listed in windows_entra_tenant_ids.
	tenantID   string
	signingKey *rsa.PrivateKey
}

// Real Entra tokens are signed by Microsoft, so a Fleet server only accepts these when it runs in dev mode with
// FLEET_DEV_AZURE_JWT_JWKS_URI pointing at windows_entra_loadtest_jwks.json (the public half of this key).
//
//go:embed windows_entra_loadtest_key.pem
var windowsEntraKeyPEM []byte

func windowsEntraSigningKey() (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(windowsEntraKeyPEM)
	if block == nil {
		return nil, errors.New("windows entra signing key: no PEM block")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("windows entra signing key: %w", err)
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("windows entra signing key: got %T, want RSA", parsed)
	}
	return key, nil
}

// windowsEntraJWKS returns the public JWK set for the embedded key, as Fleet fetches it.
func windowsEntraJWKS(ctx context.Context, key *rsa.PrivateKey) ([]byte, error) {
	jwk, err := jwkset.NewJWKFromKey(key, jwkset.JWKOptions{Metadata: jwkset.JWKMetadataOptions{KID: windowsEntraKeyID}})
	if err != nil {
		return nil, err
	}
	set := jwkset.NewMemoryStorage()
	if err := set.KeyWrite(ctx, jwk); err != nil {
		return nil, err
	}
	return set.JSONPublic(ctx)
}
