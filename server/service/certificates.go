package service

import (
	"context"
	"strconv"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

type listCertificateTemplatesRequest struct {
	TeamID  uint `query:"team_id"`
	Page    uint `query:"page,optional"`
	PerPage uint `query:"per_page,optional"`
}

type listCertificateTemplatesResponse struct {
	Certificates []*fleet.CertificateTemplateResponseSummary `json:"certificates"`
	Err          error                                       `json:"error,omitempty"`
	Meta         *fleet.PaginationMetadata                   `json:"meta"`
}

func (r listCertificateTemplatesResponse) Error() error { return r.Err }

func listCertificateTemplatesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*listCertificateTemplatesRequest)
	certificates, err := svc.ListCertificateTemplates(ctx, req.TeamID, req.Page, req.PerPage)
	if err != nil {
		return listCertificateTemplatesResponse{Err: err}, nil
	}
	return listCertificateTemplatesResponse{Certificates: certificates}, nil
}

func (svc *Service) ListCertificateTemplates(ctx context.Context, teamID uint, page uint, perPage uint) ([]*fleet.CertificateTemplateResponseSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	certificates, err := svc.ds.GetCertificateTemplatesByTeamID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	return certificates, nil
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
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	certificate, err := svc.ds.GetCertificateTemplateById(ctx, id)
	if err != nil {
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

func (svc *Service) ApplyCertificateTemplateSpecs(ctx context.Context, specs []*fleet.CertificateRequestSpec) error {
	// TODO: What is the right authorization here?
	// svc.authz.Authorize(ctx, &fleet.Certificate{TeamID: tmID}, fleet.ActionWrite) ?
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return err
	}

	var certificates []*fleet.CertificateTemplate
	for _, spec := range specs {
		var teamID uint
		if spec.Team != "" {
			parsed, err := strconv.ParseUint(spec.Team, 10, 0)
			if err != nil {
				return err
			}
			teamID = uint(parsed)
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
	IDs []uint `json:"ids"`
	// TeamID uint   `json:"team_id"` ??
}

type deleteCertificateTemplateSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteCertificateTemplateSpecsResponse) Error() error { return r.Err }

func deleteCertificateTemplateSpecsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteCertificateTemplateSpecsRequest)
	err := svc.DeleteCertificateTemplateSpecs(ctx, req.IDs)
	if err != nil {
		return deleteCertificateTemplateSpecsResponse{Err: err}, nil
	}
	return deleteCertificateTemplateSpecsResponse{}, nil
}

func (svc *Service) DeleteCertificateTemplateSpecs(ctx context.Context, certificateTemplateIDs []uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionWrite); err != nil {
		return err
	}
	return svc.ds.BatchDeleteCertificateTemplates(ctx, certificateTemplateIDs)
}

// replaceCertificateVariables replaces FLEET_VAR_* variables in the subject name with actual host values
func (svc *Service) replaceCertificateVariables(ctx context.Context, subjectName string, host *fleet.Host) (string, error) {
	fleetVars := variables.Find(subjectName)
	if len(fleetVars) == 0 {
		return subjectName, nil
	}

	result := subjectName
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case "HOST_UUID":
			result = fleet.FleetVarHostUUIDRegexp.ReplaceAllString(result, host.UUID)
		case "HOST_HARDWARE_SERIAL":
			result = fleet.FleetVarHostHardwareSerialRegexp.ReplaceAllString(result, host.HardwareSerial)
		case "HOST_END_USER_IDP_USERNAME":
			users, err := fleet.GetEndUsers(ctx, svc.ds, host.ID)
			if err != nil {
				return "", ctxerr.Wrapf(ctx, err, "getting host end users for variable %s", fleetVar)
			}
			if len(users) == 0 || users[0].IdpUserName == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have an IDP username for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostEndUserIDPUsernameRegexp.ReplaceAllString(result, users[0].IdpUserName)
		}
	}

	return result, nil
}
