package fleet

import (
	"slices"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
)

// ConfigLists holds the parts of a fleet's config whose source of truth is
// their own tables, so they're the same whether they were added via YAML, the
// UI, or the API.
type ConfigLists struct {
	AppleProfiles    []MDMProfileSpec
	WindowsProfiles  []MDMProfileSpec
	AndroidProfiles  []MDMProfileSpec
	Scripts          []string
	BootstrapPackage string
	SetupAssistant   string
	SetupScript      string
	// VolumePurchasingProgram is a global setting, so it's only set for "No fleet" (0).
	VolumePurchasingProgram []MDMAppleVolumePurchasingProgramInfo

	// Hidden lists are returned as null: the user can't read them on their own
	// endpoints, so the config and fleet responses mustn't expose them either.
	Hidden ConfigListsHidden
}

type ConfigListsHidden struct {
	Profiles                bool
	Scripts                 bool // also covers the setup experience script
	BootstrapPackage        bool
	SetupAssistant          bool
	VolumePurchasingProgram bool
}

// NewConfigLists returns empty, non-nil lists so responses always render
// them as [] rather than null.
func NewConfigLists() *ConfigLists {
	return &ConfigLists{
		AppleProfiles:   []MDMProfileSpec{},
		WindowsProfiles: []MDMProfileSpec{},
		AndroidProfiles: []MDMProfileSpec{},
		Scripts:         []string{},

		VolumePurchasingProgram: []MDMAppleVolumePurchasingProgramInfo{},
	}
}

func (l *ConfigLists) ApplyToAppConfig(ac *AppConfig) {
	if l == nil {
		l = NewConfigLists()
	}
	l.applyToMDM(&ac.MDM.MacOSSettings, &ac.MDM.WindowsSettings, &ac.MDM.AndroidSettings, &ac.MDM.MacOSSetup)
	ac.Scripts = sliceOrNull(l.Scripts, l.Hidden.Scripts)
	ac.MDM.VolumePurchasingProgram = sliceOrNull(l.VolumePurchasingProgram, l.Hidden.VolumePurchasingProgram)
}

func (l *ConfigLists) ApplyToTeamConfig(tc *TeamConfig) {
	if l == nil {
		l = NewConfigLists()
	}
	l.applyToMDM(&tc.MDM.MacOSSettings, &tc.MDM.WindowsSettings, &tc.MDM.AndroidSettings, &tc.MDM.MacOSSetup)
	tc.Scripts = sliceOrNull(l.Scripts, l.Hidden.Scripts)
}

func (l *ConfigLists) applyToMDM(mac *MacOSSettings, win *WindowsSettings, android *AndroidSettings, setup *MacOSSetup) {
	mac.CustomSettings = nil
	if !l.Hidden.Profiles {
		mac.CustomSettings = l.AppleProfiles
	}
	win.CustomSettings = sliceOrNull(l.WindowsProfiles, l.Hidden.Profiles)
	android.CustomSettings = sliceOrNull(l.AndroidProfiles, l.Hidden.Profiles)
	setup.BootstrapPackage = stringOrNull(l.BootstrapPackage, l.Hidden.BootstrapPackage)
	setup.MacOSSetupAssistant = stringOrNull(l.SetupAssistant, l.Hidden.SetupAssistant)
	setup.Script = stringOrNull(l.SetupScript, l.Hidden.Scripts)
}

func sliceOrNull[T any](s []T, hidden bool) optjson.Slice[T] {
	if hidden {
		return optjson.Slice[T]{Set: true}
	}
	return optjson.SetSlice(s)
}

func stringOrNull(s string, hidden bool) optjson.String {
	if hidden {
		return optjson.String{Set: true}
	}
	return optjson.SetString(s)
}

// Config and fleet responses list saved profiles by name, without a path. A
// request that sends them back mustn't replace the profiles recorded from YAML,
// or store entries that MDMProfileSpecsMatch can't compare.
func WithoutEchoedProfileSpecs(incoming, recorded []MDMProfileSpec) []MDMProfileSpec {
	if !slices.ContainsFunc(incoming, hasNoPath) {
		return incoming
	}
	kept := slices.DeleteFunc(slices.Clone(incoming), hasNoPath)
	if len(kept) == 0 {
		return recorded
	}
	return kept
}

func WithoutEchoedProfileSpecsOpt(incoming, recorded optjson.Slice[MDMProfileSpec]) optjson.Slice[MDMProfileSpec] {
	if !slices.ContainsFunc(incoming.Value, hasNoPath) {
		return incoming
	}
	kept := slices.DeleteFunc(slices.Clone(incoming.Value), hasNoPath)
	if len(kept) == 0 {
		return recorded
	}
	return optjson.SetSlice(kept)
}

func hasNoPath(s MDMProfileSpec) bool {
	return s.Path == "" && s.Paths == ""
}

func VolumePurchasingProgramFromTokens(tokens []*VPPTokenDB) []MDMAppleVolumePurchasingProgramInfo {
	var vpp []MDMAppleVolumePurchasingProgramInfo
	for _, token := range tokens {
		teamNames := make([]string, 0, len(token.Teams))
		for _, team := range token.Teams {
			teamNames = append(teamNames, team.Name)
		}
		// A token assigned to all fleets has no fleet names to list.
		if token.Teams != nil && len(token.Teams) == 0 {
			teamNames = []string{DisplayNameAllTeams}
		}
		vpp = append(vpp, MDMAppleVolumePurchasingProgramInfo{
			Location: token.Location,
			Teams:    teamNames,
		})
	}
	return vpp
}
