package winget

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

//go:embed dependencies.ps1
var dependenciesScriptTemplate string

type dependencies struct {
	PackageDependencies []packageDependency `yaml:"PackageDependencies,omitempty"`
}

type packageDependency struct {
	PackageIdentifier string `yaml:"PackageIdentifier"`
	MinimumVersion    string `yaml:"MinimumVersion,omitempty"`
}

const (
	dependencyKindVCRedist      = "vcredist"
	dependencyKindWebView2      = "webview2"
	dependencyKindDotNetDesktop = "dotnet-desktop"

	// dependencyArchHost picks the installer matching the host's architecture at install time.
	dependencyArchHost = "host"
)

// supportedDependency is a runtime the install script knows how to detect and install
// silently as SYSTEM. Anything else a winget manifest declares is left alone.
type supportedDependency struct {
	kind string
	name string
	// arch is a fixed architecture, dependencyArchHost, or "" for the app's own.
	arch  string
	major string
	// arguments are ours, never the manifest's: they end up in a script that runs as SYSTEM.
	arguments string
	// downloadHosts are the only hosts the pinned installer URL may point at.
	downloadHosts []string
}

var (
	vcRedistDependency      = regexp.MustCompile(`^Microsoft\.VCRedist\.2015\+\.(x64|x86|arm64)$`)
	dotNetDesktopDependency = regexp.MustCompile(`^Microsoft\.DotNet\.DesktopRuntime\.(\d{1,2})$`)

	dependencyVersion = regexp.MustCompile(`^\d+(\.\d+){0,3}$`)
	dependencySHA256  = regexp.MustCompile(`^[0-9A-Fa-f]{64}$`)
	dependencyURLPath = regexp.MustCompile(`^[A-Za-z0-9._~/%+-]+$`)
)

func lookupSupportedDependency(packageIdentifier string) (supportedDependency, bool) {
	if m := vcRedistDependency.FindStringSubmatch(packageIdentifier); m != nil {
		return supportedDependency{
			kind:          dependencyKindVCRedist,
			name:          fmt.Sprintf("Microsoft Visual C++ Redistributable (%s)", m[1]),
			arch:          m[1],
			arguments:     "/install /quiet /norestart",
			downloadHosts: []string{"download.visualstudio.microsoft.com"},
		}, true
	}
	if m := dotNetDesktopDependency.FindStringSubmatch(packageIdentifier); m != nil {
		return supportedDependency{
			kind:          dependencyKindDotNetDesktop,
			name:          fmt.Sprintf("Microsoft .NET Desktop Runtime %s", m[1]),
			major:         m[1],
			arguments:     "/install /quiet /norestart",
			downloadHosts: []string{"download.microsoft.com", "builds.dotnet.microsoft.com", "download.visualstudio.microsoft.com"},
		}, true
	}
	if packageIdentifier == "Microsoft.EdgeWebView2Runtime" {
		return supportedDependency{
			kind:          dependencyKindWebView2,
			name:          "Microsoft Edge WebView2 Runtime",
			arch:          dependencyArchHost,
			arguments:     "/silent /install",
			downloadHosts: []string{"msedge.sf.dl.delivery.mp.microsoft.com"},
		}, true
	}
	return supportedDependency{}, false
}

type dependencyInstaller struct {
	arch   string
	url    string
	sha256 string
}

type resolvedDependency struct {
	supportedDependency
	minVersion string
	installers []dependencyInstaller
}

// resolveDependencies turns the dependencies an app's winget manifest declares into
// pinned installers for the ones the install script can handle.
func (i *wingetIngester) resolveDependencies(ctx context.Context, declared []packageDependency, appArch string) ([]resolvedDependency, error) {
	var resolved []resolvedDependency
	for _, d := range declared {
		supported, ok := lookupSupportedDependency(d.PackageIdentifier)
		if !ok {
			i.logger.WarnContext(ctx, "dependency is not installed by the install script", "dependency", d.PackageIdentifier)
			continue
		}
		if d.MinimumVersion != "" && !dependencyVersion.MatchString(d.MinimumVersion) {
			return nil, ctxerr.Errorf(ctx, "dependency %s has an unsupported minimum version %q", d.PackageIdentifier, d.MinimumVersion)
		}
		if supported.arch == "" {
			supported.arch = appArch
		}

		m, err := i.dependencyManifest(ctx, d.PackageIdentifier)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "getting manifest for dependency %s", d.PackageIdentifier)
		}

		arches := []string{supported.arch}
		if supported.arch == dependencyArchHost {
			arches = []string{arch64Bit, "arm64", arch32Bit}
		}
		dep := resolvedDependency{supportedDependency: supported, minVersion: d.MinimumVersion}
		for _, arch := range arches {
			inst, err := selectDependencyInstaller(m, arch, supported.downloadHosts)
			if err != nil {
				return nil, ctxerr.Wrapf(ctx, err, "selecting %s installer for dependency %s", arch, d.PackageIdentifier)
			}
			dep.installers = append(dep.installers, inst)
		}

		// Some manifests list the same runtime for several architectures; keep one of each.
		if !slices.ContainsFunc(resolved, func(r resolvedDependency) bool {
			return r.kind == dep.kind && r.arch == dep.arch && r.major == dep.major
		}) {
			resolved = append(resolved, dep)
		}
	}
	return resolved, nil
}

