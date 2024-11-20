package service

import (
	"context"
	"encoding/pem"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// //////////////////////////////////////////////////////////////////////////////
// POST /fleet/certificate_mgmt/certificate
// //////////////////////////////////////////////////////////////////////////////

type uploadCertRequest struct {
	// TODO: Add an identifier
	File *multipart.FileHeader
}

func (uploadCertRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadCertRequest{}
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["certificate"] == nil || len(r.MultipartForm.File["certificate"]) == 0 {
		return nil, &fleet.BadRequestError{
			Message:     "certificate multipart field is required",
			InternalErr: err,
		}
	}

	decoded.File = r.MultipartForm.File["certificate"][0]

	return &decoded, nil
}

type uploadCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r uploadCertResponse) error() error {
	return r.Err
}

func (r uploadCertResponse) Status() int { return http.StatusAccepted }

func uploadCertEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uploadCertRequest)
	file, err := req.File.Open()
	if err != nil {
		return uploadCertResponse{Err: err}, nil
	}
	defer file.Close()

	if err := svc.UploadCert(ctx, file); err != nil {
		return &uploadCertResponse{Err: err}, nil
	}

	return &uploadMDMAppleAPNSCertResponse{}, nil
}

func (svc *Service) UploadCert(ctx context.Context, cert io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return err
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return ctxerr.New(ctx,
			"Couldn't upload certificate. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
	}

	if cert == nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("certificate", "Invalid certificate. Please provide a valid certificate."))
	}

	// Get cert file bytes
	certBytes, err := io.ReadAll(cert)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading certificate")
	}

	// Validate cert
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("certificate", "Invalid certificate. Please provide a valid certificate."))
	}

	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return err
	}

	// TODO: Parse the certificate to determine expiration date

	// delete the old certificate and insert the new one
	// TODO(roberto): replacing the certificate should be done in a single transaction in the DB
	err = svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAuthenticationCertificate})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting old apns cert from db")
	}
	err = svc.ds.InsertMDMConfigAssets(ctx, []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetAuthenticationCertificate, Value: certBytes},
	}, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "writing cert to db")
	}

	return nil

}
