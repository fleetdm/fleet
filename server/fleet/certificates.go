package fleet

type CertificateRequestSpec struct {
	Name                   string `json:"name"`
	Team                   string `json:"team"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	SubjectName            string `json:"subject_name"`
}

type Certificate struct {
	Name                   string
	TeamID                 uint
	CertificateAuthorityID uint
	SubjectName            string
}

type CertificateSummary struct {
	ID                     int    `json:"id"`
	CertificateAuthorityId uint   `json:"certificate_authority_id"`
	Name                   string `json:"name"`
	SubjectName            string `json:"subject_name"`
}
