package fleet

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"
)

// Auth contains methods to fetch information from a valid SSO SAMLResponse
type Auth interface {
	// UserID returns the Subject Name Identifier associated with the request,
	// this can be an email address, an entity identifier, or any other valid
	// Name Identifier as described in the spec:
	// http://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf
	//
	// Fleet requires users to configure this value to be the email of the Subject
	UserID() string
	// UserDisplayName finds a display name in the SSO response Attributes, there
	// isn't a defined spec for this, so the return value is in a best-effort
	// basis
	UserDisplayName() string
	// AssertionAttributes returns the attributes of the SAML response.
	AssertionAttributes() []SAMLAttribute
}

// SAMLAttribute holds the name and values of a custom attribute.
type SAMLAttribute struct {
	Name   string
	Values []SAMLAttributeValue
}

// SAMLAttributeValue holds the type and value of a custom attribute.
type SAMLAttributeValue struct {
	// Type is the type of attribute value.
	Type string
	// Value is the actual value of the attribute.
	Value string
}

type SSOSession struct {
	Token       string
	RedirectURL string
}

// SessionSSOSettings SSO information used prior to authentication.
type SessionSSOSettings struct {
	// IDPName is a human readable name for the IDP
	IDPName string `json:"idp_name"`
	// IDPImageURL https link to a logo image for the IDP.
	IDPImageURL string `json:"idp_image_url"`
	// SSOEnabled true if single sign on is enabled.
	SSOEnabled bool `json:"sso_enabled"`
}

// Session is the model object which represents what an active session is
type Session struct {
	CreateTimestamp
	ID         uint
	AccessedAt time.Time `db:"accessed_at"`
	UserID     uint      `json:"user_id" db:"user_id"`
	Key        string
	APIOnly    *bool `json:"-" db:"api_only"`
}

func (s Session) AuthzType() string {
	return "session"
}

// SSORolesInfo holds the configuration parsed from SAML custom attributes.
//
// `Global` and `Teams` are never both set (by design, users must be either global
// or member of teams).
type SSORolesInfo struct {
	// Global holds the role for the Global domain.
	Global *string
	// Teams holds the roles for teams.
	Teams []TeamRole
}

// TeamRole holds a user's role on a Team.
type TeamRole struct {
	// ID is the unique identifier of the team.
	ID uint
	// Role is the role of the user in the team.
	Role string
}

func (s SSORolesInfo) verify() error {
	if s.Global != nil && len(s.Teams) > 0 {
		return errors.New("cannot set both global and team roles")
	}
	// Check for duplicate entries for the same team.
	// This is just in case some IdP allows duplicating attributes.
	teamSet := make(map[uint]struct{})
	for _, teamRole := range s.Teams {
		if _, ok := teamSet[teamRole.ID]; ok {
			return fmt.Errorf("duplicate team entry: %d", teamRole.ID)
		}
		teamSet[teamRole.ID] = struct{}{}
	}
	return nil
}

// IsSet returns whether any role attributes were set.
func (s SSORolesInfo) IsSet() bool {
	return s.Global != nil || len(s.Teams) != 0
}

const (
	globalUserRoleSSOAttrName     = "FLEET_JIT_USER_ROLE_GLOBAL"
	teamUserRoleSSOAttrNamePrefix = "FLEET_JIT_USER_ROLE_TEAM_"
	ssoAttrNullRoleValue          = "null"
)

// RolesFromSSOAttributes loads Global and Team roles from SAML custom attributes.
//   - Custom attribute `FLEET_JIT_USER_ROLE_GLOBAL` is used for setting global role.
//   - Custom attributes of the form `FLEET_JIT_USER_ROLE_TEAM_<TEAM_ID>` are used
//     for setting role for a team with ID <TEAM_ID>.
//
// For both attributes currently supported values are `admin`, `maintainer`, `observer`,
// `observer_plus` and `null`. A `null` value is used to ignore the attribute.
func RolesFromSSOAttributes(attributes []SAMLAttribute) (SSORolesInfo, error) {
	ssoRolesInfo := SSORolesInfo{}
	for _, attribute := range attributes {
		switch {
		case attribute.Name == globalUserRoleSSOAttrName:
			role, err := parseRole(attribute.Values)
			if err != nil {
				return SSORolesInfo{}, fmt.Errorf("parse global role: %w", err)
			}
			if role == ssoAttrNullRoleValue {
				// If the role is set to the null value then the attribute is ignored.
				continue
			}
			ssoRolesInfo.Global = ptr.String(role)
		case strings.HasPrefix(attribute.Name, teamUserRoleSSOAttrNamePrefix):
			teamIDSuffix := strings.TrimPrefix(attribute.Name, teamUserRoleSSOAttrNamePrefix)
			teamID, err := strconv.ParseUint(teamIDSuffix, 10, 32)
			if err != nil {
				return SSORolesInfo{}, fmt.Errorf("parse team ID: %w", err)
			}
			teamRole, err := parseRole(attribute.Values)
			if err != nil {
				return SSORolesInfo{}, fmt.Errorf("parse team role: %w", err)
			}
			if teamRole == ssoAttrNullRoleValue {
				// If the role is set to the null value then the attribute is ignored.
				continue
			}
			ssoRolesInfo.Teams = append(ssoRolesInfo.Teams, TeamRole{
				ID:   uint(teamID),
				Role: teamRole,
			})
		default:
			continue
		}
	}
	if err := ssoRolesInfo.verify(); err != nil {
		return SSORolesInfo{}, err
	}
	return ssoRolesInfo, nil
}

func parseRole(values []SAMLAttributeValue) (string, error) {
	if len(values) == 0 {
		return "", errors.New("empty role")
	}
	// Using last value by default.
	value := values[len(values)-1].Value
	if value != RoleAdmin &&
		value != RoleMaintainer &&
		value != RoleObserver &&
		value != RoleObserverPlus &&
		value != ssoAttrNullRoleValue {
		return "", fmt.Errorf("invalid role: %s", value)
	}
	return value, nil
}
