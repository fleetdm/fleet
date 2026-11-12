package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/require"
)

func TestWithoutEchoedProfileSpecs(t *testing.T) {
	recorded := []MDMProfileSpec{{Path: "lib/a.mobileconfig"}}
	cases := []struct {
		name         string
		incoming     []MDMProfileSpec
		want         []MDMProfileSpec
		keepRecorded bool
	}{
		{name: "nil", incoming: nil, want: nil},
		{name: "empty", incoming: []MDMProfileSpec{}, want: []MDMProfileSpec{}},
		{
			name:     "paths only",
			incoming: []MDMProfileSpec{{Path: "lib/b.mobileconfig"}, {Paths: "lib/*.json"}},
			want:     []MDMProfileSpec{{Path: "lib/b.mobileconfig"}, {Paths: "lib/*.json"}},
		},
		{name: "echoed only", incoming: []MDMProfileSpec{{Name: "Wi-Fi"}, {Name: "Passcode"}}, keepRecorded: true},
		{
			name:     "mixed",
			incoming: []MDMProfileSpec{{Name: "Wi-Fi"}, {Path: "lib/b.mobileconfig", Name: "B"}},
			want:     []MDMProfileSpec{{Path: "lib/b.mobileconfig", Name: "B"}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := c.want
			if c.keepRecorded {
				want = recorded
			}
			require.Equal(t, want, WithoutEchoedProfileSpecs(c.incoming, recorded))
			require.Equal(t, optjson.SetSlice(want), WithoutEchoedProfileSpecsOpt(optjson.SetSlice(c.incoming), optjson.SetSlice(recorded)))
		})
	}

	// The incoming slice isn't modified.
	incoming := []MDMProfileSpec{{Name: "Wi-Fi"}, {Path: "lib/b.mobileconfig"}}
	WithoutEchoedProfileSpecs(incoming, nil)
	require.Equal(t, []MDMProfileSpec{{Name: "Wi-Fi"}, {Path: "lib/b.mobileconfig"}}, incoming)
}

func TestConfigListsApply(t *testing.T) {
	var ac AppConfig
	ac.MDM.MacOSSettings.CustomSettings = []MDMProfileSpec{{Path: "stale.mobileconfig"}}
	ac.Scripts = optjson.SetSlice([]string{"stale.sh"})
	ac.MDM.MacOSSetup.BootstrapPackage = optjson.SetString("https://example.com/stale.pkg")

	var nilLists *ConfigLists
	nilLists.ApplyToAppConfig(&ac)
	require.Equal(t, []MDMProfileSpec{}, ac.MDM.MacOSSettings.CustomSettings)
	require.Equal(t, optjson.SetSlice([]MDMProfileSpec{}), ac.MDM.WindowsSettings.CustomSettings)
	require.Equal(t, optjson.SetSlice([]string{}), ac.Scripts)
	require.Equal(t, optjson.SetString(""), ac.MDM.MacOSSetup.BootstrapPackage)
	require.Equal(t, optjson.SetSlice([]MDMAppleVolumePurchasingProgramInfo{}), ac.MDM.VolumePurchasingProgram)

	l := NewConfigLists()
	l.WindowsProfiles = []MDMProfileSpec{{Name: "Firewall"}}
	l.Scripts = []string{"hello.sh"}
	l.SetupAssistant = "assistant.json"
	var tc TeamConfig
	l.ApplyToTeamConfig(&tc)
	require.Equal(t, optjson.SetSlice([]MDMProfileSpec{{Name: "Firewall"}}), tc.MDM.WindowsSettings.CustomSettings)
	require.Equal(t, optjson.SetSlice([]string{"hello.sh"}), tc.Scripts)
	require.Equal(t, optjson.SetString("assistant.json"), tc.MDM.MacOSSetup.MacOSSetupAssistant)
}

func TestConfigListsApplyHidden(t *testing.T) {
	filled := func() *ConfigLists {
		l := NewConfigLists()
		l.AppleProfiles = []MDMProfileSpec{{Name: "Wi-Fi"}}
		l.WindowsProfiles = []MDMProfileSpec{{Name: "Firewall"}}
		l.Scripts = []string{"hello.sh"}
		l.SetupScript = "setup.sh"
		l.BootstrapPackage = "bootstrap.pkg"
		l.SetupAssistant = "assistant.json"
		l.VolumePurchasingProgram = []MDMAppleVolumePurchasingProgramInfo{{Location: "loc", Teams: []string{"Engineering"}}}
		return l
	}
	l := filled()
	l.Hide(ConfigListsHidden{Profiles: true, Scripts: true, BootstrapPackage: true, SetupAssistant: true, VolumePurchasingProgram: true})
	require.Equal(t, ConfigLists{Hidden: l.Hidden}, *l)

	var ac AppConfig
	l.ApplyToAppConfig(&ac)
	require.Nil(t, ac.MDM.MacOSSettings.CustomSettings)
	require.Equal(t, optjson.Slice[MDMProfileSpec]{Set: true}, ac.MDM.WindowsSettings.CustomSettings)
	require.Equal(t, optjson.Slice[MDMProfileSpec]{Set: true}, ac.MDM.AndroidSettings.CustomSettings)
	require.Equal(t, optjson.Slice[string]{Set: true}, ac.Scripts)
	require.Equal(t, optjson.String{Set: true}, ac.MDM.MacOSSetup.Script)
	require.Equal(t, optjson.String{Set: true}, ac.MDM.MacOSSetup.BootstrapPackage)
	require.Equal(t, optjson.String{Set: true}, ac.MDM.MacOSSetup.MacOSSetupAssistant)
	require.Equal(t, optjson.Slice[MDMAppleVolumePurchasingProgramInfo]{Set: true}, ac.MDM.VolumePurchasingProgram)

	b, err := json.Marshal(ac.MDM.MacOSSetup.BootstrapPackage)
	require.NoError(t, err)
	require.JSONEq(t, `null`, string(b))

	// Each list is hidden on its own.
	l = filled()
	l.Hide(ConfigListsHidden{Profiles: true})
	require.Nil(t, l.AppleProfiles)
	require.Equal(t, []string{"hello.sh"}, l.Scripts)
	var tc TeamConfig
	l.ApplyToTeamConfig(&tc)
	require.Nil(t, tc.MDM.MacOSSettings.CustomSettings)
	require.Equal(t, optjson.SetSlice([]string{"hello.sh"}), tc.Scripts)
	require.Equal(t, optjson.SetString("setup.sh"), tc.MDM.MacOSSetup.Script)
	require.Equal(t, optjson.SetString("assistant.json"), tc.MDM.MacOSSetup.MacOSSetupAssistant)
}

func TestVolumePurchasingProgramFromTokens(t *testing.T) {
	require.Nil(t, VolumePurchasingProgramFromTokens(nil))
	require.Equal(t, []MDMAppleVolumePurchasingProgramInfo{
		{Location: "all", Teams: []string{DisplayNameAllTeams}},
		{Location: "some", Teams: []string{TeamNameNoTeam, "Engineering"}},
		{Location: "none", Teams: []string{}},
	}, VolumePurchasingProgramFromTokens([]*VPPTokenDB{
		{Location: "all", Teams: []TeamTuple{}},
		{Location: "some", Teams: []TeamTuple{{ID: 0, Name: TeamNameNoTeam}, {ID: 1, Name: "Engineering"}}},
		{Location: "none"},
	}))
}
