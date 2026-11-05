package s3

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/cloudstorage"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestWrapAPIError(t *testing.T) {
	store := &s3store{bucket: "my-bucket"}

	withStatus := func(status int, err error) error {
		return &smithy.OperationError{
			ServiceID:     "S3",
			OperationName: "PutObject",
			Err: &smithyhttp.ResponseError{
				Response: &smithyhttp.Response{Response: &http.Response{StatusCode: status}},
				Err:      err,
			},
		}
	}

	cases := []struct {
		name    string
		err     error
		wantMsg string // empty means the error is returned unchanged
	}{
		{
			name:    "no such bucket",
			err:     withStatus(http.StatusNotFound, &types.NoSuchBucket{Message: new("The specified bucket does not exist")}),
			wantMsg: "Cloud storage is misconfigured. S3: api error NoSuchBucket: The specified bucket does not exist (bucket: my-bucket)",
		},
		{
			name:    "invalid access key",
			err:     withStatus(http.StatusForbidden, &smithy.GenericAPIError{Code: "InvalidAccessKeyId", Message: "The AWS Access Key Id you provided does not exist in our records."}),
			wantMsg: "Cloud storage is misconfigured. S3: api error InvalidAccessKeyId: The AWS Access Key Id you provided does not exist in our records. (bucket: my-bucket)",
		},
		{
			name:    "wrong region keeps the vendor hint",
			err:     withStatus(http.StatusMovedPermanently, &smithy.GenericAPIError{Code: "AuthorizationHeaderMalformed", Message: "the region 'us-east-1' is wrong; expecting 'eu-west-1'"}),
			wantMsg: "Cloud storage is misconfigured. S3: api error AuthorizationHeaderMalformed: the region 'us-east-1' is wrong; expecting 'eu-west-1' (bucket: my-bucket)",
		},
		{
			name:    "forbidden HEAD response without body",
			err:     withStatus(http.StatusForbidden, &smithy.GenericAPIError{Code: "Forbidden", Message: "UnknownError"}),
			wantMsg: "Cloud storage is misconfigured. S3: api error Forbidden: Forbidden (bucket: my-bucket)",
		},
		{
			name:    "unknown code with 403 status",
			err:     withStatus(http.StatusForbidden, &smithy.GenericAPIError{Code: "SomethingNew", Message: "nope"}),
			wantMsg: "Cloud storage is misconfigured. S3: api error SomethingNew: nope (bucket: my-bucket)",
		},
		{
			name: "non-configuration API error is unchanged",
			err:  withStatus(http.StatusServiceUnavailable, &smithy.GenericAPIError{Code: "SlowDown", Message: "Please reduce your request rate."}),
		},
		{
			name: "non-API error is unchanged",
			err:  errors.New("connection reset"),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := store.wrapAPIError(c.err)
			if c.wantMsg == "" {
				require.Equal(t, c.err, got)
				return
			}

			wrapped := ctxerr.Wrap(t.Context(), got, "uploading")
			require.Equal(t, c.wantMsg, ctxerr.Cause(wrapped).Error())
			var mis *cloudstorage.MisconfiguredError
			require.ErrorAs(t, wrapped, &mis)
			var apiErr smithy.APIError
			require.ErrorAs(t, wrapped, &apiErr)
		})
	}

	require.NoError(t, store.wrapAPIError(nil))
}

// misconfiguredCases has one entry per code in misconfiguredCodes, with the
// status S3 uses for it and a typical vendor message.
var misconfiguredCases = []struct {
	code    string
	status  int
	message string
}{
	{"AccessDenied", http.StatusForbidden, "Access Denied"},
	{"AccountProblem", http.StatusForbidden, "There is a problem with your AWS account that prevents the operation from completing successfully."},
	{"AllAccessDisabled", http.StatusForbidden, "All access to this Amazon S3 resource has been disabled."},
	{"AuthorizationHeaderMalformed", http.StatusBadRequest, "the region 'us-east-1' is wrong; expecting 'eu-west-1'"},
	{"BadRequest", http.StatusBadRequest, "Bad Request"},
	{"ExpiredToken", http.StatusBadRequest, "The provided token has expired."},
	{"Forbidden", http.StatusForbidden, "Forbidden"},
	{"InvalidAccessKeyId", http.StatusForbidden, "The AWS Access Key Id you provided does not exist in our records."},
	{"InvalidBucketName", http.StatusBadRequest, "The specified bucket is not valid."},
	{"InvalidSecurity", http.StatusForbidden, "The provided security credentials are not valid."},
	{"InvalidToken", http.StatusBadRequest, "The provided token is malformed or otherwise invalid."},
	{"MovedPermanently", http.StatusMovedPermanently, "Moved Permanently"},
	{"NoSuchBucket", http.StatusNotFound, "The specified bucket does not exist"},
	{"PermanentRedirect", http.StatusMovedPermanently, "The bucket you are attempting to access must be addressed using the specified endpoint. Please send all future requests to this endpoint."},
	{"SignatureDoesNotMatch", http.StatusForbidden, "The request signature we calculated does not match the signature you provided. Check your key and signing method."},
}

