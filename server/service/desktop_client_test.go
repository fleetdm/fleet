package service

import (
	"bytes"
	"io/ioutil"
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
		Body:       ioutil.NopCloser(bytes.NewBufferString(m.resBody)),
	}

	return res, nil
}

func TestDeviceClientListPolicies(t *testing.T) {
	client, err := NewDesktopClient("https://test.com", "test-token", true, "")
	require.NoError(t, err)

	mockRequestDoer := &mockHttpClient{}
	client.http = mockRequestDoer

	t.Run("with wrong license", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusPaymentRequired
		_, err = client.GetPayload()
		require.ErrorIs(t, err, ErrMissingLicense)
	})

	t.Run("with failing policies", func(t *testing.T) {
		mockRequestDoer.statusCode = http.StatusOK
		mockRequestDoer.resBody = `{"failing_policies_count": 1}`
		res, err := client.GetPayload()
		require.NoError(t, err)
		require.Equal(t, uint(1), res.FailingPolicies)
	})
}
