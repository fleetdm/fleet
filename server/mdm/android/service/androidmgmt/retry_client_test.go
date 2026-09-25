package androidmgmt_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

var (
	errTooManyRequests = &googleapi.Error{Code: http.StatusTooManyRequests, Message: "quota exceeded"}
	shortRetryDelays   = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
)

func TestRetryClientNil(t *testing.T) {
	require.Nil(t, androidmgmt.NewRetryClient(nil, nil))
}

func TestRetryClientIssueCommand(t *testing.T) {
	wantOp := &androidmanagement.Operation{Name: "enterprises/e/devices/d/operations/o"}
	tests := []struct {
		name          string
		errs          []error // error returned by each successive call; nil means success
		wantCalls     int
		wantErrCode   int // 0 means no error expected
		wantOperation bool
	}{
		{name: "success first try", errs: []error{nil}, wantCalls: 1, wantOperation: true},
		{name: "429 then success", errs: []error{errTooManyRequests, nil}, wantCalls: 2, wantOperation: true},
		{name: "429 three times then success", errs: []error{errTooManyRequests, errTooManyRequests, errTooManyRequests, nil}, wantCalls: 4, wantOperation: true},
		{name: "429 until retries exhausted", errs: []error{errTooManyRequests, errTooManyRequests, errTooManyRequests, errTooManyRequests}, wantCalls: 4, wantErrCode: http.StatusTooManyRequests},
		{name: "wrapped 429 is retried", errs: []error{fmt.Errorf("issuing command: %w", errTooManyRequests), nil}, wantCalls: 2, wantOperation: true},
		{name: "429 then other error", errs: []error{errTooManyRequests, &googleapi.Error{Code: http.StatusInternalServerError}}, wantCalls: 2, wantErrCode: http.StatusInternalServerError},
		{name: "bad request is not retried", errs: []error{&googleapi.Error{Code: http.StatusBadRequest}}, wantCalls: 1, wantErrCode: http.StatusBadRequest},
		{name: "not found is not retried", errs: []error{&googleapi.Error{Code: http.StatusNotFound}}, wantCalls: 1, wantErrCode: http.StatusNotFound},
		{name: "server error is not retried", errs: []error{&googleapi.Error{Code: http.StatusInternalServerError}}, wantCalls: 1, wantErrCode: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			inner := &mock.Client{}
			inner.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, deviceName string, command *androidmanagement.Command) (*androidmanagement.Operation, error) {
				require.Less(t, calls, len(tt.errs), "unexpected extra call")
				err := tt.errs[calls]
				calls++
				if err != nil {
					return nil, err
				}
				return wantOp, nil
			}
			client := androidmgmt.NewRetryClientWithDelays(inner, shortRetryDelays)

			op, err := client.EnterprisesDevicesIssueCommand(t.Context(), "enterprises/e/devices/d", &androidmanagement.Command{Type: "LOCK"})
			assert.Equal(t, tt.wantCalls, calls)
			if tt.wantErrCode == 0 {
				require.NoError(t, err)
			} else {
				var ae *googleapi.Error
				require.ErrorAs(t, err, &ae)
				assert.Equal(t, tt.wantErrCode, ae.Code)
			}
			if tt.wantOperation {
				assert.Same(t, wantOp, op)
			} else {
				assert.Nil(t, op)
			}
		})
	}
}

func TestRetryClientStopsWhenContextDone(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var calls int
	inner := &mock.Client{}
	inner.EnterprisesDevicesGetFunc = func(context.Context, string) (*androidmanagement.Device, error) {
		calls++
		cancel()
		return nil, errTooManyRequests
	}
	client := androidmgmt.NewRetryClientWithDelays(inner, []time.Duration{time.Hour})

	start := time.Now()
	device, err := client.EnterprisesDevicesGet(ctx, "enterprises/e/devices/d")
	assert.Less(t, time.Since(start), time.Minute)
	assert.Equal(t, 1, calls)
	assert.Nil(t, device)
	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, androidmgmt.IsTooManyRequestsError(err))
	assert.NotContains(t, err.Error(), "\n", "error is persisted in single-line fields")
}

