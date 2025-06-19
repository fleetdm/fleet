package winget

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
			return nil, ctxerr.Wrapf(ctx, err, "unmarshal app input file: %s", f.Name())
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
	var selectedInstaller *installer
	var installScript, uninstallScript string
	productCode := m.ProductCode

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

	for _, installer := range m.Installers {
		level.Debug(i.logger).Log("msg", "checking installer", "arch", installer.Architecture, "type", installer.InstallerType, "locale", installer.InstallerLocale, "scope", installer.Scope)
		installerType := m.InstallerType
		if installerType == "" || isVendorType(installerType) {
			installerType = installer.InstallerType
		}

		if installerType == "" || isVendorType(installerType) {
			// try to get it from the URL
			installerType = strings.Trim(filepath.Ext(installer.InstallerURL), ".")
		}

		scope := m.Scope
		if scope == "" {
			scope = installer.Scope
			if scope == "" {
				if installerType == installerTypeMSI {
					scope = machineScope
				}
			}
		}

		if !isFileType(installerType) && scope == machineScope {
			// assume we're an MSI
			installerType = installerTypeMSI
		}

		if input.InstallerLocale == "" {
			// We only care about the locale if one is specified
			installer.InstallerLocale = ""
		}

		if installer.Architecture == input.InstallerArch &&
			scope == input.InstallerScope &&
			installer.InstallerLocale == input.InstallerLocale &&
			installerType == input.InstallerType {
			selectedInstaller = &installer
			break
		}

	}

	if selectedInstaller == nil {
		return nil, ctxerr.New(ctx, "failed to find installer for app")
	}

	if input.InstallerType == installerTypeMSI && input.InstallerScope == machineScope {
		if installScript == "" {
			installScript = file.GetInstallScript(installerTypeMSI)
		}
	}

	if (input.InstallerType == installerTypeMSI || input.UninstallType == installerTypeMSI) && input.InstallerScope == machineScope {
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

	if productCode == "" {
		productCode = selectedInstaller.ProductCode
	}

	productCode = strings.Split(productCode, ".")[0]

	out.Name = input.Name
	out.Slug = input.Slug
	out.InstallerURL = selectedInstaller.InstallerURL
	out.UniqueIdentifier = input.UniqueIdentifier
	out.DefaultCategories = input.DefaultCategories
	out.SHA256 = "no_check"
	if !input.IgnoreHash {
		out.SHA256 = strings.ToLower(selectedInstaller.InstallerSha256) // maintain consistency with darwin outputs SHAs
	}
	out.Version = m.PackageVersion
	publisher := l.Publisher
	if input.ProgramPublisher != "" {
		publisher = input.ProgramPublisher
	}
	name := l.PackageName
	if input.UniqueIdentifier != "" {
		name = input.UniqueIdentifier
	}
	existsTemplate := "SELECT 1 FROM programs WHERE name = '%s' AND publisher = '%s';"
	if input.FuzzyMatchName {
		existsTemplate = "SELECT 1 FROM programs WHERE name LIKE '%s %%' AND publisher = '%s';"
	}
	out.Queries = maintained_apps.FMAQueries{
		Exists: fmt.Sprintf(existsTemplate, name, publisher),
	}
	out.InstallScript = installScript
	out.UninstallScript = preProcessUninstallScript(uninstallScript, productCode)
	out.InstallScriptRef = maintained_apps.GetScriptRef(out.InstallScript)
	out.UninstallScriptRef = maintained_apps.GetScriptRef(out.UninstallScript)
	out.Frozen = input.Frozen

	return &out, nil
}

var packageIDRegex = regexp.MustCompile(`((("\$PACKAGE_ID")|(\$PACKAGE_ID))(?P<suffix>\W|$))|(("\${PACKAGE_ID}")|(\${PACKAGE_ID}))`)

func preProcessUninstallScript(uninstallScript, productCode string) string {
	code := fmt.Sprintf("\"%s\"", productCode)
	return packageIDRegex.ReplaceAllString(uninstallScript, fmt.Sprintf("%s${suffix}", code))
}

// these are installer types that correspond to software vendors, not the actual installer type
// (like exe or msi).
var vendorTypes = map[string]struct{}{
	installerTypeWix:      {},
	installerTypeNullSoft: {},
	installerTypeInno:     {},
}

func isVendorType(installerType string) bool {
	_, ok := vendorTypes[installerType]
	return ok
}

var fileTypes = map[string]struct{}{
	installerTypeMSI:  {},
	installerTypeMSIX: {},
	installerTypeExe:  {},
}

func isFileType(installerType string) bool {
	_, ok := fileTypes[installerType]
	return ok
}

type inputApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// PackageIdentifier is the identifier used by winget. It's composed of a vendor part (e.g.
	// AgileBits) and an app part (e.g. 1Password), joined by a "."
	PackageIdentifier string `json:"package_identifier"`
	// The value matching programs.name for the primary app package in osquery
	UniqueIdentifier    string `json:"unique_identifier"`
	InstallScriptPath   string `json:"install_script_path"`
	UninstallScriptPath string `json:"uninstall_script_path"`
	InstallerArch       string `json:"installer_arch"`
	InstallerType       string `json:"installer_type"`
	InstallerScope      string `json:"installer_scope"`
	InstallerLocale     string `json:"installer_locale"`
	ProgramPublisher    string `json:"program_publisher"`
	UninstallType       string `json:"uninstall_type"`
	FuzzyMatchName      bool   `json:"fuzzy_match_name"`
	// Whether to use "no_check" instead of the app's hash (e.g. for non-pinned download URLs)
	IgnoreHash        bool     `json:"ignore_hash"`
	DefaultCategories []string `json:"default_categories"`
	Frozen            bool     `json:"frozen"`
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
	InstallerLocale        string                   `yaml:"InstallerLocale"`
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
	machineScope          = "machine"
	userScope             = "user"
	installerTypeMSI      = "msi"
	installerTypeMSIX     = "msix"
	installerTypeExe      = "exe"
	installerTypeWix      = "wix"
	installerTypeNullSoft = "nullsoft"
	installerTypeInno     = "inno"
	arch64Bit             = "x64"
	arch32Bit             = "x86"
)
