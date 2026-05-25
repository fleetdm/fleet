package fleet

// ConditionalAccessIDPAssets is used for authorization checks on Okta IdP assets
// (signing certificate, Apple profile, etc.).
// It represents the resource being accessed when downloading these assets.
type ConditionalAccessIDPAssets struct{}

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessIDPAssets) AuthzType() string {
	return "conditional_access_idp_assets"
}

// ConditionalAccessOktaProfileIdentifier is the top-level PayloadIdentifier
// of the .mobileconfig profile delivered for Okta conditional access. It is
// also the value stored in host_mdm_apple_profiles.profile_identifier for
// hosts that have received the profile, and is used to detect when an
// InstallProfile ack applies to the Okta CA profile.
const ConditionalAccessOktaProfileIdentifier = "com.fleetdm.conditional-access-okta"

// ConditionalAccessOktaCertificateCN is the Subject CN of the SCEP
// certificate issued for Okta conditional access. The duplicate-cert
// cleanup script matches certificates in the keychain by this CN.
const ConditionalAccessOktaCertificateCN = "Fleet conditional access for Okta"

// OktaCACleanupTarget identifies where to run the Okta conditional access
// keychain-cleanup script after a successful InstallProfile ack for the
// Okta CA profile.
type OktaCACleanupTarget struct {
	HostID        uint   `db:"host_id"`
	UserShortName string `db:"user_short_name"`
}
