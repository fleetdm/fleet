package maintained_apps

import "context"

type InputApp struct {
	Name             string `json:"name"`
	Identifier       string `json:"identifier"`
	UniqueIdentifier string `json:"unique_identifier"`
	InstallerFormat  string `json:"installer_format"`
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
}

type OutputFile struct {
	Versions []*OutputApp      `json:"versions"`
	Refs     map[string]string `json:"refs"`
}

type Ingester interface {
	IngestApps(ctx context.Context) error
}
