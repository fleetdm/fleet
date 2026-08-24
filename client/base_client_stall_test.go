package client

import (
	"context"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// blockingReader serves `data`, then blocks every subsequent Read until Close
// is called — simulating a connection that stalls mid-download.
type blockingReader struct {
	data      []byte
	pos       int
	closed    chan struct{}
	closeOnce sync.Once
}

func (r *blockingReader) Read(p []byte) (int, error) {
	if r.pos < len(r.data) {
		n := copy(p, r.data[r.pos:])
		r.pos += n
		return n, nil
	}
	<-r.closed // block until the stall watchdog closes the body
	return 0, io.ErrClosedPipe
}

func (r *blockingReader) Close() error {
	r.closeOnce.Do(func() { close(r.closed) })
	return nil
}

// trickleReader delivers `chunks` single bytes, sleeping `delay` before each —
// simulating a slow-but-healthy download.
type trickleReader struct {
	chunks int
	delay  time.Duration
	sent   int
}

func (r *trickleReader) Read(p []byte) (int, error) {
	if r.sent >= r.chunks {
		return 0, io.EOF
	}
	time.Sleep(r.delay)
	r.sent++
	p[0] = 'x'
	return 1, nil
}

func (r *trickleReader) Close() error { return nil }

func fileResp(dir string, body io.ReadCloser, stall time.Duration) (*FileResponse, *http.Response) {
	fr := &FileResponse{DestPath: dir, StallTimeout: stall}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       body,
		Header:     http.Header{"Content-Disposition": []string{`attachment;filename="installer.pkg"`}},
	}
	return fr, resp
}

func TestFileResponseStallTimeout(t *testing.T) {
	t.Run("aborts a stalled download as a retryable timeout", func(t *testing.T) {
		fr, resp := fileResp(t.TempDir(), &blockingReader{data: []byte("partial"), closed: make(chan struct{})}, 500*time.Millisecond)
		// Shorten the watchdog poll floor so the sub-second stall is caught fast.
		fr.stallCheckInterval = 50 * time.Millisecond

		start := time.Now()
		err := fr.Handle(resp)
		elapsed := time.Since(start)

		require.Error(t, err)
		// Must be classified transient so the installer retries instead of failing setup.
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.GreaterOrEqual(t, elapsed, 400*time.Millisecond, "should wait ~StallTimeout before aborting")
		require.Less(t, elapsed, 3*time.Second, "should abort, not hang")
	})

	t.Run("does not abort a slow-but-progressing download", func(t *testing.T) {
		// Bytes arrive every 200ms, well under the 1s stall timeout.
		fr, resp := fileResp(t.TempDir(), &trickleReader{chunks: 5, delay: 200 * time.Millisecond}, 1*time.Second)

		err := fr.Handle(resp)
		require.NoError(t, err)

		data, err := os.ReadFile(fr.DestFilePath)
		require.NoError(t, err)
		require.Len(t, data, 5, "the full download should have completed")
	})

	t.Run("disabled when StallTimeout is zero (progress path unchanged)", func(t *testing.T) {
		fr, resp := fileResp(t.TempDir(), &trickleReader{chunks: 3, delay: 10 * time.Millisecond}, 0)

		err := fr.Handle(resp)
		require.NoError(t, err)

		data, err := os.ReadFile(fr.DestFilePath)
		require.NoError(t, err)
		require.Len(t, data, 3)
	})
}
