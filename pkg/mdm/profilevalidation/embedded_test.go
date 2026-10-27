package profilevalidation

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedSchema checks the committed outputs/profile_schema.json against what the generator
// must produce from Apple's release (the ticket's conditions of satisfaction), so a bad regeneration
// fails here rather than at server startup.
func TestEmbeddedSchema(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)

	assert.Equal(t, FormatVersion, s.FormatVersion)
	assert.Regexp(t, `^[0-9a-f]{40}$`, s.Version.Apple.Commit)
	assert.NotEmpty(t, s.Version.Apple.Release)
	assert.False(t, s.Version.GeneratedAt.IsZero())

	var apple, allowed int
	for _, pt := range s.AppleProfiles {
		if pt.Source == "apple" {
			apple++
		}
		// Counted by reason, not source: once Apple ships an allowed type, the entry becomes
		// Apple's but keeps the allow entry's reason.
		if pt.Override != "" {
			allowed++
		}
	}
	assert.Equal(t, 20, allowed, "seeded allow entries")
	// The ticket's counts are for Release-v27.0. Later releases are covered by Parse's floors, so the
	// daily update PR doesn't fail just because Apple added a type.
	if s.Version.Apple.Release == "Release-v27.0" {
		assert.Equal(t, 119, apple, "Apple payload types")
		assert.Len(t, s.AppleDeclarations, 59)
	}

	t.Run("envelope kept apart from payload types", func(t *testing.T) {
		require.NotNil(t, s.AppleEnvelope.TopLevel)
		require.NotNil(t, s.AppleEnvelope.CommonPayloadKeys)
		assert.NotNil(t, s.AppleEnvelope.TopLevel.Keys["PayloadContent"])
		assert.NotNil(t, s.AppleEnvelope.CommonPayloadKeys.Keys["PayloadUUID"])
		assert.NotContains(t, s.AppleProfiles, "TopLevel")
		assert.NotContains(t, s.AppleProfiles, "CommonPayloadKeys")
	})

	t.Run("one entry per payload type", func(t *testing.T) {
		mcx := s.AppleProfile("com.apple.MCX")
		require.NotNil(t, mcx)
		// Keys from different MCX(...) files land in the one entry.
		assert.NotNil(t, mcx.Keys["timeServer"])
		assert.NotNil(t, mcx.Keys["com.apple.EnergySaver.portable.BatteryPower"])
		sso := s.AppleProfile("com.apple.extensiblesso")
		require.NotNil(t, sso)
		// The general file's looser rangelist wins over the Kerberos variant's.
		assert.ElementsMatch(t, []any{"Credential", "Redirect"}, sso.Keys["Type"].RangeList)
		assert.NotNil(t, s.AppleProfile("loginwindow"), "loginwindow.yaml is its own type")
		assert.NotNil(t, s.AppleProfile("com.apple.loginwindow"))
	})

	t.Run("recursive types use ref", func(t *testing.T) {
		home := s.AppleProfile("com.apple.homescreenlayout")
		require.NotNil(t, home)
		assert.Equal(t, "Dock[]", home.Keys["Dock"].Items.Keys["Pages"].Items.Items.Ref)
		assert.Equal(t, "allowList[]", s.AppleProfile("com.apple.applicationaccess.new").Keys["allowList"].Items.Keys["subApps"].Items.Ref)
		bookmarks := s.AppleDeclaration("com.apple.configuration.safari.bookmarks")
		require.NotNil(t, bookmarks)
		assert.Contains(t, string(mustJSON(t, bookmarks)), `"ref":"ManagedBookmarks[].Bookmarks[].Folder"`)
	})

	t.Run("constraints where upstream has them", func(t *testing.T) {
		format := s.AppleProfile("com.apple.eas.account").Keys["PayloadCertificateUUID"].Format
		require.NotEmpty(t, format)
		assert.Regexp(t, format, "AB12CD34-AB12-AB12-AB12-AB12CD34EF56")
		assert.Equal(t, []any{"Application", "Folder", "WebClip"}, s.AppleProfile("com.apple.homescreenlayout").Keys["Dock"].Items.Keys["Type"].RangeList)
		airPrint := s.AppleProfile("com.apple.applicationaccess").Keys["allowAirPrint"]
		assert.True(t, airPrint.Platforms[AppleOSiOS].Supervised)
		assert.Equal(t, "n/a", airPrint.Platforms[AppleOSmacOS].Introduced)
		delay := s.AppleProfile("com.apple.applicationaccess").Keys["enforcedSoftwareUpdateDelay"].Range
		assert.Equal(t, &Range{Min: new(1.0), Max: new(90.0)}, delay)
	})

	t.Run("declarations", func(t *testing.T) {
		require.NotNil(t, s.AppleDeclarationEnvelope)
		assert.NotContains(t, s.AppleDeclarationEnvelope.Keys, "ServerToken", "set by Fleet's server, never validated")
		assert.NotNil(t, s.AppleDeclarationEnvelope.Keys["Identifier"])
		passcode := s.AppleDeclaration("com.apple.configuration.passcode.settings")
		require.NotNil(t, passcode)
		assert.Equal(t, &Range{Min: new(0.0), Max: new(16.0)}, passcode.Keys["MinimumLength"].Range)
		assert.Nil(t, s.AppleProfile("com.apple.configuration.passcode.settings"), "profiles and declarations don't mix")
	})

	t.Run("overrides are applied and marked", func(t *testing.T) {
		chrome := s.AppleProfile("com.google.Chrome")
		require.NotNil(t, chrome)
		assert.Equal(t, "override", chrome.Source)
		assert.True(t, chrome.Open)
		assert.NotEmpty(t, chrome.Override)
		enabled := s.AppleEnvelope.CommonPayloadKeys.Keys["PayloadEnabled"]
		require.NotNil(t, enabled)
		assert.NotEmpty(t, enabled.Override)
		assert.Equal(t, []any{"System", "User"}, s.AppleDeclarationEnvelope.Keys["PayloadScope"].RangeList)
	})
}

