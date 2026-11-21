package winget

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	testVCRedistURL = "https://download.visualstudio.microsoft.com/download/pr/1/VC_redist.x64.exe"
	testVCRedistSHA = "9C02C9ED120F671B900E6B9471D890E3B2414117A7E53F4AE2E855466FA2B7C9"
)

func TestLookupSupportedDependency(t *testing.T) {
	cases := []struct {
		id        string
		supported bool
		kind      string
		arch      string
		major     string
	}{
		{id: "Microsoft.VCRedist.2015+.x64", supported: true, kind: dependencyKindVCRedist, arch: "x64"},
		{id: "Microsoft.VCRedist.2015+.arm64", supported: true, kind: dependencyKindVCRedist, arch: "arm64"},
		{id: "Microsoft.DotNet.DesktopRuntime.8", supported: true, kind: dependencyKindDotNetDesktop, major: "8"},
		{id: "Microsoft.DotNet.DesktopRuntime.10", supported: true, kind: dependencyKindDotNetDesktop, major: "10"},
		{id: "Microsoft.EdgeWebView2Runtime", supported: true, kind: dependencyKindWebView2, arch: dependencyArchHost},
		{id: "Microsoft.VCRedist.2010.x86"},
		{id: "Microsoft.VCRedist.2015+.x64.Extra"},
		{id: "Microsoft.DotNet.Runtime.8"},
		{id: "Git.Git"},
	}
	for _, tt := range cases {
		t.Run(tt.id, func(t *testing.T) {
			got, ok := lookupSupportedDependency(tt.id)
			require.Equal(t, tt.supported, ok)
			require.Equal(t, tt.kind, got.kind)
			require.Equal(t, tt.arch, got.arch)
			require.Equal(t, tt.major, got.major)
		})
	}
}

func TestSelectDependencyInstaller(t *testing.T) {
	hosts := []string{"download.visualstudio.microsoft.com"}
	manifest := func(installers ...installer) installerManifest {
		return installerManifest{PackageVersion: "14.0", Installers: installers}
	}

	t.Run("picks the machine-scope installer for the architecture", func(t *testing.T) {
		got, err := selectDependencyInstaller(manifest(
			installer{Architecture: "arm64", InstallerURL: "https://download.visualstudio.microsoft.com/arm64.exe", InstallerSha256: testVCRedistSHA},
			installer{Architecture: "x64", Scope: "user", InstallerURL: "https://download.visualstudio.microsoft.com/user.exe", InstallerSha256: testVCRedistSHA},
			installer{Architecture: "x64", Scope: "machine", InstallerURL: testVCRedistURL, InstallerSha256: strings.ToLower(testVCRedistSHA)},
		), "x64", hosts)
		require.NoError(t, err)
		require.Equal(t, dependencyInstaller{arch: "x64", url: testVCRedistURL, sha256: testVCRedistSHA}, got)
	})

	t.Run("no installer for the architecture", func(t *testing.T) {
		_, err := selectDependencyInstaller(manifest(
			installer{Architecture: "arm64", InstallerURL: testVCRedistURL, InstallerSha256: testVCRedistSHA},
		), "x64", hosts)
		require.ErrorContains(t, err, "no machine-scope x64 installer")
	})

	rejected := map[string]installer{
		"plain http":      {InstallerURL: "http://download.visualstudio.microsoft.com/a.exe", InstallerSha256: testVCRedistSHA},
		"unexpected host": {InstallerURL: "https://example.com/a.exe", InstallerSha256: testVCRedistSHA},
		"lookalike host":  {InstallerURL: "https://download.visualstudio.microsoft.com.example.com/a.exe", InstallerSha256: testVCRedistSHA},
		"credentials":     {InstallerURL: "https://user@download.visualstudio.microsoft.com/a.exe", InstallerSha256: testVCRedistSHA},
		"query string":    {InstallerURL: "https://download.visualstudio.microsoft.com/a.exe?x=1", InstallerSha256: testVCRedistSHA},
		"quote in path":   {InstallerURL: "https://download.visualstudio.microsoft.com/a'; calc; '.exe", InstallerSha256: testVCRedistSHA},
		"subexpression":   {InstallerURL: "https://download.visualstudio.microsoft.com/$(calc).exe", InstallerSha256: testVCRedistSHA},
		"short hash":      {InstallerURL: testVCRedistURL, InstallerSha256: "ABCD"},
		"non-hex hash":    {InstallerURL: testVCRedistURL, InstallerSha256: strings.Repeat("Z", 64)},
	}
	for name, inst := range rejected {
		t.Run("rejects "+name, func(t *testing.T) {
			inst.Architecture = "x64"
			_, err := selectDependencyInstaller(manifest(inst), "x64", hosts)
			require.Error(t, err)
		})
	}
}

