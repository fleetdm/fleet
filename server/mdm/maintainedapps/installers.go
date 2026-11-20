package maintained_apps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/version"
)

// InstallerTimeout is the timeout duration for downloading and adding a maintained app.
const InstallerTimeout = 15 * time.Minute

// DownloadInstaller downloads the maintained app installer located at the given URL.
func DownloadInstaller(ctx context.Context, installerURL string, client *http.Client) (*fleet.TempFileReader, string, error) {
	// validate the URL before doing the request
	u, err := url.ParseRequestURI(installerURL)
	if err != nil {
		return nil, "", fleet.NewInvalidArgumentError(
			"fleet_maintained_app.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) is invalid", installerURL),
		)
	}

	tfr, filename, isHTML, err := downloadInstaller(ctx, installerURL, "", client)
	if err != nil {
		return nil, "", err
	}
	// Some vendors (jamovi) serve a web page unless the download is linked from
	// their own site. Sending a Referer up front isn't safe: SourceForge answers
	// it with its HTML download page instead of redirecting to a mirror.
	if isHTML && u.Host != "" {
		_ = tfr.Close()
		tfr, filename, isHTML, err = downloadInstaller(ctx, installerURL, u.Scheme+"://"+u.Host+"/", client)
		if err != nil {
			return nil, "", err
		}
	}
	// Otherwise a moved or removed download would surface as a hash mismatch.
	if isHTML {
		_ = tfr.Close()
		return nil, "", fleet.NewInvalidArgumentError(
			"fleet_maintained_app.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) returned a web page instead of an installer. The vendor may have moved or removed this download.", installerURL),
		)
	}

	return tfr, filename, nil
}

func downloadInstaller(ctx context.Context, installerURL, referer string, client *http.Client) (tfr *fleet.TempFileReader, filename string, isHTML bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, installerURL, nil)
	if err != nil {
		return nil, "", false, ctxerr.Wrapf(ctx, err, "creating request for URL %s", installerURL)
	}
	// Vendor CDNs disagree on which clients to allow: some Cloudflare-fronted
	// hosts 403 a User-Agent that starts with "Go-http-client", while Akamai
	// (downloads.tableau.com) 403s any User-Agent without a known HTTP library
	// token. Leading with fleet/<version> and keeping the Go token satisfies both.
	req.Header.Set("User-Agent", installerUserAgent())
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", false, ctxerr.Wrapf(ctx, err, "performing request for URL %s", installerURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", false, fleet.NewInvalidArgumentError(
			"fleet_maintained_app.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) doesn't exist. Please make sure that URLs are publicy accessible to the internet.", installerURL),
		)
	}

	// Allow all 2xx and 3xx status codes in this pass.
	if resp.StatusCode > 400 {
		return nil, "", false, fleet.NewInvalidArgumentError(
			"fleet_maintained_app.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) received response status code %d.", installerURL, resp.StatusCode),
		)
	}

	tfr, err = fleet.NewTempFileReader(resp.Body, nil)
	if err != nil {
		return nil, "", false, ctxerr.Wrapf(ctx, err, "reading installer %q contents", installerURL)
	}

	isHTML, err = isHTMLFile(tfr)
	if err != nil {
		_ = tfr.Close()
		return nil, "", false, ctxerr.Wrapf(ctx, err, "inspecting installer %q contents", installerURL)
	}

	return tfr, FilenameFromResponse(resp), isHTML, nil
}

func isHTMLFile(tfr *fleet.TempFileReader) (bool, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(tfr, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return false, err
	}
	if err := tfr.Rewind(); err != nil {
		return false, err
	}
	return strings.HasPrefix(http.DetectContentType(head[:n]), "text/html"), nil
}

func installerUserAgent() string {
	return "fleet/" + version.Version().Version + " Go-http-client/1.1"
}

func FilenameFromResponse(resp *http.Response) string {
	var filename string
	cdh, ok := resp.Header["Content-Disposition"]
	if ok && len(cdh) > 0 {
		_, params, err := mime.ParseMediaType(cdh[0])
		if err == nil {
			filename = params["filename"]
		} else {
			// fallback for responses that include a filename in their content-disposition header
			// but the header isn't technically RFC compliant
			cdhParts := strings.Split(cdh[0], "filename=")
			if len(cdhParts) > 1 {
				unescapedFilename, err := url.QueryUnescape(cdhParts[1])
				if err == nil {
					filename = unescapedFilename
				}
			}
		}
	}

	// Fall back on extracting the filename from the URL
	// This is OK for the first 20 apps we support, but we should do something more robust once we
	// support more apps.
	if filename == "" && resp.Request.URL.Path != "" {
		filename = path.Base(resp.Request.URL.Path)
	}

	return filename
}
