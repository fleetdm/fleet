package fleet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DefaultIDPConnectionName is the directory used when no named connection is
// configured. Existing SCIM users are assigned to it.
const DefaultIDPConnectionName = "default"

type idpConnectionContextKey struct{}

// NewContextWithIDPConnection scopes SCIM reads and writes to one connection.
func NewContextWithIDPConnection(ctx context.Context, id uint) context.Context {
	return context.WithValue(ctx, idpConnectionContextKey{}, id)
}

// IDPConnectionFromContext returns the SCIM directory selected for this request.
func IDPConnectionFromContext(ctx context.Context) (uint, bool) {
	id, ok := ctx.Value(idpConnectionContextKey{}).(uint)
	return id, ok && id != 0
}

// HashSCIMToken is the stored form of a connection's SCIM bearer token.
func HashSCIMToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// MDMIdentityProvider is one named SAML connection. Several fleets can reference
// the same name. At most one connection is the organization default.
type MDMIdentityProvider struct {
	// Name is the stable reference fleets store. It is not the SAML idp_name.
	Name string `json:"name"`
	SSOProviderSettings
	// Default marks the connection used by fleets that do not name one.
	Default bool `json:"default"`
}

// EffectiveSettings returns the SAML settings for this connection. An empty
// idp_name falls back to Name so a connection is complete without repeating it.
func (p MDMIdentityProvider) EffectiveSettings() SSOProviderSettings {
	s := p.SSOProviderSettings
	if strings.TrimSpace(s.IDPName) == "" {
		s.IDPName = p.Name
	}
	return s
}

// Configured reports whether the SAML settings have the fields Fleet requires
// to start an end-user sign-in.
func (s SSOProviderSettings) Configured() bool {
	return strings.TrimSpace(s.EntityID) != "" &&
		strings.TrimSpace(s.IDPName) != "" &&
		(strings.TrimSpace(s.Metadata) != "" || strings.TrimSpace(s.MetadataURL) != "")
}

// IdentityProviderByName returns the connection with that name.
func (m MDM) IdentityProviderByName(name string) (MDMIdentityProvider, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return MDMIdentityProvider{}, false
	}
	for _, p := range m.IdentityProviders {
		if p.Name == name {
			return p, true
		}
	}
	return MDMIdentityProvider{}, false
}

// DefaultIdentityProvider returns the connection marked default.
func (m MDM) DefaultIdentityProvider() (MDMIdentityProvider, bool) {
	for _, p := range m.IdentityProviders {
		if p.Default {
			return p, true
		}
	}
	return MDMIdentityProvider{}, false
}

// EndUserSSOSettings resolves the SAML settings for a fleet. providerName empty
// uses the default connection, then the legacy end_user_authentication settings.
// A name that is not defined is an error and does not fall through to the default.
func (m MDM) EndUserSSOSettings(providerName string) (SSOProviderSettings, error) {
	providerName = strings.TrimSpace(providerName)
	if providerName != "" {
		p, ok := m.IdentityProviderByName(providerName)
		if !ok {
			return SSOProviderSettings{}, fmt.Errorf("identity provider %q is not configured", providerName)
		}
		s := p.EffectiveSettings()
		if !s.Configured() {
			return SSOProviderSettings{}, fmt.Errorf("identity provider %q is incomplete", providerName)
		}
		return s, nil
	}
	if p, ok := m.DefaultIdentityProvider(); ok {
		s := p.EffectiveSettings()
		if !s.Configured() {
			return SSOProviderSettings{}, fmt.Errorf("default identity provider %q is incomplete", p.Name)
		}
		return s, nil
	}
	if m.EndUserAuthentication.IsEmpty() {
		return SSOProviderSettings{}, fmt.Errorf("organization not configured to use sso")
	}
	return m.EndUserAuthentication.SSOProviderSettings, nil
}

