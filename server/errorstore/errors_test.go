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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	pkgErrors "github.com/pkg/errors" //nolint:depguard
	"github.com/rotisserie/eris"      //nolint:depguard
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func alwaysErrors() error { return pkgErrors.New("always errors") }

func alwaysCallsAlwaysErrors() error { return alwaysErrors() }

func alwaysErisErrors() error { return eris.New("always eris errors") }

func alwaysNewError(eh *Handler) error {
	err := eris.New("always new errors")
	eh.Store(err)
	return err
}

func alwaysNewErrorTwo(eh *Handler) error {
	err := eris.New("always new errors two")
	eh.Store(err)
	return err
}

func alwaysWrappedErr() error { return eris.Wrap(io.EOF, "always EOF") }

func TestHashErr(t *testing.T) {
	t.Run("without stack trace, same error is same hash", func(t *testing.T) {
		err1 := alwaysErrors()
		err2 := alwaysCallsAlwaysErrors()
		assert.Equal(t, hashError(err1), hashError(err2))
	})

	t.Run("different location, same error is different hash", func(t *testing.T) {
		err1 := alwaysErisErrors()
		err2 := alwaysErisErrors()
		assert.NotEqual(t, hashError(err1), hashError(err2))
	})

	t.Run("same error, wrapped, same hash", func(t *testing.T) {
		eris1 := alwaysErisErrors()

		w1, w2 := fmt.Errorf("wrap: %w", eris1), pkgErrors.Wrap(eris1, "wrap")
		h1, h2 := hashError(w1), hashError(w2)
		assert.Equal(t, h1, h2)
	})

	t.Run("generates json", func(t *testing.T) {
		var m map[string]interface{}

		generatedErr := pkgErrors.New("some err")
		res, jsonBytes, err := hashAndMarshalError(generatedErr)
		require.NoError(t, err)
		assert.Equal(t, "mWoqz7iS1IPOZXGhpzHLl_DVQOyemWxCmvkpLz8uEZk=", res)
		assert.True(t, strings.HasPrefix(jsonBytes, `{
  "external": "some err`))
		require.NoError(t, json.Unmarshal([]byte(jsonBytes), &m))

		generatedErr2 := pkgErrors.New("some other err")
		res, jsonBytes, err = hashAndMarshalError(generatedErr2)
		require.NoError(t, err)
		assert.Equal(t, "8AXruOzQmQLF4H3SrzLxXSwFQgZ8DcbkoF1owo0RhTs=", res)
		assert.True(t, strings.HasPrefix(jsonBytes, `{
  "external": "some other err`))
		require.NoError(t, json.Unmarshal([]byte(jsonBytes), &m))
	})
}

