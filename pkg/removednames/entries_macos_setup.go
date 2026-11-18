package removednames

func init() { register(macosSetupEntries...) }

var macosSetupEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_setup", New: "setup_experience"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "bootstrap_package", New: "macos_bootstrap_package"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_setup_assistant", New: "apple_setup_assistant"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "enable_release_device_manually", New: "apple_enable_release_device_manually"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "manual_agent_install", New: "macos_manual_agent_install"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "script", New: "macos_script"},

	yamlEntry("org_settings.mdm.macos_setup", "org_settings.mdm.setup_experience"),
	yamlEntry("org_settings.mdm.setup_experience.bootstrap_package", "org_settings.mdm.setup_experience.macos_bootstrap_package"),
	yamlEntry("org_settings.mdm.setup_experience.macos_setup_assistant", "org_settings.mdm.setup_experience.apple_setup_assistant"),
	yamlEntry("org_settings.mdm.setup_experience.enable_release_device_manually", "org_settings.mdm.setup_experience.apple_enable_release_device_manually"),
	yamlEntry("org_settings.mdm.setup_experience.manual_agent_install", "org_settings.mdm.setup_experience.macos_manual_agent_install"),
	yamlEntry("org_settings.mdm.setup_experience.script", "org_settings.mdm.setup_experience.macos_script"),
	yamlEntry("controls.macos_setup", "controls.setup_experience"),
	yamlEntry("controls.setup_experience.bootstrap_package", "controls.setup_experience.macos_bootstrap_package"),
	yamlEntry("controls.setup_experience.macos_setup_assistant", "controls.setup_experience.apple_setup_assistant"),
	yamlEntry("controls.setup_experience.enable_release_device_manually", "controls.setup_experience.apple_enable_release_device_manually"),
	yamlEntry("controls.setup_experience.manual_agent_install", "controls.setup_experience.macos_manual_agent_install"),
	yamlEntry("controls.setup_experience.script", "controls.setup_experience.macos_script"),
	yamlEntry("settings.mdm.macos_setup", "settings.mdm.setup_experience"),
	yamlEntry("settings.mdm.setup_experience.bootstrap_package", "settings.mdm.setup_experience.macos_bootstrap_package"),
	yamlEntry("settings.mdm.setup_experience.macos_setup_assistant", "settings.mdm.setup_experience.apple_setup_assistant"),
	yamlEntry("settings.mdm.setup_experience.enable_release_device_manually", "settings.mdm.setup_experience.apple_enable_release_device_manually"),
	yamlEntry("settings.mdm.setup_experience.manual_agent_install", "settings.mdm.setup_experience.macos_manual_agent_install"),
	yamlEntry("settings.mdm.setup_experience.script", "settings.mdm.setup_experience.macos_script"),
}
