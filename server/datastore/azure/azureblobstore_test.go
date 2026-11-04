package azure

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/stretchr/testify/require"
)

func TestNewAPIError(t *testing.T) {
	ctx := t.Context()

	newRespErr := func(t *testing.T, method string, status int, code, body string) error {
		req, err := http.NewRequest(method, "https://acct.blob.core.windows.net/container/blob", nil)
		require.NoError(t, err)
		resp := &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}
		if code != "" {
			resp.Header.Set("x-ms-error-code", code)
		}
		return runtime.NewResponseError(resp)
	}

	cases := []struct {
		name    string
		method  string
		status  int
		code    string
		body    string
		wantMsg string
	}{
		{
			name:   "container not found with XML body",
			method: http.MethodPut,
			status: http.StatusNotFound,
			code:   string(bloberror.ContainerNotFound),
			body: `<?xml version="1.0" encoding="utf-8"?><Error><Code>ContainerNotFound</Code>` +
				"<Message>The specified container does not exist.\nRequestId:abc\nTime:2026-01-01T00:00:00.000Z</Message></Error>",
			wantMsg: "api error ContainerNotFound: The specified container does not exist.",
		},
		{
			name:    "authentication failed without body",
			method:  http.MethodHead,
			status:  http.StatusForbidden,
			code:    string(bloberror.AuthenticationFailed),
			wantMsg: "api error AuthenticationFailed: Forbidden",
		},
		{
			name:    "no error code and no body",
			method:  http.MethodGet,
			status:  http.StatusInternalServerError,
			wantMsg: "api error 500: Internal Server Error",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ctxerr.Wrap(ctx, newAPIError(newRespErr(t, c.method, c.status, c.code, c.body)), "uploading")
			require.Equal(t, c.wantMsg, ctxerr.Cause(err).Error())
			if c.code != "" {
				require.True(t, bloberror.HasCode(err, bloberror.Code(c.code)))
			}
		})
	}

	plainErr := errors.New("not an azure error")
	require.Equal(t, plainErr, newAPIError(plainErr))
}

func TestAPIErrorMisconfigured(t *testing.T) {
	checkTestEnv(t)
	ctx := t.Context()

	t.Run("container does not exist", func(t *testing.T) {
		store, err := NewSoftwareInstallerStore(config.AzureConfig{
			SoftwareInstallersAccountName:  testAccountName,
			SoftwareInstallersAccountKey:   testAccountKey,
			SoftwareInstallersContainer:    "no-such-container",
			SoftwareInstallersContainerURL: testContainerURL,
		})
		require.NoError(t, err)

		err = store.Put(ctx, "installer", bytes.NewReader([]byte("content")))
		require.Error(t, err)
		require.True(t, bloberror.HasCode(err, bloberror.ContainerNotFound))
		require.Equal(t, "api error ContainerNotFound: The specified container does not exist.", ctxerr.Cause(err).Error())
	})

	t.Run("wrong account key", func(t *testing.T) {
		store, err := NewSoftwareInstallerStore(config.AzureConfig{
			SoftwareInstallersAccountName:  testAccountName,
			SoftwareInstallersAccountKey:   base64.StdEncoding.EncodeToString(make([]byte, 64)),
			SoftwareInstallersContainer:    "wrong-key-container",
			SoftwareInstallersContainerURL: testContainerURL,
		})
		require.NoError(t, err)

		_, err = store.Exists(ctx, "installer")
		require.Error(t, err)
		var responseError *azcore.ResponseError
		require.ErrorAs(t, err, &responseError)
		require.NotEmpty(t, responseError.ErrorCode)
		msg := ctxerr.Cause(err).Error()
		require.True(t, strings.HasPrefix(msg, "api error "+responseError.ErrorCode+":"), msg)
		require.NotContains(t, msg, "RequestId")
		require.NotContains(t, msg, "\n")
	})
}
