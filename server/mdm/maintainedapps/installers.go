package maintainedapps

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// DownloadInstaller downloads the maintained app installer located at the given URL.
func DownloadInstaller(ctx context.Context, installerURL string, maxSize int64) ([]byte, error) {
	// validate the URL before doing the request
	_, err := url.ParseRequestURI(installerURL)
	if err != nil {
		return nil, fleet.NewInvalidArgumentError(
			"software.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) is invalid", installerURL),
		)
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	client.Transport = fleethttp.NewSizeLimitTransport(maxSize)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, installerURL, nil)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "creating request for URL %s", installerURL)
	}

	resp, err := client.Do(req)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
			return nil, fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("Couldn't download maintained app installer. URL (%q). The maximum file size is %d MB", installerURL, maxSize/(1024*1024)),
			)
		}

		return nil, ctxerr.Wrapf(ctx, err, "performing request for URL %s", installerURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fleet.NewInvalidArgumentError(
			"software.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) doesn't exist. Please make sure that URLs are publicy accessible to the internet.", installerURL),
		)
	}

	// Allow all 2xx and 3xx status codes in this pass.
	if resp.StatusCode > 400 {
		return nil, fleet.NewInvalidArgumentError(
			"software.url",
			fmt.Sprintf("Couldn't download maintained app installer. URL (%q) received response status code %d.", installerURL, resp.StatusCode),
		)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// the max size error can be received either at client.Do or here when
		// reading the body if it's caught via a limited body reader.
		var maxBytesErr *http.MaxBytesError
		if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
			return nil, fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("Couldn't download maintained app installer. URL (%q). The maximum file size is %d MB", installerURL, maxSize/(1024*1024)),
			)
		}
		return nil, ctxerr.Wrapf(ctx, err, "reading installer %q contents", installerURL)
	}

	return bodyBytes, nil
}
