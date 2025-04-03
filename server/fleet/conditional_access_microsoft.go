package fleet

type ConditionalAccessMicrosoftIntegration struct {
	TenantID          string `db:"tenant_id"`
	ProxyServerSecret string `db:"proxy_server_secret"`
	SetupDone         bool   `db:"setup_done"`
}

func (c *ConditionalAccessMicrosoftIntegration) AuthzType() string {
	return "conditional_access_microsoft"
}

type HostConditionalAccessStatus struct {
	HostID    uint   `db:"host_id"`
	DeviceID  string `db:"device_id"`
	Compliant *bool  `db:"compliant"`

	DisplayName string `db:"display_name"`
	OSVersion   string `db:"os_version"`

	UpdateCreateTimestamps
}
