package fleet

import "time"

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

// ConditionalAccessSCEPPath is the path of Fleet's conditional access SCEP server.
const ConditionalAccessSCEPPath = "/api/fleet/conditional_access/scep"

// ConditionalAccessSCEPChallengeTTL is how long a one-time conditional access SCEP challenge is valid. As with Apple
// renewal challenges, a host can defer the profile (NotNow), and every re-send carries the same challenge until it's
// consumed or expires.
const ConditionalAccessSCEPChallengeTTL = 30 * 24 * time.Hour

// ConditionalAccessSCEPChallengePurpose marks one-time challenges for Fleet's conditional access SCEP server, on every
// platform. They're stored with the Apple MDM SCEP challenges, and Fleet's Apple MDM SCEP server never accepts them.
const ConditionalAccessSCEPChallengePurpose AppleMDMCertPurpose = "conditional_access"

// ConditionalAccessManagedCertificateCAName is the ca_name of the host_mdm_managed_certificates rows for certificates
// issued by Fleet's conditional access SCEP server.
const ConditionalAccessManagedCertificateCAName = "Conditional access"

type ConditionalAccessWindowsProfiles struct {
	SCEPProfile    string `json:"scep_profile"`
	BrowserProfile string `json:"browser_profile"`
}

// OktaCACleanupTarget identifies where to run the Okta conditional access
// keychain-cleanup script after a successful InstallProfile ack for the
// Okta CA profile.
type OktaCACleanupTarget struct {
	HostID        uint   `db:"host_id"`
	UserShortName string `db:"user_short_name"`
}
