package service

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgLogoAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool // PUT and DELETE
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
		},
		{
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
		},
		{
			"team gitops",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"user without roles",
			&fleet.User{ID: 777},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			authedCtx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.UploadOrgLogo(authedCtx, fleet.OrgLogoModeLight, bytes.NewReader([]byte{}))
			checkOrgLogoAuth(t, tt.shouldFailWrite, err)

			err = svc.DeleteOrgLogo(authedCtx, fleet.OrgLogoModeLight)
			checkOrgLogoAuth(t, tt.shouldFailWrite, err)

			// GET is public — never an authz failure regardless of viewer.
			_, _, err = svc.GetOrgLogo(authedCtx, fleet.OrgLogoModeLight)
			checkOrgLogoAuth(t, false, err)
		})
	}

	// GET should also work without any viewer in the context (login page
	// case). It may still fail downstream because no store is wired, but
	// that's not an authz failure.
	t.Run("public GET without viewer", func(t *testing.T) {
		_, _, err := svc.GetOrgLogo(ctx, fleet.OrgLogoModeLight)
		checkOrgLogoAuth(t, false, err)
	})
}

func checkOrgLogoAuth(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	var forbidden *authz.Forbidden
	if shouldFail {
		require.Error(t, err)
		require.ErrorAs(t, err, &forbidden, "expected authz Forbidden, got %T: %v", err, err)
		return
	}
	if err != nil {
		require.NotErrorAs(t, err, &forbidden,
			"expected non-authz error, got authz Forbidden: %v", err)
	}
}

func TestValidateOrgLogoBytesSVG(t *testing.T) {
	t.Parallel()

	const minSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="1" height="1"></svg>`

	t.Run("accepts a minimal SVG", func(t *testing.T) {
		require.NoError(t, validateOrgLogoBytes([]byte(minSVG)))
	})

	t.Run("accepts an SVG with XML declaration and inline style", func(t *testing.T) {
		body := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
  <style>.a { fill: red; }</style>
  <rect class="a" width="10" height="10"/>
</svg>`
		require.NoError(t, validateOrgLogoBytes([]byte(body)))
	})

	t.Run("accepts xlink:href fragment references", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><defs><circle id="c" r="5"/></defs><use xlink:href="#c"/></svg>`
		require.NoError(t, validateOrgLogoBytes([]byte(body)))
	})

	t.Run("accepts a real-world SVG (CSS logo from the public web)", func(t *testing.T) {
		body, err := os.ReadFile("testdata/icons/org_logo_css.svg")
		require.NoError(t, err)
		require.NoError(t, validateOrgLogoBytes(body))
	})

	t.Run("rejects oversized SVG before parsing", func(t *testing.T) {
		// Bytes don't need to be a real SVG — the size gate fires
		// first regardless of looksLikeSVG.
		body := append([]byte("<svg>"), bytes.Repeat([]byte("a"), int(orgLogoMaxFileSize))...)
		err := validateOrgLogoBytes(body)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "100KB or less")
	})

	t.Run("rejects <script>", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "<script>")
	})

	t.Run("rejects <foreignObject>", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><div xmlns="http://www.w3.org/1999/xhtml">x</div></foreignObject></svg>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "foreignobject")
	})

	t.Run("rejects on* event handlers", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect width="1" height="1"/></svg>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event-handler")
	})

	t.Run("rejects javascript: href", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink"><a xlink:href="javascript:alert(1)"><rect width="1" height="1"/></a></svg>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "javascript:")
	})

	t.Run("rejects data: href", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><a href="data:text/html,&lt;script&gt;alert(1)&lt;/script&gt;"><rect width="1" height="1"/></a></svg>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "data:")
	})

	t.Run("rejects DOCTYPE", func(t *testing.T) {
		body := `<?xml version="1.0"?>
<!DOCTYPE svg [<!ENTITY xxe SYSTEM "file:///etc/passwd">]>
<svg xmlns="http://www.w3.org/2000/svg"><text>&xxe;</text></svg>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, strings.ToLower(err.Error()), "doctype")
	})

	t.Run("rejects malformed XML", func(t *testing.T) {
		body := `<svg xmlns="http://www.w3.org/2000/svg"><rect`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid SVG")
	})

	t.Run("rejects non-svg root that still tripped the sniffer", func(t *testing.T) {
		// looksLikeSVG just searches for "<svg" anywhere in the head;
		// a wrapper element must not let a document slip past with
		// the sniffer satisfied but the parsed root non-svg.
		body := `<html><body><svg/></body></html>`
		err := validateOrgLogoBytes([]byte(body))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "root")
	})
}

func TestContentTypeForBytesSVG(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		body string
		want string
	}{
		{"plain svg", `<svg xmlns="http://www.w3.org/2000/svg"/>`, "image/svg+xml"},
		{"svg after xml decl", `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`, "image/svg+xml"},
		{"svg after BOM and whitespace", "\xEF\xBB\xBF\n  <svg/>", "image/svg+xml"},
		{"uppercase root tag", "<SVG/>", "image/svg+xml"},
		{"not an svg", `<html><body/></html>`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, contentTypeForBytes([]byte(tc.body)))
		})
	}
}
