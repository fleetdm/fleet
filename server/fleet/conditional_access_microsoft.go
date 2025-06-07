package fleet

// ConditionalAccessMicrosoftIntegrations holds settings for a "Conditional access" integration.
type ConditionalAccessMicrosoftIntegration struct {
	// TenantID is the Entra's tenant ID.
	TenantID string `db:"tenant_id"`
	// ProxyServerSecret is the secret used to authenticate a Cloud instance.
	ProxyServerSecret string `db:"proxy_server_secret"`
	// SetupDone is true when the Entra admin has consented and the tenant has been provisioned.
	SetupDone bool `db:"setup_done"`
}

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessMicrosoftIntegration) AuthzType() string {
	return "conditional_access_microsoft"
}

// HostConditionalAccessStatus holds "Conditional access" status for a host.
type HostConditionalAccessStatus struct {
	// HostID is the host's ID.
	HostID uint `db:"host_id"`

	// DeviceID is Entra's Device ID assigned when the device first logs in to Entra (obtained using a detail query).
	DeviceID string `db:"device_id"`
	// DeviceID is Entra's User Principal Name that logged in the device (obtained using a detail query).
	UserPrincipalName string `db:"user_principal_name"`

	// Managed holds the last "DeviceManagementState" reported to Entra.
	// It is true if the host is MDM enrolled, false otherwise.
	//
	// This field is used to know if Fleet needs to update the status on Entra.
	Managed *bool `db:"managed"`
	// Compliant holds the last "complianceStatus" reported to Entra.
	// It is true if all configured policies are passing.
	//
	// This field is used to know if Fleet needs to update the status on Entra.
	Compliant *bool `db:"compliant"`

	// DisplayName is the host's display name to reported to Entra.
	DisplayName string `db:"display_name"`
	// OSVersion is the host's OS version reported to Entra
	OSVersion string `db:"os_version"`

	// UpdateCreateTimestamps holds the timestamps of the entry.
	UpdateCreateTimestamps
}
