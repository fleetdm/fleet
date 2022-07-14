package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type installerRequest struct {
	EnrollSecret string `url:"enroll_secret"`
	Kind         string `url:"kind"`
	Desktop      bool   `query:"desktop,optional"`
}

////////////////////////////////////////////////////////////////////////////////
// Check if a prebuilt Orbit installer is available
////////////////////////////////////////////////////////////////////////////////

type checkInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r checkInstallerResponse) error() error { return r.Err }

func checkInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*installerRequest)

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

	// Undocumented FLEET_DEMO environment variable, as this endpoint is intended only to be
	// used in the Fleet Sandbox demo environment.
	if os.Getenv("FLEET_DEMO") != "1" {
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
		return notFoundError{}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Retrieve an Orbit installer from storage
////////////////////////////////////////////////////////////////////////////////

type getInstallerResponse struct {
	Err error `json:"error,omitempty"`

	// file fields below are used in hijackRender for the response
	fileReader *io.ReadCloser
	fileLength *int64
}

func (r getInstallerResponse) error() error { return r.Err }

func (r getInstallerResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(*r.fileLength, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment")

	// OK to ignore the error here as writing anything on `http.ResponseWriter`
	// sets the status code to 200 (and it can't be changed.)
	// Clients should rely on matching content-length with the header provided
	io.Copy(w, *r.fileReader)
	(*r.fileReader).Close()
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
func (svc *Service) GetInstaller(ctx context.Context, installer fleet.Installer) (*io.ReadCloser, *int64, error) {
	if err := svc.authz.Authorize(ctx, &fleet.EnrollSecret{}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// Undocumented FLEET_DEMO environment variable, as this endpoint is intended only to be
	// used in the Fleet Sandbox demo environment.
	if os.Getenv("FLEET_DEMO") != "1" {
		return nil, nil, errors.New("this endpoint only enabled in demo mode")
	}

	if svc.installerStore == nil {
		return nil, nil, ctxerr.New(ctx, "installer storage has not been configured")
	}

	_, err := svc.ds.VerifyEnrollSecret(ctx, installer.EnrollSecret)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "finding a matching enroll secret")
	}

	reader, length, err := svc.installerStore.Get(ctx, installer)
	if err != nil {
		return nil, nil, ctxerr.New(ctx, "unable to retrieve installer from store")
	}

	return reader, length, nil
}
