package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/httpsig"
	"github.com/fleetdm/fleet/v4/ee/server/service/hostidentity/types"
	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
)

func TestHTTPMessageSignAuth(t *testing.T) {
	var nextCalled bool
	var nextCtx context.Context
	next := func(ctx context.Context, request any) (response any, err error) {
		nextCalled = true
		nextCtx = ctx
		return nil, nil
	}
	// Pass in a nil service interface. We shouldn't hit the place where it gets called while
	// http only signatures
	endpoint := AuthenticatedUser(nil, next)

	tcs := []struct {
		Name          string
		Path          string
		Err           string
		Called        bool
		HostIdentCert *types.HostIdentityCertificate
	}{
		{
			Name: "no http auth path",
			Path: "/some/path",
			Err:  "no auth token",
		},
		{
			Name:          "no http auth path with cert",
			Path:          "/some/path",
			Err:           "no auth token",
			HostIdentCert: &types.HostIdentityCertificate{},
		},
		{
			Name: "no http auth path with good cert context",
			Path: "/some/path",
			Err:  "no auth token",
			HostIdentCert: &types.HostIdentityCertificate{
				NotValidAfter: time.Now().Add(24 * time.Hour),
				HostID:        ptr.Uint(1),
			},
		},
		{
			Name: "auth path with no cert",
			Path: "/osquery/",
			Err:  "no auth token",
		},
		{
			Name: "auth path with good cert context",
			Path: "/osquery/",
			Err:  "",
			HostIdentCert: &types.HostIdentityCertificate{
				NotValidAfter: time.Now().Add(24 * time.Hour),
				HostID:        ptr.Uint(1),
			},
			Called: true,
		},
		{
			Name: "auth path with good cert context 2",
			Path: "/api/fleet/orbit/foo",
			Err:  "",
			HostIdentCert: &types.HostIdentityCertificate{
				NotValidAfter: time.Now().Add(24 * time.Hour),
				HostID:        ptr.Uint(1),
			},
			Called: true,
		},
		{
			Name: "auth path with good cert context 3",
			Path: "/api/v1/fleet/certificate_authorities/3/request_certificate",
			Err:  "",
			HostIdentCert: &types.HostIdentityCertificate{
				NotValidAfter: time.Now().Add(24 * time.Hour),
				HostID:        ptr.Uint(1),
			},
			Called: true,
		},
		{
			Name:          "auth path with zeroed cert context",
			Path:          "/osquery/",
			Err:           "host identity certificate expired",
			HostIdentCert: &types.HostIdentityCertificate{},
		},
		{
			Name: "auth path with expired cert context",
			Path: "/osquery/",
			Err:  "host identity certificate expired",
			HostIdentCert: &types.HostIdentityCertificate{
				NotValidAfter: time.Now().Add(-24 * time.Hour),
				HostID:        ptr.Uint(1),
			},
		},
		{
			Name: "auth path with missing host id",
			Path: "/osquery/",
			Err:  "identity certificate is not linked to a specific host",
			HostIdentCert: &types.HostIdentityCertificate{
				NotValidAfter: time.Now().Add(24 * time.Hour),
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			nextCalled = false
			nextCtx = nil
			ctx := context.Background()
			ctx = authz.NewContext(ctx, &authz.AuthorizationContext{})
			ctx = context.WithValue(ctx, kithttp.ContextKeyRequestPath, tc.Path)
			if tc.HostIdentCert != nil {
				ctx = httpsig.NewContext(ctx, *tc.HostIdentCert)
			}

			_, err := endpoint(ctx, nil)
			if tc.Err == "" {
				assert.NoError(t, err)
			} else {
				authErr := &fleet.AuthFailedError{}
				authHeaderErr := &fleet.AuthHeaderRequiredError{}
				if errors.As(err, &authErr) {
					assert.Contains(t, authErr.Internal(), tc.Err)
				} else if errors.As(err, &authHeaderErr) {
					assert.Contains(t, authHeaderErr.Internal(), tc.Err)
				} else {
					assert.ErrorContains(t, err, tc.Err)
				}
			}
			assert.Equal(t, tc.Called, nextCalled)
			if tc.Called {
				assert.NotNil(t, ctx)
				auth, ok := authz.FromContext(nextCtx)
				assert.True(t, ok)
				assert.Equal(t, authz.AuthnHTTPMessageSignature, auth.AuthnMethod())
			}
		})
	}
}
