package fleet

import "github.com/fleetdm/fleet/v4/pkg/optjson"

// The config only records what YAML last applied for these items. That goes
// stale as soon as one is added or removed in the UI or API, so responses
// leave them out and clients read each item on its own endpoint. The stored
// values stay: the server compares against them to know what a request
// changed.

func (c *AppConfig) ClearRecordedLists() {
	c.Scripts = optjson.Slice[string]{}
	c.MDM.VolumePurchasingProgram = optjson.Slice[MDMAppleVolumePurchasingProgramInfo]{}
	clearRecordedMDMLists(&c.MDM.MacOSSettings, &c.MDM.WindowsSettings, &c.MDM.AndroidSettings, &c.MDM.MacOSSetup)
}

func (t *Team) ClearRecordedLists() {
	if t == nil {
		return
	}
	t.Config.Scripts = optjson.Slice[string]{}
	clearRecordedMDMLists(&t.Config.MDM.MacOSSettings, &t.Config.MDM.WindowsSettings, &t.Config.MDM.AndroidSettings, &t.Config.MDM.MacOSSetup)
}

func clearRecordedMDMLists(mac *MacOSSettings, win *WindowsSettings, android *AndroidSettings, setup *MacOSSetup) {
	mac.CustomSettings = nil
	win.CustomSettings = optjson.Slice[MDMProfileSpec]{}
	android.CustomSettings = optjson.Slice[MDMProfileSpec]{}
	setup.BootstrapPackage = optjson.String{}
	setup.MacOSSetupAssistant = optjson.String{}
	setup.Script = optjson.String{}
}
