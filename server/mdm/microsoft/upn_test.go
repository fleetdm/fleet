package microsoft_mdm

import (
	"strings"
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

func TestIsValidEntraUPN(t *testing.T) {
	for _, upn := range []string{
		"user@example.com",
		"o'brien@example.com",
		"first.last@example.com",
	} {
		require.True(t, IsValidEntraUPN(upn), upn)
	}
	for _, upn := range []string{
		"user+tag@example.com",
		"user%40@example.com",
		"user.@example.com",
		strings.Repeat("a", 65) + "@example.com",   // local part over 64
		"user@" + strings.Repeat("b", 45) + ".com", // domain over 48
		"DESKTOP-ABC",
		"",
	} {
		require.False(t, IsValidEntraUPN(upn), upn)
	}
	require.True(t, IsValidEntraUPN(strings.Repeat("a", 64)+"@"+strings.Repeat("b", 44)+".com")) // 64 and 48, 113 total
}
