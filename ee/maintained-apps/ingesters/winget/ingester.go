package winget

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	feednvd "github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd/tools/cvefeed/nvd"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/go-github/v37/github"
	"gopkg.in/yaml.v2"
)

func IngestApps(ctx context.Context, logger kitlog.Logger, inputsPath string) ([]*maintained_apps.FMAManifestApp, error) {
	level.Info(logger).Log("msg", "starting winget app data ingestion")
	// Read from our list of apps we should be ingesting
	files, err := os.ReadDir(inputsPath)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reading winget input data directory")
	}

	var manifestApps []*maintained_apps.FMAManifestApp

	githubHTTPClient := fleethttp.NewGithubClient()
	githubClient := github.NewClient(githubHTTPClient)
	opts := &github.RepositoryContentGetOptions{
		Ref: "master",
	}

	i := &wingetIngester{
		githubClient: githubClient,
		ghClientOpts: opts,
		logger:       logger,
	}

	for _, f := range files {

		fileBytes, err := os.ReadFile(path.Join(inputsPath, f.Name()))
		if err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "reading app input file", map[string]any{"file_name": f.Name()})
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "unmarshal app input file", map[string]any{"file_name": f.Name()})
		}

		if input.Slug == "" {
			return nil, ctxerr.NewWithData(ctx, "missing slug for app", map[string]any{"file_name": f.Name()})
		}

		if input.UniqueIdentifier == "" {
			return nil, ctxerr.NewWithData(ctx, "missing unique identifier for app", map[string]any{"file_name": f.Name()})
		}

		if input.Name == "" {
			return nil, ctxerr.NewWithData(ctx, "missing name for app", map[string]any{"file_name": f.Name()})
		}

		if input.PackageIdentifier == "" {
			return nil, ctxerr.NewWithData(ctx, "missing package identifier for app", map[string]any{"file_name": f.Name()})
		}

		level.Info(logger).Log("msg", "ingesting winget app", "name", input.Name)

		outApp, err := i.ingestOne(ctx, input)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "ingesting winget app")
		}

		manifestApps = append(manifestApps, outApp)
	}

	return manifestApps, nil
}

type wingetIngester struct {
	githubClient *github.Client
	ghClientOpts *github.RepositoryContentGetOptions
	logger       kitlog.Logger
}

func (i *wingetIngester) ingestOne(ctx context.Context, input inputApp) (*maintained_apps.FMAManifestApp, error) {
	// this is the path within the winget GitHub repo where the manifests are located
	dirPath := path.Join(
		"manifests",
		strings.ToLower(input.PackageIdentifier[:1]),
		strings.ReplaceAll(input.PackageIdentifier, ".", "/"),
	)

	_, repoContents, _, err := i.githubClient.Repositories.GetContents(ctx,
		"microsoft",
		"winget-pkgs",
		dirPath,
		i.ghClientOpts,
	)
	if err != nil {
		return nil, fmt.Errorf("get data from winget repo: %w", err)
	}

	// sort the list of directories in descending order
	slices.SortFunc(repoContents, func(a, b *github.RepositoryContent) int { return feednvd.SmartVerCmp(b.GetName(), a.GetName()) })

	// this directory has the latest version data in it
	latestVersionDir := repoContents[0]
	if latestVersionDir.GetName() == "" {
		return nil, ctxerr.New(ctx, "latest version for app not found")
	}

	// this is the path to the specific manifest file we need
	installerManifestPath := path.Join(
		dirPath,
		latestVersionDir.GetName(),
		fmt.Sprintf("%s.installer.yaml", input.PackageIdentifier),
	)

	fileContents, _, _, err := i.githubClient.Repositories.GetContents(ctx,
		"microsoft",
		"winget-pkgs",
		installerManifestPath,
		i.ghClientOpts,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting the ")
	}

	contents, err := fileContents.GetContent()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting winget manifest file contents")
	}

	var m installerManifest
	if err := yaml.Unmarshal([]byte(contents), &m); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling winget manifest")
	}

	localeManifestPath := path.Join(dirPath, latestVersionDir.GetName(), fmt.Sprintf("%s.locale.en-US.yaml", input.PackageIdentifier))
	fileContents, _, _, err = i.githubClient.Repositories.GetContents(ctx,
		"microsoft",
		"winget-pkgs",
		localeManifestPath,
		i.ghClientOpts,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting winget manifest locale file contents")
	}

	contents, err = fileContents.GetContent()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting locale manifest contents")
	}

	var l localeManifest
	if err := yaml.Unmarshal([]byte(contents), &l); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling winget locale manifest")
	}

	var out maintained_apps.FMAManifestApp

	// TODO: handle non-machine scope (aka .exe installers)
	var installScript, uninstallScript, installerURL, productCode, sha256 string

	// Some data is present on the top-level object, so try to grab that first
	if m.InstallerType == installerTypeMSI || m.Scope == machineScope {
		productCode = m.ProductCode
		installScript = file.GetInstallScript(m.InstallerType)
		uninstallScript = file.GetUninstallScript(m.InstallerType)
	}

	if productCode == "" {
		for _, afe := range m.AppsAndFeaturesEntries {
			if afe.ProductCode != "" {
				productCode = afe.ProductCode
			}
		}
	}

	// Walk through the installers and get any data we missed
	for _, installer := range m.Installers {
		if (installer.Scope == machineScope || m.Scope == machineScope || (installer.Scope == "" && m.Scope == "")) &&
			installer.Architecture == arch64Bit {
			// Use the first machine scoped installer
			installerURL = installer.InstallerURL
			sha256 = installer.InstallerSha256
			installerType := installer.InstallerType
			if installerType == "" || installerType == installerTypeWix {
				// try to get it from the URL
				// TODO: this may not work in all situations
				// TODO: also, "wix" might always mean an MSI installer, in which case we should
				// just use that
				installerType = strings.Trim(filepath.Ext(installerURL), ".")
			}
			if installScript == "" {
				installScript = file.GetInstallScript(installerType)
			}
			if uninstallScript == "" {
				uninstallScript = file.GetUninstallScript(installerType)
			}
			if productCode == "" {
				productCode = installer.ProductCode
			}
			break
		}
	}

	out.Name = input.Name
	out.Slug = input.Slug
	out.InstallerURL = installerURL
	out.UniqueIdentifier = input.UniqueIdentifier
	out.SHA256 = strings.ToLower(sha256) // maintain consistency with darwin outputs SHAs
	out.Version = m.PackageVersion
	out.Queries = maintained_apps.FMAQueries{
		Exists: fmt.Sprintf("SELECT 1 FROM programs WHERE name = '%s' AND publisher = '%s';", l.PackageName, l.Publisher),
	}
	out.InstallScript = installScript
	out.UninstallScript = uninstallScript
	out.InstallScriptRef = maintained_apps.GetScriptRef(installScript)
	out.UninstallScriptRef = maintained_apps.GetScriptRef(uninstallScript)

	return &out, nil
}

type inputApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// PackageIdentifier is the identifier used by winget. It's composed of a vendor part (e.g.
	// AgileBits) and an app part (e.g. 1Password), joined by a "."
	PackageIdentifier string `json:"package_identifier"`
	UniqueIdentifier  string `json:"unique_identifier"`
}

type installerManifest struct {
	PackageIdentifier      string                   `yaml:"PackageIdentifier"`
	PackageVersion         string                   `yaml:"PackageVersion"`
	Installers             []installer              `yaml:"Installers"`
	InstallerType          string                   `yaml:"InstallerType"`
	AppsAndFeaturesEntries []appsAndFeaturesEntries `yaml:"AppsAndFeaturesEntries,omitempty"`
	ProductCode            string                   `yaml:"ProductCode"`
	Scope                  string                   `yaml:"Scope"`
}

type installer struct {
	Architecture string `yaml:"Architecture"`
	// InstallerType is the filetype of the installer. Either "exe" or "msi".
	InstallerType          string                   `yaml:"InstallerType"`
	Scope                  string                   `yaml:"Scope"`
	InstallerURL           string                   `yaml:"InstallerUrl"`
	InstallerSha256        string                   `yaml:"InstallerSha256"`
	InstallModes           []string                 `yaml:"InstallModes,omitempty"`
	InstallerSwitches      installerSwitches        `yaml:"InstallerSwitches,omitempty"`
	ProductCode            string                   `yaml:"ProductCode"`
	AppsAndFeaturesEntries []appsAndFeaturesEntries `yaml:"AppsAndFeaturesEntries,omitempty"`
}
type installerSwitches struct {
	Silent             string `yaml:"Silent"`
	SilentWithProgress string `yaml:"SilentWithProgress"`
}

type appsAndFeaturesEntries struct {
	Publisher   string `yaml:"Publisher"`
	ProductCode string `yaml:"ProductCode"`
	UpgradeCode string `yaml:"UpgradeCode"`
}

type localeManifest struct {
	PackageIdentifier   string   `yaml:"PackageIdentifier"`
	PackageVersion      string   `yaml:"PackageVersion"`
	PackageLocale       string   `yaml:"PackageLocale"`
	Publisher           string   `yaml:"Publisher"`
	PublisherURL        string   `yaml:"PublisherUrl"`
	PublisherSupportURL string   `yaml:"PublisherSupportUrl"`
	PrivacyURL          string   `yaml:"PrivacyUrl"`
	Author              string   `yaml:"Author"`
	PackageName         string   `yaml:"PackageName"`
	PackageURL          string   `yaml:"PackageUrl"`
	License             string   `yaml:"License"`
	LicenseURL          string   `yaml:"LicenseUrl"`
	Copyright           string   `yaml:"Copyright"`
	CopyrightURL        string   `yaml:"CopyrightUrl"`
	ShortDescription    string   `yaml:"ShortDescription"`
	Description         string   `yaml:"Description"`
	Tags                []string `yaml:"Tags"`
	PurchaseURL         string   `yaml:"PurchaseUrl"`
	ManifestType        string   `yaml:"ManifestType"`
	ManifestVersion     string   `yaml:"ManifestVersion"`
}

const (
	machineScope     = "machine"
	installerTypeMSI = "msi"
	installerTypeWix = "wix"
	arch64Bit        = "x64"
)
