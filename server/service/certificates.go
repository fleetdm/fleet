package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
)

type createCertificateTemplateRequest struct {
	Name                   string `json:"name"`
	TeamID                 uint   `json:"team_id"` // If not provided, intentionally defaults to 0 aka "No team"
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

func (svc *Service) CreateCertificateTemplate(ctx context.Context, name string, teamID uint, certificateAuthorityID uint, subjectName string) (*fleet.CertificateTemplateResponse, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := validateCertificateTemplateFleetVariables(subjectName); err != nil {
		return nil, &fleet.BadRequestError{Message: err.Error()}
	}

	// Get the CA to validate its existence and type.
	ca, err := svc.ds.GetCertificateAuthorityByID(ctx, certificateAuthorityID, false)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting certificate authority")
	}

	if ca.Type != string(fleet.CATypeCustomSCEPProxy) {
		return nil, &fleet.BadRequestError{Message: "Currently, only the custom_scep_proxy certificate authority is supported."}
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

	// Create pending certificate template records for all enrolled Android hosts in the team
	if _, err := svc.ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, savedTemplate.ID, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating pending certificate templates for existing hosts")
	}

	activity := fleet.ActivityTypeCreatedCertificate{
		Name: name,
	}
	if teamID != 0 {
		team, err := svc.ds.TeamLite(ctx, teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting team")
		}
		if team != nil {
			activity.TeamID = &team.ID
			activity.TeamName = &team.Name
		}
	}
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		activity,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating activity for new certificate template")
	}

	return savedTemplate, nil
}

type listCertificateTemplatesRequest struct {
	fleet.ListOptions

	// If not provided, intentionally defaults to 0 aka "No team"
	TeamID uint `query:"team_id,optional"`
}

type listCertificateTemplatesResponse struct {
	Certificates []*fleet.CertificateTemplateResponseSummary `json:"certificates"`
	Err          error                                       `json:"error,omitempty"`
	Meta         *fleet.PaginationMetadata                   `json:"meta"`
}

func (r listCertificateTemplatesResponse) Error() error { return r.Err }

func listCertificateTemplatesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listCertificateTemplatesRequest)
	certificates, paginationMetaData, err := svc.ListCertificateTemplates(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return listCertificateTemplatesResponse{Err: err}, nil
	}
	return listCertificateTemplatesResponse{Certificates: certificates, Meta: paginationMetaData}, nil
}

func (svc *Service) ListCertificateTemplates(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]*fleet.CertificateTemplateResponseSummary, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// cursor-based pagination is not supported
	opts.After = ""

	// custom ordering is not supported, always by sort by id
	opts.OrderKey = "certificate_templates.id"
	opts.OrderDirection = fleet.OrderAscending

	// no matching query support
	opts.MatchQuery = ""

	// always include metadata
	opts.IncludeMetadata = true

	certificates, metaData, err := svc.ds.GetCertificateTemplatesByTeamID(ctx, teamID, opts)
	if err != nil {
		return nil, nil, err
	}

	return certificates, metaData, nil
}

type getDeviceCertificateTemplateRequest struct {
	ID uint `url:"id"`
}

type getDeviceCertificateTemplateResponse struct {
	Certificate *fleet.CertificateTemplateResponseForHost `json:"certificate"`
	Err         error                                     `json:"error,omitempty"`
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

func (svc *Service) GetDeviceCertificateTemplate(ctx context.Context, id uint) (*fleet.CertificateTemplateResponseForHost, error) {
	// skipauth: This endpoint uses node key authentication instead of user authentication.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "missing host from request context")
	}

	certificate, err := svc.ds.GetCertificateTemplateByIdForHost(ctx, id, host.UUID)
	if err != nil {
		return nil, err
	}

	// team_id = 0 for hosts without a team
	hostTeamID := uint(0)
	if host.TeamID != nil {
		hostTeamID = *host.TeamID
	}
	if certificate.TeamID != hostTeamID {
		return nil, fleet.NewPermissionError("host does not have access to this certificate template")
	}

	subjectName, err := svc.replaceCertificateVariables(ctx, certificate.SubjectName, host)
	if err != nil {
		// If the certificate variables cannot be replaced, mark the certificate as failed.
		errorMsg := fmt.Sprintf("Could not replace certificate variables: %s", err.Error())
		if err := svc.ds.UpsertCertificateStatus(
			ctx,
			host.UUID,
			certificate.ID,
			fleet.MDMDeliveryFailed,
			&errorMsg,
			fleet.MDMOperationTypeInstall,
		); err != nil {
			return nil, err
		}
		certificate.Status = fleet.CertificateTemplateFailed
		return certificate, nil
	}
	certificate.SubjectName = subjectName

	return certificate, nil
}

