package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEndUserSSOSettingsResolution(t *testing.T) {
	okta := MDMIdentityProvider{
		Name: "Okta",
		SSOProviderSettings: SSOProviderSettings{
			EntityID:    "https://okta.example",
			MetadataURL: "https://okta.example/metadata",
		},
		Default: true,
	}
	entra := MDMIdentityProvider{
		Name: "Entra",
		SSOProviderSettings: SSOProviderSettings{
			EntityID:    "https://entra.example",
			IDPName:     "Entra ID",
			MetadataURL: "https://entra.example/metadata",
		},
	}
	mdm := MDM{
		IdentityProviders: []MDMIdentityProvider{okta, entra},
		EndUserAuthentication: MDMEndUserAuthentication{SSOProviderSettings: SSOProviderSettings{
			EntityID: "legacy", IDPName: "Legacy", Metadata: "<xml/>",
		}},
	}

	t.Run("fleet name selects that connection", func(t *testing.T) {
		got, err := mdm.EndUserSSOSettings("Entra")
		require.NoError(t, err)
		require.Equal(t, "Entra ID", got.IDPName)
		require.Equal(t, "https://entra.example", got.EntityID)
	})

	t.Run("empty name uses the default and fills idp_name", func(t *testing.T) {
		got, err := mdm.EndUserSSOSettings("")
		require.NoError(t, err)
		require.Equal(t, "Okta", got.IDPName)
		require.Equal(t, "https://okta.example", got.EntityID)
	})

	t.Run("unknown name does not fall through", func(t *testing.T) {
		_, err := mdm.EndUserSSOSettings("Missing")
		require.ErrorContains(t, err, "not configured")
		require.False(t, mdm.EndUserAuthAvailable("Missing"))
	})

	t.Run("no connections uses legacy settings", func(t *testing.T) {
		legacy := MDM{EndUserAuthentication: mdm.EndUserAuthentication}
		got, err := legacy.EndUserSSOSettings("")
		require.NoError(t, err)
		require.Equal(t, "legacy", got.EntityID)
		require.True(t, legacy.EndUserAuthAvailable(""))
	})

	t.Run("empty org and no legacy is unavailable", func(t *testing.T) {
		empty := MDM{}
		_, err := empty.EndUserSSOSettings("")
		require.Error(t, err)
		require.False(t, empty.EndUserAuthAvailable(""))
	})
}

func TestNormalizeAndValidateIdentityProviders(t *testing.T) {
	t.Run("fills idp name and accepts one default", func(t *testing.T) {
		mdm := MDM{IdentityProviders: []MDMIdentityProvider{{
			Name:    " Okta ",
			Default: true,
			SSOProviderSettings: SSOProviderSettings{
				EntityID:    " https://okta.example ",
				MetadataURL: "https://okta.example/metadata",
			},
		}}}
		invalid := &InvalidArgumentError{}
		mdm.NormalizeAndValidateIdentityProviders(invalid)
		require.False(t, invalid.HasErrors())
		require.Equal(t, "Okta", mdm.IdentityProviders[0].Name)
		require.Equal(t, "Okta", mdm.IdentityProviders[0].IDPName)
		require.Equal(t, "https://okta.example", mdm.IdentityProviders[0].EntityID)
	})

	t.Run("rejects two defaults and a duplicate name", func(t *testing.T) {
		mdm := MDM{IdentityProviders: []MDMIdentityProvider{
			{Name: "Okta", Default: true, SSOProviderSettings: SSOProviderSettings{EntityID: "a", MetadataURL: "https://a.example"}},
			{Name: "Okta", Default: true, SSOProviderSettings: SSOProviderSettings{EntityID: "b", MetadataURL: "https://b.example"}},
		}}
		invalid := &InvalidArgumentError{}
		mdm.NormalizeAndValidateIdentityProviders(invalid)
		require.True(t, invalid.HasErrors())
		reasons := invalid.Error()
		for _, item := range invalid.Invalid() {
			reasons += item["reason"]
		}
		require.Contains(t, reasons, "duplicate name")
		require.Contains(t, reasons, "only one identity provider")
	})
}

func TestSSOFleetRef(t *testing.T) {
	ref := SSOFleetRef("secret", 42)
	id, ok := ParseSSOFleetRef("secret", ref)
	require.True(t, ok)
	require.Equal(t, uint(42), id)

	_, ok = ParseSSOFleetRef("other", ref)
	require.False(t, ok)

	unassigned := SSOFleetRef("secret", 0)
	id, ok = ParseSSOFleetRef("secret", unassigned)
	require.True(t, ok)
	require.Equal(t, uint(0), id)

	_, ok = ParseSSOFleetRef("secret", "not-a-ref")
	require.False(t, ok)
}
