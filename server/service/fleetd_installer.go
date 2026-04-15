package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
)

type getFleetdInstallerResponse struct {
	Err     error `json:"error,omitempty"`
	payload *fleet.DownloadFleetdInstallerPayload
}

func (r getFleetdInstallerResponse) Error() error { return r.Err }

func (r getFleetdInstallerResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(int(r.payload.Size)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.payload.Filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	if n, err := io.Copy(w, r.payload.Installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
	r.payload.Installer.Close()
}

func getFleetdInstallerEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetFleetdInstallerRequest)
	payload, err := svc.GetFleetdInstallerPkg(ctx, req.TeamID)
	if err != nil {
		return getFleetdInstallerResponse{Err: err}, nil
	}
	return getFleetdInstallerResponse{payload: payload}, nil
}

func (svc *Service) GetFleetdInstallerPkg(ctx context.Context, _ uint) (*fleet.DownloadFleetdInstallerPayload, error) {
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}
