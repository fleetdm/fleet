package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

type createAndroidCACertificateRequest struct {
	TeamID           uint     `json:"team_id" renameto:"fleet_id"`
	Name             string   `json:"name"`
	CertificatePEM   string   `json:"certificate_pem"`
	LabelsIncludeAll []string `json:"labels_include_all"`
	LabelsIncludeAny []string `json:"labels_include_any"`
	LabelsExcludeAny []string `json:"labels_exclude_any"`

	FileHeader *multipart.FileHeader `json:"-"`
	FileBytes  []byte                `json:"-"`
}

func (createAndroidCACertificateRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(platform_http.MaxMultipartFormSize) //nolint:gosec // G120: bounded by the route limit
		if err != nil {
			return nil, &fleet.BadRequestError{
				Message:     "failed to parse multipart form",
				InternalErr: err,
			}
		}

		decoded := createAndroidCACertificateRequest{}
		if len(r.MultipartForm.File["certificate"]) > 0 {
			decoded.FileHeader = r.MultipartForm.File["certificate"][0]
			f, err := decoded.FileHeader.Open()
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "opening uploaded certificate file")
			}
			defer f.Close()
			bytes, err := io.ReadAll(f)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "reading uploaded certificate file")
			}
			decoded.FileBytes = bytes
		}

		if val := r.MultipartForm.Value["name"]; len(val) > 0 {
			decoded.Name = val[0]
		}
		if val := r.MultipartForm.Value["team_id"]; len(val) > 0 {
			teamID, _ := strconv.ParseUint(val[0], 10, 32)
			decoded.TeamID = uint(teamID)
		} else if val := r.MultipartForm.Value["fleet_id"]; len(val) > 0 {
			teamID, _ := strconv.ParseUint(val[0], 10, 32)
			decoded.TeamID = uint(teamID)
		}

		if val := r.MultipartForm.Value["labels_include_all"]; len(val) > 0 {
			_ = json.Unmarshal([]byte(val[0]), &decoded.LabelsIncludeAll)
		}
		if val := r.MultipartForm.Value["labels_include_any"]; len(val) > 0 {
			_ = json.Unmarshal([]byte(val[0]), &decoded.LabelsIncludeAny)
		}
		if val := r.MultipartForm.Value["labels_exclude_any"]; len(val) > 0 {
			_ = json.Unmarshal([]byte(val[0]), &decoded.LabelsExcludeAny)
		}

		return &decoded, nil
	}

	var decoded createAndroidCACertificateRequest
	if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("invalid json body: %v", err)}
	}
	if decoded.CertificatePEM != "" {
		decoded.FileBytes = []byte(decoded.CertificatePEM)
	}
	return &decoded, nil
}

type createAndroidCACertificateResponse struct {
	Certificate *fleet.AndroidCACertificateResponse `json:"certificate,omitempty"`
	Err         error                               `json:"error,omitempty"`
}

func (r createAndroidCACertificateResponse) Error() error { return r.Err }

func createAndroidCACertificateEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createAndroidCACertificateRequest)
	cert := &fleet.AndroidCACertificate{
		TeamID:           req.TeamID,
		Name:             req.Name,
		LabelsIncludeAll: req.LabelsIncludeAll,
		LabelsIncludeAny: req.LabelsIncludeAny,
		LabelsExcludeAny: req.LabelsExcludeAny,
	}

	created, err := svc.CreateAndroidCACertificate(ctx, cert, req.FileBytes)
	if err != nil {
		return createAndroidCACertificateResponse{Err: err}, nil
	}
	return createAndroidCACertificateResponse{Certificate: created.ToResponse()}, nil
}

func (svc *Service) CreateAndroidCACertificate(ctx context.Context, cert *fleet.AndroidCACertificate, certBytes []byte) (*fleet.AndroidCACertificate, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AndroidCACertificate{TeamID: cert.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	lic, err := svc.License(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting license")
	}
	if !lic.IsPremium() {
		return nil, fleet.ErrMissingLicense
	}

	parsed, err := fleet.ValidateAndParseAndroidCACertificate(certBytes)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(cert.Name)
	if name == "" {
		if parsed.CommonName != "" {
			name = parsed.CommonName
		} else {
			name = "Android CA Certificate"
		}
	}
	if err := validateCertificateTemplateName(name); err != nil {
		return nil, err
	}

	cert.Name = name
	cert.CertificatePEM = parsed.CertificatePEM
	cert.SHA256Fingerprint = parsed.SHA256Fingerprint
	cert.Subject = parsed.Subject
	cert.Issuer = parsed.Issuer
	cert.NotValidBefore = &parsed.NotValidBefore
	cert.NotValidAfter = &parsed.NotValidAfter

	created, err := svc.ds.CreateAndroidCACertificate(ctx, cert)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating android ca certificate")
	}
	if created == nil {
		return nil, ctxerr.New(ctx, "failed to create android ca certificate")
	}

	activity := fleet.ActivityTypeAddedCertificate{
		Name: created.Name,
	}
	if created.TeamID != 0 {
		team, err := svc.ds.TeamLite(ctx, created.TeamID)
		if err == nil && team != nil {
			activity.TeamID = &team.ID
			activity.TeamName = &team.Name
		}
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		svc.logger.ErrorContext(ctx, "failed to create activity for added ca certificate", "err", err)
	}

	return created, nil
}

