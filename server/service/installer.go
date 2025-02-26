package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/gorilla/mux"
)

////////////////////////////////////////////////////////////////////////////////
// Retrieve an Orbit installer from storage
////////////////////////////////////////////////////////////////////////////////

type getInstallerRequest struct {
	Kind         string
	EnrollSecret string
	Desktop      bool
}

func (getInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	k, ok := mux.Vars(r)["kind"]
	if !ok {
		return "", endpoint_utils.ErrBadRoute
	}

	return getInstallerRequest{
		Kind:         k,
		EnrollSecret: r.FormValue("enroll_secret"),
		Desktop:      r.FormValue("desktop") == "true",
	}, nil
}

type getInstallerResponse struct {
	Err error `json:"error,omitempty"`

	// file fields below are used in hijackRender for the response
	fileReader io.ReadCloser
	fileLength int64
	fileExt    string
}

func (r getInstallerResponse) Error() error { return r.Err }

func (r getInstallerResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.fileLength, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="fleet-osquery.%s"`, r.fileExt))

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

func getInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(getInstallerRequest)

	fileReader, fileLength, err := svc.GetInstaller(ctx, fleet.Installer{
		EnrollSecret: req.EnrollSecret,
		Kind:         req.Kind,
		Desktop:      req.Desktop,
	})
	if err != nil {
		return getInstallerResponse{Err: err}, nil
	}

	return getInstallerResponse{fileReader: fileReader, fileLength: fileLength, fileExt: req.Kind}, nil
}

// GetInstaller retrieves a blob containing the installer binary
func (svc *Service) GetInstaller(ctx context.Context, installer fleet.Installer) (io.ReadCloser, int64, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return nil, int64(0), err
	}

	if !svc.SandboxEnabled() {
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

////////////////////////////////////////////////////////////////////////////////
// Check if a prebuilt Orbit installer is available
////////////////////////////////////////////////////////////////////////////////

type checkInstallerRequest struct {
	Kind         string `url:"kind"`
	Desktop      bool   `query:"desktop,optional"`
	EnrollSecret string `query:"enroll_secret"`
}

type checkInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r checkInstallerResponse) Error() error { return r.Err }

func checkInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*checkInstallerRequest)

	err := svc.CheckInstallerExistence(ctx, fleet.Installer{
		EnrollSecret: req.EnrollSecret,
		Kind:         req.Kind,
		Desktop:      req.Desktop,
	})
	if err != nil {
		return checkInstallerResponse{Err: err}, nil
	}

	return checkInstallerResponse{}, nil
}

// CheckInstallerExistence checks if an installer exists in the configured storage
func (svc *Service) CheckInstallerExistence(ctx context.Context, installer fleet.Installer) error {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return err
	}

	if !svc.SandboxEnabled() {
		return errors.New("this endpoint only enabled in demo mode")
	}

	if svc.installerStore == nil {
		return ctxerr.New(ctx, "installer storage has not been configured")
	}

	_, err := svc.ds.VerifyEnrollSecret(ctx, installer.EnrollSecret)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cannot find a matching enroll secret")
	}

	exists, err := svc.installerStore.Exists(ctx, installer)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking installer existence")
	}

	if !exists {
		return newNotFoundError()
	}

	return nil
}
