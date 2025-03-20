package winget

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
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

	for _, f := range files {

		fileBytes, err := os.ReadFile(path.Join(inputsPath, f.Name()))
		if err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "reading app input file", map[string]any{"file_name": f.Name()})
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "unmarshal app input file", map[string]any{"file_name": f.Name()})
		}

		level.Info(logger).Log("msg", "ingesting winget app", "name", input.Name)

		// this is the path within the winget GitHub repo where the manifests are located
		packageIdentParts := strings.Split(input.PackageIdentifier, ".")
		if len(packageIdentParts) != 2 {
			return nil, ctxerr.NewWithData(ctx, "invalid package identifier for app", map[string]any{"package_identifier": input.PackageIdentifier, "app": input.Name})
		}

		dirPath := path.Join(
			"manifests",
			// string(bytes.ToLower([]byte{input.PackageIdentifier[0]})),
			strings.ToLower(input.PackageIdentifier[:1]),
			packageIdentParts[0],
			packageIdentParts[1],
		)

		_, contents, _, err := githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			dirPath,
			opts,
		)
		if err != nil {
			return nil, fmt.Errorf("get data from winget repo: %w", err)
		}

		// sort the list of directories in descending order
		slices.SortFunc(contents, func(a, b *github.RepositoryContent) int { return feednvd.SmartVerCmp(b.GetName(), a.GetName()) })

		// this directory has the latest version data in it
		latestVersionDir := contents[0]
		if latestVersionDir.GetName() == "" {
			level.Warn(logger).Log("msg", "latest version not found", "app", input.Name)
			continue
		}

		// this is the path to the specific manifest file we need
		filePath := path.Join(
			dirPath,
			latestVersionDir.GetName(),
			fmt.Sprintf("%s.installer.yaml", input.PackageIdentifier),
		)

		fileContents, _, _, err := githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			filePath,
			opts,
		)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting the ")
		}

		manifestContents, err := fileContents.GetContent()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting winget manifest file contents")
		}

		var m wingetManifest
		if err := yaml.Unmarshal([]byte(manifestContents), &m); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshaling winget manifest")
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

		// Walk through the installers and get any data we missed
		for _, installer := range m.Installers {
			if (installer.Scope == machineScope || m.Scope == machineScope) || installer.Architecture == arch64Bit {
				// Use the first machine scoped installer
				installerURL = installer.InstallerURL
				sha256 = installer.InstallerSha256
				installerType := installer.InstallerType
				if installerType == "" {
					// try to get it from the URL
					urlParts := strings.Split(installerURL, ".")
					if len(urlParts) > 1 {
						if urlParts[len(urlParts)-1] == installerTypeMSI {
							installerType = installerTypeMSI
						}
					}
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
		fmt.Printf("productCode: %v\n", productCode)
		out.Queries = maintained_apps.FMAQueries{
			Exists: fmt.Sprintf("SELECT 1 FROM programs WHERE identifying_number = '%s';", productCode),
		}
		out.InstallScript = installScript
		out.UninstallScript = uninstallScript
		out.InstallScriptRef = maintained_apps.GetScriptRef(installScript)
		out.UninstallScriptRef = maintained_apps.GetScriptRef(uninstallScript)

		manifestApps = append(manifestApps, &out)
	}

	return manifestApps, nil
}

type inputApp struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// PackageIdentifier is the identifier used by winget. It's composed of a vendor part (e.g.
	// AgileBits) and an app part (e.g. 1Password), joined by a "."
	PackageIdentifier string `json:"package_identifier"`
	UniqueIdentifier  string `json:"unique_identifier"`
}

type wingetManifest struct {
	PackageIdentifier      string                   `yaml:"PackageIdentifier"`
	PackageVersion         string                   `yaml:"PackageVersion"`
	Installers             []wingetInstaller        `yaml:"Installers"`
	InstallerType          string                   `yaml:"InstallerType"`
	AppsAndFeaturesEntries []appsAndFeaturesEntries `yaml:"AppsAndFeaturesEntries,omitempty"`
	ProductCode            string                   `yaml:"ProductCode"`
	Scope                  string                   `yaml:"Scope"`
}

type wingetInstaller struct {
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

const (
	machineScope     = "machine"
	installerTypeMSI = "msi"
	arch64Bit        = "x64"
)
