package azure

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/cloudstorage"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

// TestNewAPIError checks the message built from Azure response errors.
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
			wantMsg: "Cloud storage is misconfigured. Azure: api error ContainerNotFound: The specified container does not exist. (container: container)",
		},
		{
			name:    "authentication failed without body",
			method:  http.MethodHead,
			status:  http.StatusForbidden,
			code:    string(bloberror.AuthenticationFailed),
			wantMsg: "Cloud storage is misconfigured. Azure: api error AuthenticationFailed: Forbidden (container: container)",
		},
		{
			name:    "forbidden without error code",
			method:  http.MethodGet,
			status:  http.StatusForbidden,
			wantMsg: "Cloud storage is misconfigured. Azure: api error 403: Forbidden (container: container)",
		},
		{
			name:   "server busy is not a misconfiguration",
			method: http.MethodPut,
			status: http.StatusServiceUnavailable,
			code:   string(bloberror.ServerBusy),
			body: `<?xml version="1.0" encoding="utf-8"?><Error><Code>ServerBusy</Code>` +
				"<Message>The server is busy.\nRequestId:abc</Message></Error>",
			wantMsg: "api error ServerBusy: The server is busy.",
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
			err := ctxerr.Wrap(ctx, newAPIError(newRespErr(t, c.method, c.status, c.code, c.body), "container"), "uploading")
			require.Equal(t, c.wantMsg, ctxerr.Cause(err).Error())
			if c.code != "" {
				require.True(t, bloberror.HasCode(err, bloberror.Code(c.code)))
			}
		})
	}

	plainErr := errors.New("not an azure error")
	require.Equal(t, plainErr, newAPIError(plainErr, "container"))
}

// misconfiguredCases has one entry per code in misconfiguredCodes, with the
// status Azure uses for it and a typical vendor message.
var misconfiguredCases = []struct {
	code    bloberror.Code
	status  int
	message string
}{
	{bloberror.AccountIsDisabled, http.StatusForbidden, "The specified account is disabled."},
	{bloberror.AuthenticationFailed, http.StatusForbidden, "Server failed to authenticate the request. Make sure the value of Authorization header is formed correctly including the signature."},
	{bloberror.AuthorizationFailure, http.StatusForbidden, "This request is not authorized to perform this operation."},
	{bloberror.AuthorizationPermissionMismatch, http.StatusForbidden, "This request is not authorized to perform this operation using this permission."},
	{bloberror.AuthorizationProtocolMismatch, http.StatusForbidden, "This request is not authorized to perform this operation using this protocol."},
	{bloberror.AuthorizationResourceTypeMismatch, http.StatusForbidden, "This request is not authorized to perform this operation using this resource type."},
	{bloberror.AuthorizationServiceMismatch, http.StatusForbidden, "This request is not authorized to perform this operation using this service."},
	{bloberror.AuthorizationSourceIPMismatch, http.StatusForbidden, "This request is not authorized to perform this operation using this source IP address."},
	{bloberror.ContainerDisabled, http.StatusForbidden, "The specified container is disabled."},
	{bloberror.ContainerNotFound, http.StatusNotFound, "The specified container does not exist."},
	{bloberror.InsufficientAccountPermissions, http.StatusForbidden, "The account being accessed does not have sufficient permissions to execute this operation."},
	{bloberror.InvalidAuthenticationInfo, http.StatusUnauthorized, "Server failed to authenticate the request. Please refer to the information in the www-authenticate header."},
}

