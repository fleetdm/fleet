package android

type SignupDetails struct {
	Url  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

type Enterprise struct {
	ID           uint   `db:"id"`
	SignupName   string `db:"signup_name"`
	EnterpriseID string `db:"enterprise_id"`
}

func (e Enterprise) Name() string {
	return "enterprises/" + e.EnterpriseID
}

type EnrollmentToken struct {
	Value string `json:"value"`
}
