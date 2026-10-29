package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaintainedAppArchSiblingSlug(t *testing.T) {
	cases := map[string]string{
		"google-chrome/windows":         "google-chrome-arm64/windows",
		"google-chrome-arm64/windows":   "google-chrome/windows",
		"firefox@nightly-arm64/windows": "firefox@nightly/windows",
		"google-chrome/darwin":          "",
		"-arm64/windows":                "",
		"google-chrome":                 "",
		"":                              "",
	}
	for slug, want := range cases {
		require.Equal(t, want, MaintainedAppArchSiblingSlug(slug), slug)
	}

	require.True(t, MaintainedAppsAreArchSiblings("slack/windows", "slack-arm64/windows"))
	require.True(t, MaintainedAppsAreArchSiblings("slack-arm64/windows", "slack/windows"))
	require.False(t, MaintainedAppsAreArchSiblings("slack/windows", "slack/windows"))
	require.False(t, MaintainedAppsAreArchSiblings("slack/windows", "zoom-arm64/windows"))
	require.False(t, MaintainedAppsAreArchSiblings("slack/darwin", "slack-arm64/darwin"))
	require.False(t, MaintainedAppsAreArchSiblings("", ""))

	require.True(t, MaintainedAppSlugIsARM64("slack-arm64/windows"))
	require.False(t, MaintainedAppSlugIsARM64("slack/windows"))
	require.False(t, MaintainedAppSlugIsARM64("slack-arm64/darwin"))
}

func TestMaintainedAppArchFitForHost(t *testing.T) {
	armHost := &Host{Platform: "windows", CPUType: "ARM"}
	x64Host := &Host{Platform: "windows", CPUType: "x86_64"}
	unknownHost := &Host{Platform: "windows"}
	macHost := &Host{Platform: "darwin", CPUType: "arm64e"}

	require.True(t, armHost.IsWindowsARM())
	require.True(t, (&Host{Platform: "windows", CPUType: "arm64"}).IsWindowsARM())
	require.False(t, x64Host.IsWindowsARM())
	require.False(t, unknownHost.IsWindowsARM())
	require.False(t, macHost.IsWindowsARM())

	cases := []struct {
		host           *Host
		slug           string
		hasArchSibling bool
		want           MaintainedAppArchFit
	}{
		{armHost, "slack-arm64/windows", true, MaintainedAppArchNative},
		{armHost, "slack/windows", true, MaintainedAppArchEmulated},
		{x64Host, "slack/windows", true, MaintainedAppArchNative},
		{x64Host, "slack-arm64/windows", true, MaintainedAppArchIncompatible},
		// A host that hasn't reported its CPU counts as x64.
		{unknownHost, "slack-arm64/windows", true, MaintainedAppArchIncompatible},
		{unknownHost, "slack/windows", true, MaintainedAppArchNative},
		// An ARM64 build that's its title's only build still can't run on other hosts.
		{x64Host, "slack-arm64/windows", false, MaintainedAppArchIncompatible},
		{unknownHost, "slack-arm64/windows", false, MaintainedAppArchIncompatible},
		{armHost, "slack-arm64/windows", false, MaintainedAppArchNative},
		// An x64 build that's its title's only build is what an ARM host gets.
		{armHost, "slack/windows", false, MaintainedAppArchNative},
		// Not a Windows FMA.
		{armHost, "", true, MaintainedAppArchNative},
		{macHost, "slack/darwin", true, MaintainedAppArchNative},
	}
	for _, c := range cases {
		require.Equal(t, c.want, MaintainedAppArchFitForHost(c.host, c.slug, c.hasArchSibling), "%s on %+v", c.slug, c.host)
	}
}
