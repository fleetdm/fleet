package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type createCertificateTemplateRequest struct {
	Name                   string `json:"name"`
	TeamID                 uint   `json:"team_id"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
}
type createCertificateTemplateResponse struct {
	ID                     uint   `json:"id"`
	Name                   string `json:"name"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
	Err                    error  `json:"error,omitempty"`
}

func (r createCertificateTemplateResponse) Error() error { return r.Err }

func createCertificateTemplateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createCertificateTemplateRequest)
	certificate, err := svc.CreateCertificateTemplate(ctx, req.Name, req.TeamID, req.CertificateAuthorityId, req.SubjectName)
	if err != nil {
		return createCertificateTemplateResponse{Err: err}, nil
	}
	return createCertificateTemplateResponse{
		ID:                     certificate.ID,
		Name:                   certificate.Name,
		CertificateAuthorityId: certificate.CertificateAuthorityId,
		SubjectName:            certificate.SubjectName,
	}, nil
}

func (svc *Service) CreateCertificateTemplate(ctx context.Context, name string, teamID uint, certificateAuthorityID uint, subjectName string) (*fleet.CertificateTemplateResponseFull, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := validateCertificateTemplateFleetVariables(subjectName); err != nil {
		return nil, &fleet.BadRequestError{Message: err.Error()}
	}

	certTemplate := &fleet.CertificateTemplate{
		Name:                   name,
		TeamID:                 teamID,
		CertificateAuthorityID: certificateAuthorityID,
		SubjectName:            subjectName,
	}

	savedTemplate, err := svc.ds.CreateCertificateTemplate(ctx, certTemplate)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating certificate template")
	}

	return savedTemplate, nil
}

type listCertificateTemplatesRequest struct {
	TeamID  uint `query:"team_id"`
	Page    int  `query:"page,optional"`
	PerPage int  `query:"per_page,optional"`
}

type listCertificateTemplatesResponse struct {
	Certificates []*fleet.CertificateTemplateResponseSummary `json:"certificates"`
	Err          error                                       `json:"error,omitempty"`
	Meta         *fleet.PaginationMetadata                   `json:"meta"`
}

func (r listCertificateTemplatesResponse) Error() error { return r.Err }

func listCertificateTemplatesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listCertificateTemplatesRequest)
	certificates, paginationMetaData, err := svc.ListCertificateTemplates(ctx, req.TeamID, req.Page, req.PerPage)
	if err != nil {
		return listCertificateTemplatesResponse{Err: err}, nil
	}
	return listCertificateTemplatesResponse{Certificates: certificates, Meta: paginationMetaData}, nil
}

func (svc *Service) ListCertificateTemplates(ctx context.Context, teamID uint, page int, perPage int) ([]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	certificates, paginationMetaData, err := svc.ds.GetCertificateTemplatesByTeamID(ctx, teamID, page, perPage)
	if err != nil {
		return nil, nil, err
	}

	return certificates, paginationMetaData, nil
}

type getDeviceCertificateTemplateRequest struct {
	ID      uint   `url:"id"`
	NodeKey string `query:"node_key"`
}

func (r *getDeviceCertificateTemplateRequest) hostNodeKey() string {
	return r.NodeKey
}

type getDeviceCertificateTemplateResponse struct {
	Certificate *fleet.CertificateTemplateResponseFull `json:"certificate"`
	Err         error                                  `json:"error,omitempty"`
}

func (r getDeviceCertificateTemplateResponse) Error() error { return r.Err }

func getDeviceCertificateTemplateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getDeviceCertificateTemplateRequest)
	certificate, err := svc.GetDeviceCertificateTemplate(ctx, req.ID)
	if err != nil {
		return getDeviceCertificateTemplateResponse{Err: err}, nil
	}
	return getDeviceCertificateTemplateResponse{Certificate: certificate}, nil
}

func (svc *Service) GetDeviceCertificateTemplate(ctx context.Context, id uint) (*fleet.CertificateTemplateResponseFull, error) {
	// skipauth: This endpoint uses node key authentication instead of user authentication.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "missing host from request context")
	}

	certificate, err := svc.ds.GetCertificateTemplateById(ctx, id)
	if err != nil {
		return nil, err
	}

	if certificate.TeamID != 0 && (host.TeamID == nil || *host.TeamID != certificate.TeamID) {
		return nil, fleet.NewPermissionError("host does not have access to this certificate template")
	}

	subjectName, err := svc.replaceCertificateVariables(ctx, certificate.SubjectName, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "replacing certificate variables")
	}
	certificate.SubjectName = subjectName

	return certificate, nil
}

type getCertificateTemplateRequest struct {
	ID       uint    `url:"id"`
	HostUUID *string `query:"host_uuid,optional"`
}

type getCertificateTemplateResponse struct {
	Certificate *fleet.CertificateTemplateResponseFull `json:"certificate"`
	Err         error                                  `json:"error,omitempty"`
}

func (r getCertificateTemplateResponse) Error() error { return r.Err }

func getCertificateTemplateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getCertificateTemplateRequest)
	certificate, err := svc.GetCertificateTemplate(ctx, req.ID, req.HostUUID)
	if err != nil {
		return getCertificateTemplateResponse{Err: err}, nil
	}
	return getCertificateTemplateResponse{Certificate: certificate}, nil
}

