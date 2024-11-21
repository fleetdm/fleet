package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"regexp"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
)

const rsaKeySize = 2048

// //////////////////////////////////////////////////////////////////////////////
// GET /fleet/certificate_mgmt/certificates
// //////////////////////////////////////////////////////////////////////////////

type getCertificatesResponse struct {
	Certificates []fleet.PKICertificate `json:"certificates"`
	Err          error                  `json:"error,omitempty"`
}

func (r getCertificatesResponse) error() error { return r.Err }

func getCertificatesEndpoint(ctx context.Context, _ interface{}, svc fleet.Service) (errorer, error) {
	certs, err := svc.GetPKICertificates(ctx)
	return &getCertificatesResponse{Certificates: certs, Err: err}, nil
}

func (svc *Service) GetPKICertificates(ctx context.Context) ([]fleet.PKICertificate, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	certs, err := svc.ds.ListPKICertificates(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pki certificates")
	}
	return certs, nil
}

// //////////////////////////////////////////////////////////////////////////////
// GET /fleet/certificate_mgmt/certificate/{pki_name}/request_csr
// //////////////////////////////////////////////////////////////////////////////

type getCertCSRRequest struct {
	Name string `url:"pki_name"`
}

type getCertCSRResponse struct {
	CSR []byte `json:"csr"` // base64 encoded
	Err error  `json:"error,omitempty"`
}

func (r getCertCSRResponse) error() error { return r.Err }

func getCertCSREndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getCertCSRRequest)
	csr64, err := svc.GetCertCSR(ctx, req.Name)
	if err != nil {
		return &getCertCSRResponse{Err: err}, nil
	}

	return &getCertCSRResponse{CSR: csr64}, nil
}

func (svc *Service) GetCertCSR(ctx context.Context, nameEscaped string) ([]byte, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	name, err := url.PathUnescape(nameEscaped)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("pki_name", "Invalid pki_name. Please provide a valid pki_name."))
	}
	if len(name) > 255 {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("pki_name", "pki_name too long. Please provide a valid pki_name."))
	}

	privateKey := svc.config.Server.PrivateKey
	if testSetEmptyPrivateKey {
		privateKey = ""
	}

	if len(privateKey) == 0 {
		return nil, ctxerr.Wrap(ctx,
			&fleet.BadRequestError{Message: "Couldn't download signed CSR. Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key"})
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	// Check if we have existing cert and keys
	pkiCert, err := svc.ds.GetPKICertificate(ctx, name)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "loading existing pki cert")
	}
	var key *rsa.PrivateKey
	if pkiCert == nil {
		key, err = rsa.GenerateKey(rand.Reader, rsaKeySize)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "generate new private key")
		}
		// Create new PKI certificate
		pkiCert = &fleet.PKICertificate{
			Name: name,
			Key:  apple_mdm.EncodePrivateKeyPEM(key),
		}
		err = svc.ds.SavePKICertificate(ctx, pkiCert)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "saving new pki cert")
		}
	} else {
		block, _ := pem.Decode(pkiCert.Key)
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshaling saved key")
		}
	}

	// Generate new CSR every time this is called
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	csr, err := apple_mdm.GenerateAPNSCSR(appConfig.OrgInfo.OrgName, vc.Email(), key)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate new CSR")
	}

	return csr.Raw, nil
}

// //////////////////////////////////////////////////////////////////////////////
// POST /fleet/certificate_mgmt/certificate/{pki_name}
// //////////////////////////////////////////////////////////////////////////////

var certificatePathRegexp = regexp.MustCompile(`/certificate/(?P<pki_name>.*)$`)

type uploadCertRequest struct {
	Name string
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

	// regex to get and validate the name
	matches := certificatePathRegexp.FindStringSubmatch(r.URL.Path)
	for i, name := range certificatePathRegexp.SubexpNames() {
		if name == "pki_name" {
			certName, err := url.QueryUnescape(matches[i])
			if err != nil {
				return nil, &fleet.BadRequestError{
					Message:     "certificate pki_name has invalid format",
					InternalErr: err,
				}
			}
			decoded.Name = fleet.Preprocess(certName)
		}
	}

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

	// TODO: Parse the certificate to determine expiration date and fingerprint
	// h := sha256.New()
	// _, _ = io.Copy(h, bytes.NewReader(certBytes)) // writes to a Hash can never fail
	// sha256Hash := hex.EncodeToString(h.Sum(nil))

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

// //////////////////////////////////////////////////////////////////////////////
// GET /fleet/certificate_mgmt/certificate/{pki_name}
// //////////////////////////////////////////////////////////////////////////////

type getCertRequest struct {
}

func getCertEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	// TODO: Implementation
	_ = request.(*getCertRequest)
	return nil, nil
}

// //////////////////////////////////////////////////////////////////////////////
// DELETE /fleet/certificate_mgmt/certificate/{pki_name}
// //////////////////////////////////////////////////////////////////////////////

type deleteCertRequest struct{}

type deleteCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteCertResponse) error() error {
	return r.Err
}

func deleteCertEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	if err := svc.DeleteMDMAppleAPNSCert(ctx); err != nil {
		return &deleteMDMAppleAPNSCertResponse{Err: err}, nil
	}

	return &deleteMDMAppleAPNSCertResponse{}, nil
}

func (svc *Service) DeleteCert(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return err
	}

	err := svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetAPNSCert,
		fleet.MDMAssetAPNSKey,
		fleet.MDMAssetCACert,
		fleet.MDMAssetCAKey,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting apple mdm assets")
	}

	// flip the app config flag
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	appCfg.MDM.EnabledAndConfigured = false

	return svc.ds.SaveAppConfig(ctx, appCfg)
}
