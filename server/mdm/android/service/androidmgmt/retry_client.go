package androidmgmt

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
)

// defaultRetryDelays are the waits between attempts after AMAPI returns a 429. Google asks callers to wait
// at least 60 seconds before retrying a quota error.
var defaultRetryDelays = []time.Duration{60 * time.Second, 120 * time.Second, 240 * time.Second}

// retryClient wraps a Client and retries calls that fail because the AMAPI quota was exceeded.
type retryClient struct {
	// next is a named field rather than an embedded Client so that a method added to Client does not
	// silently bypass the retry.
	next   Client
	logger *slog.Logger
	delays []time.Duration
}

// Compile-time check to ensure that retryClient implements Client.
var _ Client = &retryClient{}

// NewRetryClient wraps client so that AMAPI calls rejected with a 429 are retried with exponential
// backoff, until the retries are exhausted or ctx is done. Calls made with a WithoutRetry context are not
// retried. It returns nil if client is nil.
func NewRetryClient(client Client, logger *slog.Logger) Client {
	if client == nil {
		return nil
	}
	return newRetryClient(client, logger, defaultRetryDelays)
}

func newRetryClient(client Client, logger *slog.Logger, delays []time.Duration) *retryClient {
	return &retryClient{next: client, logger: logger, delays: delays}
}

type withoutRetryKey struct{}

// WithoutRetry returns a context under which the client returned by NewRetryClient makes a single attempt
// and returns 429 errors immediately. Use it for background work that already retries on its next run, so
// a quota error doesn't stall the rest of the batch or other jobs sharing the same runner.
func WithoutRetry(ctx context.Context) context.Context {
	return context.WithValue(ctx, withoutRetryKey{}, true)
}

// RetryDisabled reports whether ctx was returned by WithoutRetry.
func RetryDisabled(ctx context.Context) bool {
	disabled, _ := ctx.Value(withoutRetryKey{}).(bool)
	return disabled
}

func withRetry[T any](ctx context.Context, r *retryClient, method string, fn func() (T, error)) (T, error) {
	ret, err := fn()
	if RetryDisabled(ctx) {
		return ret, err
	}
	for attempt, delay := range r.delays {
		if !IsTooManyRequestsError(err) {
			break
		}
		// Jitter spreads out the retries of Fleet servers that share the proxy's quota, so they don't all
		// land at the start of the next quota window.
		if half := int64(delay / 2); half > 0 {
			delay += time.Duration(rand.Int64N(half)) //nolint:gosec // jitter does not need a secure source
		}
		r.logger.WarnContext(ctx, "AMAPI quota exceeded, retrying", "method", method, "retry", attempt+1, "delay", delay)
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			var zero T
			return zero, fmt.Errorf("%w: %w", ctx.Err(), err)
		}
		ret, err = fn()
		if err == nil {
			r.logger.InfoContext(ctx, "AMAPI call succeeded after retry", "method", method, "retry", attempt+1)
		}
	}
	return ret, err
}

