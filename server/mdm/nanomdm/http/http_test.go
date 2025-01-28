package http

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHTTPServerTimeoutError(t *testing.T) {
	// ensure that a read timeout error is properly detected
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := http.StatusOK
		if _, err := io.ReadAll(r.Body); err != nil {
			var toErr interface{ Timeout() bool }
			if errors.As(err, &toErr) && toErr.Timeout() {
				code = http.StatusRequestTimeout
			} else {
				code = http.StatusInternalServerError
			}
		}
		w.WriteHeader(code)
	}))

	srv.Config.ReadTimeout = time.Second
	srv.Start()
	defer srv.Close()

	req, err := http.NewRequest("POST", srv.URL, slowReader{b: []byte("slowly send this")})
	require.NoError(t, err)
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	code := res.StatusCode
	require.Equal(t, http.StatusRequestTimeout, code)
}

type slowReader struct {
	b []byte
}

func (s slowReader) Read(p []byte) (n int, err error) {
	if len(s.b) == 0 {
		return 0, io.EOF
	}

	time.Sleep(200 * time.Millisecond)
	n = copy(p, s.b[:len(s.b)/2])
	return n, nil
}