func (svc *Service) GetCertificateTemplate(ctx context.Context, id uint, hostUUID *string) (*fleet.CertificateTemplateResponseFull, error) {
	certificate, err := svc.ds.GetCertificateTemplateById(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: certificate.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if hostUUID != nil {
		host, err := svc.ds.HostByIdentifier(ctx, *hostUUID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting host for variable replacement")
		}

		subjectName, err := svc.replaceCertificateVariables(ctx, certificate.SubjectName, host)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "replacing certificate variables")
		}
		certificate.SubjectName = subjectName
	}

	return certificate, nil
}

type deleteCertificateTemplateRequest struct {
	ID uint `url:"id"`
}
type deleteCertificateTemplateResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteCertificateTemplateResponse) Error() error { return r.Err }

func deleteCertificateTemplateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteCertificateTemplateRequest)
	err := svc.DeleteCertificateTemplate(ctx, req.ID)
	if err != nil {
		return deleteCertificateTemplateResponse{Err: err}, nil
	}
	return deleteCertificateTemplateResponse{}, nil
}

func (svc *Service) DeleteCertificateTemplate(ctx context.Context, certificateTemplateID uint) error {
	certificate, err := svc.ds.GetCertificateTemplateById(ctx, certificateTemplateID)
	if err != nil {
		return err
	}

	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: certificate.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteCertificateTemplate(ctx, certificateTemplateID)
}

type applyCertificateTemplateSpecsRequest struct {
	Specs []*fleet.CertificateRequestSpec `json:"specs"`
}

type applyCertificateTemplateSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyCertificateTemplateSpecsResponse) Error() error { return r.Err }

func applyCertificateTemplateSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*applyCertificateTemplateSpecsRequest)
	err := svc.ApplyCertificateTemplateSpecs(ctx, req.Specs)
	if err != nil {
		return applyCertificateTemplateSpecsResponse{Err: err}, nil
	}
	return applyCertificateTemplateSpecsResponse{}, nil
}

func (svc *Service) checkCertificateTemplateSpecAuthorization(ctx context.Context, specs []*fleet.CertificateRequestSpec) error {
	teamIDs := make(map[uint]bool)
	for _, spec := range specs {
		var teamID uint
		if spec.Team != "" {
			parsed, err := strconv.ParseUint(spec.Team, 10, 0)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "parsing team ID")
			}
			teamID = uint(parsed)
		}
		teamIDs[teamID] = true
	}

	for teamID := range teamIDs {
		if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) ApplyCertificateTemplateSpecs(ctx context.Context, specs []*fleet.CertificateRequestSpec) error {
	if err := svc.checkCertificateTemplateSpecAuthorization(ctx, specs); err != nil {
		return err
	}

	var certificates []*fleet.CertificateTemplate
	for _, spec := range specs {
		var teamID uint
		if spec.Team != "" {
			parsed, err := strconv.ParseUint(spec.Team, 10, 0)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "parsing team ID")
			}
			teamID = uint(parsed)
		}

		// Validate Fleet variables in subject name
		if err := validateCertificateTemplateFleetVariables(spec.SubjectName); err != nil {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%s (certificate %s)", err.Error(), spec.Name)}
		}

		cert := &fleet.CertificateTemplate{
			Name:                   spec.Name,
			CertificateAuthorityID: spec.CertificateAuthorityId,
			SubjectName:            spec.SubjectName,
			TeamID:                 teamID,
		}

		certificates = append(certificates, cert)
	}

	return svc.ds.BatchUpsertCertificateTemplates(ctx, certificates)
}

type deleteCertificateTemplateSpecsRequest struct {
	IDs    []uint `json:"ids"`
	TeamID uint   `json:"team_id"`
}

type deleteCertificateTemplateSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteCertificateTemplateSpecsResponse) Error() error { return r.Err }

func deleteCertificateTemplateSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteCertificateTemplateSpecsRequest)
	err := svc.DeleteCertificateTemplateSpecs(ctx, req.IDs, req.TeamID)
	if err != nil {
		return deleteCertificateTemplateSpecsResponse{Err: err}, nil
	}
	return deleteCertificateTemplateSpecsResponse{}, nil
}

func (svc *Service) DeleteCertificateTemplateSpecs(ctx context.Context, certificateTemplateIDs []uint, teamID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}
	return svc.ds.BatchDeleteCertificateTemplates(ctx, certificateTemplateIDs)
}

type updateCertificateStatusRequest struct {
	CertificateTemplateID uint   `url:"id"`
	NodeKey               string `json:"node_key"`
	Status                string `json:"status"`
}

func (r *updateCertificateStatusRequest) hostNodeKey() string {
	return r.NodeKey
}

type updateCertificateStatusResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateCertificateStatusResponse) Error() error { return r.Err }

func updateCertificateStatusEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req, ok := request.(*updateCertificateStatusRequest)
	if !ok {
		return nil, errors.New("invalid request")
	}

	err := svc.UpdateCertificateStatus(ctx, req.CertificateTemplateID, fleet.MDMDeliveryStatus(req.Status))
	if err != nil {
		return updateCertificateStatusResponse{Err: err}, nil
	}

	return updateCertificateStatusResponse{}, nil
}

func (svc *Service) UpdateCertificateStatus(ctx context.Context, certificateTemplateID uint, status fleet.MDMDeliveryStatus) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return err
	}

	// Validate the status.
	if !status.IsValid() {
		return fleet.NewInvalidArgumentError("status", string(status))
	}

	return svc.ds.UpdateCertificateStatus(ctx, host.UUID, certificateTemplateID, status)
}