func withRetryNoResult(ctx context.Context, r *retryClient, method string, fn func() error) error {
	_, err := withRetry(ctx, r, method, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

func (r *retryClient) SignupURLsCreate(ctx context.Context, serverURL, callbackURL string) (*android.SignupDetails, error) {
	return withRetry(ctx, r, "SignupURLsCreate", func() (*android.SignupDetails, error) {
		return r.next.SignupURLsCreate(ctx, serverURL, callbackURL)
	})
}

// EnterprisesCreate is deliberately not retried: each attempt creates a new PubSub topic and subscription
// before creating the enterprise, so a retried 429 would leave orphaned ones behind. It runs from the
// admin's signup callback, so the admin can retry it.
func (r *retryClient) EnterprisesCreate(ctx context.Context, req EnterprisesCreateRequest) (EnterprisesCreateResponse, error) {
	return r.next.EnterprisesCreate(ctx, req)
}

func (r *retryClient) EnterprisesPoliciesPatch(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts PoliciesPatchOpts) (*androidmanagement.Policy, error) {
	return withRetry(ctx, r, "EnterprisesPoliciesPatch", func() (*androidmanagement.Policy, error) {
		return r.next.EnterprisesPoliciesPatch(ctx, policyName, policy, opts)
	})
}

func (r *retryClient) EnterprisesDevicesPatch(ctx context.Context, deviceName string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
	return withRetry(ctx, r, "EnterprisesDevicesPatch", func() (*androidmanagement.Device, error) {
		return r.next.EnterprisesDevicesPatch(ctx, deviceName, device)
	})
}

func (r *retryClient) EnterprisesDevicesGet(ctx context.Context, deviceName string) (*androidmanagement.Device, error) {
	return withRetry(ctx, r, "EnterprisesDevicesGet", func() (*androidmanagement.Device, error) {
		return r.next.EnterprisesDevicesGet(ctx, deviceName)
	})
}

func (r *retryClient) EnterprisesDevicesDelete(ctx context.Context, deviceName string) error {
	return withRetryNoResult(ctx, r, "EnterprisesDevicesDelete", func() error {
		return r.next.EnterprisesDevicesDelete(ctx, deviceName)
	})
}

func (r *retryClient) EnterprisesDevicesIssueCommand(ctx context.Context, deviceName string, command *androidmanagement.Command) (*androidmanagement.Operation, error) {
	return withRetry(ctx, r, "EnterprisesDevicesIssueCommand", func() (*androidmanagement.Operation, error) {
		return r.next.EnterprisesDevicesIssueCommand(ctx, deviceName, command)
	})
}

// EnterprisesDevicesOperationsGet is deliberately not retried: the command reconciler, its only caller,
// stops its run on a 429 and resumes on the next run rather than waiting out the quota.
func (r *retryClient) EnterprisesDevicesOperationsGet(ctx context.Context, operationName string) (*androidmanagement.Operation, error) {
	return r.next.EnterprisesDevicesOperationsGet(ctx, operationName)
}

func (r *retryClient) EnterprisesDevicesListPartial(ctx context.Context, enterpriseName string, pageToken string) (*androidmanagement.ListDevicesResponse, error) {
	return withRetry(ctx, r, "EnterprisesDevicesListPartial", func() (*androidmanagement.ListDevicesResponse, error) {
		return r.next.EnterprisesDevicesListPartial(ctx, enterpriseName, pageToken)
	})
}

func (r *retryClient) EnterprisesEnrollmentTokensCreate(ctx context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
	return withRetry(ctx, r, "EnterprisesEnrollmentTokensCreate", func() (*androidmanagement.EnrollmentToken, error) {
		return r.next.EnterprisesEnrollmentTokensCreate(ctx, enterpriseName, token)
	})
}

func (r *retryClient) EnterpriseDelete(ctx context.Context, enterpriseName string) error {
	return withRetryNoResult(ctx, r, "EnterpriseDelete", func() error {
		return r.next.EnterpriseDelete(ctx, enterpriseName)
	})
}

func (r *retryClient) EnterprisesList(ctx context.Context, serverURL string) ([]*androidmanagement.Enterprise, error) {
	return withRetry(ctx, r, "EnterprisesList", func() ([]*androidmanagement.Enterprise, error) {
		return r.next.EnterprisesList(ctx, serverURL)
	})
}

func (r *retryClient) EnterprisesApplications(ctx context.Context, enterpriseName, packageName string) (*androidmanagement.Application, error) {
	return withRetry(ctx, r, "EnterprisesApplications", func() (*androidmanagement.Application, error) {
		return r.next.EnterprisesApplications(ctx, enterpriseName, packageName)
	})
}

func (r *retryClient) EnterprisesPoliciesModifyPolicyApplications(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
	return withRetry(ctx, r, "EnterprisesPoliciesModifyPolicyApplications", func() (*androidmanagement.Policy, error) {
		return r.next.EnterprisesPoliciesModifyPolicyApplications(ctx, policyName, appPolicies)
	})
}

func (r *retryClient) EnterprisesPoliciesRemovePolicyApplications(ctx context.Context, policyName string, packageNames []string) (*androidmanagement.Policy, error) {
	return withRetry(ctx, r, "EnterprisesPoliciesRemovePolicyApplications", func() (*androidmanagement.Policy, error) {
		return r.next.EnterprisesPoliciesRemovePolicyApplications(ctx, policyName, packageNames)
	})
}

func (r *retryClient) EnterprisesWebAppsCreate(ctx context.Context, enterpriseName string, webApp *androidmanagement.WebApp) (*androidmanagement.WebApp, error) {
	return withRetry(ctx, r, "EnterprisesWebAppsCreate", func() (*androidmanagement.WebApp, error) {
		return r.next.EnterprisesWebAppsCreate(ctx, enterpriseName, webApp)
	})
}

func (r *retryClient) SetAuthenticationSecret(secret string) error {
	return r.next.SetAuthenticationSecret(secret)
}
