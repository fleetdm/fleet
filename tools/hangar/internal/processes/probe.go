package processes

import (
	"context"
	"crypto/tls"
	"net"
	"strconv"
	"time"
)

// ServeTCPCheck reports whether something is listening and able to complete a
// TLS handshake on host:port. Fleet's dev server speaks TLS on 8080 with a
// self-signed cert, so a raw TCP connect would leave a "TLS handshake error:
// EOF" in fleet's log every probe — we do a real handshake (accepting any
// cert) so the probe is silent. 1.5s budget for connect + handshake.
func ServeTCPCheck(host string, port uint16) bool {
	if host == "" {
		host = "127.0.0.1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	dialer := tls.Dialer{Config: &tls.Config{InsecureSkipVerify: true}} //nolint:gosec // dev probe, any cert is fine
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
