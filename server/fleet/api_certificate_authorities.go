package fleet

import "net/http"

type ListCertificateAuthoritiesRequest struct{}

type ListCertificateAuthoritiesResponse struct {
	CertificateAuthorities []*CertificateAuthoritySummary `json:"certificate_authorities"`
	Err                    error                          `json:"error,omitempty"`
}

func (r ListCertificateAuthoritiesResponse) Error() error { return r.Err }

type GetCertificateAuthorityRequest struct {
	ID uint `url:"id"`
}

type GetCertificateAuthorityResponse struct {
	*CertificateAuthority
	Err error `json:"error,omitempty"`
}

func (r GetCertificateAuthorityResponse) Error() error { return r.Err }

type CreateCertificateAuthorityRequest struct {
	CertificateAuthorityPayload
}

type CreateCertificateAuthorityResponse struct {
	ID   uint   `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type CAType `json:"type,omitempty"`
	Err  error  `json:"error,omitempty"`
}

func (r CreateCertificateAuthorityResponse) Error() error { return r.Err }

type DeleteCertificateAuthorityRequest struct {
	ID uint `url:"id"`
}

type DeleteCertificateAuthorityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r DeleteCertificateAuthorityResponse) Error() error { return r.Err }

func (r DeleteCertificateAuthorityResponse) Status() int { return http.StatusNoContent }

type UpdateCertificateAuthorityRequest struct {
	ID uint `url:"id"`
	CertificateAuthorityUpdatePayload
}

type UpdateCertificateAuthorityResponse struct {
	Err error `json:"error,omitempty"`
}

func (r UpdateCertificateAuthorityResponse) Error() error { return r.Err }

type RequestCertificateRequest struct {
	RequestCertificatePayload
}

type RequestCertificateResponse struct {
	Certificate string `json:"certificate"`
	Err         error  `json:"error,omitempty"`
}

func (r RequestCertificateResponse) Error() error { return r.Err }

type BatchApplyCertificateAuthoritiesRequest struct {
	CertificateAuthorities GroupedCertificateAuthorities `json:"certificate_authorities"`
	DryRun                 bool                          `json:"dry_run"`
}

type BatchApplyCertificateAuthoritiesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r BatchApplyCertificateAuthoritiesResponse) Error() error { return r.Err }

type GetCertificateAuthoritiesSpecRequest struct {
	IncludeSecrets bool `query:"include_secrets,optional"`
}

type GetCertificateAuthoritiesSpecResponse struct {
	CertificateAuthorities *GroupedCertificateAuthorities `json:"certificate_authorities"`
	Err                    error                          `json:"error,omitempty"`
}

func (r GetCertificateAuthoritiesSpecResponse) Error() error { return r.Err }
