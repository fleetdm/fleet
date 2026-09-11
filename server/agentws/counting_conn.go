package agentws

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
)

// countingConn wraps the connection underlying a WebSocket and counts raw
// bytes in/out. Because it wraps the hijacked net.Conn (post-TLS), the counts
// include WebSocket frame overhead and control frames (pings/pongs) — on an
// idle notification channel the keepalives are most of the traffic, so
// counting at the message layer would show nothing.
type countingConn struct {
	net.Conn
	bytesIn  atomic.Int64
	bytesOut atomic.Int64
}

func (c *countingConn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	c.bytesIn.Add(int64(n))
	return n, err
}

func (c *countingConn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	c.bytesOut.Add(int64(n))
	return n, err
}

// hijackCountingResponseWriter intercepts the Hijack done by the WebSocket
// upgrader and hands it a countingConn instead of the raw connection. The
// upgrader builds its own buffered reader/writer on the returned conn (we set
// a non-zero ReadBufferSize), so all subsequent I/O is counted.
type hijackCountingResponseWriter struct {
	http.ResponseWriter
}

var _ http.Hijacker = (*hijackCountingResponseWriter)(nil)

func (w *hijackCountingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	netConn, brw, err := hj.Hijack()
	if err != nil {
		return nil, nil, err
	}
	return &countingConn{Conn: netConn}, brw, nil
}
