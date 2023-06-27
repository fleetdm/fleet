package service

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

type mockHttpClient struct {
	resBody    string
	statusCode int
	err        error
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	if m.err != nil {
		return nil, m.err
	}

	res := &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(bytes.NewBufferString(m.resBody)),
	}

	return res, nil
}

func TestDeviceClientGetDesktopPayload(t *testing.T) {
	client, err := NewDeviceClient("https://test.com", true, "", nil, "")
	token := "test_token"
	require.NoError(t, err)

	mockRequestDoer := &mockHttpClient{}
	client.http = mockRequestDoer

	t.Run("with wrong license", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusPaymentRequired
		_, err = client.DesktopSummary(token)
		require.ErrorIs(t, err, ErrMissingLicense)
	})

	t.Run("with no failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{}`
		result, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, 0, *result.FailingPolicies)
		require.False(t, result.Notifications.NeedsMDMMigration)
	})

	t.Run("with failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"failing_policies_count": 1}`
		result, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, 1, *result.FailingPolicies)
		require.False(t, result.Notifications.NeedsMDMMigration)
	})

	t.Run("with flag to enable MDM migration", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"failing_policies_count": 15, "notifications": {"needs_mdm_migration": true}}`
		result, err := client.DesktopSummary(token)
		require.NoError(t, err)
		require.EqualValues(t, 15, *result.FailingPolicies)
		require.True(t, result.Notifications.NeedsMDMMigration)
	})
}
