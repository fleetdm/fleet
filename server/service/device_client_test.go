package service

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
	client, err := NewDeviceClient("https://test.com", "test-token", true, "", fleet.CapabilityMap{})
	require.NoError(t, err)

	mockRequestDoer := &mockHttpClient{}
	client.http = mockRequestDoer

	t.Run("with wrong license", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusPaymentRequired
		_, err = client.ListDevicePolicies()
		require.ErrorIs(t, err, ErrMissingLicense)
	})

	t.Run("with empty policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"policies": []}`
		policies, err := client.ListDevicePolicies()
		require.NoError(t, err)
		require.Len(t, policies, 0)
	})

	t.Run("with policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"policies": [{"id": 1}]}`
		policies, err := client.ListDevicePolicies()
		require.NoError(t, err)
		require.Len(t, policies, 1)
		require.Equal(t, uint(1), policies[0].ID)
	})
}
