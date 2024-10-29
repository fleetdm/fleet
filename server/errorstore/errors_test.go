package errorstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	pkgErrors "github.com/pkg/errors" //nolint:depguard
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	eh   = ctxerr.MockHandler{}
	ctxb = context.Background()
	ctx  = ctxerr.NewContext(ctxb, eh)
)

func alwaysErrors() error { return pkgErrors.New("always errors") }

func alwaysCallsAlwaysErrors() error { return alwaysErrors() }

func alwaysFleetErrors() error { return ctxerr.New(ctx, "always fleet errors") }

func alwaysNewError(eh *Handler) error {
	err := ctxerr.New(ctx, "always new errors")
	eh.Store(err)
	return err
}

func alwaysNewErrorTwo(eh *Handler) error {
	err := ctxerr.New(ctx, "always new errors two")
	eh.Store(err)
	return err
}

func alwaysWrappedErr() error { return ctxerr.Wrap(ctx, io.EOF, "always EOF") }

func TestHashErr(t *testing.T) {
	t.Run("without stack trace, same error is same hash", func(t *testing.T) {
		err1 := alwaysErrors()
		err2 := alwaysCallsAlwaysErrors()
		assert.Equal(t, hashError(err1), hashError(err2))
	})

	t.Run("different location, same error is different hash", func(t *testing.T) {
		err1 := alwaysFleetErrors()
		err2 := alwaysFleetErrors()
		assert.NotEqual(t, hashError(err1), hashError(err2))
	})

	t.Run("same error, wrapped, same hash", func(t *testing.T) {
		ferror1 := alwaysFleetErrors()

		w1, w2 := fmt.Errorf("wrap: %w", ferror1), pkgErrors.Wrap(ferror1, "wrap")
		h1, h2 := hashError(w1), hashError(w2)
		assert.Equal(t, h1, h2)
	})

	t.Run("generates json", func(t *testing.T) {
		var m []interface{}

		generatedErr := pkgErrors.New("some err")
		res, jsonBytes, err := hashAndMarshalError(generatedErr)
		require.NoError(t, err)
		assert.Equal(t, "mWoqz7iS1IPOZXGhpzHLl_DVQOyemWxCmvkpLz8uEZk=", res)
		require.NoError(t, json.Unmarshal([]byte(jsonBytes), &m))

		generatedErr2 := pkgErrors.New("some other err")
		res, jsonBytes, err = hashAndMarshalError(generatedErr2)
		require.NoError(t, err)
		assert.Equal(t, "8AXruOzQmQLF4H3SrzLxXSwFQgZ8DcbkoF1owo0RhTs=", res)
		require.NoError(t, json.Unmarshal([]byte(jsonBytes), &m))
	})
}

func TestHashErrFleetError(t *testing.T) {
	t.Run("Marshal", func(t *testing.T) {
		var m []interface{}

		generatedErr := ctxerr.New(ctx, "some err")
		res, jsonBytes, err := hashAndMarshalError(generatedErr)
		require.NoError(t, err)
		assert.NotEmpty(t, res)
		require.NoError(t, json.Unmarshal([]byte(jsonBytes), &m))
	})

	t.Run("HashWrapped", func(t *testing.T) {
		// hashing a fleet error that wraps a root error hashes to the same
		// value if it is from the same location, even if wrapped differently
		// afterwards.
		err := alwaysWrappedErr()
		werr1, werr2 := pkgErrors.Wrap(err, "wrap pkg"), fmt.Errorf("wrap fmt: %w", err)
		wantHash := hashError(err)
		h1, h2 := hashError(werr1), hashError(werr2)
		assert.Equal(t, wantHash, h1)
		assert.Equal(t, wantHash, h2)
	})

	t.Run("HashNew", func(t *testing.T) {
		err := alwaysFleetErrors()
		werr := ctxerr.Wrap(ctx, err, "wrap ctxterr")
		werr1, werr2 := pkgErrors.Wrap(err, "wrap pkg"), fmt.Errorf("wrap fmt: %w", err)
		wantHash := hashError(err)
		h0, h1, h2 := hashError(werr), hashError(werr1), hashError(werr2)
		assert.Equal(t, wantHash, h0)
		assert.Equal(t, wantHash, h1)
		assert.Equal(t, wantHash, h2)
	})

	t.Run("HashSameRootDifferentLocation", func(t *testing.T) {
		err1 := alwaysWrappedErr()
		err2 := func() error { return ctxerr.Wrap(ctx, io.EOF, "always EOF") }()
		err3 := func() error { return ctxerr.Wrap(ctx, io.EOF, "always EOF") }()
		h1, h2, h3 := hashError(err1), hashError(err2), hashError(err3)
		assert.NotEqual(t, h1, h2)
		assert.NotEqual(t, h1, h3)
		assert.NotEqual(t, h2, h3)
	})
}

