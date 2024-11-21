package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
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
	"github.com/smallstep/pkcs7"
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

func (uploadCertRequest) DecodeRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	if err := svc.UploadCert(ctx, req.Name, file); err != nil {
		return &uploadCertResponse{Err: err}, nil
	}

	return &uploadMDMAppleAPNSCertResponse{}, nil
}

func (svc *Service) UploadCert(ctx context.Context, nameEscaped string, cert io.ReadSeeker) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return err
	}

	name, err := url.PathUnescape(nameEscaped)
	if err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("pki_name", "Invalid pki_name. Please provide a valid pki_name."))
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
	if len(certBytes) == 0 {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("certificate", "Empty certificate. Please provide a valid certificate."))
	}

	// Convert from PEM to DER format if needed
	block, _ := pem.Decode(certBytes)
	if block != nil {
		// Assume that certificate need to be converted from PEM to DER
		certBytes = block.Bytes
	}

	// Validate cert
	p7, err := pkcs7.Parse(certBytes)
	if err != nil {
		return ctxerr.Wrap(ctx,
			fleet.NewInvalidArgumentError("certificate",
				fmt.Sprintf("Invalid PKCS7 certificate. Please provide a valid certificate. %s", err.Error())))
	}
	if len(p7.Certificates) == 0 {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("certificate", "No certificate found. Please provide a valid certificate."))
	}

	// Get the saved certificate
	pkiCert, err := svc.ds.GetPKICertificate(ctx, name)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading existing pki private key")
	}

	x509Cert := p7.Certificates[0]
	pkiCert.Cert = x509Cert.Raw
	pkiCert.NotValidAfter = &x509Cert.NotAfter

	h := sha256.New()
	_, _ = io.Copy(h, bytes.NewReader(x509Cert.Raw)) // writes to a Hash can never fail
	sha256Hash := hex.EncodeToString(h.Sum(nil))
	pkiCert.Sha256 = &sha256Hash

	err = svc.ds.SavePKICertificate(ctx, pkiCert)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "saving new pki cert")
	}

	return nil

}

// //////////////////////////////////////////////////////////////////////////////
// DELETE /fleet/certificate_mgmt/certificate/{pki_name}
// //////////////////////////////////////////////////////////////////////////////

type deleteCertRequest struct {
	Name string `url:"pki_name"`
}

type deleteCertResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteCertResponse) error() error {
	return r.Err
}

func deleteCertEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteCertRequest)
	err := svc.DeleteCert(ctx, req.Name)
	return &deleteCertResponse{Err: err}, nil
}

func (svc *Service) DeleteCert(ctx context.Context, nameEscaped string) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionWrite); err != nil {
		return err
	}

	name, err := url.PathUnescape(nameEscaped)
	if err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("pki_name", "Invalid pki_name. Please provide a valid pki_name."))
	}

	err = svc.ds.DeletePKICertificate(ctx, name)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting pki cert")
	}
	return nil
}
