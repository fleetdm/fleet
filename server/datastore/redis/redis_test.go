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
			&netError{error: io.EOF, timeout: true, allowedCalls: 1}, 10, 2, 250 * time.Millisecond, 750 * time.Millisecond,
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
