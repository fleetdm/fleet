package redis

import (
	"errors"
	"io"
	"testing"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
)

type netError struct {
	error
	timeout      bool
	temporary    bool
	allowedCalls int // once this reaches 0, mockDial does not return an error
	countCalls   int
}

func (t *netError) Timeout() bool   { return t.timeout }
func (t *netError) Temporary() bool { return t.temporary }

var errFromConn = errors.New("SUCCESS")

type redisConn struct{}

func (redisConn) Close() error                                       { return errFromConn }
func (redisConn) Err() error                                         { return errFromConn }
func (redisConn) Do(_ string, _ ...interface{}) (interface{}, error) { return nil, errFromConn }
func (redisConn) Send(_ string, _ ...interface{}) error              { return errFromConn }
func (redisConn) Flush() error                                       { return errFromConn }
func (redisConn) Receive() (interface{}, error)                      { return nil, errFromConn }

func TestConnectRetry(t *testing.T) {
	mockDial := func(err error) func(net, addr string, opts ...redigo.DialOption) (redigo.Conn, error) {
		return func(net, addr string, opts ...redigo.DialOption) (redigo.Conn, error) {
			var ne *netError
			if errors.As(err, &ne) {
				ne.countCalls++
				if ne.allowedCalls <= 0 {
					return redisConn{}, nil
				}
				ne.allowedCalls--
			}
			return nil, err
		}
	}

	cases := []struct {
		err       error
		retries   int
		wantCalls int
		min, max  time.Duration
	}{
		// the min-max time intervals are based on the backoff default configuration as
		// used in the Dial func of the redis pool. It starts with 500ms interval,
		// multiplies by 1.5 on each attempt, and has a randomization of 0.5 that must
		// be accounted for. Example ranges of intervals are given at
		// https://github.com/fleetdm/fleet/pull/1962#issue-729635664
		// and were used to calculate the (approximate) expected range.
		{
			io.EOF, 0, 1, 0, 100 * time.Millisecond,
		}, // non-retryable, no retry configured
		{
			&netError{error: io.EOF, timeout: true, allowedCalls: 10}, 0, 1, 0, 100 * time.Millisecond,
		}, // retryable, but no retry configured
		{
			io.EOF, 3, 1, 0, 100 * time.Millisecond,
		}, // non-retryable, retry configured
		{
			&netError{error: io.EOF, timeout: true, allowedCalls: 10}, 2, 3, 625 * time.Millisecond, 3500 * time.Millisecond,
		}, // retryable, retry configured
		{
			&netError{error: io.EOF, temporary: true, allowedCalls: 10}, 2, 3, 625 * time.Millisecond, 3500 * time.Millisecond,
		}, // retryable, retry configured
		{
			&netError{error: io.EOF, allowedCalls: 10}, 2, 1, 0, 100 * time.Millisecond,
		}, // net error, but non-retryable
		{
			&netError{error: io.EOF, timeout: true, allowedCalls: 1}, 10, 2, 250 * time.Millisecond, 800 * time.Millisecond,
		}, // retryable, but succeeded after one retry
	}
	for _, c := range cases {
		t.Run(c.err.Error(), func(t *testing.T) {
			start := time.Now()
			_, err := NewPool(PoolConfig{
				Server:               "127.0.0.1:12345",
				ConnectRetryAttempts: c.retries,
				testRedisDialFunc:    mockDial(c.err),
			})
			diff := time.Since(start)
			require.GreaterOrEqual(t, diff, c.min)
			require.LessOrEqual(t, diff, c.max)
			require.Error(t, err)

			wantErr := io.EOF
			var ne *netError
			if errors.As(c.err, &ne) {
				require.Equal(t, c.wantCalls, ne.countCalls)
				if ne.allowedCalls == 0 {
					wantErr = errFromConn
				}
			} else {
				require.Equal(t, c.wantCalls, 1)
			}

			// the error is returned as part of the cluster.Refresh error, hence the
			// check with Contains.
			require.Contains(t, err.Error(), wantErr.Error())
		})
	}
}

