package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/require"
)

func TestFileVaultPromptEnablementAt(t *testing.T) {
	cases := []struct {
		name string
		in   optjson.String
		want string
	}{
		{"unset", optjson.String{}, FileVaultPromptEnablementAtLogin},
		{"empty", optjson.SetString(""), FileVaultPromptEnablementAtLogin},
		{"explicit login", optjson.SetString("login"), FileVaultPromptEnablementAtLogin},
		{"explicit logout", optjson.SetString("logout"), FileVaultPromptEnablementAtLogout},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := MDM{FileVault: &MDMFileVaultSettings{PromptEnablementAt: c.in}}
			require.Equal(t, c.want, m.FileVaultPromptEnablementAt())

			tm := TeamMDM{FileVault: &MDMFileVaultSettings{PromptEnablementAt: c.in}}
			require.Equal(t, c.want, tm.FileVaultPromptEnablementAt())
		})
	}
}

func TestValidFileVaultPromptEnablementAt(t *testing.T) {
	for _, v := range []string{"", "login", "logout"} {
		require.True(t, ValidFileVaultPromptEnablementAt(v), "expected %q to be valid", v)
	}
	for _, v := range []string{"LOGIN", "logon", "setup", "true", "1"} {
		require.False(t, ValidFileVaultPromptEnablementAt(v), "expected %q to be invalid", v)
	}
}
