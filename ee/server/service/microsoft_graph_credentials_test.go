package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
	"github.com/stretchr/testify/assert"
)

// The end-to-end behavior of the credential endpoints is covered in server/service, which builds the full core plus
// premium stack. This covers the message classification directly, because it is unexported and the three cases have
// genuinely different remedies for the admin.
func TestMicrosoftGraphVerifyMessage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		err      error
		contains string
	}{
		{"permission", &msgraph.Error{StatusCode: http.StatusForbidden}, "DeviceManagementServiceConfig.Read.All"},
		{"auth", &msgraph.Error{StatusCode: http.StatusUnauthorized}, "rejected the credential"},
		{"transient", &msgraph.Error{StatusCode: http.StatusBadGateway}, "temporarily unavailable"},
		{"non-graph error", errors.New("dial tcp: timeout"), "Couldn't connect to Microsoft Graph"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, microsoftGraphVerifyMessage(tc.err), tc.contains)
		})
	}
}
