package errors

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/pubsub"
	kitlog "github.com/go-kit/kit/log"
	pkgErrors "github.com/pkg/errors"
	"github.com/rotisserie/eris"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func alwaysErrors() error { return pkgErrors.New("always errors") }

func alwaysCallsAlwaysErrors() error { return alwaysErrors() }

func alwaysErisErrors() error { return eris.New("always eris errors") }

func alwaysCallsAlwaysErisErrors() error { return alwaysErisErrors() }

func alwaysNewError() error { return New(eris.New("always new errors")) }

func alwaysNewErrorTwo() error { return New(eris.New("always new errors two")) }

func TestHashErr(t *testing.T) {
	t.Run("same error, same hash", func(t *testing.T) {
		err1 := alwaysErrors()
		err2 := alwaysCallsAlwaysErrors()
		assert.Equal(t, hashErrorLocation(err1), hashErrorLocation(err2))
		eris1 := alwaysErisErrors()
		eris2 := alwaysCallsAlwaysErisErrors()
		assert.Equal(t, hashErrorLocation(eris1), hashErrorLocation(eris2))
		assert.NotEqual(t, err1, eris1)
		assert.NotEmpty(t, err1)
		assert.NotEmpty(t, eris2)
	})

	t.Run("generates json", func(t *testing.T) {
		generatedErr := pkgErrors.New("some err")
		res, jsonBytes, err := hashErr(generatedErr)
		require.NoError(t, err)
		assert.Equal(t, "uNq0wUK9_ATMiyFvkXhrVoREEBDJjRtuPLJ9xJ2R7vI=", res)
		assert.True(t, strings.HasPrefix(jsonBytes, `{
  "external": "some err`))

		generatedErr2 := pkgErrors.New("some other err")
		res, jsonBytes, err = hashErr(generatedErr2)
		require.NoError(t, err)
		assert.Equal(t, "Dtkt3vUS5WAyVmYOPY113I-30fK0Mx7ZtqOxRsqmjmk=", res)
		assert.True(t, strings.HasPrefix(jsonBytes, `{
  "external": "some other err`))
	})
}

func TestHashErrEris(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	generatedErr := eris.New("some err")
	res, jsonBytes, err := hashErr(generatedErr)
	require.NoError(t, err)
	assert.Equal(t, "ur33yN3AcASxv0gjHnuQUKY1HA8gPEajVZ8RWL2VXA8=", res)
	assert.Equal(t, fmt.Sprintf(`{
  "root": {
    "message": "some err",
    "stack": [
      "errors.TestHashErrEris:%s/errors_test.go:33"
    ]
  }
}`, wd), jsonBytes)
}

func TestErroHandler(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	t.Run("works if the error handler is down", func(t *testing.T) {
		doneCh := make(chan struct{})
		go func() {
			New(pkgErrors.New("test"))
			close(doneCh)
		}()
		ticker := time.NewTicker(2 * time.Second)
		select {
		case <-doneCh:
		case <-ticker.C:
			t.FailNow()
		}
	})

	store, teardown := pubsub.SetupRedisForTest(t, false)
	defer teardown()

	eh := NewHandler(ctx, store.Pool(), kitlog.NewNopLogger())
	eh.Flush()

	t.Run("collects errors", func(t *testing.T) {
		alwaysNewError()
		alwaysNewError() // and it doesnt repeat them

		errors, err := eh.Flush()
		require.NoError(t, err)
		require.Len(t, errors, 1)
		assert.Equal(t, fmt.Sprintf(`{
  "root": {
    "message": "always new errors",
    "stack": [
      "errors.TestErroHandler.func2:%s/errors_test.go:108",
      "errors.alwaysNewError:%s/errors_test.go:27"
    ]
  }
}`, wd, wd), errors[0])

		// and then errors are gone
		errors, err = eh.Flush()
		require.NoError(t, err)
		assert.Len(t, errors, 0)
	})

	t.Run("collects different errors", func(t *testing.T) {
		alwaysNewError()
		alwaysNewError() // and it doesnt repeat them
		alwaysNewErrorTwo()

		time.Sleep(1 * time.Second)

		errors, err := eh.Flush()
		require.NoError(t, err)
		require.Len(t, errors, 2)
		assert.Equal(t, fmt.Sprintf(`{
  "root": {
    "message": "always new errors two",
    "stack": [
      "errors.TestErroHandler.func3:%s/errors_test.go:132",
      "errors.alwaysNewErrorTwo:%s/errors_test.go:29"
    ]
  }
}`, wd, wd), errors[0])
		assert.Equal(t, fmt.Sprintf(`{
  "root": {
    "message": "always new errors",
    "stack": [
      "errors.TestErroHandler.func3:%s/errors_test.go:131",
      "errors.alwaysNewError:%s/errors_test.go:27"
    ]
  }
}`, wd, wd), errors[1])
	})
}
