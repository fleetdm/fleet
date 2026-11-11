package removednames

func init() { register(macosSetupEntries...) }

var macosSetupEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_setup", New: "setup_experience"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "bootstrap_package", New: "macos_bootstrap_package"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "macos_setup_assistant", New: "apple_setup_assistant"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "enable_release_device_manually", New: "apple_enable_release_device_manually"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "manual_agent_install", New: "macos_manual_agent_install"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "script", New: "macos_script"},

	yamlEntry("controls.macos_setup", "controls.setup_experience"),
	yamlEntry("controls.setup_experience.bootstrap_package", "controls.setup_experience.macos_bootstrap_package"),
	yamlEntry("controls.setup_experience.macos_setup_assistant", "controls.setup_experience.apple_setup_assistant"),
	yamlEntry("controls.setup_experience.enable_release_device_manually", "controls.setup_experience.apple_enable_release_device_manually"),
	yamlEntry("controls.setup_experience.manual_agent_install", "controls.setup_experience.macos_manual_agent_install"),
	yamlEntry("controls.setup_experience.script", "controls.setup_experience.macos_script"),
}
