package apnsmock

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Preflight reports what a mock needs before it can start. Starting without
// these fails a moment later: the mock exits when it can't reach Redis, and
// `go run` fails without the package or with the port taken.
type Preflight struct {
	// HasSource is false when repo (server 1's clone, which the mock runs
	// from) is unset or predates cmd/apple-apns-mock.
	HasSource bool `json:"has_source"`
	// RedisUp is true when the server's Redis answers PING.
	RedisUp bool `json:"redis_up"`
	// PortFree is true when nothing listens on the mock's port.
	PortFree bool `json:"port_free"`
}

// Check runs the checks for a mock run from repo, on the Redis at redisPort,
// listening on port.
func Check(repo string, redisPort, port uint16) Preflight {
	return Preflight{
		HasSource: hasSource(repo),
		RedisUp:   redisUp(redisPort),
		PortFree:  portFree(port),
	}
}

func hasSource(repo string) bool {
	if strings.TrimSpace(repo) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(repo, "cmd", "apple-apns-mock"))
	return err == nil && info.IsDir()
}

// redisUp sends PING, because a port that accepts connections may still be
// Docker's proxy for a container that isn't ready.
func redisUp(port uint16) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port))), 500*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	if _, err := conn.Write([]byte("PING\r\n")); err != nil {
		return false
	}
	line, err := bufio.NewReader(conn).ReadString('\n')
	return err == nil && strings.TrimSpace(line) == "+PONG"
}

// portFree matches the mock's --listen 127.0.0.1:<port>.
func portFree(port uint16) bool {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port))))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
