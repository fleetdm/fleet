package winget

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// maxVendorRedirects bounds the redirect chain followed from a latest_installer_url.
	maxVendorRedirects = 5
	// vendorDownloadTimeout keeps a stalled download from hanging the whole ingest run.
	vendorDownloadTimeout = 30 * time.Minute
)

func validateSource(input inputApp) error {
	switch {
	case input.PackageIdentifier != "" && input.LatestInstallerURL != "":
		return errors.New("package_identifier and latest_installer_url can't both be set")
	case input.PackageIdentifier == "" && input.LatestInstallerURL == "":
		return errors.New("missing package identifier for app")
	case input.LatestInstallerURL != "" && !strings.HasPrefix(input.LatestInstallerURL, "https://"):
		return errors.New("latest_installer_url must be an https URL")
	case input.LatestInstallerURL != "" && input.ProgramPublisher == "":
		// There's no winget locale manifest to take the publisher from.
		return errors.New("latest_installer_url requires program_publisher")
	case input.LatestInstallerURL != "" && input.InstallerType != installerTypeMSI && input.InstallerType != installerTypeExe:
		return errors.New("latest_installer_url supports installer_type msi or exe")
	}
	return nil
}

// transientVendorError is a vendor failure that's likely to clear by a later run: a
// rate limit, a server error, or a network failure.
type transientVendorError struct {
	url    string
	status int
	cause  error
}

func (e *transientVendorError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("vendor request to %s failed: %v", e.url, e.cause)
	}
	return fmt.Sprintf("vendor request to %s returned %d", e.url, e.status)
}

func (e *transientVendorError) Unwrap() error { return e.cause }

// asTransientNetworkError marks connection failures, timeouts, and truncated downloads
// as transient, and returns any other error unchanged.
func asTransientNetworkError(rawURL string, err error) error {
	if _, ok := errors.AsType[net.Error](err); ok || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, context.DeadlineExceeded) {
		return &transientVendorError{url: rawURL, cause: err}
	}
	return err
}

func isTransientVendorError(err error) bool {
	_, ok := errors.AsType[*transientVendorError](err)
	return ok
}

func checkVendorStatus(rawURL string, resp *http.Response) error {
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return &transientVendorError{url: rawURL, status: resp.StatusCode}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("vendor request to %s returned %d", rawURL, resp.StatusCode)
	}
	return nil
}

// vendorInstaller is what the ingester needs from a vendor installer file.
type vendorInstaller struct {
	version     string
	sha256      string
	productCode string
	upgradeCode string
}

// vendorManifests builds the manifests ingestOne needs from the installer that
// input.LatestInstallerURL currently redirects to.
func (i *wingetIngester) vendorManifests(ctx context.Context, input inputApp) (installerManifest, localeManifest, error) {
	installerURL, err := i.resolveLatestInstallerURL(ctx, input.LatestInstallerURL)
	if err != nil {
		return installerManifest{}, localeManifest{}, err
	}

	inst, err := i.cachedVendorInstaller(input, installerURL)
	if err != nil {
		return installerManifest{}, localeManifest{}, ctxerr.Wrap(ctx, err, "reading current output")
	}
	if inst == nil {
		i.logger.InfoContext(ctx, "downloading vendor installer", "url", installerURL)
		inst, err = i.downloadVendorInstaller(ctx, installerURL, input.InstallerType)
		if err != nil {
			return installerManifest{}, localeManifest{}, err
		}
	}

	m := installerManifest{
		PackageVersion: inst.version,
		ProductCode:    inst.productCode,
		Installers: []installer{{
			Architecture:    input.InstallerArch,
			InstallerType:   input.InstallerType,
			Scope:           input.InstallerScope,
			InstallerLocale: input.InstallerLocale,
			InstallerURL:    installerURL,
			InstallerSha256: inst.sha256,
		}},
	}
	if inst.upgradeCode != "" {
		m.AppsAndFeaturesEntries = []appsAndFeaturesEntries{{UpgradeCode: inst.upgradeCode}}
	}
	l := localeManifest{PackageName: input.Name, Publisher: input.ProgramPublisher}
	return m, l, nil
}

