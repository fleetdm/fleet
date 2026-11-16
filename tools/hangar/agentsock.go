package main

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/agentapi"
)

// maxSockPath is sun_path's size on macOS, including the trailing NUL.
const maxSockPath = 104

// startAgentSocket serves h on <dataDir>/agent/hangar.sock to same-uid peers
// only. The caller holds the data-dir lock, so an existing socket file is
// stale and gets replaced.
func startAgentSocket(dataDir string, h http.Handler) (stop func(), err error) {
	dir := filepath.Join(dataDir, "agent")
	sock := filepath.Join(dir, "hangar.sock")
	if len(sock) >= maxSockPath {
		return nil, fmt.Errorf("agent socket path %s is %d bytes, over the %d-byte limit; agent API disabled", sock, len(sock), maxSockPath-1)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return nil, fmt.Errorf("securing %s: %w", dir, err)
	}
	if err := os.Remove(sock); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("removing stale %s: %w", sock, err)
	}
	ln, err := net.Listen("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", sock, err)
	}
	if err := os.Chmod(sock, 0o600); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("securing %s: %w", sock, err)
	}
	srv := &http.Server{Handler: h, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		if err := srv.Serve(peerListener{ln}); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("agent socket stopped", "err", err)
		}
	}()
	slog.Info("agent API listening", "socket", sock)
	return func() { _ = srv.Close() }, nil
}

// peerListener drops connections from other uids.
type peerListener struct{ net.Listener }

func (l peerListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}
		uid, err := agentapi.PeerUID(c.(*net.UnixConn))
		if err == nil && uid == uint32(os.Getuid()) {
			return c, nil
		}
		slog.Warn("agent socket: rejected peer", "uid", uid, "err", err)
		_ = c.Close()
	}
}
