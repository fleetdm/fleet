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

		// TODO: fully implement this ingester, right now it's just a stub/noop

		// get the data from the repo in github
		githubHTTPClient := fleethttp.NewGithubClient()
		githubClient := github.NewClient(githubHTTPClient)

		_, contents, _, err := githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			// TODO(JVE): make this path calculation a function
			fmt.Sprintf("manifests/%s/%s/%s", string(bytes.ToLower([]byte{input.Vendor[0]})), input.Vendor, input.Name),
			&github.RepositoryContentGetOptions{
				Ref: "master",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("get data from winget repo request: %w", err)
		}

		slices.SortFunc(contents, func(a, b *github.RepositoryContent) int { return feednvd.SmartVerCmp(b.GetName(), a.GetName()) })

		latestVersionDir := contents[0]

		fileContents, _, _, err := githubClient.Repositories.GetContents(ctx,
			"microsoft",
			"winget-pkgs",
			// TODO(JVE): make this path calculation a function
			fmt.Sprintf("manifests/%s/%s/%s/%s/%s.%s.installer.yaml", string(bytes.ToLower([]byte{input.Vendor[0]})), input.Vendor, input.Name, latestVersionDir.GetName(), input.Vendor, input.Name),
			&github.RepositoryContentGetOptions{
				Ref: "master",
			},
		)
		if err != nil {
			return nil, fmt.Errorf("get data from winget repo request: %w", err)
		}

		fmt.Printf("fileContents.GetName(): %v\n", fileContents.GetName())
		manifestContents, err := fileContents.GetContent()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting winget manifest file contents")
		}

		var m wingetManifest
		if err := yaml.Unmarshal([]byte(manifestContents), &m); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshaling winget manifest")
		}

		var installerData wingetInstaller
		for _, installer := range m.Installers {
			// TODO(JVE): handle the user scope case (e.g. Notion)
			if installer.Scope == machineScope {
				// fmt.Printf("machine scoped installer.InstallerURL: %v\n", installer.InstallerURL)
				// installerURL = installer.InstallerURL
				installerData = installer
			}
		}

		manifestApps = append(manifestApps, &maintained_apps.FMAManifestApp{
			Name:             input.Name,
			Slug:             input.Slug,
			InstallerURL:     installerData.InstallerURL,
			SHA256:           installerData.InstallerSha256,
			Version:          m.PackageVersion,
			UniqueIdentifier: m.PackageIdentifier, // TODO(JVE): is this true?
		})

	}

	return manifestApps, nil
}

type inputApp struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Vendor string `json:"vendor"`
}

type wingetManifest struct {
	PackageIdentifier string            `yaml:"PackageIdentifier"`
	PackageVersion    string            `yaml:"PackageVersion"`
	Installers        []wingetInstaller `yaml:"Installers"`
}

type wingetInstaller struct {
	Architecture           string                   `yaml:"Architecture"`
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
