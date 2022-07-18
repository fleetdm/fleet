package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type installerRequest struct {
	EnrollSecret string `url:"enroll_secret"`
	Kind         string `url:"kind"`
	Desktop      bool   `query:"desktop,optional"`
}

////////////////////////////////////////////////////////////////////////////////
// Retrieve an Orbit installer from storage
////////////////////////////////////////////////////////////////////////////////

type getInstallerResponse struct {
	Err error `json:"error,omitempty"`

	// file fields below are used in hijackRender for the response
	fileReader io.ReadCloser
	fileLength int64
}

func (r getInstallerResponse) error() error { return r.Err }

func (r getInstallerResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.fileLength, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	wl, err := io.Copy(w, r.fileReader)
	if err != nil {
		logging.WithExtras(ctx, "s3_copy_error", err, "bytes_copied", wl)
	}
	r.fileReader.Close()
}

func getInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*installerRequest)

	fileReader, fileLength, err := svc.GetInstaller(ctx, fleet.Installer{
		EnrollSecret: req.EnrollSecret,
		Kind:         req.Kind,
		Desktop:      req.Desktop,
	})

	if err != nil {
		return getInstallerResponse{Err: err}, nil
	}

	return getInstallerResponse{fileReader: fileReader, fileLength: fileLength}, nil
}

// GetInstaller retrieves a blob containing the installer binary
func (svc *Service) GetInstaller(ctx context.Context, installer fleet.Installer) (io.ReadCloser, int64, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return nil, int64(0), err
	}

	// Undocumented FLEET_DEMO environment variable, as this endpoint is intended only to be
	// used in the Fleet Sandbox demo environment.
	if os.Getenv("FLEET_DEMO") != "1" {
		return nil, int64(0), errors.New("this endpoint only enabled in demo mode")
	}

	if svc.installerStore == nil {
		return nil, int64(0), ctxerr.New(ctx, "installer storage has not been configured")
	}

	_, err := svc.ds.VerifyEnrollSecret(ctx, installer.EnrollSecret)
	if err != nil {
		return nil, int64(0), ctxerr.Wrap(ctx, err, "finding a matching enroll secret")
	}

	reader, length, err := svc.installerStore.Get(ctx, installer)
	if err != nil {
		return nil, int64(0), ctxerr.Wrap(ctx, err, "unable to retrieve installer from store")
	}

	return reader, length, nil
}
