package maintained_apps

import "context"

type InputApp struct {
	// Name is the user-friendly name of the app.
	Name string `json:"name"`
	// SourceIdentifier is the identifier in the source data for the app (e.g. homebrew token).
	SourceIdentifier string `json:"source_identifier"`
	// UniqueIdentifier is the app's unique identifier on its platform (e.g. bundle ID on macOS).
	UniqueIdentifier string `json:"unique_identifier"`
	// InstallerFormat is the installer format used for installing this app.
	InstallerFormat string `json:"installer_format"`
}

const ExistsKey = "exists"

// TODO(JVE): better name?
type OutputApp struct {
	Version            string            `json:"version"`
	Queries            map[string]string `json:"queries"`
	InstallerURL       string            `json:"installer_url"`
	UniqueIdentifier   string            `json:"unique_identifier"`
	InstallScriptRef   string            `json:"install_script_ref"`
	UninstallScriptRef string            `json:"uninstall_script_ref"`
	Sha256             string            `json:"sha256"`
	Description        string            `json:"-"`
}

type OutputFile struct {
	Versions []*OutputApp      `json:"versions"`
	Refs     map[string]string `json:"refs"`
}

type Ingester interface {
	IngestApps(ctx context.Context) error
}

type OutputAppsFileApp struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Platform         string `json:"platform"`
	UniqueIdentifier string `json:"unique_identifier"`
	Description      string `json:"description"`
}

type OutputAppsFile struct {
	Version int                 `json:"version"`
	Apps    []OutputAppsFileApp `json:"apps"`
}