type listAndroidCACertificatesRequest struct {
	fleet.ListOptions
	TeamID uint `query:"team_id,optional" renameto:"fleet_id"`
}

type listAndroidCACertificatesResponse struct {
	Certificates []*fleet.AndroidCACertificateResponse `json:"certificates"`
	Meta         *fleet.PaginationMetadata             `json:"meta,omitempty"`
	Err          error                                 `json:"error,omitempty"`
}

func (r listAndroidCACertificatesResponse) Error() error { return r.Err }

func listAndroidCACertificatesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listAndroidCACertificatesRequest)
	certificates, meta, err := svc.ListAndroidCACertificates(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return listAndroidCACertificatesResponse{Err: err}, nil
	}

	var responses []*fleet.AndroidCACertificateResponse
	for _, c := range certificates {
		responses = append(responses, c.ToResponse())
	}
	return listAndroidCACertificatesResponse{Certificates: responses, Meta: meta}, nil
}

func (svc *Service) ListAndroidCACertificates(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.AndroidCACertificate, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AndroidCACertificate{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, nil, err
	}

	opts.OrderKey = "id"
	opts.OrderDirection = fleet.OrderAscending
	opts.IncludeMetadata = true

	return svc.ds.ListAndroidCACertificates(ctx, teamID, opts)
}

type getAndroidCACertificateRequest struct {
	ID uint `url:"id"`
}

type getAndroidCACertificateResponse struct {
	Certificate *fleet.AndroidCACertificate `json:"certificate,omitempty"`
	Err         error                       `json:"error,omitempty"`
}

func (r getAndroidCACertificateResponse) Error() error { return r.Err }

func getAndroidCACertificateEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getAndroidCACertificateRequest)
	cert, err := svc.GetAndroidCACertificate(ctx, req.ID)
	if err != nil {
		return getAndroidCACertificateResponse{Err: err}, nil
	}
	return getAndroidCACertificateResponse{Certificate: cert}, nil
}

func (svc *Service) GetAndroidCACertificate(ctx context.Context, id uint) (*fleet.AndroidCACertificate, error) {
	cert, err := svc.ds.GetAndroidCACertificateByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := svc.authz.Authorize(ctx, &fleet.AndroidCACertificate{TeamID: cert.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return cert, nil
}

type deleteAndroidCACertificateRequest struct {
	ID uint `url:"id"`
}

type deleteAndroidCACertificateResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteAndroidCACertificateResponse) Error() error { return r.Err }

func deleteAndroidCACertificateEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteAndroidCACertificateRequest)
	err := svc.DeleteAndroidCACertificate(ctx, req.ID)
	if err != nil {
		return deleteAndroidCACertificateResponse{Err: err}, nil
	}
	return deleteAndroidCACertificateResponse{}, nil
}

func (svc *Service) DeleteAndroidCACertificate(ctx context.Context, id uint) error {
	cert, err := svc.ds.GetAndroidCACertificateByID(ctx, id)
	if err != nil {
		return err
	}
	if err := svc.authz.Authorize(ctx, &fleet.AndroidCACertificate{TeamID: cert.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteAndroidCACertificate(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting android ca certificate")
	}

	activity := fleet.ActivityTypeDeletedCertificate{
		Name: cert.Name,
	}
	if cert.TeamID != 0 {
		team, err := svc.ds.TeamLite(ctx, cert.TeamID)
		if err == nil && team != nil {
			activity.TeamID = &team.ID
			activity.TeamName = &team.Name
		}
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		svc.logger.ErrorContext(ctx, "failed to create activity for deleted ca certificate", "err", err)
	}

	return nil
}

// /////////////////////////////////////////////////////////////////////////////
// Agent Endpoints (Node-Key Authenticated)

type getDeviceCACertificateRequest struct {
	ID uint `url:"id"`
}

type getDeviceCACertificateResponse struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	CertificatePEM string `json:"certificate_pem"`
	SHA256         string `json:"sha256"`
	Err            error  `json:"error,omitempty"`
}

func (r getDeviceCACertificateResponse) Error() error { return r.Err }

func getDeviceCACertificateEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getDeviceCACertificateRequest)
	cert, err := svc.GetDeviceCACertificate(ctx, req.ID)
	if err != nil {
		return getDeviceCACertificateResponse{Err: err}, nil
	}
	return getDeviceCACertificateResponse{
		ID:             cert.ID,
		Name:           cert.Name,
		CertificatePEM: cert.CertificatePEM,
		SHA256:         cert.SHA256Fingerprint,
	}, nil
}

