package maintained_apps

import "context"

// Ingester is responsible for ingesting the metadata for maintained apps for a given platform.
// Each platform may have multiple sources for metadata (e.g. homebrew and autopkg for macOS). Each
// source must have its own SourceIngester implementation.
type Ingester interface {
	IngestApps(ctx context.Context) error
}

// SourceIngester is resposible for the actual ingesting of metadata from a source and converting it
// to our FMA manifest format.
type SourceIngester interface {
	IngestOne(ctx context.Context, app InputApp) (*FMAManifestApp, map[string]string, error)
}

// SourceHomebrew indicates that the app metadata comes from homebrew.
const SourceHomebrew = "homebrew"

type InputApp struct {
	// Name is the user-friendly name of the app.
	Name string `json:"name"`
	// SourceIdentifier is the identifier in the source data for the app (e.g. homebrew token).
	SourceIdentifier string `json:"source_identifier"`
	// UniqueIdentifier is the app's unique identifier on its platform (e.g. bundle ID on macOS).
	UniqueIdentifier string `json:"unique_identifier"`
	// InstallerFormat is the installer format used for installing this app.
	InstallerFormat string `json:"installer_format"`
	// Source is the source for the FMA metadata, e.g. homebrew, autopkg, winget, etc.
	Source string `json:"source"`
}

// ExistsKey is the key used for an osquery query that checks if the app exists on a host.
const ExistsKey = "exists"

type FMAManifestApp struct {
	Version            string            `json:"version"`
	Queries            map[string]string `json:"queries"`
	InstallerURL       string            `json:"installer_url"`
	UniqueIdentifier   string            `json:"unique_identifier"`
	InstallScriptRef   string            `json:"install_script_ref"`
	UninstallScriptRef string            `json:"uninstall_script_ref"`
	SHA256             string            `json:"sha256"`
	// Description is an optional description of the app and what it does.
	Description string `json:"-"`
}

type FMAManifestFile struct {
	Versions []*FMAManifestApp `json:"versions"`
	Refs     map[string]string `json:"refs"`
}

type FMAListFileApp struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Platform         string `json:"platform"`
	UniqueIdentifier string `json:"unique_identifier"`
	Description      string `json:"description"`
}

type FMAListFile struct {
	Version int              `json:"version"`
	Apps    []FMAListFileApp `json:"apps"`
}
