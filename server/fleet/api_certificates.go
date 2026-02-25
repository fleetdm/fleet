package fleet

import "time"

type CreateCertificateTemplateRequest struct {
	Name                   string `json:"name"`
	TeamID                 uint   `json:"team_id" renameto:"fleet_id"` // If not provided, intentionally defaults to 0 aka "No team"
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
}

type CreateCertificateTemplateResponse struct {
	ID                     uint   `json:"id"`
	Name                   string `json:"name"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
	Err                    error  `json:"error,omitempty"`
}

func (r CreateCertificateTemplateResponse) Error() error { return r.Err }

type ListCertificateTemplatesRequest struct {
	ListOptions

	// If not provided, intentionally defaults to 0 aka "No team"
	TeamID uint `query:"team_id,optional" renameto:"fleet_id"`
}

type ListCertificateTemplatesResponse struct {
	Certificates []*CertificateTemplateResponseSummary `json:"certificates"`
	Err          error                                 `json:"error,omitempty"`
	Meta         *PaginationMetadata                   `json:"meta"`
}

func (r ListCertificateTemplatesResponse) Error() error { return r.Err }

type GetDeviceCertificateTemplateRequest struct {
	ID uint `url:"id"`
}

type GetDeviceCertificateTemplateResponse struct {
	Certificate *CertificateTemplateResponseForHost `json:"certificate"`
	Err         error                               `json:"error,omitempty"`
}

func (r GetDeviceCertificateTemplateResponse) Error() error { return r.Err }

type GetCertificateTemplateRequest struct {
	ID uint `url:"id"`
}

type GetCertificateTemplateResponse struct {
	Certificate *CertificateTemplateResponse `json:"certificate"`
	Err         error                        `json:"error,omitempty"`
}

func (r GetCertificateTemplateResponse) Error() error { return r.Err }

type DeleteCertificateTemplateRequest struct {
	ID uint `url:"id"`
}

type DeleteCertificateTemplateResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteCertificateTemplateResponse) Error() error { return r.Err }

type ApplyCertificateTemplateSpecsRequest struct {
	Specs []*CertificateRequestSpec `json:"specs"`
}

type ApplyCertificateTemplateSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r ApplyCertificateTemplateSpecsResponse) Error() error { return r.Err }

type DeleteCertificateTemplateSpecsRequest struct {
	IDs    []uint `json:"ids"`
	TeamID uint   `json:"team_id" renameto:"fleet_id"` // If not provided, intentionally defaults to 0 aka "No team"
}

type DeleteCertificateTemplateSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteCertificateTemplateSpecsResponse) Error() error { return r.Err }

type UpdateCertificateStatusRequest struct {
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

type UpdateCertificateStatusResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UpdateCertificateStatusResponse) Error() error { return r.Err }
