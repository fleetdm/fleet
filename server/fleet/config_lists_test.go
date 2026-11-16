package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClearRecordedLists(t *testing.T) {
	profiles := []MDMProfileSpec{{Path: "profile"}}
	macOSSettings := MacOSSettings{CustomSettings: profiles}
	windowsSettings := WindowsSettings{CustomSettings: optjson.SetSlice(profiles)}
	androidSettings := AndroidSettings{CustomSettings: optjson.SetSlice(profiles)}
	macOSSetup := MacOSSetup{
		BootstrapPackage:    optjson.SetString("https://example.com/bootstrap.pkg"),
		MacOSSetupAssistant: optjson.SetString("assistant.json"),
		Script:              optjson.SetString("script.sh"),
	}

	recordedKeysByObject := map[string][]string{
		"macos_settings":   {"custom_settings", "configuration_profiles"},
		"apple_settings":   {"custom_settings", "configuration_profiles"},
		"windows_settings": {"custom_settings", "configuration_profiles"},
		"android_settings": {"custom_settings", "configuration_profiles"},
		"macos_setup":      {"bootstrap_package", "macos_setup_assistant", "script"},
		"setup_experience": {"macos_bootstrap_package", "apple_setup_assistant", "macos_script"},
	}

	// Responses carry every renamed key under both names, which the response encoder adds.
	marshalToMap := func(t *testing.T, value any) map[string]any {
		raw, err := json.Marshal(value)
		require.NoError(t, err)
		raw = endpointer.DuplicateJSONKeys(raw, endpointer.ExtractAliasRules(value))
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(raw, &decoded))
		return decoded
	}

	// Returns the recorded keys found, so the same walk proves they were there before clearing.
	recordedKeysPresent := func(t *testing.T, config map[string]any, topLevelMDMKeys ...string) []string {
		var present []string
		if _, ok := config["scripts"]; ok {
			present = append(present, "scripts")
		}
		mdm, ok := config["mdm"].(map[string]any)
		require.True(t, ok)
		for _, key := range topLevelMDMKeys {
			if _, ok := mdm[key]; ok {
				present = append(present, "mdm."+key)
			}
		}
		for objectName, keys := range recordedKeysByObject {
			object, ok := mdm[objectName].(map[string]any)
			if !ok {
				continue
			}
			for _, key := range keys {
				if _, ok := object[key]; ok {
					present = append(present, "mdm."+objectName+"."+key)
				}
			}
		}
		return present
	}

	requireOtherKeysKept := func(t *testing.T, config map[string]any) {
		mdm := config["mdm"].(map[string]any)
		appleSettings, ok := mdm["apple_settings"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, appleSettings, "enable_disk_encryption")
		setupExperience, ok := mdm["setup_experience"].(map[string]any)
		require.True(t, ok)
		assert.Contains(t, setupExperience, "enable_end_user_authentication")
		assert.Contains(t, mdm["windows_settings"], "enable_disk_encryption")
		assert.Contains(t, mdm["android_settings"], "certificates")
	}

	t.Run("app config", func(t *testing.T) {
		appConfig := &AppConfig{
			Scripts: optjson.SetSlice([]string{"script.sh"}),
			MDM: MDM{
				VolumePurchasingProgram: optjson.SetSlice([]MDMAppleVolumePurchasingProgramInfo{{Location: "location", Teams: []string{"fleet"}}}),
				MacOSSettings:           macOSSettings,
				WindowsSettings:         windowsSettings,
				AndroidSettings:         androidSettings,
				MacOSSetup:              macOSSetup,
			},
		}
		require.ElementsMatch(t, []string{
			"scripts",
			"mdm.volume_purchasing_program",
			"mdm.macos_settings.custom_settings",
			"mdm.apple_settings.configuration_profiles",
			"mdm.windows_settings.custom_settings",
			"mdm.windows_settings.configuration_profiles",
			"mdm.android_settings.custom_settings",
			"mdm.android_settings.configuration_profiles",
			"mdm.macos_setup.bootstrap_package",
			"mdm.macos_setup.macos_setup_assistant",
			"mdm.macos_setup.script",
			"mdm.setup_experience.macos_bootstrap_package",
			"mdm.setup_experience.apple_setup_assistant",
			"mdm.setup_experience.macos_script",
		}, recordedKeysPresent(t, marshalToMap(t, appConfig), "volume_purchasing_program"))

		appConfig.ClearRecordedLists()

		cleared := marshalToMap(t, appConfig)
		require.Empty(t, recordedKeysPresent(t, cleared, "volume_purchasing_program"))
		requireOtherKeysKept(t, cleared)
	})

	t.Run("fleet", func(t *testing.T) {
		team := &Team{
			Name: "fleet",
			Config: TeamConfig{
				Scripts: optjson.SetSlice([]string{"script.sh"}),
				MDM: TeamMDM{
					MacOSSettings:   macOSSettings,
					WindowsSettings: windowsSettings,
					AndroidSettings: androidSettings,
					MacOSSetup:      macOSSetup,
				},
			},
		}
		require.ElementsMatch(t, []string{
			"scripts",
			"mdm.macos_settings.custom_settings",
			"mdm.apple_settings.configuration_profiles",
			"mdm.windows_settings.custom_settings",
			"mdm.windows_settings.configuration_profiles",
			"mdm.android_settings.custom_settings",
			"mdm.android_settings.configuration_profiles",
			"mdm.macos_setup.bootstrap_package",
			"mdm.macos_setup.macos_setup_assistant",
			"mdm.macos_setup.script",
			"mdm.setup_experience.macos_bootstrap_package",
			"mdm.setup_experience.apple_setup_assistant",
			"mdm.setup_experience.macos_script",
		}, recordedKeysPresent(t, marshalToMap(t, team)))

		team.ClearRecordedLists()

		cleared := marshalToMap(t, team)
		require.Empty(t, recordedKeysPresent(t, cleared))
		requireOtherKeysKept(t, cleared)
	})

	t.Run("nil fleet", func(t *testing.T) {
		require.NotPanics(t, func() { (*Team)(nil).ClearRecordedLists() })
	})
}