func (i *wingetIngester) dependencyManifest(ctx context.Context, packageIdentifier string) (installerManifest, error) {
	if m, ok := i.dependencyManifests[packageIdentifier]; ok {
		return m, nil
	}
	m, _, err := i.latestInstallerManifest(ctx, packageIdentifier)
	if err != nil {
		return installerManifest{}, err
	}
	if i.dependencyManifests == nil {
		i.dependencyManifests = make(map[string]installerManifest)
	}
	i.dependencyManifests[packageIdentifier] = m
	return m, nil
}

func selectDependencyInstaller(m installerManifest, arch string, allowedHosts []string) (dependencyInstaller, error) {
	for _, inst := range m.Installers {
		scope := inst.Scope
		if scope == "" {
			scope = m.Scope
		}
		if inst.Architecture != arch || scope == userScope {
			continue
		}

		u, err := url.Parse(inst.InstallerURL)
		if err != nil {
			return dependencyInstaller{}, fmt.Errorf("parsing installer URL: %w", err)
		}
		if u.Scheme != "https" || u.User != nil || !slices.Contains(allowedHosts, strings.ToLower(u.Host)) ||
			u.RawQuery != "" || u.Fragment != "" || !dependencyURLPath.MatchString(u.EscapedPath()) {
			return dependencyInstaller{}, fmt.Errorf("installer URL %q is not an expected download location", inst.InstallerURL)
		}
		if !dependencySHA256.MatchString(inst.InstallerSha256) {
			return dependencyInstaller{}, fmt.Errorf("installer SHA-256 %q is not valid", inst.InstallerSha256)
		}
		return dependencyInstaller{arch: arch, url: u.String(), sha256: strings.ToUpper(inst.InstallerSha256)}, nil
	}
	return dependencyInstaller{}, fmt.Errorf("no machine-scope %s installer in version %s", arch, m.PackageVersion)
}

// prependDependencies returns installScript preceded by a block that installs any
// missing dependency before the app's own installer runs.
func prependDependencies(installScript, appName string, deps []resolvedDependency) string {
	if len(deps) == 0 {
		return installScript
	}

	var entries strings.Builder
	for _, d := range deps {
		var installers []string
		for _, inst := range d.installers {
			installers = append(installers, fmt.Sprintf("%s = @{ Url = %s; Sha256 = %s }",
				inst.arch, psQuote(inst.url), psQuote(inst.sha256)))
		}
		fmt.Fprintf(&entries,
			"        @{ Name = %s; Kind = %s; Arch = %s; Major = %s; MinVersion = %s; Arguments = %s\n"+
				"           Installers = @{ %s } }\n",
			psQuote(d.name), psQuote(d.kind), psQuote(d.arch), psQuote(d.major), psQuote(d.minVersion),
			psQuote(d.arguments), strings.Join(installers, "; "))
	}

	block := strings.NewReplacer(
		"{{APP_NAME}}", psQuote(appName),
		"{{DEPENDENCIES}}", strings.TrimRight(entries.String(), "\n"),
	).Replace(dependenciesScriptTemplate)
	return block + installScript
}

// psQuote returns s as a single-quoted PowerShell string literal.
func psQuote(s string) string {
	// PowerShell also treats typographic single quotes as string delimiters.
	return "'" + strings.NewReplacer("'", "''", "‘", "‘‘", "’", "’’", "‚", "‚‚", "‛", "‛‛").Replace(s) + "'"
}
