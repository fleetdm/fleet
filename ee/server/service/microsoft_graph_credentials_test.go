package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
	"github.com/stretchr/testify/assert"
)

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
