package maintainedapps

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
)

type AppDownloader struct {
	logger kitlog.Logger
	store  fleet.SoftwareInstallerStore
	client *http.Client
}

// Given an app's URL
// Download the installer
// If it is > 3GB, return an error
// Store the installer in S3

const maxInstallerSizeBytes int64 = 1024 * 1024 * 1024 * 3 // 3GB

// TODO(JVE): is this a good name?
func NewAppDownloader(ctx context.Context, store fleet.SoftwareInstallerStore, logger kitlog.Logger) *AppDownloader {
	client := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	client.Transport = fleethttp.NewSizeLimitTransport(maxInstallerSizeBytes)
	return &AppDownloader{
		logger: logger,
		store:  store,
		client: client,
	}
}

func (d *AppDownloader) Download(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := d.client.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return ctxerr.NewWithData(ctx, "request to download installer failed", map[string]any{"status_code": res.StatusCode})
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading request body")
	}

	slog.With("filename", "server/mdm/maintainedapps/get_installer.go", "func", "Download").Info("JVE_LOG: got response: ", "length", res.ContentLength)

	if err := d.store.Put(ctx, "foobar", bytes.NewReader(body)); err != nil {
		return ctxerr.Wrap(ctx, err, "upload maintained app installer to S3")
	}

	return nil
}