type getCertificateTemplateRequest struct {
	ID uint `url:"id"`
}

type getCertificateTemplateResponse struct {
	Certificate *fleet.CertificateTemplateResponse `json:"certificate"`
	Err         error                              `json:"error,omitempty"`
}

func (r getCertificateTemplateResponse) Error() error { return r.Err }

func getCertificateTemplateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getCertificateTemplateRequest)
	certificate, err := svc.GetCertificateTemplate(ctx, req.ID)
	if err != nil {
		return getCertificateTemplateResponse{Err: err}, nil
	}
	return getCertificateTemplateResponse{Certificate: certificate}, nil
}

func (svc *Service) GetCertificateTemplate(ctx context.Context, id uint) (*fleet.CertificateTemplateResponse, error) {
	certificate, err := svc.ds.GetCertificateTemplateById(ctx, id)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, err
	}

	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: certificate.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
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
		return ctxerr.Wrap(ctx, err, "getting certificate template")
	}

	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: certificate.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "authorizing user for certificate template deletion")
	}

	if err := svc.ds.DeleteCertificateTemplate(ctx, certificateTemplateID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting certificate template")
	}

	if err := svc.ds.SetHostCertificateTemplatesToPendingRemove(ctx, certificateTemplateID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting host certificate templates to pending remove")
	}

	activity := fleet.ActivityTypeDeletedCertificate{
		Name: certificate.Name,
	}
	if certificate.TeamID != 0 {
		team, err := svc.ds.TeamLite(ctx, certificate.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team")
		}
		if team != nil {
			activity.TeamID = &team.ID
			activity.TeamName = &team.Name
		}
	}
	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		activity,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for deleting certificate template")
	}

	return nil
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

func (svc *Service) resolveTeamNamesForSpecs(ctx context.Context, specs []*fleet.CertificateRequestSpec) (map[string]uint, error) {
	teamNameToID := make(map[string]uint)

	for _, spec := range specs {
		if _, ok := teamNameToID[spec.Team]; ok {
			continue
		}

		// Handle empty string and "No team" as teamID = 0
		if spec.Team == "" || spec.Team == "No team" {
			teamNameToID[spec.Team] = 0
			continue
		}

		team, err := svc.ds.TeamByName(ctx, spec.Team)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting team by name")
		}
		teamNameToID[spec.Team] = team.ID
	}

	return teamNameToID, nil
}

func (svc *Service) checkCertificateTemplateSpecAuthorization(ctx context.Context, teamNameToID map[string]uint) error {
	for _, teamID := range teamNameToID {
		if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
			return err
		}
	}

	return nil
}

func (svc *Service) ApplyCertificateTemplateSpecs(ctx context.Context, specs []*fleet.CertificateRequestSpec) error {
	teamNameToID, err := svc.resolveTeamNamesForSpecs(ctx, specs)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return err
	}

	if err := svc.checkCertificateTemplateSpecAuthorization(ctx, teamNameToID); err != nil {
		return err
	}

	// Get all of the CAs.
	cas, err := svc.ds.ListCertificateAuthorities(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting all certificate authorities")
	}
	casByID := make(map[uint]*fleet.CertificateAuthoritySummary)
	for _, ca := range cas {
		casByID[ca.ID] = ca
	}

	var certificates []*fleet.CertificateTemplate
	for _, spec := range specs {
		// Get the CA to validate its existence and type.
		ca, ok := casByID[spec.CertificateAuthorityId]
		if !ok {
			return &fleet.BadRequestError{Message: fmt.Sprintf("certificate authority with ID %d not found (certificate %s)", spec.CertificateAuthorityId, spec.Name)}
		}

		if ca.Type != string(fleet.CATypeCustomSCEPProxy) {
			return &fleet.BadRequestError{Message: fmt.Sprintf("Ccertificate `%s`: Currently, only the custom_scep_proxy certificate authority is supported.", spec.Name)}
		}

		// Validate Fleet variables in subject name
		if err := validateCertificateTemplateFleetVariables(spec.SubjectName); err != nil {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%s (certificate %s)", err.Error(), spec.Name)}
		}

		teamID := teamNameToID[spec.Team]

		cert := &fleet.CertificateTemplate{
			Name:                   spec.Name,
			CertificateAuthorityID: spec.CertificateAuthorityId,
			SubjectName:            spec.SubjectName,
			TeamID:                 teamID,
		}

		certificates = append(certificates, cert)
	}

	teamsModified, err := svc.ds.BatchUpsertCertificateTemplates(ctx, certificates)
	if err != nil {
		return err
	}

	// Create pending certificate template records for all enrolled Android hosts in each team.
	for _, cert := range certificates {
		// Get the template ID by querying for it (BatchUpsert doesn't return IDs)
		tmpl, err := svc.ds.GetCertificateTemplateByTeamIDAndName(ctx, cert.TeamID, cert.Name)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting certificate template by team ID and name")
		}
		// Safe to call even for existing templates (it will be a no-op for hosts that already have records)
		if _, err := svc.ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, tmpl.ID, cert.TeamID); err != nil {
			return ctxerr.Wrap(ctx, err, "creating pending certificate templates for existing hosts")
		}
	}

	// Only create activity for teams that actually had certificates affected
	for _, teamID := range teamsModified {
		var tmID *uint
		var tmName *string
		if teamID != 0 {
			team, err := svc.ds.TeamLite(ctx, teamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting team for activity")
			}
			tmID = &team.ID
			tmName = &team.Name
		}

		if err := svc.NewActivity(
			ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedAndroidCertificate{
				TeamID:   tmID,
				TeamName: tmName,
			}); err != nil {
			return ctxerr.Wrap(ctx, err, "logging activity for edited android certificate")
		}
	}

	return nil
}

