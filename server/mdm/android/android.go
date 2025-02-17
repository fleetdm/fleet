package android

type SignupDetails struct {
	Url  string
	Name string
}

type Enterprise struct {
	ID           uint   `db:"id"`
	EnterpriseID string `db:"enterprise_id"`
}

func (e Enterprise) Name() string {
	return "enterprises/" + e.EnterpriseID
}

func (e Enterprise) IsValid() bool {
	return e.EnterpriseID != ""
}

func (e Enterprise) AuthzType() string {
	return "android_enterprise"
}

type EnterpriseDetails struct {
	Enterprise
	SignupName  string `db:"signup_name"`
	TopicID     string `db:"topic_id"`
	SignupToken string `db:"signup_token"`
}

type EnrollmentToken struct {
	Value string `json:"value"`
}

type Host struct {
	HostID            uint   `db:"host_id"`
	FleetEnterpriseID uint   `db:"enterprise_id"`
	DeviceID          string `db:"device_id"`
}