func TestUnwrapAll(t *testing.T) {
	root := sql.ErrNoRows
	werr := pkgErrors.Wrap(root, "pkg wrap")
	gerr := fmt.Errorf("fmt wrap: %w", werr)
	eerr := ctxerr.Wrap(ctx, gerr, "fleet wrap")
	eerr2 := ctxerr.Wrap(ctx, eerr, "fleet wrap 2")

	uw := ctxerr.Cause(eerr2)
	assert.Equal(t, uw, root)
	assert.Nil(t, ctxerr.Cause(nil))
}

func TestErrorHandler(t *testing.T) {
	t.Run("works if the error handler is down", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		eh := newTestHandler(ctx, nil, kitlog.NewNopLogger(), time.Minute, nil, nil)

		doneCh := make(chan struct{})
		go func() {
			eh.Store(pkgErrors.New("test"))
			close(doneCh)
		}()

		// should not even block in the call to Store as there is no handler running
		ticker := time.NewTicker(1 * time.Second)
		select {
		case <-doneCh:
		case <-ticker.C:
			t.FailNow()
		}
	})

	t.Run("works if the error storage is disabled", func(t *testing.T) {
		eh := newTestHandler(context.Background(), nil, kitlog.NewNopLogger(), -1, nil, nil)

		doneCh := make(chan struct{})
		go func() {
			eh.Store(pkgErrors.New("test"))
			close(doneCh)
		}()

		// should not even block in the call to Store as there is no handler running
		ticker := time.NewTicker(1 * time.Second)
		select {
		case <-doneCh:
		case <-ticker.C:
			t.FailNow()
		}
	})

	wd, err := os.Getwd()
	require.NoError(t, err)
	wd = regexp.QuoteMeta(wd)

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, "error:", false, false, false)
		t.Run("collects errors", func(t *testing.T) { testErrorHandlerCollectsErrors(t, pool, wd, false) })
		t.Run("collects different errors", func(t *testing.T) { testErrorHandlerCollectsDifferentErrors(t, pool, wd, false) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, "error:", true, true, false)
		t.Run("collects errors", func(t *testing.T) { testErrorHandlerCollectsErrors(t, pool, wd, false) })
		t.Run("collects different errors", func(t *testing.T) { testErrorHandlerCollectsDifferentErrors(t, pool, wd, false) })
	})
}

func testErrorHandlerCollectsErrors(t *testing.T, pool fleet.RedisPool, wd string, flush bool) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	chGo, chDone := make(chan struct{}), make(chan struct{})

	var storeCalls int32 = 3
	testOnStart := func() {
		close(chGo)
	}
	testOnStore := func(err error) {
		require.NoError(t, err)
		if atomic.AddInt32(&storeCalls, -1) == 0 {
			close(chDone)
		}
	}
	eh := newTestHandler(ctx, pool, kitlog.NewNopLogger(), time.Minute, testOnStart, testOnStore)

	<-chGo

	for i := 0; i < 3; i++ {
		alwaysNewError(eh) //nolint:errcheck
	}

	<-chDone

	errors, err := eh.Retrieve(flush)
	require.NoError(t, err)
	require.Len(t, errors, 1)

	assert.Regexp(t, regexp.MustCompile(`\[
  \{
    "message": "always new errors",
    "data": \{
      "timestamp": ".+"
    \},
    "stack": \[
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.alwaysNewError \(errors_test\.go\:\d+\)",
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.testErrorHandlerCollectsErrors \(errors_test\.go\:\d+\)",
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.TestErrorHandler\.func\d\.\d \(errors_test\.go\:\d+\)",
      ".+",
      ".+"
    \]
  \}
\]`), string(errors[0].Chain))

	errors, err = eh.Retrieve(flush)
	require.NoError(t, err)
	if flush {
		assert.Len(t, errors, 0)
	} else {
		assert.Len(t, errors, 1)
	}

	// ensure we clear errors before returning
	_, err = eh.Retrieve(true)
	require.NoError(t, err)
}

