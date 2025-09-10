package maintained_apps

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"

	kitlog "github.com/go-kit/log"
)

// Ingester is responsible for ingesting the metadata for maintained apps for a given platform.
// Each platform may have multiple sources for metadata (e.g. homebrew and autopkg for macOS). Each
// source must have its own Ingester implementation.
type Ingester func(ctx context.Context, logger kitlog.Logger, inputsPath string, slugFilter string) ([]*FMAManifestApp, error)

const OutputPath = "ee/maintained-apps/outputs"

type FMAQueries struct {
	Exists string `json:"exists"`
}

type FMAManifestApp struct {
	Version            string     `json:"version"`
	Queries            FMAQueries `json:"queries"`
	InstallerURL       string     `json:"installer_url"`
	UniqueIdentifier   string     `json:"unique_identifier,omitempty"`
	InstallScriptRef   string     `json:"install_script_ref"`
	UninstallScriptRef string     `json:"uninstall_script_ref"`
	InstallScript      string     `json:"-"`
	UninstallScript    string     `json:"-"`
	SHA256             string     `json:"sha256"`
	Slug               string     `json:"-"`
	Name               string     `json:"-"`
	DefaultCategories  []string   `json:"default_categories"`
	Frozen             bool       `json:"-"`
}

func (a *FMAManifestApp) Platform() string {
	parts := strings.Split(a.Slug, "/")
	if len(parts) != 2 {
		return ""
	}

	return parts[1]
}

func (a *FMAManifestApp) SlugAppName() string {
	parts := strings.Split(a.Slug, "/")
	if len(parts) != 2 {
		return ""
	}

	return parts[0]
}

func (a *FMAManifestApp) IsEmpty() bool {
	return a.Version == "" &&
		a.InstallerURL == "" &&
		a.UniqueIdentifier == "" &&
		a.InstallScriptRef == "" &&
		a.UninstallScriptRef == "" &&
		a.SHA256 == "" &&
		a.Queries == (FMAQueries{})
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

func GetScriptRef(script string) string {
	h := sha256.New()
	_, _ = io.Copy(h, strings.NewReader(script)) // writes to a Hash can never fail
	return hex.EncodeToString(h.Sum(nil))[:8]
}