type deleteCertificateTemplateSpecsRequest struct {
	IDs    []uint `json:"ids"`
	TeamID uint   `json:"team_id"` // If not provided, intentionally defaults to 0 aka "No team"
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

	deletedRows, err := svc.ds.BatchDeleteCertificateTemplates(ctx, certificateTemplateIDs)
	if err != nil {
		return err
	}

	if !deletedRows {
		return nil
	}

	// Delete or mark the certificate templates as pending removal for all android hosts
	for _, certificateTemplateID := range certificateTemplateIDs {
		if err := svc.ds.SetHostCertificateTemplatesToPendingRemove(ctx, certificateTemplateID); err != nil {
			return ctxerr.Wrap(ctx, err, "setting host certificate templates to pending remove")
		}
	}

	// Only create activity if rows were actually deleted
	var tmID *uint
	var tmName *string
	if teamID != 0 {
		team, err := svc.ds.TeamLite(ctx, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team for activity")
		}
		tmID = &team.ID
		tmName = &team.Name
	}

	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedAndroidCertificate{
			TeamID:   tmID,
			TeamName: tmName,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edited android certificate")
	}

	return nil
}

type updateCertificateStatusRequest struct {
	CertificateTemplateID uint   `url:"id"`
	Status                string `json:"status"`
	// OperationType is optional and defaults to "install" if not provided.
	OperationType *string `json:"operation_type,omitempty"`
	// Detail provides additional information about the status change.
	// For example, it can be used to provide a reason for a failed status change.
	Detail *string `json:"detail,omitempty"`
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

	err := svc.UpdateCertificateStatus(ctx, req.CertificateTemplateID, fleet.MDMDeliveryStatus(req.Status), req.Detail, req.OperationType)
	if err != nil {
		return updateCertificateStatusResponse{Err: err}, nil
	}

	return updateCertificateStatusResponse{}, nil
}

func (svc *Service) UpdateCertificateStatus(
	ctx context.Context,
	certificateTemplateID uint,
	status fleet.MDMDeliveryStatus,
	detail *string,
	operationType *string,
) error {
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

	// Default operation_type to "install" if not provided.
	opType := fleet.MDMOperationTypeInstall
	if operationType != nil && *operationType != "" {
		opType = fleet.MDMOperationType(*operationType)
	}

	if !opType.IsValid() {
		return fleet.NewInvalidArgumentError("operation_type", string(opType))
	}

	// Use GetHostCertificateTemplateRecord to query the host_certificate_templates table directly,
	// allowing status updates even when the parent certificate_template has been deleted.
	record, err := svc.ds.GetHostCertificateTemplateRecord(ctx, host.UUID, certificateTemplateID)
	if err != nil {
		return err
	}

	if record.Status != fleet.CertificateTemplateDelivered {
		level.Info(svc.logger).Log("msg", "ignoring certificate status update for non-delivered certificate", "host_uuid", host.UUID, "certificate_template_id", certificateTemplateID, "current_status", record.Status, "new_status", status)
		return nil
	}

	if record.OperationType != opType {
		level.Info(svc.logger).Log("msg", "ignoring certificate status update for different operation type", "host_uuid", host.UUID, "certificate_template_id", certificateTemplateID, "current_operation_type", record.OperationType, "new_operation_type", opType)
		return nil
	}

	return svc.ds.UpsertCertificateStatus(ctx, host.UUID, certificateTemplateID, status, detail, opType)
}