func testErrorHandlerCollectsDifferentErrors(t *testing.T, pool fleet.RedisPool, wd string, flush bool) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	var storeCalls int32 = 5

	chGo, chDone := make(chan struct{}), make(chan struct{})

	testOnStart := func() {
		close(chGo)
	}
	testOnStore := func(err error) {
		require.NoError(t, err)
		if atomic.AddInt32(&storeCalls, -1) == 0 {
			close(chDone)
		}
	}

	eh := newTestHandler(ctx, pool, kitlog.NewNopLogger(), time.Minute, testOnStart, testOnStore)

	<-chGo

	// those two errors are different because from a different strack trace
	// (different line)
	alwaysNewError(eh) //nolint:errcheck
	alwaysNewError(eh) //nolint:errcheck

	// while those two are the same, only one gets store
	for i := 0; i < 2; i++ {
		alwaysNewError(eh) //nolint:errcheck
	}

	alwaysNewErrorTwo(eh) //nolint:errcheck

	<-chDone

	errors, err := eh.Retrieve(flush)
	require.NoError(t, err)
	require.Len(t, errors, 4)

	// order is not guaranteed by scan keys
	for _, jsonErr := range errors {
		msg := string(jsonErr.Chain)
		if strings.Contains(msg, "new errors two") {
			assert.Regexp(t, regexp.MustCompile(`\[
  \{
    "message": "always new errors two",
    "data": \{
      "timestamp": ".+"
    \},
    "stack": \[
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.alwaysNewErrorTwo \(errors_test\.go\:\d+\)",
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.testErrorHandlerCollectsDifferentErrors \(errors_test\.go\:\d+\)",
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.TestErrorHandler\.func\d\.\d \(errors_test\.go\:\d+\)",
      ".+",
      ".+"
    \]
  \}
\]`), msg)
		} else {
			assert.Regexp(t, regexp.MustCompile(`\[
  \{
    "message": "always new errors",
    "data": \{
      "timestamp": ".+"
    \},
    "stack": \[
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.alwaysNewError \(errors_test\.go\:\d+\)",
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.testErrorHandlerCollectsDifferentErrors \(errors_test\.go\:\d+\)",
      "github\.com\/fleetdm\/fleet\/v4\/server\/errorstore\.TestErrorHandler\.func\d.\d \(errors_test\.go\:\d+\)",
      ".+",
      ".+"
    \]
  \}
\]`), msg)
		}
	}

	// ensure we clear errors before returning
	_, err = eh.Retrieve(true)
	require.NoError(t, err)
}

func TestHttpHandler(t *testing.T) {
	setupTest := func(t *testing.T) *Handler {
		pool := redistest.SetupRedis(t, "error:", false, false, false)
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()

		var storeCalls int32 = 3

		chGo, chDone := make(chan struct{}), make(chan struct{})
		testOnStart := func() {
			close(chGo)
		}
		testOnStore := func(err error) {
			require.NoError(t, err)
			if atomic.AddInt32(&storeCalls, -1) == 0 {
				close(chDone)
			}
		}

		eh := newTestHandler(ctx, pool, kitlog.NewNopLogger(), time.Minute, testOnStart, testOnStore)

		<-chGo
		// simulate two errors, one happening twice
		err1 := ctxerr.New(ctx, "err1")
		err2 := ctxerr.New(ctx, "err2")
		eh.Store(err1)
		eh.Store(err2)
		eh.Store(err1)
		<-chDone

		return eh
	}

	type errResp struct {
		Count int
		Chain []struct{ Message string }
	}

	var errs []errResp

	sortByCount := func(errs []errResp) {
		sort.Slice(errs, func(i, j int) bool {
			return errs[i].Count > errs[j].Count
		})
	}

	t.Run("retrieves errors", func(t *testing.T) {
		eh := setupTest(t)
		req := httptest.NewRequest("GET", "/", nil)
		res := httptest.NewRecorder()
		eh.ServeHTTP(res, req)

		require.Equal(t, res.Code, 200)
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errs))
		require.Len(t, errs, 2)
		require.NotEmpty(t, errs[0].Chain[0].Message)
		require.NotEmpty(t, errs[1].Chain[0].Message)

		sortByCount(errs)
		require.Equal(t, 2, errs[0].Count)
		require.Equal(t, 1, errs[1].Count)
	})

	t.Run("flushes errors after retrieving if the flush flag is true", func(t *testing.T) {
		eh := setupTest(t)
		req := httptest.NewRequest("GET", "/?flush=true", nil)
		res := httptest.NewRecorder()
		eh.ServeHTTP(res, req)

		require.Equal(t, res.Code, 200)
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errs))
		require.Len(t, errs, 2)
		require.NotEmpty(t, errs[0].Chain[0].Message)
		require.NotEmpty(t, errs[1].Chain[0].Message)

		sortByCount(errs)
		require.Equal(t, 2, errs[0].Count)
		require.Equal(t, 1, errs[1].Count)

		req = httptest.NewRequest("GET", "/?flush=true", nil)
		res = httptest.NewRecorder()
		eh.ServeHTTP(res, req)
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errs))
		require.Len(t, errs, 0)
	})

	t.Run("fails with correct status code if the flush flag is invalid", func(t *testing.T) {
		eh := setupTest(t)
		req := httptest.NewRequest("GET", "/?flush=invalid", nil)
		res := httptest.NewRecorder()
		eh.ServeHTTP(res, req)

		require.Equal(t, res.Code, 400)
	})
}