func TestPrependDependencies(t *testing.T) {
	require.Equal(t, "install", prependDependencies("install", "Foo", nil))

	dep, ok := lookupSupportedDependency("Microsoft.VCRedist.2015+.x64")
	require.True(t, ok)
	script := prependDependencies("install", "Foo's ‘App’", []resolvedDependency{{
		supportedDependency: dep,
		minVersion:          "14.42.0.0",
		installers:          []dependencyInstaller{{arch: "x64", url: testVCRedistURL, sha256: testVCRedistSHA}},
	}})

	require.True(t, strings.HasSuffix(script, "\ninstall"), "the app's own script must follow the dependency block")
	require.Contains(t, script, "$appName = 'Foo''s ‘‘App’’'")
	require.Contains(t, script, "Kind = 'vcredist'; Arch = 'x64'; Major = ''; MinVersion = '14.42.0.0'; Arguments = '/install /quiet /norestart'")
	require.Contains(t, script, "Installers = @{ x64 = @{ Url = '"+testVCRedistURL+"'; Sha256 = '"+testVCRedistSHA+"' } }")
	require.NotContains(t, script, "{{")
}

func TestIngestDependencies(t *testing.T) {
	tempDir := t.TempDir()
	installScript := path.Join(tempDir, "install_script.ps1")
	uninstallScript := path.Join(tempDir, "uninstall_script.ps1")
	require.NoError(t, os.WriteFile(installScript, []byte("app install"), 0o644))
	require.NoError(t, os.WriteFile(uninstallScript, []byte("app uninstall"), 0o644))

	const (
		webViewX64   = "https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/1/X64.exe"
		webViewARM64 = "https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/2/ARM64.exe"
		webViewX86   = "https://msedge.sf.dl.delivery.mp.microsoft.com/filestreamingservice/files/3/X86.exe"
	)

	var appManifest, vcRedistManifest installerManifest
	requests := map[string]int{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "/repos/microsoft/winget-pkgs/contents/manifests/"
		p := strings.TrimPrefix(r.URL.Path, prefix)
		requests[p]++

		writeDir := func() {
			content := []github.RepositoryContent{{Name: new("1.0"), Type: new("dir")}}
			if err := json.NewEncoder(w).Encode(content); err != nil {
				t.Errorf("encoding fixture: %v", err)
			}
		}
		writeYAML := func(v any) {
			bytes, err := yaml.Marshal(v)
			if err != nil {
				t.Errorf("marshaling fixture: %v", err)
				return
			}
			str := string(bytes)
			if err := json.NewEncoder(w).Encode(&github.RepositoryContent{Name: new("f"), Content: &str}); err != nil {
				t.Errorf("encoding fixture: %v", err)
			}
		}

		switch p {
		case "f/Foo", "m/Microsoft/VCRedist/2015+/x64", "m/Microsoft/EdgeWebView2Runtime":
			writeDir()
		case "f/Foo/1.0/Foo.installer.yaml":
			writeYAML(appManifest)
		case "f/Foo/1.0/Foo.locale.en-US.yaml":
			writeYAML(localeManifest{PackageName: "foo", Publisher: "Bar, Inc."})
		case "m/Microsoft/VCRedist/2015+/x64/1.0/Microsoft.VCRedist.2015+.x64.installer.yaml":
			writeYAML(vcRedistManifest)
		case "m/Microsoft/EdgeWebView2Runtime/1.0/Microsoft.EdgeWebView2Runtime.installer.yaml":
			writeYAML(installerManifest{PackageVersion: "1.0", Installers: []installer{
				{Architecture: "x64", Scope: "user", InstallerURL: webViewX64, InstallerSha256: strings.Repeat("0", 64)},
				{Architecture: "x64", Scope: "machine", InstallerURL: webViewX64, InstallerSha256: strings.Repeat("1", 64)},
				{Architecture: "arm64", Scope: "machine", InstallerURL: webViewARM64, InstallerSha256: strings.Repeat("2", 64)},
				{Architecture: "x86", Scope: "machine", InstallerURL: webViewX86, InstallerSha256: strings.Repeat("3", 64)},
			}})
		default:
			w.WriteHeader(http.StatusNotFound)
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	newIngester := func() *wingetIngester {
		gc := github.NewClient(srv.Client())
		u, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		gc.BaseURL = u
		return &wingetIngester{logger: slog.New(slog.DiscardHandler), githubClient: gc}
	}

	input := inputApp{
		Name:                "Foo",
		UniqueIdentifier:    "Foo",
		PackageIdentifier:   "Foo",
		Slug:                "foo/windows",
		InstallerArch:       "x64",
		InstallerType:       "exe",
		InstallerScope:      "machine",
		InstallScriptPath:   installScript,
		UninstallScriptPath: uninstallScript,
	}
	appWith := func(root, perInstaller []packageDependency) installerManifest {
		return installerManifest{
			InstallerType:  "exe",
			PackageVersion: "1.0",
			Scope:          "machine",
			Dependencies:   dependencies{PackageDependencies: root},
			Installers: []installer{{
				Architecture:    "x64",
				InstallerURL:    "https://example.com/foo.exe",
				InstallerSha256: "AAAA",
				Dependencies:    dependencies{PackageDependencies: perInstaller},
			}},
		}
	}
	vcRedistManifest = installerManifest{PackageVersion: "14.52", Scope: "machine", Installers: []installer{
		{Architecture: "x64", InstallerURL: testVCRedistURL, InstallerSha256: testVCRedistSHA},
	}}

	t.Run("no dependencies leaves the script alone", func(t *testing.T) {
		appManifest = appWith(nil, nil)
		out, err := newIngester().ingestOne(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, "app install", out.InstallScript)
	})

	t.Run("supported dependencies are pinned ahead of the app's script", func(t *testing.T) {
		appManifest = appWith(nil, []packageDependency{
			{PackageIdentifier: "Microsoft.VCRedist.2015+.x64", MinimumVersion: "14.42.34433.0"},
			{PackageIdentifier: "Git.Git", MinimumVersion: "2.24.1.2"},
			{PackageIdentifier: "Microsoft.EdgeWebView2Runtime"},
			{PackageIdentifier: "Microsoft.VCRedist.2015+.x64"},
		})
		i := newIngester()
		out, err := i.ingestOne(t.Context(), input)
		require.NoError(t, err)

		require.True(t, strings.HasSuffix(out.InstallScript, "\napp install"))
		require.Contains(t, out.InstallScript, "$appName = 'Foo'")
		require.Contains(t, out.InstallScript, "MinVersion = '14.42.34433.0'")
		require.Contains(t, out.InstallScript, "x64 = @{ Url = '"+testVCRedistURL+"'; Sha256 = '"+testVCRedistSHA+"' }")
		require.Equal(t, 1, strings.Count(out.InstallScript, "Kind = 'vcredist'"), "a repeated dependency is listed once")
		// WebView2 follows the host's architecture, so every machine-scope build is pinned.
		require.Contains(t, out.InstallScript, "Kind = 'webview2'; Arch = 'host'")
		require.Contains(t, out.InstallScript, "x64 = @{ Url = '"+webViewX64+"'; Sha256 = '"+strings.Repeat("1", 64)+"' }")
		require.Contains(t, out.InstallScript, "arm64 = @{ Url = '"+webViewARM64+"'")
		require.Contains(t, out.InstallScript, "x86 = @{ Url = '"+webViewX86+"'")
		require.NotContains(t, out.InstallScript, "Git")
		require.Equal(t, "app uninstall", out.UninstallScript)

		before := requests["m/Microsoft/VCRedist/2015+/x64"]
		_, err = i.ingestOne(t.Context(), input)
		require.NoError(t, err)
		require.Equal(t, before, requests["m/Microsoft/VCRedist/2015+/x64"], "a dependency's manifest is fetched once per run")
	})

	t.Run("an installer's own dependencies replace the manifest-wide ones", func(t *testing.T) {
		appManifest = appWith(
			[]packageDependency{{PackageIdentifier: "Microsoft.EdgeWebView2Runtime"}},
			[]packageDependency{{PackageIdentifier: "Microsoft.VCRedist.2015+.x64"}},
		)
		out, err := newIngester().ingestOne(t.Context(), input)
		require.NoError(t, err)
		require.Contains(t, out.InstallScript, "Kind = 'vcredist'")
		require.NotContains(t, out.InstallScript, "Kind = 'webview2'")

		appManifest = appWith([]packageDependency{{PackageIdentifier: "Microsoft.EdgeWebView2Runtime"}}, nil)
		out, err = newIngester().ingestOne(t.Context(), input)
		require.NoError(t, err)
		require.Contains(t, out.InstallScript, "Kind = 'webview2'")
	})

	t.Run("a malformed minimum version fails the app", func(t *testing.T) {
		appManifest = appWith(nil, []packageDependency{{PackageIdentifier: "Microsoft.VCRedist.2015+.x64", MinimumVersion: "14'; calc; '"}})
		_, err := newIngester().ingestOne(t.Context(), input)
		require.ErrorContains(t, err, "unsupported minimum version")
	})

	t.Run("a dependency installer from an unexpected host fails the app", func(t *testing.T) {
		appManifest = appWith(nil, []packageDependency{{PackageIdentifier: "Microsoft.VCRedist.2015+.x64"}})
		good := vcRedistManifest
		t.Cleanup(func() { vcRedistManifest = good })
		vcRedistManifest = installerManifest{PackageVersion: "14.52", Scope: "machine", Installers: []installer{
			{Architecture: "x64", InstallerURL: "https://example.com/VC_redist.x64.exe", InstallerSha256: testVCRedistSHA},
		}}
		_, err := newIngester().ingestOne(t.Context(), input)
		require.ErrorContains(t, err, "not an expected download location")
	})
}
