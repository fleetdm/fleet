// Package netinfo answers questions about the host's own network identity.
//
// It exists because more than one part of Hangar needs the LAN address: SCEP
// enrollment URLs handed to a phone, and the python file server someone points
// another device at. Both care for the same reason — the address a *different*
// machine has to dial — so the logic lives here rather than in either feature.
package netinfo

import "net"

// LanIP returns the host's primary outbound IPv4, matching what
// `ipconfig getifaddr en0` reports in practice. Empty on failure, which callers
// should treat as "unknown" rather than an error: it's display detail, and a
// laptop that's between networks legitimately has no answer.
//
// Note this changes as the machine moves between networks, so callers should
// read it when it matters (on start, on display) rather than caching it for
// the life of the process.
func LanIP() string {
	// A UDP "connect" picks the source IP via the routing table without sending
	// any packets — no traffic leaves the machine and 8.8.8.8 is never reached.
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return addr.IP.String()
	}
	return ""
}