func TestHashErrEris(t *testing.T) {
	t.Run("Marshal", func(t *testing.T) {
		wd, err := os.Getwd()
		require.NoError(t, err)

		generatedErr := eris.New("some err")
		res, jsonBytes, err := hashAndMarshalError(generatedErr)
		require.NoError(t, err)
		assert.NotEmpty(t, res)

		assert.Regexp(t, regexp.MustCompile(fmt.Sprintf(`\{
  "root": \{
    "message": "some err",
    "stack": \[
      "errorstore.TestHashErrEris\.func\d+:%s/errors_test\.go:\d+"
    \]
  \}
\}`, regexp.QuoteMeta(wd))), jsonBytes)
	})

	t.Run("HashWrapped", func(t *testing.T) {
		// hashing an eris error that wraps a root error hashes to the same
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
		err := alwaysErisErrors()
		werr := eris.Wrap(err, "wrap eris")
		werr1, werr2 := pkgErrors.Wrap(err, "wrap pkg"), fmt.Errorf("wrap fmt: %w", err)
		wantHash := hashError(err)
		h0, h1, h2 := hashError(werr), hashError(werr1), hashError(werr2)
		assert.Equal(t, wantHash, h0)
		assert.Equal(t, wantHash, h1)
		assert.Equal(t, wantHash, h2)
	})

	t.Run("HashSameRootDifferentLocation", func(t *testing.T) {
		err1 := alwaysWrappedErr()
		err2 := func() error { return eris.Wrap(io.EOF, "always EOF") }()
		err3 := func() error { return eris.Wrap(io.EOF, "always EOF") }()
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
	eerr := eris.Wrap(gerr, "eris wrap")
	eerr2 := eris.Wrap(eerr, "eris wrap 2")

	uw := eris.Cause(eerr2)
	assert.Equal(t, uw, root)
	assert.Nil(t, eris.Cause(nil))
}

func TestErrorHandler(t *testing.T) {
	// Skipped until error publishing is re-enabled.
	t.Skip()

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

	wd, err := os.Getwd()
	require.NoError(t, err)
	wd = regexp.QuoteMeta(wd)

	t.Run("standalone", func(t *testing.T) {
		pool := redistest.SetupRedis(t, false, false, false)
		t.Run("collects errors", func(t *testing.T) { testErrorHandlerCollectsErrors(t, pool, wd) })
		t.Run("collects different errors", func(t *testing.T) { testErrorHandlerCollectsDifferentErrors(t, pool, wd) })
	})

	t.Run("cluster", func(t *testing.T) {
		pool := redistest.SetupRedis(t, true, true, false)
		t.Run("collects errors", func(t *testing.T) { testErrorHandlerCollectsErrors(t, pool, wd) })
		t.Run("collects different errors", func(t *testing.T) { testErrorHandlerCollectsDifferentErrors(t, pool, wd) })
	})
}

func testErrorHandlerCollectsErrors(t *testing.T, pool fleet.RedisPool, wd string) {
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
		alwaysNewError(eh)
	}

	<-chDone

	errors, err := eh.Flush()
	require.NoError(t, err)
	require.Len(t, errors, 1)

	assert.Regexp(t, regexp.MustCompile(fmt.Sprintf(`\{
  "root": \{
    "message": "always new errors",
    "stack": \[
      "errorstore\.TestErrorHandler\.func\d\.\d+:%s/errors_test\.go:\d+",
      "errorstore\.testErrorHandlerCollectsErrors:%[1]s/errors_test\.go:\d+",
      "errorstore\.alwaysNewError:%s/errors_test\.go:\d+"
    \]
  \}`, wd, wd)), errors[0])

	// and then errors are gone
	errors, err = eh.Flush()
	require.NoError(t, err)
	assert.Len(t, errors, 0)
}

func testErrorHandlerCollectsDifferentErrors(t *testing.T, pool fleet.RedisPool, wd string) {
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
	alwaysNewError(eh)
	alwaysNewError(eh)

	// while those two are the same, only one gets store
	for i := 0; i < 2; i++ {
		alwaysNewError(eh)
	}

	alwaysNewErrorTwo(eh)

	<-chDone

	errors, err := eh.Flush()
	require.NoError(t, err)
	require.Len(t, errors, 4)

	// order is not guaranteed by scan keys
	for _, jsonErr := range errors {
		if strings.Contains(jsonErr, "new errors two") {
			assert.Regexp(t, regexp.MustCompile(fmt.Sprintf(`\{
  "root": \{
    "message": "always new errors two",
    "stack": \[
      "errorstore\.TestErrorHandler\.func\d\.\d+:%s/errors_test\.go:\d+",
      "errorstore\.testErrorHandlerCollectsDifferentErrors:%[1]s/errors_test\.go:\d+",
      "errorstore\.alwaysNewErrorTwo:%[1]s/errors_test\.go:\d+"
    \]
  \}`, wd)), jsonErr)
		} else {
			assert.Regexp(t, regexp.MustCompile(fmt.Sprintf(`\{
  "root": \{
    "message": "always new errors",
    "stack": \[
      "errorstore\.TestErrorHandler\.func\d\.\d+:%s/errors_test\.go:\d+",
      "errorstore\.testErrorHandlerCollectsDifferentErrors:%[1]s/errors_test\.go:\d+",
      "errorstore\.alwaysNewError:%[1]s/errors_test\.go:\d+"
    \]
  \}`, wd)), jsonErr)
		}
	}
}

func TestHttpHandler(t *testing.T) {
	// Skipped until error publishing is re-enabled.
	t.Skip()

	pool := redistest.SetupRedis(t, false, false, false)
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	var storeCalls int32 = 2

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
	// store two errors
	alwaysNewError(eh)
	alwaysNewErrorTwo(eh)
	<-chDone

	req := httptest.NewRequest("GET", "/", nil)
	res := httptest.NewRecorder()
	eh.ServeHTTP(res, req)

	require.Equal(t, res.Code, 200)
	var errs []struct {
		Root struct {
			Message string
		}
		Wrap []struct {
			Message string
		}
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &errs))
	require.Len(t, errs, 2)
	require.NotEmpty(t, errs[0].Root.Message)
	require.NotEmpty(t, errs[1].Root.Message)
}
