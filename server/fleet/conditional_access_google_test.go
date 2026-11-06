package fleet

import (
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