// azureErrorBody returns the XML body Azure sends with an error.
func azureErrorBody(code bloberror.Code, message string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?><Error><Code>%s</Code><Message>%s`+"\nRequestId:abc\nTime:2026-01-01T00:00:00.000Z</Message></Error>", code, message)
}

// TestNewAPIErrorAllMisconfiguredCodes checks the message for every misconfigured code.
func TestNewAPIErrorAllMisconfiguredCodes(t *testing.T) {
	require.Len(t, misconfiguredCases, len(misconfiguredCodes), "every misconfigured code needs a test case")

	for _, c := range misconfiguredCases {
		t.Run(string(c.code), func(t *testing.T) {
			require.Contains(t, misconfiguredCodes, c.code)

			req, err := http.NewRequest(http.MethodPut, "https://acct.blob.core.windows.net/container/blob", nil)
			require.NoError(t, err)
			resp := &http.Response{
				StatusCode: c.status,
				Status:     http.StatusText(c.status),
				Header:     http.Header{"X-Ms-Error-Code": []string{string(c.code)}},
				Body:       io.NopCloser(strings.NewReader(azureErrorBody(c.code, c.message))),
				Request:    req,
			}

			err = ctxerr.Wrap(t.Context(), newAPIError(runtime.NewResponseError(resp), "my-container"), "uploading")
			require.Equal(t, fmt.Sprintf("Cloud storage is misconfigured. Azure: api error %s: %s (container: my-container)", c.code, c.message), ctxerr.Cause(err).Error())
			require.True(t, bloberror.HasCode(err, c.code), "the SDK error must stay reachable")
			require.True(t, cloudstorage.IsMisconfigured(err))
		})
	}
}

// TestNewAPIErrorOtherCodesAreNotMisconfigured checks that other codes get no prefix.
func TestNewAPIErrorOtherCodesAreNotMisconfigured(t *testing.T) {
	for _, code := range []bloberror.Code{bloberror.ServerBusy, bloberror.InternalError, bloberror.BlobNotFound, bloberror.InvalidInput, bloberror.OperationTimedOut} {
		t.Run(string(code), func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPut, "https://acct.blob.core.windows.net/container/blob", nil)
			require.NoError(t, err)
			resp := &http.Response{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"X-Ms-Error-Code": []string{string(code)}},
				Body:       io.NopCloser(strings.NewReader(azureErrorBody(code, "something"))),
				Request:    req,
			}
			err = newAPIError(runtime.NewResponseError(resp), "my-container")
			require.False(t, cloudstorage.IsMisconfigured(err))
			require.Equal(t, fmt.Sprintf("api error %s: something", code), ctxerr.Cause(err).Error())
		})
	}
}

// newErrorServer returns an Azure Blob-compatible test server that answers
// every request with the given error, except for blob listing, which succeeds
// with one old blob so that Cleanup reaches the delete step. HEAD responses
// have no body, like Azure.
func newErrorServer(t *testing.T, code bloberror.Code, message string, status int, listSucceeds bool) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if listSucceeds && r.URL.Query().Get("comp") == "list" {
			w.Header().Set("Content-Type", "application/xml")
			_, _ = io.WriteString(w, `<?xml version="1.0" encoding="utf-8"?><EnumerationResults><Blobs><Blob><Name>software-installers/old</Name><Properties><Last-Modified>Mon, 01 Jan 2001 00:00:00 GMT</Last-Modified></Properties></Blob></Blobs><NextMarker/></EnumerationResults>`)
			return
		}
		w.Header().Set("x-ms-error-code", string(code))
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			_, _ = io.WriteString(w, azureErrorBody(code, message))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// newErrorTestConfig returns an Azure config that points at the test server.
func newErrorTestConfig(endpoint string) config.AzureConfig {
	return config.AzureConfig{
		SoftwareInstallersAccountName:  "acct",
		SoftwareInstallersAccountKey:   base64.StdEncoding.EncodeToString([]byte("test-key")),
		SoftwareInstallersContainer:    "my-container",
		SoftwareInstallersContainerURL: endpoint,
	}
}

// TestStoreOperationsMisconfiguredErrors checks, for every misconfiguration
// code, the message returned by each store operation against a server that
// returns that error.
func TestStoreOperationsMisconfiguredErrors(t *testing.T) {
	for _, c := range misconfiguredCases {
		t.Run(string(c.code), func(t *testing.T) {
			srv := newErrorServer(t, c.code, c.message, c.status, false)
			cfg := newErrorTestConfig(srv.URL)
			installers, err := NewSoftwareInstallerStore(cfg)
			require.NoError(t, err)
			logos, err := NewOrgLogoStore(cfg)
			require.NoError(t, err)

			want := fmt.Sprintf("Cloud storage is misconfigured. Azure: api error %s: %s (container: my-container)", c.code, c.message)
			// HEAD responses carry no body, so only the status text is known.
			wantHead := fmt.Sprintf("Cloud storage is misconfigured. Azure: api error %s: %s (container: my-container)", c.code, http.StatusText(c.status))

			check := func(t *testing.T, err error, wantMsg string) {
				t.Helper()
				require.Error(t, err)
				require.Equal(t, wantMsg, ctxerr.Cause(err).Error())
				require.True(t, cloudstorage.IsMisconfigured(err))
				require.True(t, bloberror.HasCode(err, c.code), "the SDK error must stay reachable")
			}

			t.Run("put", func(t *testing.T) {
				err := installers.Put(t.Context(), "installer", bytes.NewReader([]byte("content")))
				check(t, err, want)
			})
			t.Run("get", func(t *testing.T) {
				_, _, err := installers.Get(t.Context(), "installer")
				check(t, err, want)
			})
			t.Run("exists", func(t *testing.T) {
				_, err := installers.Exists(t.Context(), "installer")
				check(t, err, wantHead)
			})
			t.Run("cleanup list", func(t *testing.T) {
				_, err := installers.Cleanup(t.Context(), nil, time.Now())
				check(t, err, want)
			})
			t.Run("cleanup delete", func(t *testing.T) {
				srv := newErrorServer(t, c.code, c.message, c.status, true)
				store, err := NewSoftwareInstallerStore(newErrorTestConfig(srv.URL))
				require.NoError(t, err)
				_, err = store.Cleanup(t.Context(), nil, time.Now())
				check(t, err, want)
			})
			t.Run("org logo delete", func(t *testing.T) {
				check(t, logos.Delete(t.Context(), fleet.OrgLogoModeLight), want)
			})
		})
	}
}

// TestStoreOperationsBlobNotFound checks that a missing blob is not an error.
func TestStoreOperationsBlobNotFound(t *testing.T) {
	srv := newErrorServer(t, bloberror.BlobNotFound, "The specified blob does not exist.", http.StatusNotFound, false)
	cfg := newErrorTestConfig(srv.URL)
	installers, err := NewSoftwareInstallerStore(cfg)
	require.NoError(t, err)
	logos, err := NewOrgLogoStore(cfg)
	require.NoError(t, err)

	_, _, err = installers.Get(t.Context(), "installer")
	require.True(t, fleet.IsNotFound(err))

	exists, err := installers.Exists(t.Context(), "installer")
	require.NoError(t, err)
	require.False(t, exists)

	require.NoError(t, logos.Delete(t.Context(), fleet.OrgLogoModeLight))
}

// TestStoreOperationsOtherErrorsAreNotMisconfigured checks that other Azure errors get no prefix.
func TestStoreOperationsOtherErrorsAreNotMisconfigured(t *testing.T) {
	srv := newErrorServer(t, bloberror.InvalidInput, "Bad input", http.StatusBadRequest, false)
	store, err := NewSoftwareInstallerStore(newErrorTestConfig(srv.URL))
	require.NoError(t, err)

	err = store.Put(t.Context(), "installer", bytes.NewReader([]byte("content")))
	require.Error(t, err)
	require.False(t, cloudstorage.IsMisconfigured(err))
	require.Equal(t, "api error InvalidInput: Bad input", ctxerr.Cause(err).Error())

	_, err = store.Cleanup(t.Context(), nil, time.Now())
	require.Error(t, err)
	require.False(t, cloudstorage.IsMisconfigured(err))
}

// TestAPIErrorMisconfigured checks the message against Azurite when AZURE_STORAGE_TEST is set.
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
		require.Equal(t, "Cloud storage is misconfigured. Azure: api error ContainerNotFound: The specified container does not exist. (container: no-such-container)", ctxerr.Cause(err).Error())
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
		require.True(t, strings.HasPrefix(msg, "Cloud storage is misconfigured. Azure: api error "+responseError.ErrorCode+":"), msg)
		require.NotContains(t, msg, "RequestId")
		require.NotContains(t, msg, "\n")
	})
}
