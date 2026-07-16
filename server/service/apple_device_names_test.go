package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveHostNameIDPVars(t *testing.T) {
	ds := new(mock.Store)
	var scimCalls int
	ds.ScimUserByHostIDFunc = func(_ context.Context, _ uint) (*fleet.ScimUser, error) {
		scimCalls++
		return &fleet.ScimUser{
			ID:         1,
			UserName:   "jdoe@corp.com",
			GivenName:  new("Jane"),
			FamilyName: new("Doe"),
			Department: new("Eng"),
			Groups:     []fleet.ScimUserGroup{{DisplayName: "Admins"}, {DisplayName: "Eng"}},
		}, nil
	}
	ds.ListHostDeviceMappingFunc = func(_ context.Context, _ uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}

	t.Run("multiple IdP vars resolve from a single fetch", func(t *testing.T) {
		scimCalls = 0
		// The template repeats and mixes IdP vars, including _USERNAME and its
		// longer _USERNAME_LOCAL_PART sibling, to exercise the longest-first
		// substitution order.
		name, detail, err := resolveHostNameIDPVars(t.Context(), ds,
			"u=$FLEET_VAR_HOST_END_USER_IDP_USERNAME;lp=${FLEET_VAR_HOST_END_USER_IDP_USERNAME_LOCAL_PART};d=$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT;g=$FLEET_VAR_HOST_END_USER_IDP_GROUPS",
			42)
		require.NoError(t, err)
		require.Empty(t, detail)
		require.Equal(t, "u=jdoe@corp.com;lp=jdoe;d=Eng;g=Admins,Eng", name)
		require.Equal(t, 1, scimCalls, "end users must be fetched once regardless of the number of IdP variables")
	})

	t.Run("no IdP vars needs no fetch and leaves other tokens untouched", func(t *testing.T) {
		scimCalls = 0
		name, detail, err := resolveHostNameIDPVars(t.Context(), ds, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL", 42)
		require.NoError(t, err)
		require.Empty(t, detail)
		// identity variables are resolved elsewhere, so they're passed through here
		require.Equal(t, "WS-$FLEET_VAR_HOST_HARDWARE_SERIAL", name)
		require.Zero(t, scimCalls, "no datastore fetch when the template has no IdP variables")
	})

	t.Run("missing IdP field fails with the profile-style detail", func(t *testing.T) {
		ds.ScimUserByHostIDFunc = func(_ context.Context, _ uint) (*fleet.ScimUser, error) {
			return &fleet.ScimUser{ID: 1, UserName: "jdoe@corp.com"}, nil // no department
		}
		name, detail, err := resolveHostNameIDPVars(t.Context(), ds, "$FLEET_VAR_HOST_END_USER_IDP_DEPARTMENT", 42)
		require.NoError(t, err)
		require.Empty(t, name)
		require.Contains(t, detail, "no IdP department for this host")
	})

	t.Run("no IdP user fails with the username detail", func(t *testing.T) {
		ds.ScimUserByHostIDFunc = func(_ context.Context, _ uint) (*fleet.ScimUser, error) {
			return nil, nil // no SCIM user mapped
		}
		_, detail, err := resolveHostNameIDPVars(t.Context(), ds, "$FLEET_VAR_HOST_END_USER_IDP_USERNAME", 42)
		require.NoError(t, err)
		require.Contains(t, detail, "no IdP username for this host")
	})
}
