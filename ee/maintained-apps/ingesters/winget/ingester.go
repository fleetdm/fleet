package winget

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"

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
			return nil, ctxerr.WrapWithData(ctx, err, "reading app input file", map[string]any{"fileName": f.Name()})
		}

		var input inputApp
		if err := json.Unmarshal(fileBytes, &input); err != nil {
			return nil, ctxerr.WrapWithData(ctx, err, "unmarshal app input file", map[string]any{"fileName": f.Name()})
		}

		level.Info(logger).Log("msg", "ingesting winget app", "name", input.Name)

		// this is the path within the winget GitHub repo where the manifests are located
		dirPath := path.Join("manifests", string(bytes.ToLower([]byte{input.Vendor[0]})), input.Vendor, input.Name)

		_, contents, _, err := githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			dirPath,
			opts,
		)
		if err != nil {
			return nil, fmt.Errorf("get data from winget repo request: %w", err)
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
		filePath := path.Join(dirPath, latestVersionDir.GetName(), fmt.Sprintf("%s.%s.installer.yaml", input.Vendor, input.Name))

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

		var installerData wingetInstaller
		var installScript, uninstallScript, existsQuery string
		for _, installer := range m.Installers {
			// TODO: handle non-machine scope (aka .exe installers)
			if installer.Scope == machineScope {
				installerData = installer
				installScript = file.GetInstallScript(installer.InstallerType)
				uninstallScript = file.GetUninstallScript(installer.InstallerType)
				existsQuery = fmt.Sprintf("SELECT 1 FROM programs WHERE identifying_number = '%s';", installerData.ProductCode)
			}
		}

		manifestApps = append(manifestApps, &maintained_apps.FMAManifestApp{
			Name:             input.Name,
			Slug:             input.Slug,
			InstallerURL:     installerData.InstallerURL,
			SHA256:           installerData.InstallerSha256,
			Version:          m.PackageVersion,
			UniqueIdentifier: input.UniqueIdentifier,
			Queries: maintained_apps.FMAQueries{
				Exists: existsQuery,
			},
			InstallScript:      installScript,
			UninstallScript:    uninstallScript,
			InstallScriptRef:   maintained_apps.GetScriptRef(installScript),
			UninstallScriptRef: maintained_apps.GetScriptRef(uninstallScript),
		})
	}

	return manifestApps, nil
}

type inputApp struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Vendor           string `json:"vendor"`
	UniqueIdentifier string `json:"unique_identifier"`
}

type wingetManifest struct {
	PackageIdentifier string            `yaml:"PackageIdentifier"`
	PackageVersion    string            `yaml:"PackageVersion"`
	Installers        []wingetInstaller `yaml:"Installers"`
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

const machineScope = "machine"
