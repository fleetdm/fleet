package service

import (
	"context"
	"net/http"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type conditionalAccessGetIDPSigningCertRequest struct{}

type conditionalAccessGetIDPSigningCertResponse struct {
	CertPEM []byte
	Err     error `json:"error,omitempty"`
}

func (r conditionalAccessGetIDPSigningCertResponse) Error() error { return r.Err }

func (r conditionalAccessGetIDPSigningCertResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.CertPEM)), 10))
	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "attachment; filename=\"fleet-idp-signing-cert.pem\"")

	// OK to just log the error here as writing anything on `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the header provided
	n, err := w.Write(r.CertPEM)
	if err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_written", n)
	}
}

func conditionalAccessGetIDPSigningCertEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	certPEM, err := svc.ConditionalAccessGetIDPSigningCert(ctx)
	if err != nil {
		return conditionalAccessGetIDPSigningCertResponse{Err: err}, nil
	}
	return conditionalAccessGetIDPSigningCertResponse{
		CertPEM: certPEM,
	}, nil
}

func (svc *Service) ConditionalAccessGetIDPSigningCert(ctx context.Context) (certPEM []byte, err error) {
	// Check user is authorized to read conditional access Okta IdP certificate
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessIDPCert{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}

	// Load IdP certificate from mdm_config_assets
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetConditionalAccessIDPCert,
	}, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to load IdP certificate")
	}

	certAsset, ok := assets[fleet.MDMAssetConditionalAccessIDPCert]
	if !ok {
		return nil, ctxerr.New(ctx, "IdP certificate not configured")
	}

	return certAsset.Value, nil
}
