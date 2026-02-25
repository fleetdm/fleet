package fleet

type SetupRequest struct {
	Admin        *UserPayload `json:"admin"`
	OrgInfo      *OrgInfo     `json:"org_info"`
	ServerURL    *string      `json:"server_url,omitempty"`
	EnrollSecret *string      `json:"osquery_enroll_secret,omitempty"`
}

type SetupResponse struct {
	Admin        *User    `json:"admin,omitempty"`
	OrgInfo      *OrgInfo `json:"org_info,omitempty"`
	ServerURL    *string  `json:"server_url"`
	EnrollSecret *string  `json:"osquery_enroll_secret"`
	Token        *string  `json:"token,omitempty"`
	Err          error    `json:"error,omitempty"`
}

func (r SetupResponse) Error() error { return r.Err }