func TestEmbeddedSchemaWindows(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)

	v := s.Version.Windows
	assert.Regexp(t, `^DDFv2\w+$`, v.Release)
	assert.Regexp(t, `^https://download\.microsoft\.com/download/.+/`+v.Release+`\.zip$`, v.URL)
	assert.Regexp(t, `^[0-9a-f]{64}$`, v.SHA256)

	tv := s.Version.WindowsTemplates
	assert.Regexp(t, `^[0-9a-f]{64}$`, tv.SHA256)
	assert.NotEmpty(t, tv.GeneratedAt)

	var settings, interior, templates int
	for locURI, n := range s.Windows {
		if n.XSD != "" {
			assert.NotEmpty(t, s.WindowsXSDs[n.XSD], "%s references a missing xsd", locURI)
		}
		switch {
		case n.Source == WindowsSourceADMXTemplate:
			templates++
		case n.Source != WindowsSourceWindows:
			t.Errorf("%s has unexpected source %q", locURI, n.Source)
		case slices.Equal(n.Format, WindowsFormats{WindowsFormatNode}):
			interior++
		default:
			settings++
		}
	}
	// Exact counts are for the drop the ticket was checked against; later drops are covered by
	// Parse's floor, so the daily update PR doesn't fail just because Microsoft added nodes.
	if v.Release == "DDFv2Feb2026" {
		assert.Equal(t, "bf667d895af4a8c8ab5a31065ce0e28ea2f8b649c4dc416f452f62fd1c42ff14", v.SHA256)
		assert.Equal(t, 5146, settings)
		assert.Equal(t, 809, interior)
		assert.Len(t, s.WindowsXSDs, 7)
	}
	// 1,575 policies, written once per scope they apply to.
	if tv.SHA256 == "c834b85a777dcbbcf66943daa97012c0d6da9780fd8f71176c81071e711f5559" {
		assert.Equal(t, 2570, templates)
	}

	node := func(locURI string) *WindowsNode {
		n := s.WindowsNode(locURI)
		require.NotNil(t, n, locURI)
		return n
	}
	int1 := WindowsFormats{WindowsFormatInt}
	chr := WindowsFormats{WindowsFormatChr}

	t.Run("ticket examples", func(t *testing.T) {
		assert.Equal(t, &WindowsNode{Format: int1, Scope: WindowsScopeDevice, Source: WindowsSourceWindows, Range: &Range{Min: new(0.0), Max: new(999.0)}, MinBuild: []string{"10.0.10240"}},
			node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock"))
		assert.Equal(t, `^(\d\.)?(\d)$`, node("./Device/Vendor/MSFT/DMClient/Provider/*/SyncApplicationVersion").RegEx)
		assert.Equal(t, []string{"0x0", "0x1", "0x2", "0x4"}, node("./Device/Vendor/MSFT/DMClient/Provider/*/FirstSyncStatus/BlockInStatusPage").Flag)
		configure := node("./Device/Vendor/MSFT/Policy/Config/LocalUsersAndGroups/Configure")
		assert.Equal(t, chr, configure.Format)
		assert.Contains(t, s.WindowsXSDs[configure.XSD], "xs:schema")
		assert.True(t, node("./Device/Vendor/MSFT/Policy/Config/RemoteAssistance/SolicitedRemoteAssistance").ADMXBacked)
		assert.Equal(t, []string{"0", "1"}, node("./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type").Enum)
	})

	t.Run("dynamic instances and interior nodes", func(t *testing.T) {
		assert.Same(t, s.Windows["./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type"],
			node("./Vendor/MSFT/Firewall/MdmStore/FirewallRules/BlockTelnet/Action/Type"))
		// Profiles <Add> the instance before setting its children, as Fleet's SCEP profiles do.
		scep := node("./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID")
		assert.Equal(t, WindowsFormats{WindowsFormatNode}, scep.Format)
		// On an interior "*" node the regex constrains the instance's name.
		assert.Equal(t, "^[^|/]*$", node("./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*").RegEx)
	})

	t.Run("ADMX template policies", func(t *testing.T) {
		const homepage = "Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation"
		assert.Equal(t, &WindowsNode{Format: chr, Scope: WindowsScopeDevice, Source: WindowsSourceADMXTemplate}, node("./Device/"+homepage))
		assert.Equal(t, &WindowsNode{Format: chr, Scope: WindowsScopeUser, Source: WindowsSourceADMXTemplate}, node("./User/"+homepage))
		// Paths use the profile generator's AppNames. Fleet's docs install Chrome as "Chrome", which
		// only reaches the DDF's generic Policy/Config/*/* node: #55792's matcher has to handle it.
		docs := node("./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotification")
		assert.Equal(t, WindowsSourceWindows, docs.Source)
		assert.Equal(t, WindowsFormats{WindowsFormatNull}, docs.Format)
	})

	t.Run("scope, builds, and deprecation", func(t *testing.T) {
		assert.Equal(t, WindowsScopeUser, node("./User/Vendor/MSFT/Policy/Config/Education/PrinterNames").Scope)
		// Applicability inherited from an ancestor.
		assert.Equal(t, []string{"10.0.10240"}, node("./Device/Vendor/MSFT/DMClient/Provider/*/SyncApplicationVersion").MinBuild)
		var lists, deprecated int
		for _, n := range s.Windows {
			if len(n.MinBuild) > 1 {
				lists++
			}
			if n.Deprecated != nil {
				deprecated++
			}
		}
		assert.Greater(t, lists, 1000, "backported builds are kept as lists")
		assert.NotZero(t, deprecated)
	})
}

func mustJSON(t *testing.T, pt *ApplePayloadType) []byte {
	t.Helper()
	b, err := json.Marshal(pt)
	require.NoError(t, err)
	return b
}