// resolveLatestInstallerURL follows the vendor's redirect to the versioned installer
// URL. The latest URL itself is rejected as the installer URL: its content changes
// with every release, so the output's SHA-256 would go stale.
func (i *wingetIngester) resolveLatestInstallerURL(ctx context.Context, latestURL string) (string, error) {
	client := *i.vendorClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

	current := latestURL
	for hops := 0; ; hops++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, current, nil)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "creating vendor request")
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", ctxerr.Wrap(ctx, asTransientNetworkError(current, err), "requesting vendor installer URL")
		}
		resp.Body.Close()

		if resp.StatusCode < 300 || resp.StatusCode >= 400 {
			if err := checkVendorStatus(current, resp); err != nil {
				return "", ctxerr.Wrap(ctx, err, "resolving latest installer URL")
			}
			if hops == 0 {
				return "", ctxerr.Errorf(ctx, "latest_installer_url %s didn't redirect to a versioned installer", latestURL)
			}
			return current, nil
		}

		if hops >= maxVendorRedirects {
			return "", ctxerr.Errorf(ctx, "latest_installer_url %s redirected more than %d times", latestURL, maxVendorRedirects)
		}
		loc, err := resp.Location()
		if err != nil {
			return "", ctxerr.Wrapf(ctx, err, "reading redirect from %s", current)
		}
		if loc.Scheme != "https" {
			return "", ctxerr.Errorf(ctx, "latest_installer_url %s redirected to non-HTTPS URL %s", latestURL, loc)
		}
		current = loc.String()
	}
}

// cachedVendorInstaller returns the installer recorded in the app's current output when
// it already points at installerURL, or nil when the installer has to be downloaded.
func (i *wingetIngester) cachedVendorInstaller(input inputApp, installerURL string) (*vendorInstaller, error) {
	b, err := os.ReadFile(path.Join(i.outputPath, input.Slug+".json"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out maintained_apps.FMAManifestFile
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal %s output: %w", input.Slug, err)
	}
	if len(out.Versions) == 0 {
		return nil, nil
	}
	v := out.Versions[0]
	if v.InstallerURL != installerURL || v.Version == "" || v.SHA256 == "" || v.SHA256 == "no_check" {
		return nil, nil
	}
	// The output doesn't record an MSI's product code, so reuse it only when the generated
	// uninstall script goes by upgrade code instead.
	if input.InstallerType == installerTypeMSI && (v.UpgradeCode == "" || input.UninstallScriptPath != "") {
		return nil, nil
	}
	return &vendorInstaller{version: v.Version, sha256: v.SHA256, upgradeCode: v.UpgradeCode}, nil
}

// downloadVendorInstaller downloads the installer to a temporary file and reads its
// version (ProductVersion), SHA-256, and for an MSI its product and upgrade codes.
func (i *wingetIngester) downloadVendorInstaller(ctx context.Context, installerURL, installerType string) (*vendorInstaller, error) {
	ctx, cancel := context.WithTimeout(ctx, vendorDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, installerURL, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating vendor installer request")
	}
	resp, err := i.vendorClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, asTransientNetworkError(installerURL, err), "downloading vendor installer")
	}
	defer resp.Body.Close()
	if err := checkVendorStatus(installerURL, resp); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "downloading vendor installer")
	}

	tfr, err := fleet.NewTempFileReader(resp.Body, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, asTransientNetworkError(installerURL, err), "saving vendor installer")
	}
	defer tfr.Close()

	meta, err := file.ExtractInstallerMetadata(tfr)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "reading vendor installer metadata from %s", installerURL)
	}
	if meta.Extension != installerType {
		return nil, ctxerr.Errorf(ctx, "vendor installer %s is an %s file, not %s", installerURL, meta.Extension, installerType)
	}
	inst := &vendorInstaller{
		version:     strings.TrimSpace(meta.Version),
		sha256:      hex.EncodeToString(meta.SHASum),
		upgradeCode: meta.UpgradeCode,
	}
	if inst.version == "" {
		return nil, ctxerr.Errorf(ctx, "vendor installer %s has no ProductVersion", installerURL)
	}
	if installerType == installerTypeMSI && len(meta.PackageIDs) > 0 {
		inst.productCode = meta.PackageIDs[0]
	}
	return inst, nil
}
