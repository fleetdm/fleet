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

func IngestApps(ctx context.Context, logger kitlog.Logger, inputsPath string, slugFilter string) ([]*maintained_apps.FMAManifestApp, error) {
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
		if f.IsDir() {
			continue
		}

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

		if slugFilter != "" && !strings.Contains(input.Slug, slugFilter) {
			continue
		}

		level.Info(logger).Log("msg", "ingesting winget app", "name", input.Name)

		outApp, err := i.ingestOne(ctx, input)
		if err != nil {
			level.Warn(logger).Log("msg", "failed to ingest app", "err", err, "name", input.Name)
			continue
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
		return nil, ctxerr.Wrap(ctx, err, "downloading file contents for installer manifest")
	}

	contents, err := fileContents.GetContent()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "extracting installer manifest file contents")
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

	var installScript, uninstallScript string

	// if we have a provided install script, use that
	if input.InstallScriptPath != "" {
		scriptBytes, err := os.ReadFile(input.InstallScriptPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading provided install script file")
		}

		installScript = string(scriptBytes)
	}

	if input.UninstallScriptPath != "" {
		scriptBytes, err := os.ReadFile(input.UninstallScriptPath)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading provided uninstall script file")
		}

		uninstallScript = string(scriptBytes)
	}

	// Some data is present on the top-level object, so try to grab that first
	if m.InstallerType == installerTypeMSI || m.Scope == machineScope {
		installScript = file.GetInstallScript(m.InstallerType)
		uninstallScript = file.GetUninstallScript(m.InstallerType)
	}

	installersByArch := map[string]map[string]*installer{
		arch64Bit: make(map[string]*installer),
		arch32Bit: make(map[string]*installer),
	}

	// Walk through the installers and get any data we missed
	for _, installer := range m.Installers {
		if installer.Architecture != arch32Bit && installer.Architecture != arch64Bit {
			// we don't care about arm or other architectures
			continue
		}

		// get the installer type
		installerType := m.InstallerType
		if installerType == "" || installerType == installerTypeWix {
			installerType = installer.InstallerType
		}

		if installerType == "" || installerType == installerTypeWix {
			// try to get it from the URL
			// TODO: this may not work in all situations
			// TODO: also, "wix" might always mean an MSI installer, in which case we should
			// just use that
			installerType = strings.Trim(filepath.Ext(installer.InstallerURL), ".")
		}

		if installerType == installerTypeMSIX {
			// skip MSIX for now
			continue
		}

		// get the installer scope
		scope := m.Scope
		if scope == "" {
			scope = installer.Scope
			if scope == "" {
				if installerType == installerTypeMSI {
					scope = machineScope
				}
			}
		}
		fmt.Printf("scope: %v\n", scope)
		fmt.Printf("installerType: %v\n", installerType)

		installersByArch[installer.Architecture][scope] = &installer

	}

	selectedScope := machineScope
	inst, ok := installersByArch[arch64Bit][machineScope]
	if !ok {
		inst, ok = installersByArch[arch64Bit][userScope]
		selectedScope = userScope
	}
	fmt.Printf("selectedScope: %v\n", selectedScope)

	if inst == nil {
		return nil, ctxerr.New(ctx, "no suitable installer found for app")
	}

	if selectedScope == machineScope {
		if installScript == "" {
			installScript = file.GetInstallScript(installerTypeMSI)
		}
		if uninstallScript == "" {
			uninstallScript = file.GetUninstallScript(installerTypeMSI)
		}

	}

	if installScript == "" {
		return nil, ctxerr.New(ctx, "no install script found for app, aborting")
	}

	if uninstallScript == "" {
		return nil, ctxerr.New(ctx, "no uninstall script found for app, aborting")
	}

	out.Name = input.Name
	out.Slug = input.Slug
	out.InstallerURL = inst.InstallerURL
	out.UniqueIdentifier = input.UniqueIdentifier
	out.SHA256 = strings.ToLower(inst.InstallerSha256) // maintain consistency with darwin outputs SHAs
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
	PackageIdentifier   string `json:"package_identifier"`
	UniqueIdentifier    string `json:"unique_identifier"`
	InstallScriptPath   string `json:"install_script_path"`
	UninstallScriptPath string `json:"uninstall_script_path"`
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
	machineScope      = "machine"
	userScope         = "user"
	installerTypeMSI  = "msi"
	installerTypeMSIX = "msix"
	installerTypeWix  = "wix"
	arch64Bit         = "x64"
	arch32Bit         = "x86"
)
