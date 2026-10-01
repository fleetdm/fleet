package netinfo

import (
	"net"
	"testing"
)

// The result depends on the host's routing table, so this asserts the contract
// rather than a value: either a usable IPv4, or "" when there's no route (a
// machine between networks, or a sandboxed CI runner). Anything else — an IPv6
// address, a host:port pair, a partial string — would break the callers that
// interpolate it straight into a URL.
func TestLanIPIsAUsableIPv4OrEmpty(t *testing.T) {
	got := LanIP()
	if got == "" {
		t.Skip("no outbound route on this host; nothing to assert")
	}

	ip := net.ParseIP(got)
	if ip == nil {
		t.Fatalf("LanIP() = %q, which does not parse as an IP", got)
	}
	if ip.To4() == nil {
		t.Errorf("LanIP() = %q, want IPv4 (callers build host:port from it)", got)
	}
	if ip.IsUnspecified() {
		t.Errorf("LanIP() = %q, which no other machine can dial", got)
	}
}
