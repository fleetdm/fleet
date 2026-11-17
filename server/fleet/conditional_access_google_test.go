package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGoogleConditionalAccessIntegrationValidate(t *testing.T) {
	errNames := func(invalid *InvalidArgumentError) []string {
		var names []string
		for _, e := range invalid.Errors {
			names = append(names, e.name)
		}
		return names
	}

	t.Run("both fields are required", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		(&GoogleConditionalAccessIntegration{}).Validate(invalid)
		require.ElementsMatch(t, []string{"customer_id", "admin_email"}, errNames(invalid))
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
			(&GoogleConditionalAccessIntegration{CustomerID: tc.in, AdminEmail: "admin@example.com"}).Validate(invalid)
			require.Equal(t, !tc.valid, invalid.HasErrors(), tc.in)
		}
	})

	t.Run("trims", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		intg := &GoogleConditionalAccessIntegration{CustomerID: " C02cajzvg", AdminEmail: " admin@example.com "}
		intg.Validate(invalid)
		require.False(t, invalid.HasErrors())
		require.Equal(t, "C02cajzvg", intg.CustomerID)
		require.Equal(t, "admin@example.com", intg.AdminEmail)
	})

	t.Run("malformed admin email", func(t *testing.T) {
		for _, email := range []string{"Admin <admin@example.com>", "admin"} {
			invalid := &InvalidArgumentError{}
			(&GoogleConditionalAccessIntegration{CustomerID: "C02cajzvg", AdminEmail: email}).Validate(invalid)
			require.Equal(t, []string{"admin_email"}, errNames(invalid), email)
		}
	})
}
