package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Certificate template name validation constants
const (
	maxCertificateTemplateNameLength = 255
)

// certificateTemplateNameRegex allows only letters, numbers, spaces, dashes, and underscores
var certificateTemplateNameRegex = regexp.MustCompile(`^[a-zA-Z0-9 \-_]+$`)

func validateCertificateTemplateSubjectName(subjectName string) error {
	if strings.TrimSpace(subjectName) == "" {
		return &fleet.BadRequestError{Message: "Certificate template subject name is required."}
	}
	return nil
}

// validateCertificateTemplateName validates the certificate template name.
// Returns a BadRequestError if validation fails.
func validateCertificateTemplateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return &fleet.BadRequestError{Message: "Certificate template name is required."}
	}

	if len(name) > maxCertificateTemplateNameLength {
		return &fleet.BadRequestError{Message: fmt.Sprintf("Certificate template name is too long. Maximum is %d characters.", maxCertificateTemplateNameLength)}
	}

	if !certificateTemplateNameRegex.MatchString(name) {
		return &fleet.BadRequestError{Message: "Invalid certificate template name. Only letters, numbers, spaces, dashes, and underscores are allowed."}
	}

	return nil
}

type createCertificateTemplateRequest struct {
	Name                   string `json:"name"`
	TeamID                 uint   `json:"team_id" renameto:"fleet_id"` // If not provided, intentionally defaults to 0 aka "No team"
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
	SubjectAlternativeName string `json:"subject_alternative_name,omitempty"`
}
type createCertificateTemplateResponse struct {
	ID                     uint   `json:"id"`
	Name                   string `json:"name"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
	SubjectAlternativeName string `json:"subject_alternative_name,omitempty"`
	Err                    error  `json:"error,omitempty"`
}

func (r createCertificateTemplateResponse) Error() error { return r.Err }

func createCertificateTemplateEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createCertificateTemplateRequest)
	certificate, err := svc.CreateCertificateTemplate(ctx, req.Name, req.TeamID, req.CertificateAuthorityId, req.SubjectName, req.SubjectAlternativeName)
	if err != nil {
		return createCertificateTemplateResponse{Err: err}, nil
	}
	return createCertificateTemplateResponse{
		ID:                     certificate.ID,
		Name:                   certificate.Name,
		CertificateAuthorityId: certificate.CertificateAuthorityId,
		SubjectName:            certificate.SubjectName,
		SubjectAlternativeName: certificate.SubjectAlternativeName,
	}, nil
}

func (svc *Service) CreateCertificateTemplate(ctx context.Context, name string, teamID uint, certificateAuthorityID uint, subjectName string, subjectAlternativeName string) (*fleet.CertificateTemplateResponse, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Certificate templates require a custom SCEP CA, and CAs are Premium-only (see
	// server/service/certificate_authorities.go core stubs). Reject any create on Free up front.
	lic, err := svc.License(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting license")
	}
	if !lic.IsPremium() {
		return nil, fleet.ErrMissingLicense
	}

	// Validate certificate template name
	if err := validateCertificateTemplateName(name); err != nil {
		return nil, err
	}

	if err := validateCertificateTemplateSubjectName(subjectName); err != nil {
		return nil, err
	}

	if err := validateCertificateTemplateFleetVariables(subjectName); err != nil {
		return nil, &fleet.BadRequestError{Message: err.Error()}
	}

	// Validate the optional SAN: format (token shape, KEY allow-list, length cap) and any
	// $FLEET_VAR_* references against the same allow-list as subject_name.
	if strings.TrimSpace(subjectAlternativeName) != "" {
		if err := validateCertificateTemplateSubjectAlternativeName(subjectAlternativeName, ""); err != nil {
			return nil, err
		}
		if err := validateCertificateTemplateFleetVariables(subjectAlternativeName); err != nil {
			return nil, fleet.NewInvalidArgumentError("subject_alternative_name", err.Error())
		}
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
		SubjectAlternativeName: subjectAlternativeName,
	}

	savedTemplate, err := svc.ds.CreateCertificateTemplate(ctx, certTemplate)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating certificate template")
	}

	// Create pending certificate template records for all enrolled Android hosts in the team
	if _, err := svc.ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, savedTemplate.ID, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating pending certificate templates for existing hosts")
	}

	activity := fleet.ActivityTypeAddedCertificate{
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
	TeamID uint `query:"team_id,optional" renameto:"fleet_id"`
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
	opts.OrderKey = "id"
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

	// Memo for the host's end-user list, shared between subject_name and subject_alternative_name
	// expansion so we don't double-fetch when both reference $FLEET_VAR_HOST_END_USER_IDP_USERNAME.
	var endUsersMemo []fleet.HostEndUser

	subjectName, ok, err := svc.expandCertVar(ctx, certificate, certificate.SubjectName,
		"Could not replace certificate variables", host, &endUsersMemo)
	if err != nil {
		return nil, err
	}
	if !ok {
		return certificate, nil
	}
	certificate.SubjectName = subjectName

	if certificate.SubjectAlternativeName != "" {
		san, ok, err := svc.expandCertVar(ctx, certificate, certificate.SubjectAlternativeName,
			"Could not replace certificate variables in subject_alternative_name", host, &endUsersMemo)
		if err != nil {
			return nil, err
		}
		if !ok {
			return certificate, nil
		}
		certificate.SubjectAlternativeName = san
	}

	// On-demand challenge creation for delivered status.
	// If FleetChallenge is nil or empty, create one now (the challenge TTL starts from this moment).
	if certificate.Status == fleet.CertificateTemplateDelivered {
		if certificate.FleetChallenge == nil || *certificate.FleetChallenge == "" {
			challenge, err := svc.ds.GetOrCreateFleetChallengeForCertificateTemplate(ctx, host.UUID, id)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "create fleet challenge on-demand")
			}
			certificate.FleetChallenge = &challenge
		}
	}

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

	// Certificate templates require a custom SCEP CA, and CAs are Premium-only.
	if len(specs) > 0 {
		lic, err := svc.License(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting license")
		}
		if !lic.IsPremium() {
			return fleet.ErrMissingLicense
		}
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
		// Validate certificate template name
		if err := validateCertificateTemplateName(spec.Name); err != nil {
			return err
		}

		if err := validateCertificateTemplateSubjectName(spec.SubjectName); err != nil {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%s (certificate %s)", err.Error(), spec.Name)}
		}

		// Get the CA to validate its existence and type.
		ca, ok := casByID[spec.CertificateAuthorityId]
		if !ok {
			return &fleet.BadRequestError{Message: fmt.Sprintf("certificate authority with ID %d not found (certificate %s)", spec.CertificateAuthorityId, spec.Name)}
		}

		if ca.Type != string(fleet.CATypeCustomSCEPProxy) {
			return &fleet.BadRequestError{Message: fmt.Sprintf("Certificate `%s`: Currently, only the custom_scep_proxy certificate authority is supported.", spec.Name)}
		}

		// Validate Fleet variables in subject name
		if err := validateCertificateTemplateFleetVariables(spec.SubjectName); err != nil {
			return &fleet.BadRequestError{Message: fmt.Sprintf("%s (certificate %s)", err.Error(), spec.Name)}
		}

		// Validate the optional SAN for format and variables.
		if strings.TrimSpace(spec.SubjectAlternativeName) != "" {
			if err := validateCertificateTemplateSubjectAlternativeName(spec.SubjectAlternativeName, spec.Name); err != nil {
				return err
			}
			if err := validateCertificateTemplateFleetVariables(spec.SubjectAlternativeName); err != nil {
				return fleet.NewInvalidArgumentError("subject_alternative_name",
					fmt.Sprintf("%s (certificate %s)", err.Error(), spec.Name))
			}
		}

		teamID := teamNameToID[spec.Team]

		cert := &fleet.CertificateTemplate{
			Name:                   spec.Name,
			CertificateAuthorityID: spec.CertificateAuthorityId,
			SubjectName:            spec.SubjectName,
			SubjectAlternativeName: spec.SubjectAlternativeName,
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
	TeamID uint   `json:"team_id" renameto:"fleet_id"` // If not provided, intentionally defaults to 0 aka "No team"
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
	// Authorize team
	if err := svc.authz.Authorize(ctx, &fleet.CertificateTemplate{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}
	// Authorize all ids are on team
	certificateTemplates, err := svc.ds.GetCertificateTemplatesByIdsAndTeam(ctx, certificateTemplateIDs, teamID)
	if err != nil {
		return err
	}
	uniqueIDs := make(map[uint]struct{}, len(certificateTemplateIDs))
	for _, id := range certificateTemplateIDs {
		uniqueIDs[id] = struct{}{}
	}
	if len(uniqueIDs) != len(certificateTemplates) {
		return authz.ForbiddenWithInternal(
			"can only delete templates from team parameter",
			authz.UserFromContext(ctx),
			&fleet.CertificateTemplate{TeamID: teamID},
			fleet.ActionWrite,
		)
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
	// Certificate validity fields - reported by device after successful enrollment
	NotValidBefore *time.Time `json:"not_valid_before,omitempty"`
	NotValidAfter  *time.Time `json:"not_valid_after,omitempty"`
	Serial         *string    `json:"serial,omitempty"`
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

	// Default operation_type to "install" if not provided.
	opType := fleet.MDMOperationTypeInstall
	if req.OperationType != nil && *req.OperationType != "" {
		opType = fleet.MDMOperationType(*req.OperationType)
	}

	err := svc.UpdateCertificateStatus(ctx, &fleet.CertificateStatusUpdate{
		CertificateTemplateID: req.CertificateTemplateID,
		Status:                fleet.MDMDeliveryStatus(req.Status),
		Detail:                req.Detail,
		OperationType:         opType,
		NotValidBefore:        req.NotValidBefore,
		NotValidAfter:         req.NotValidAfter,
		Serial:                req.Serial,
	})
	if err != nil {
		return updateCertificateStatusResponse{Err: err}, nil
	}

	return updateCertificateStatusResponse{}, nil
}

func (svc *Service) UpdateCertificateStatus(ctx context.Context, update *fleet.CertificateStatusUpdate) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	// Validate the status.
	if !update.Status.IsValid() {
		return fleet.NewInvalidArgumentError("status", string(update.Status))
	}

	if !update.OperationType.IsValid() {
		return fleet.NewInvalidArgumentError("operation_type", string(update.OperationType))
	}

	// Use GetHostCertificateTemplateRecord to query the host_certificate_templates table directly,
	// allowing status updates even when the parent certificate_template has been deleted.
	record, err := svc.ds.GetHostCertificateTemplateRecord(ctx, host.UUID, update.CertificateTemplateID)
	if err != nil {
		return err
	}

	if record.OperationType != update.OperationType {
		svc.logger.InfoContext(ctx, "ignoring certificate status update for different operation type", "host_uuid", host.UUID, "certificate_template_id", update.CertificateTemplateID, "current_operation_type", record.OperationType, "new_operation_type", update.OperationType)
		return nil
	}

	// If operation_type is "remove" and status is "verified", delete the host_certificate_template row.
	// This allows deletions even when there are race conditions or status sync issues
	// (e.g., device reports removal before server transitions status).
	if update.OperationType == fleet.MDMOperationTypeRemove && update.Status == fleet.MDMDeliveryVerified {
		return svc.ds.DeleteHostCertificateTemplate(ctx, host.UUID, update.CertificateTemplateID)
	}

	if record.Status != fleet.CertificateTemplateDelivered {
		svc.logger.InfoContext(ctx, "ignoring certificate status update for non-delivered certificate", "host_uuid", host.UUID, "certificate_template_id", update.CertificateTemplateID, "current_status", record.Status, "new_status", update.Status)
		return nil
	}

	// Fill in HostUUID from context
	update.HostUUID = host.UUID

	// Log activity for install statuses (not removals). Failures are logged on every attempt
	// (including retries) so IT admins have visibility into retry attempts.
	if update.OperationType == fleet.MDMOperationTypeInstall {
		var actStatus fleet.CertificateActivityStatus
		switch update.Status {
		case fleet.MDMDeliveryVerified:
			actStatus = fleet.CertificateActivityInstalled
		case fleet.MDMDeliveryFailed:
			actStatus = fleet.CertificateActivityFailedInstall
		}
		if actStatus != "" {
			detail := ""
			if update.Detail != nil {
				detail = *update.Detail
			}
			if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeInstalledCertificate{
				HostID:                host.ID,
				HostDisplayName:       host.DisplayName(),
				CertificateTemplateID: update.CertificateTemplateID,
				CertificateName:       record.Name,
				Status:                string(actStatus),
				Detail:                detail,
			}); err != nil {
				// Log and continue since we don't want the client to fail on this.
				svc.logger.ErrorContext(ctx, "failed to create certificate install activity", "host.id", host.ID, "activity.status", actStatus,
					"err", err)
				ctxerr.Handle(ctx, err)
			}
		}
	}

	// For failed installs, automatically retry if under the retry limit.
	if update.OperationType == fleet.MDMOperationTypeInstall && update.Status == fleet.MDMDeliveryFailed {
		if record.RetryCount < fleet.MaxCertificateInstallRetries {
			detail := ""
			if update.Detail != nil {
				detail = *update.Detail
			}
			if err := svc.ds.RetryHostCertificateTemplate(ctx, host.UUID, update.CertificateTemplateID, detail); err != nil {
				return ctxerr.Wrap(ctx, err, "retrying certificate install")
			}
			return nil
		}
	}

	return svc.ds.UpsertCertificateStatus(ctx, update)
}

////////////////////////////////////////////////////////////////////////////////
// Resend Host Certificate Template
////////////////////////////////////////////////////////////////////////////////

type resendHostCertificateTemplateRequest struct {
	ID         uint `url:"id"`
	TemplateID uint `url:"template_id"`
}

type resendHostCertificateTemplateResponse struct {
	Err error `json:"error,omitempty"`
}

func (r resendHostCertificateTemplateResponse) Error() error { return r.Err }

func resendHostCertificateTemplateEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*resendHostCertificateTemplateRequest)
	err := svc.ResendHostCertificateTemplate(ctx, req.ID, req.TemplateID)
	if err != nil {
		return resendHostCertificateTemplateResponse{Err: err}, nil
	}

	return resendHostCertificateTemplateResponse{}, nil
}

func (svc *Service) ResendHostCertificateTemplate(ctx context.Context, hostID uint, templateID uint) error {
	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err)
	}

	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: host.TeamID}, fleet.ActionResend); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	template, err := svc.ds.GetCertificateTemplateByIdForHost(ctx, templateID, host.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking host certificate template")
	}

	if template.Status == fleet.CertificateTemplatePending {
		return fleet.NewUserMessageError(errors.New("Couldn't resend pending certificate template."), http.StatusBadRequest)
	}

	if err := svc.ds.ResendHostCertificateTemplate(ctx, hostID, templateID); err != nil {
		return ctxerr.Wrap(ctx, err, "resending certificate template")
	}

	certificate, err := svc.ds.GetCertificateTemplateById(ctx, templateID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting certificate details")
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeResentCertificate{
		HostID:                host.ID,
		HostDisplayName:       host.DisplayName(),
		CertificateTemplateID: certificate.ID,
		CertificateName:       certificate.Name,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity")
	}

	return nil
}
