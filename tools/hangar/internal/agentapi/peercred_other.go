//go:build !darwin

package agentapi

import (
	"errors"
	"net"
)

// PeerUID fails closed off macOS, Hangar's only platform.
func PeerUID(*net.UnixConn) (uint32, error) {
	return 0, errors.New("peer credentials are only checked on macOS")
}
