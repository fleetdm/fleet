package fleet

// ConditionalAccessIDPAssets is used for authorization checks on Okta IdP assets
// (signing certificate, Apple profile, etc.).
// It represents the resource being accessed when downloading these assets.
type ConditionalAccessIDPAssets struct{}

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessIDPAssets) AuthzType() string {
	return "conditional_access_idp_assets"
}
