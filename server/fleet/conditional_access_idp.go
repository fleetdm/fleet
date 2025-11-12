package fleet

// ConditionalAccessIDPCert is used for authorization checks on the Okta IdP signing certificate.
// It represents the resource being accessed when downloading the certificate.
type ConditionalAccessIDPCert struct{}

// AuthzType implements authz.AuthzTyper.
func (c *ConditionalAccessIDPCert) AuthzType() string {
	return "conditional_access_idp_cert"
}
