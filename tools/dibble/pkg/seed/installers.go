package seed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// installerSource describes where a curated installer fixture is fetched from
// and the SHA-256 the downloaded bytes must match. Fixtures are no longer
// committed to the repo; dibble downloads them on demand — only when seeding
// software — and caches them under the user cache dir so repeated runs stay
// offline.
type installerSource struct {
	url    string
	sha256 string
}

// testdataRef pins the fleet commit whose
// server/service/testdata/software-installers/ fixtures dibble reuses. These
// are the same package files Fleet's own tests exercise; serving them from
// raw.githubusercontent.com keeps a single source of truth and avoids
// committing binaries in this module.
const testdataRef = "8c85ef8ad3b1c67ca13486791f3b3e0cae52c565"

func testdataURL(name string) string {
	return "https://raw.githubusercontent.com/fleetdm/fleet/" + testdataRef +
		"/server/service/testdata/software-installers/" + name
}

// installerSources maps each curated fixture filename to its download source.
// The .msi and .exe entries use upstream-signed installers (python-manager,
// 7-Zip) so we exercise the Windows code paths without surfacing the Fleet
// agent itself as a custom software item; everything else reuses Fleet's
// committed test fixtures via raw GitHub.
var installerSources = map[string]installerSource{
	"7z2601.exe":              {"https://www.7-zip.org/a/7z2601.exe", "615976598f800c70827c5a47e68c2b0d2b17d048b9721ba071c8af825d2476bd"},
	"7z2601-x64.exe":          {"https://www.7-zip.org/a/7z2601-x64.exe", "d64a0468f5b5b0b0fc5b2188450bcd655b70809d97b1c4535f2884635094377d"},
	"7z2601-arm64.exe":        {"https://www.7-zip.org/a/7z2601-arm64.exe", "1fecf4e3407950939c8ffcc3e42e3039821997dea155301c75369474e5f15175"},
	"python-manager-26.2.msi": {"https://www.python.org/ftp/python/pymanager/python-manager-26.2.msi", "d2f494cafe16a40ab9d4ffb1b6c211813cfdb0b0291639676506e76ce93a271b"},
	"dummy_installer.pkg":     {testdataURL("dummy_installer.pkg"), "7f679541ccfdb56094ca76117fd7cf75071c9d8f43bfd2a6c0871077734ca7c8"},
	"EchoApp.pkg":             {testdataURL("EchoApp.pkg"), "1e83a94b801db429398b95a11f76fc5ba0e8643cb027b40a2b890592761f48f9"},
	"no_version.pkg":          {testdataURL("no_version.pkg"), "4ba383be20c1020e416958ab10e3b472a4d5532a8cd94ed720d495a9c81958fe"},
	"emacs.deb":               {testdataURL("emacs.deb"), "f2697bf4eb0418914a2f0df3dc5c17b58eb8720641cee852cd88566d40e7eaa9"},
	"ruby.deb":                {testdataURL("ruby.deb"), "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628"},
	"ruby_arm64.deb":          {testdataURL("ruby_arm64.deb"), "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628"},
	"ruby.rpm":                {testdataURL("ruby.rpm"), "3cc3e38fe8656117161fb52976eea29c8a7839b3cbe719c2c4a42b64187b5042"},
	"test.tar.gz":             {testdataURL("test.tar.gz"), "06874d845f5a7f39413c9ad562d48d334a820e6e55ad8762c78b2c3d609d0f3b"},
	"ipa_test.ipa":            {testdataURL("ipa_test.ipa"), "1dbbaf76f371ecb4c3dcdcfb53b8915b09ffe6c812586105e1ef1d421eb6fd6b"},
	"ipa_test2.ipa":           {testdataURL("ipa_test2.ipa"), "1dbbaf76f371ecb4c3dcdcfb53b8915b09ffe6c812586105e1ef1d421eb6fd6b"},
}

// installerCacheDir returns the directory dibble caches downloaded installer
// fixtures in, creating it if needed. Falls back to the OS temp dir when the
// user cache dir is unavailable.
func installerCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	dir := filepath.Join(base, "dibble", "installers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create installer cache dir: %w", err)
	}
	return dir, nil
}

// loadInstaller returns the bytes of a curated installer fixture, downloading
// and caching it on first use. A cached file whose SHA-256 matches the
// manifest is reused without hitting the network; otherwise the fixture is
// (re)downloaded, verified, and written to the cache.
func loadInstaller(log Logger, name string) ([]byte, error) {
	src, ok := installerSources[name]
	if !ok {
		return nil, fmt.Errorf("unknown installer fixture %q", name)
	}

	dir, err := installerCacheDir()
	if err != nil {
		return nil, err
	}
	cached := filepath.Join(dir, name)

	if b, err := os.ReadFile(cached); err == nil && sha256Hex(b) == src.sha256 {
		return b, nil
	}

	log.Printf("downloading installer fixture %s", name)
	b, err := downloadInstaller(src)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", name, err)
	}

	// Write atomically so a partial or interrupted download never poisons the
	// cache for the next run.
	tmp := cached + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return nil, fmt.Errorf("cache %s: %w", name, err)
	}
	if err := os.Rename(tmp, cached); err != nil {
		return nil, fmt.Errorf("cache %s: %w", name, err)
	}
	return b, nil
}

// downloadInstaller fetches src and returns its bytes only if they match the
// expected checksum, so a moved or tampered upstream artifact is rejected
// rather than uploaded to Fleet.
func downloadInstaller(src installerSource) ([]byte, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(src.url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s from %s", resp.Status, src.url)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if got := sha256Hex(b); got != src.sha256 {
		return nil, fmt.Errorf("checksum mismatch for %s: got %s want %s", src.url, got, src.sha256)
	}
	return b, nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
