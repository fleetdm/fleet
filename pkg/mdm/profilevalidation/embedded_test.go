package profilevalidation

import (
	"encoding/json"
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
	assert.Empty(t, s.Windows)
	assert.Empty(t, s.WindowsXSDs)

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

func mustJSON(t *testing.T, pt *ApplePayloadType) []byte {
	t.Helper()
	b, err := json.Marshal(pt)
	require.NoError(t, err)
	return b
}