// EndUserAuthConfigured reports whether the resolved IdP has entity ID, IdP
// name, and metadata or a metadata URL. GitOps uses this so a partial legacy
// block cannot leave end-user authentication enabled.
func (m MDM) EndUserAuthConfigured(providerName string) bool {
	providerName = strings.TrimSpace(providerName)
	if providerName != "" {
		p, ok := m.IdentityProviderByName(providerName)
		return ok && p.EffectiveSettings().Configured()
	}
	if p, ok := m.DefaultIdentityProvider(); ok {
		return p.EffectiveSettings().Configured()
	}
	return m.EndUserAuthentication.Configured()
}

// EndUserAuthAvailable reports whether end-user authentication can be turned on
// for a fleet. An empty providerName uses the org default, then the legacy
// end_user_authentication settings. Legacy settings stay available when any
// field is set, matching the historical IsEmpty check. A named connection must
// be complete.
func (m MDM) EndUserAuthAvailable(providerName string) bool {
	providerName = strings.TrimSpace(providerName)
	if providerName != "" {
		p, ok := m.IdentityProviderByName(providerName)
		return ok && p.EffectiveSettings().Configured()
	}
	if p, ok := m.DefaultIdentityProvider(); ok {
		return p.EffectiveSettings().Configured()
	}
	return !m.EndUserAuthentication.IsEmpty()
}

// NormalizeAndValidateIdentityProviders trims fields, fills an empty idp_name
// from the connection name, and reports duplicate names, more than one default,
// and incomplete connections.
func (m *MDM) NormalizeAndValidateIdentityProviders(invalid *InvalidArgumentError) {
	if m == nil || len(m.IdentityProviders) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(m.IdentityProviders))
	defaults := 0
	for i := range m.IdentityProviders {
		p := &m.IdentityProviders[i]
		p.Name = strings.TrimSpace(p.Name)
		p.EntityID = strings.TrimSpace(p.EntityID)
		p.IDPName = strings.TrimSpace(p.IDPName)
		p.Metadata = strings.TrimSpace(p.Metadata)
		p.MetadataURL = strings.TrimSpace(p.MetadataURL)
		field := fmt.Sprintf("identity_providers.%s", p.Name)
		if p.Name == "" {
			field = fmt.Sprintf("identity_providers[%d]", i)
			invalid.Append(field, "name is required")
			continue
		}
		if _, ok := seen[p.Name]; ok {
			invalid.Append(field, "duplicate name")
		}
		seen[p.Name] = struct{}{}
		if p.Default {
			defaults++
		}
		if p.EntityID == "" {
			invalid.Append(field+".entity_id", "required")
		}
		if p.Metadata == "" && p.MetadataURL == "" {
			invalid.Append(field+".metadata", "either metadata or metadata_url must be defined")
		}
		if p.MetadataURL != "" {
			u, err := url.ParseRequestURI(p.MetadataURL)
			if err != nil {
				invalid.Append(field+".metadata_url", err.Error())
			} else if u.Scheme != "https" && u.Scheme != "http" {
				invalid.Append(field+".metadata_url", "must be either https or http")
			}
		}
		if p.IDPName == "" {
			p.IDPName = p.Name
		}
	}
	if defaults > 1 {
		invalid.Append("identity_providers", "only one identity provider can be the default")
	}
}

// SSOFleetRef signs a fleet id so an enrollment profile can name its IdP
// without letting the end user swap in another fleet's id.
func SSOFleetRef(secret string, teamID uint) string {
	id := strconv.FormatUint(uint64(teamID), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(id))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return sig + "." + id
}

// ParseSSOFleetRef returns the fleet id when ref was produced by SSOFleetRef
// with the same secret. team id 0 is the unassigned fleet.
func ParseSSOFleetRef(secret, ref string) (uint, bool) {
	if secret == "" || ref == "" {
		return 0, false
	}
	sig, id, ok := strings.Cut(ref, ".")
	if !ok || sig == "" || id == "" {
		return 0, false
	}
	teamID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		return 0, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(id))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(sig)) != 1 {
		return 0, false
	}
	return uint(teamID), true
}
