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
	client, err := NewDeviceClient("https://test.com", true, "")
	token := "test_token"
	require.NoError(t, err)

	mockRequestDoer := &mockHttpClient{}
	client.http = mockRequestDoer

	t.Run("with wrong license", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusPaymentRequired
		_, err = client.NumberOfFailingPolicies(token)
		require.ErrorIs(t, err, ErrMissingLicense)
	})

	t.Run("with no failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{}`
		result, err := client.NumberOfFailingPolicies(token)
		require.NoError(t, err)
		require.Equal(t, uint(0), result)
	})

	t.Run("with failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"failing_policies_count": 1}`
		result, err := client.NumberOfFailingPolicies(token)
		require.NoError(t, err)
		require.Equal(t, uint(1), result)
	})
}