func TestWrapAPIErrorAllMisconfiguredCodes(t *testing.T) {
	store := &s3store{bucket: "my-bucket"}

	require.Len(t, misconfiguredCases, len(misconfiguredCodes), "every misconfigured code needs a test case")
	for _, c := range misconfiguredCases {
		t.Run(c.code, func(t *testing.T) {
			require.Contains(t, misconfiguredCodes, c.code)

			// 200 ensures the code alone is enough, regardless of the HTTP status.
			for _, status := range []int{c.status, http.StatusOK} {
				err := &smithy.OperationError{
					ServiceID:     "S3",
					OperationName: "PutObject",
					Err: &smithyhttp.ResponseError{
						Response: &smithyhttp.Response{Response: &http.Response{StatusCode: status}},
						Err:      &smithy.GenericAPIError{Code: c.code, Message: c.message},
					},
				}
				got := ctxerr.Wrap(t.Context(), store.wrapAPIError(err), "uploading")
				require.Equal(t, fmt.Sprintf("Cloud storage is misconfigured. S3: api error %s: %s (bucket: my-bucket)", c.code, c.message), ctxerr.Cause(got).Error())
			}
		})
	}

	t.Run("multiline message keeps the first line", func(t *testing.T) {
		err := &smithy.GenericAPIError{Code: "AccessDenied", Message: "Access Denied\nRequestId: abc\nHostId: def"}
		got := ctxerr.Cause(ctxerr.Wrap(t.Context(), store.wrapAPIError(err), "uploading"))
		require.Equal(t, "Cloud storage is misconfigured. S3: api error AccessDenied: Access Denied (bucket: my-bucket)", got.Error())
	})

	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(fmt.Sprintf("unknown code with status %d", status), func(t *testing.T) {
			err := &smithyhttp.ResponseError{
				Response: &smithyhttp.Response{Response: &http.Response{StatusCode: status}},
				Err:      &smithy.GenericAPIError{Code: "SomethingNew", Message: "nope"},
			}
			got := ctxerr.Cause(ctxerr.Wrap(t.Context(), store.wrapAPIError(err), "uploading"))
			require.Equal(t, "Cloud storage is misconfigured. S3: api error SomethingNew: nope (bucket: my-bucket)", got.Error())
		})
	}

	for _, code := range []string{"SlowDown", "InternalError", "InvalidArgument", "NoSuchKey", "BadDigest"} {
		t.Run("not misconfigured "+code, func(t *testing.T) {
			err := &smithy.GenericAPIError{Code: code, Message: "something"}
			require.Equal(t, error(err), store.wrapAPIError(err))
		})
	}
}

// newErrorServer returns an S3-compatible test server that answers every
// request with the given S3 error. HEAD responses have no body, like S3.
func newErrorServer(t *testing.T, code, message string, status int) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(status)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><Error><Code>%s</Code><Message>%s</Message><RequestId>req-1</RequestId><HostId>host-1</HostId></Error>`, code, message)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newErrorTestConfig(endpoint string) config.S3Config {
	return config.S3Config{
		SoftwareInstallersBucket:           "my-bucket",
		SoftwareInstallersPrefix:           "prefix",
		SoftwareInstallersRegion:           "us-east-1",
		SoftwareInstallersEndpointURL:      endpoint,
		SoftwareInstallersAccessKeyID:      "key",
		SoftwareInstallersSecretAccessKey:  "secret",
		SoftwareInstallersForceS3PathStyle: true,
	}
}

// TestStoreOperationsMisconfiguredErrors checks, for every misconfiguration
// code, the message returned by each store operation against a server that
// returns that error.
func TestStoreOperationsMisconfiguredErrors(t *testing.T) {
	for _, c := range misconfiguredCases {
		t.Run(c.code, func(t *testing.T) {
			srv := newErrorServer(t, c.code, c.message, c.status)
			cfg := newErrorTestConfig(srv.URL)
			installers, err := NewSoftwareInstallerStore(cfg)
			require.NoError(t, err)
			logos, err := NewOrgLogoStore(cfg)
			require.NoError(t, err)

			want := fmt.Sprintf("Cloud storage is misconfigured. S3: api error %s: %s (bucket: my-bucket)", c.code, c.message)
			// HEAD responses carry no body, so the SDK only knows the status.
			statusText := http.StatusText(c.status)
			wantHead := fmt.Sprintf("Cloud storage is misconfigured. S3: api error %s: %s (bucket: my-bucket)", strings.ReplaceAll(statusText, " ", ""), statusText)

			check := func(t *testing.T, err error, wantMsg string) {
				t.Helper()
				require.Error(t, err)
				require.Equal(t, wantMsg, ctxerr.Cause(err).Error())
				var mis *cloudstorage.MisconfiguredError
				require.ErrorAs(t, err, &mis)
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
				exists, err := installers.Exists(t.Context(), "installer")
				if c.status == http.StatusNotFound {
					// A HEAD 404 is reported as "not found", not as a failure.
					require.NoError(t, err)
					require.False(t, exists)
					return
				}
				check(t, err, wantHead)
			})
			t.Run("cleanup", func(t *testing.T) {
				_, err := installers.Cleanup(t.Context(), nil, time.Now())
				check(t, err, want)
			})
			t.Run("delete", func(t *testing.T) {
				check(t, installers.Delete(t.Context(), "installer"), want)
			})
			t.Run("org logo delete", func(t *testing.T) {
				check(t, logos.Delete(t.Context(), fleet.OrgLogoModeLight), want)
			})
		})
	}
}

func TestStoreOperationsOtherErrorsAreNotMisconfigured(t *testing.T) {
	srv := newErrorServer(t, "InvalidArgument", "Bad argument", http.StatusBadRequest)
	store, err := NewSoftwareInstallerStore(newErrorTestConfig(srv.URL))
	require.NoError(t, err)

	err = store.Put(t.Context(), "installer", bytes.NewReader([]byte("content")))
	require.Error(t, err)
	var mis *cloudstorage.MisconfiguredError
	require.NotErrorAs(t, err, &mis)
	require.NotContains(t, ctxerr.Cause(err).Error(), cloudstorage.MisconfiguredPrefix)

	_, err = store.Cleanup(t.Context(), nil, time.Now())
	require.Error(t, err)
	require.NotErrorAs(t, err, &mis)
}
