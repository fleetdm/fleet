package apnsmock

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// listen starts a TCP server on 127.0.0.1 answering each line with reply, and
// returns its port.
func listen(t *testing.T, reply string) uint16 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				if _, err := bufio.NewReader(conn).ReadString('\n'); err == nil {
					_, _ = conn.Write([]byte(reply))
				}
			}()
		}
	}()
	return uint16(l.Addr().(*net.TCPAddr).Port)
}

func freePort(t *testing.T) uint16 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := uint16(l.Addr().(*net.TCPAddr).Port)
	l.Close()
	return port
}

func TestCheck(t *testing.T) {
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "cmd", "apple-apns-mock"), 0o755); err != nil {
		t.Fatal(err)
	}
	redis := listen(t, "+PONG\r\n")
	free := freePort(t)

	if got := Check(repo, redis, free); !got.HasSource || !got.RedisUp || !got.PortFree {
		t.Errorf("all met: %+v", got)
	}
	if got := Check(t.TempDir(), redis, free); got.HasSource {
		t.Error("a repo without cmd/apple-apns-mock counted as having the source")
	}
	if got := Check("", redis, free); got.HasSource {
		t.Error("no repo counted as having the source")
	}
	// Something that accepts connections but isn't Redis.
	if got := Check(repo, listen(t, "HTTP/1.1 400 Bad Request\r\n"), free); got.RedisUp {
		t.Error("a non-Redis listener counted as Redis")
	}
	if got := Check(repo, freePort(t), free); got.RedisUp {
		t.Error("nothing listening counted as Redis")
	}
	if got := Check(repo, redis, redis); got.PortFree {
		t.Error("a port in use counted as free")
	}
}
