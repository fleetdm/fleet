package fleet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateGoogleConditionalAccessSettings(t *testing.T) {
	errNames := func(invalid *InvalidArgumentError) []string {
		var names []string
		for _, e := range invalid.Errors {
			names = append(names, e.name)
		}
		return names
	}

	t.Run("enabled requires both fields", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		ValidateGoogleConditionalAccessSettings(&GoogleConditionalAccessIntegration{Enabled: true}, "p.", invalid)
		require.ElementsMatch(t, []string{"p.customer_id", "p.admin_email"}, errNames(invalid))
	})

	t.Run("disabled and empty is valid", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		ValidateGoogleConditionalAccessSettings(&GoogleConditionalAccessIntegration{}, "p.", invalid)
		require.False(t, invalid.HasErrors())
	})

	t.Run("customer ID must start with C", func(t *testing.T) {
		for _, tc := range []struct {
			in    string
			valid bool
		}{
			{"C02cajzvg", true},
			{"Cabcdefgh", true},
			{"02cajzvg", false},
			{"c02cajzvg", false},
			{"C", false},
			{"C0123", false},
			{"C02-cajzvg", false},
			{"customers/C02cajzvg", false},
		} {
			invalid := &InvalidArgumentError{}
			ValidateGoogleConditionalAccessSettings(&GoogleConditionalAccessIntegration{CustomerID: tc.in}, "p.", invalid)
			require.Equal(t, !tc.valid, invalid.HasErrors(), tc.in)
		}
	})

	t.Run("trims", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		intg := &GoogleConditionalAccessIntegration{Enabled: true, CustomerID: " C02cajzvg", AdminEmail: " admin@example.com "}
		ValidateGoogleConditionalAccessSettings(intg, "p.", invalid)
		require.False(t, invalid.HasErrors())
		require.Equal(t, "C02cajzvg", intg.CustomerID)
		require.Equal(t, "admin@example.com", intg.AdminEmail)
	})

	t.Run("malformed values are rejected even when disabled", func(t *testing.T) {
		for _, intg := range []*GoogleConditionalAccessIntegration{
			{CustomerID: "C0-1"},
			{AdminEmail: "Admin <admin@example.com>"},
			{AdminEmail: "admin"},
		} {
			invalid := &InvalidArgumentError{}
			ValidateGoogleConditionalAccessSettings(intg, "p.", invalid)
			require.True(t, invalid.HasErrors(), "%+v", intg)
		}
	})
}

func TestValidateGoogleServiceAccountKey(t *testing.T) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pkcs8DER, err := x509.MarshalPKCS8PrivateKey(rsaKey)
	require.NoError(t, err)
	pkcs8PEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8DER}))
	pkcs1PEM := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaKey)}))
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	ecDER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	ecPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: ecDER}))

	keyJSON := func(fields map[string]any) string {
		b, err := json.Marshal(fields)
		require.NoError(t, err)
		return string(b)
	}
	key := func(privateKey any) string {
		return keyJSON(map[string]any{"type": "service_account", "client_email": "sa@example.com", "private_key": privateKey})
	}

	for _, tc := range []struct {
		name      string
		key       string
		wantNames []string
	}{
		{"valid PKCS8", key(pkcs8PEM), nil},
		{"valid PKCS1", key(pkcs1PEM), nil},
		{"missing", ``, []string{"key"}},
		{"null", `null`, []string{"key"}},
		{"not an object", `"abc"`, []string{"key"}},
		{"wrong type", keyJSON(map[string]any{"type": "authorized_user", "client_email": "sa@example.com", "private_key": pkcs8PEM}), []string{"key.type"}},
		{"no type", keyJSON(map[string]any{"client_email": "sa@example.com", "private_key": pkcs8PEM}), []string{"key.type"}},
		{"no private key", keyJSON(map[string]any{"type": "service_account", "client_email": "sa@example.com"}), []string{"key.private_key"}},
		{"blank fields", keyJSON(map[string]any{"type": "service_account", "client_email": " ", "private_key": ""}), []string{"key.client_email", "key.private_key"}},
		{"non-string field", keyJSON(map[string]any{"type": "service_account", "client_email": 1, "private_key": pkcs8PEM}), []string{"key.client_email"}},
		{"private key not PEM", key("not a key"), []string{"key.private_key"}},
		{"private key not RSA", key(ecPEM), []string{"key.private_key"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := &InvalidArgumentError{}
			ValidateGoogleServiceAccountKey(json.RawMessage(tc.key), "key", invalid)
			var names []string
			for _, e := range invalid.Errors {
				names = append(names, e.name)
			}
			require.Equal(t, tc.wantNames, names)
		})
	}
}