func TestRetryClientNotRetriedMethods(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *mock.Client, calls *int)
		call  func(ctx context.Context, c androidmgmt.Client) error
	}{
		{
			name: "EnterprisesDevicesOperationsGet",
			setup: func(m *mock.Client, calls *int) {
				m.EnterprisesDevicesOperationsGetFunc = func(context.Context, string) (*androidmanagement.Operation, error) {
					*calls++
					return nil, errTooManyRequests
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				_, err := c.EnterprisesDevicesOperationsGet(ctx, "enterprises/e/devices/d/operations/o")
				return err
			},
		},
		{
			name: "EnterprisesCreate",
			setup: func(m *mock.Client, calls *int) {
				m.EnterprisesCreateFunc = func(context.Context, androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
					*calls++
					return androidmgmt.EnterprisesCreateResponse{}, errTooManyRequests
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				_, err := c.EnterprisesCreate(ctx, androidmgmt.EnterprisesCreateRequest{})
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			inner := &mock.Client{}
			tt.setup(inner, &calls)
			client := androidmgmt.NewRetryClientWithDelays(inner, shortRetryDelays)

			err := tt.call(t.Context(), client)
			assert.Equal(t, 1, calls)
			assert.True(t, androidmgmt.IsTooManyRequestsError(err))
		})
	}
}

func TestRetryClientWithoutRetry(t *testing.T) {
	var calls int
	inner := &mock.Client{}
	inner.EnterprisesPoliciesPatchFunc = func(ctx context.Context, _ string, _ *androidmanagement.Policy, _ androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		calls++
		assert.True(t, androidmgmt.RetryDisabled(ctx))
		if calls == 1 {
			return nil, errTooManyRequests
		}
		return &androidmanagement.Policy{}, nil
	}
	client := androidmgmt.NewRetryClientWithDelays(inner, shortRetryDelays)

	require.False(t, androidmgmt.RetryDisabled(t.Context()))
	ctx := androidmgmt.WithoutRetry(t.Context())
	_, err := client.EnterprisesPoliciesPatch(ctx, "policyName", &androidmanagement.Policy{}, androidmgmt.PoliciesPatchOpts{})
	assert.Equal(t, 1, calls)
	assert.True(t, androidmgmt.IsTooManyRequestsError(err))
}

// TestRetryClientProxyTooManyRequests sends the proxy's 429 response through the real ProxyClient, to check
// that it is classified as a quota error and retried.
func TestRetryClientProxyTooManyRequests(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "/v1/enterprises/e/devices/d:issueCommand", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"code":429,"message":"rate limit exceeded"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"name":"enterprises/e/devices/d/operations/o"}`))
	}))
	defer srv.Close()

	proxy := androidmgmt.NewProxyClient(t.Context(), slog.New(slog.DiscardHandler), "license", func(name string) string {
		if name == "FLEET_DEV_ANDROID_PROXY_ENDPOINT" {
			return srv.URL + "/"
		}
		return ""
	})
	require.NotNil(t, proxy)
	client := androidmgmt.NewRetryClientWithDelays(proxy, shortRetryDelays)

	op, err := client.EnterprisesDevicesIssueCommand(t.Context(), "enterprises/e/devices/d", &androidmanagement.Command{Type: "LOCK"})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Equal(t, "enterprises/e/devices/d/operations/o", op.Name)
}

// TestRetryClientRetriesEveryMethod checks that each retried method forwards its arguments and results
// to and from the wrapped client, and retries after a 429.
func TestRetryClientRetriesEveryMethod(t *testing.T) {
	policy := &androidmanagement.Policy{Name: "policy"}
	device := &androidmanagement.Device{Name: "device"}
	opts := androidmgmt.PoliciesPatchOpts{ExcludeApps: true}

	// fail429Once returns a 429 on the first call and nil after, counting calls.
	fail429Once := func(calls *int) error {
		*calls++
		if *calls == 1 {
			return errTooManyRequests
		}
		return nil
	}

	tests := []struct {
		name  string
		setup func(t *testing.T, m *mock.Client, calls *int)
		call  func(ctx context.Context, c androidmgmt.Client) error
	}{
		{
			name: "SignupURLsCreate",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.SignupURLsCreateFunc = func(_ context.Context, serverURL, callbackURL string) (*android.SignupDetails, error) {
					assert.Equal(t, "server", serverURL)
					assert.Equal(t, "callback", callbackURL)
					return &android.SignupDetails{Url: "url"}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.SignupURLsCreate(ctx, "server", "callback")
				assert.Equal(t, "url", ret.Url)
				return err
			},
		},
		{
			name: "EnterprisesPoliciesPatch",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, p *androidmanagement.Policy, o androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
					assert.Equal(t, "policyName", policyName)
					assert.Same(t, policy, p)
					assert.Equal(t, opts, o)
					return policy, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesPoliciesPatch(ctx, "policyName", policy, opts)
				assert.Same(t, policy, ret)
				return err
			},
		},
		{
			name: "EnterprisesDevicesPatch",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesDevicesPatchFunc = func(_ context.Context, deviceName string, d *androidmanagement.Device) (*androidmanagement.Device, error) {
					assert.Equal(t, "deviceName", deviceName)
					assert.Same(t, device, d)
					return device, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesDevicesPatch(ctx, "deviceName", device)
				assert.Same(t, device, ret)
				return err
			},
		},
		{
			name: "EnterprisesDevicesGet",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesDevicesGetFunc = func(_ context.Context, deviceName string) (*androidmanagement.Device, error) {
					assert.Equal(t, "deviceName", deviceName)
					return device, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesDevicesGet(ctx, "deviceName")
				assert.Same(t, device, ret)
				return err
			},
		},
		{
			name: "EnterprisesDevicesDelete",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesDevicesDeleteFunc = func(_ context.Context, deviceName string) error {
					assert.Equal(t, "deviceName", deviceName)
					return fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				return c.EnterprisesDevicesDelete(ctx, "deviceName")
			},
		},
		{
			name: "EnterprisesDevicesIssueCommand",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesDevicesIssueCommandFunc = func(_ context.Context, deviceName string, cmd *androidmanagement.Command) (*androidmanagement.Operation, error) {
					assert.Equal(t, "deviceName", deviceName)
					assert.Equal(t, "WIPE", cmd.Type)
					return &androidmanagement.Operation{Name: "op"}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesDevicesIssueCommand(ctx, "deviceName", &androidmanagement.Command{Type: "WIPE"})
				assert.Equal(t, "op", ret.Name)
				return err
			},
		},
		{
			name: "EnterprisesDevicesListPartial",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesDevicesListPartialFunc = func(_ context.Context, enterpriseName, pageToken string) (*androidmanagement.ListDevicesResponse, error) {
					assert.Equal(t, "enterpriseName", enterpriseName)
					assert.Equal(t, "pageToken", pageToken)
					return &androidmanagement.ListDevicesResponse{NextPageToken: "next"}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesDevicesListPartial(ctx, "enterpriseName", "pageToken")
				assert.Equal(t, "next", ret.NextPageToken)
				return err
			},
		},
		{
			name: "EnterprisesEnrollmentTokensCreate",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesEnrollmentTokensCreateFunc = func(_ context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
					assert.Equal(t, "enterpriseName", enterpriseName)
					assert.Equal(t, "in", token.Name)
					return &androidmanagement.EnrollmentToken{Name: "out"}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesEnrollmentTokensCreate(ctx, "enterpriseName", &androidmanagement.EnrollmentToken{Name: "in"})
				assert.Equal(t, "out", ret.Name)
				return err
			},
		},
		{
			name: "EnterpriseDelete",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterpriseDeleteFunc = func(_ context.Context, enterpriseName string) error {
					assert.Equal(t, "enterpriseName", enterpriseName)
					return fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				return c.EnterpriseDelete(ctx, "enterpriseName")
			},
		},
		{
			name: "EnterprisesList",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesListFunc = func(_ context.Context, serverURL string) ([]*androidmanagement.Enterprise, error) {
					assert.Equal(t, "server", serverURL)
					return []*androidmanagement.Enterprise{{Name: "enterprise"}}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesList(ctx, "server")
				require.Len(t, ret, 1)
				assert.Equal(t, "enterprise", ret[0].Name)
				return err
			},
		},
		{
			name: "EnterprisesApplications",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesApplicationsFunc = func(_ context.Context, enterpriseName, packageName string) (*androidmanagement.Application, error) {
					assert.Equal(t, "enterpriseName", enterpriseName)
					assert.Equal(t, "packageName", packageName)
					return &androidmanagement.Application{Name: "app"}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesApplications(ctx, "enterpriseName", "packageName")
				assert.Equal(t, "app", ret.Name)
				return err
			},
		},
		{
			name: "EnterprisesPoliciesModifyPolicyApplications",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(_ context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
					assert.Equal(t, "policyName", policyName)
					require.Len(t, appPolicies, 1)
					assert.Equal(t, "com.example", appPolicies[0].PackageName)
					return policy, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesPoliciesModifyPolicyApplications(ctx, "policyName", []*androidmanagement.ApplicationPolicy{{PackageName: "com.example"}})
				assert.Same(t, policy, ret)
				return err
			},
		},
		{
			name: "EnterprisesPoliciesRemovePolicyApplications",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesPoliciesRemovePolicyApplicationsFunc = func(_ context.Context, policyName string, packageNames []string) (*androidmanagement.Policy, error) {
					assert.Equal(t, "policyName", policyName)
					assert.Equal(t, []string{"com.example"}, packageNames)
					return policy, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesPoliciesRemovePolicyApplications(ctx, "policyName", []string{"com.example"})
				assert.Same(t, policy, ret)
				return err
			},
		},
		{
			name: "EnterprisesWebAppsCreate",
			setup: func(t *testing.T, m *mock.Client, calls *int) {
				m.EnterprisesWebAppsCreateFunc = func(_ context.Context, enterpriseName string, webApp *androidmanagement.WebApp) (*androidmanagement.WebApp, error) {
					assert.Equal(t, "enterpriseName", enterpriseName)
					assert.Equal(t, "in", webApp.Title)
					return &androidmanagement.WebApp{Name: "out"}, fail429Once(calls)
				}
			},
			call: func(ctx context.Context, c androidmgmt.Client) error {
				ret, err := c.EnterprisesWebAppsCreate(ctx, "enterpriseName", &androidmanagement.WebApp{Title: "in"})
				assert.Equal(t, "out", ret.Name)
				return err
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			inner := &mock.Client{}
			tt.setup(t, inner, &calls)
			client := androidmgmt.NewRetryClientWithDelays(inner, shortRetryDelays)

			require.NoError(t, tt.call(t.Context(), client))
			assert.Equal(t, 2, calls)
		})
	}

	t.Run("SetAuthenticationSecret", func(t *testing.T) {
		inner := &mock.Client{}
		var got string
		inner.SetAuthenticationSecretFunc = func(secret string) error {
			got = secret
			return nil
		}
		client := androidmgmt.NewRetryClientWithDelays(inner, shortRetryDelays)
		require.NoError(t, client.SetAuthenticationSecret("secret"))
		assert.Equal(t, "secret", got)
	})
}