func TestParseElastiCacheEndpoint(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      string
		wantRegion    string
		wantCacheName string
		wantErr       bool
	}{
		{
			name:          "serverless endpoint from AWS stack with port",
			endpoint:      "fleet-iam-test-cache-6l5khx.serverless.use2.cache.amazonaws.com:6379",
			wantRegion:    "us-east-2",
			wantCacheName: "fleet-iam-test-cache",
			wantErr:       false,
		},
		{
			name:          "serverless endpoint from AWS stack without port",
			endpoint:      "fleet-iam-test-cache-6l5khx.serverless.use2.cache.amazonaws.com",
			wantRegion:    "us-east-2",
			wantCacheName: "fleet-iam-test-cache",
			wantErr:       false,
		},
		{
			name:          "standalone master endpoint from AWS stack",
			endpoint:      "master.fleet-iam-standalone.6l5khx.use2.cache.amazonaws.com:6379",
			wantRegion:    "us-east-2",
			wantCacheName: "fleet-iam-standalone",
			wantErr:       false,
		},
		{
			name:          "standalone cluster node endpoint from AWS stack",
			endpoint:      "fleet-iam-standalone-001.fleet-iam-standalone.6l5khx.use2.cache.amazonaws.com:6379",
			wantRegion:    "us-east-2",
			wantCacheName: "fleet-iam-standalone",
			wantErr:       false,
		},
		// Additional test cases for different regions
		{
			name:          "serverless endpoint us-east-1",
			endpoint:      "my-cache-abc123.serverless.use1.cache.amazonaws.com",
			wantRegion:    "us-east-1",
			wantCacheName: "my-cache",
			wantErr:       false,
		},
		{
			name:          "serverless endpoint eu-west-1",
			endpoint:      "prod-cache-xyz789.serverless.euw1.cache.amazonaws.com",
			wantRegion:    "eu-west-1",
			wantCacheName: "prod-cache",
			wantErr:       false,
		},
		// Cluster mode endpoints with different node numbers
		{
			name:          "cluster mode endpoint node 002",
			endpoint:      "my-cluster-002.my-cluster.abc123.euc1.cache.amazonaws.com:6379",
			wantRegion:    "eu-central-1",
			wantCacheName: "my-cluster",
			wantErr:       false,
		},
		{
			name:          "cluster mode endpoint node 003",
			endpoint:      "redis-cluster-003.redis-cluster.xyz789.apne1.cache.amazonaws.com",
			wantRegion:    "ap-northeast-1",
			wantCacheName: "redis-cluster",
			wantErr:       false,
		},
		// Different regions
		{
			name:          "ap-southeast-1 endpoint",
			endpoint:      "cache-name.xyz123.apse1.cache.amazonaws.com",
			wantRegion:    "ap-southeast-1",
			wantCacheName: "cache-name",
			wantErr:       false,
		},
		{
			name:          "us-west-2 endpoint",
			endpoint:      "test-cache.def456.usw2.cache.amazonaws.com:6379",
			wantRegion:    "us-west-2",
			wantCacheName: "test-cache",
			wantErr:       false,
		},
		// Edge cases
		{
			name:          "cache name with multiple hyphens",
			endpoint:      "my-long-cache-name.serverless.usw2.cache.amazonaws.com",
			wantRegion:    "us-west-2",
			wantCacheName: "my-long-cache-name",
			wantErr:       false,
		},
		{
			name:          "already full region name",
			endpoint:      "test-cache.us-east-1.cache.amazonaws.com",
			wantRegion:    "us-east-1",
			wantCacheName: "test-cache",
			wantErr:       false,
		},
		// Error cases
		{
			name:          "invalid endpoint format",
			endpoint:      "not-a-valid-endpoint",
			wantRegion:    "",
			wantCacheName: "",
			wantErr:       true,
		},
		{
			name:          "missing cache.amazonaws.com",
			endpoint:      "my-cache.use1.amazonaws.com",
			wantRegion:    "",
			wantCacheName: "",
			wantErr:       true,
		},
		{
			name:          "empty endpoint",
			endpoint:      "",
			wantRegion:    "",
			wantCacheName: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRegion, gotCacheName, err := parseElastiCacheEndpoint(tt.endpoint)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseElastiCacheEndpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotRegion != tt.wantRegion {
				t.Errorf("parseElastiCacheEndpoint() gotRegion = %v, want %v", gotRegion, tt.wantRegion)
			}
			if gotCacheName != tt.wantCacheName {
				t.Errorf("parseElastiCacheEndpoint() gotCacheName = %v, want %v", gotCacheName, tt.wantCacheName)
			}
		})
	}
}
