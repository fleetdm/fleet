package service

import (
	"errors"
	"fmt"
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
		{"transient adds retry advice", &msgraph.Error{StatusCode: http.StatusBadGateway}, "temporarily unavailable (502). Please try again."},
		{"non-graph error", errors.New("dial tcp: timeout"), "Couldn't connect to Microsoft Graph: dial tcp: timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, microsoftGraphVerifyMessage(tc.err), tc.contains)
		})
	}

	t.Run("wrapped non-graph error keeps only the root cause", func(t *testing.T) {
		wrapped := fmt.Errorf("verify credential: %w", fmt.Errorf("dial graph.microsoft.com: %w", errors.New("i/o timeout")))
		msg := microsoftGraphVerifyMessage(wrapped)
		assert.Equal(t, "Couldn't connect to Microsoft Graph: i/o timeout", msg)
		assert.NotContains(t, msg, "verify credential", "the wrap chain must never reach the UI")
	})
}
