package microsoft_mdm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidUPN(t *testing.T) {
	valid := []string{
		"user@example.com",
		"first.last@sub.example.co.uk",
		"o'brien@example.com",
		"user!name#tag^x~y@example.com",
		"user_name-1@example.org",
	}
	for _, upn := range valid {
		require.True(t, IsValidUPN(upn), upn)
	}
	invalid := []string{
		"",
		"DESKTOP-ABC",
		"user@",
		"@example.com",
		"user@example",
		"user name@example.com",
		"user@exa mple.com",
		"a1b2c3d4e5f6g7h8i9j0",
	}
	for _, upn := range invalid {
		require.False(t, IsValidUPN(upn), upn)
	}
}