func (svc *Service) GetDeviceCACertificate(ctx context.Context, id uint) (*fleet.AndroidCACertificate, error) {
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "missing host from request context")
	}

	cert, err := svc.ds.GetAndroidCACertificateByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify team match
	hostTeamID := uint(0)
	if host.TeamID != nil {
		hostTeamID = *host.TeamID
	}
	if cert.TeamID != 0 && cert.TeamID != hostTeamID {
		return nil, fleet.NewPermissionError("host does not have access to this ca certificate")
	}

	return cert, nil
}

type getDeviceCACertificateByFingerprintRequest struct {
	SHA256 string `url:"sha256"`
}

func getDeviceCACertificateByFingerprintEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getDeviceCACertificateByFingerprintRequest)
	cert, err := svc.GetDeviceCACertificateByFingerprint(ctx, req.SHA256)
	if err != nil {
		return getDeviceCACertificateResponse{Err: err}, nil
	}
	return getDeviceCACertificateResponse{
		ID:             cert.ID,
		Name:           cert.Name,
		CertificatePEM: cert.CertificatePEM,
		SHA256:         cert.SHA256Fingerprint,
	}, nil
}

func (svc *Service) GetDeviceCACertificateByFingerprint(ctx context.Context, sha256 string) (*fleet.AndroidCACertificate, error) {
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "missing host from request context")
	}

	scopedCAs, err := svc.ds.GetHostScopedAndroidCACertificates(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host scoped ca certificates")
	}

	for _, ca := range scopedCAs {
		if strings.EqualFold(ca.SHA256Fingerprint, sha256) {
			return ca, nil
		}
	}

	return nil, newNotFoundError()
}

type updateDeviceCACertificateStatusRequest struct {
	CACertificateID uint    `url:"id"`
	Status          string  `json:"status"`
	Detail          *string `json:"detail,omitempty"`
}

type updateDeviceCACertificateStatusResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateDeviceCACertificateStatusResponse) Error() error { return r.Err }

func updateDeviceCACertificateStatusEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req, ok := request.(*updateDeviceCACertificateStatusRequest)
	if !ok {
		return nil, errors.New("invalid request")
	}

	err := svc.UpdateDeviceCACertificateStatus(ctx, req.CACertificateID, fleet.MDMDeliveryStatus(req.Status), req.Detail)
	if err != nil {
		return updateDeviceCACertificateStatusResponse{Err: err}, nil
	}
	return updateDeviceCACertificateStatusResponse{}, nil
}

func (svc *Service) UpdateDeviceCACertificateStatus(ctx context.Context, caCertID uint, status fleet.MDMDeliveryStatus, detail *string) error {
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	if status != fleet.MDMDeliveryVerified && status != fleet.MDMDeliveryFailed {
		return fleet.NewInvalidArgumentError("status", string(status))
	}

	detailStr := ""
	if detail != nil {
		detailStr = *detail
	}

	if err := svc.ds.UpsertHostAndroidCACertificateStatus(ctx, host.ID, host.UUID, caCertID, status, detailStr); err != nil {
		return ctxerr.Wrap(ctx, err, "upserting host android ca certificate status")
	}

	caCert, err := svc.ds.GetAndroidCACertificateByID(ctx, caCertID)
	caName := ""
	if err == nil && caCert != nil {
		caName = caCert.Name
	}

	actStatus := fleet.CertificateActivityInstalled
	if status == fleet.MDMDeliveryFailed {
		actStatus = fleet.CertificateActivityFailedInstall
	}

	if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeInstalledCertificate{
		HostID:                host.ID,
		HostDisplayName:       host.DisplayName(),
		CertificateTemplateID: caCertID,
		CertificateName:       caName,
		Status:                string(actStatus),
		Detail:                detailStr,
	}); err != nil {
		svc.logger.ErrorContext(ctx, "failed to create activity for installed ca certificate", "err", err)
	}

	return nil
}

// /////////////////////////////////////////////////////////////////////////////
// Host Details Endpoints

type listHostAndroidCACertificatesRequest struct {
	ID uint `url:"id"`
}

type listHostAndroidCACertificatesResponse struct {
	Certificates []*fleet.HostAndroidCACertificate `json:"certificates"`
	Err          error                             `json:"error,omitempty"`
}

func (r listHostAndroidCACertificatesResponse) Error() error { return r.Err }

func listHostAndroidCACertificatesEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listHostAndroidCACertificatesRequest)
	certs, err := svc.ListHostAndroidCACertificates(ctx, req.ID)
	if err != nil {
		return listHostAndroidCACertificatesResponse{Err: err}, nil
	}
	return listHostAndroidCACertificatesResponse{Certificates: certs}, nil
}

func (svc *Service) ListHostAndroidCACertificates(ctx context.Context, hostID uint) ([]*fleet.HostAndroidCACertificate, error) {
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host")
	}
	teamID := uint(0)
	if host.TeamID != nil {
		teamID = *host.TeamID
	}
	if err := svc.authz.Authorize(ctx, &fleet.AndroidCACertificate{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListHostAndroidCACertificates(ctx, hostID)
}
