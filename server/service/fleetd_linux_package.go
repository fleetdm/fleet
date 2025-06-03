package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/goreleaser/nfpm/v2"
	"github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/rpm"
)

type getLinuxPackageRequest struct {
	PackageType string `query:"type"`
}

type getLinuxPackageResponse struct {
	packageName  string // deb/rpm full package name
	packageBytes []byte // actual deb/rpm package contents

	Err error
}

func (r getLinuxPackageResponse) Error() error { return r.Err }

func (r getLinuxPackageResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.packageBytes)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.packageName))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.packageBytes); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func getLinuxPackageEndpoint(
	ctx context.Context,
	request interface{},
	svc fleet.Service,
) (fleet.Errorer, error) {
	r := request.(*getLinuxPackageRequest)
	packageBytes, err := svc.GenerateFleetdLinuxPackage(ctx, r.PackageType)
	if err != nil {
		return getLinuxPackageResponse{Err: err}, nil
	}
	return getLinuxPackageResponse{packageBytes: packageBytes}, nil
}

func (svc *Service) GenerateFleetdLinuxPackage(ctx context.Context, packageType string) ([]byte, error) {
	// For now just re-using this.
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	var packager nfpm.Packager = deb.Default
	if packageType == "rpm" {
		packager = rpm.Default
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	enrollSecrets, err := svc.ds.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get enroll secrets")
	}
	if len(enrollSecrets) == 0 || enrollSecrets[0] == nil || enrollSecrets[0].Secret == "" {
		return nil, ctxerr.Wrap(ctx, err, "no global enroll secrets")
	}

	packageFilePath, err := packaging.BuildNFPM(packaging.Options{
		FleetURL:            appConfig.ServerSettings.ServerURL,
		EnrollSecret:        enrollSecrets[0].Secret,
		Identifier:          "com.fleetdm.orbit",
		StartService:        true,
		OrbitChannel:        "stable",
		OsquerydChannel:     "stable",
		DesktopChannel:      "stable",
		UpdateURL:           "https://updates.fleetdm.com",
		Debug:               true,
		Desktop:             true,
		OrbitUpdateInterval: 15 * time.Minute,
		EnableScripts:       true,
		HostIdentifier:      "uuid",
		Architecture:        "amd64",
		NativePlatform:      "linux",
	}, packager)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build nfpm")
	}
	b, err := os.ReadFile(packageFilePath)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read package file")
	}
	return b, nil
}
