package netsock

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

func TestCollect(t *testing.T) {
	snap := &proc.Snapshot{
		Procs: map[int]proc.Process{
			100: {PID: 100, Name: "ollama", Cmdline: "ollama serve"},
			200: {PID: 200, Name: "python", Cmdline: "python -m aider"},
			300: {PID: 300, Name: "sshd", Cmdline: "/usr/sbin/sshd -D"},
			400: {PID: 400, Name: "Electron", Cmdline: "/Applications/Claude.app/.../Electron Helper"},
		},
		Conns: []proc.Conn{
			{PID: 100, Status: "LISTEN", Type: 1, LocalIP: "127.0.0.1", LocalPort: 11434},
			{PID: 200, Status: "ESTABLISHED", Type: 1, LocalIP: "10.0.0.2", LocalPort: 5555, RemoteIP: "1.2.3.4", RemotePort: 443},
			{PID: 300, Status: "LISTEN", Type: 1, LocalIP: "0.0.0.0", LocalPort: 22},
			// Loopback IPC between Electron helpers — must NOT be flagged as egress.
			{PID: 400, Status: "ESTABLISHED", Type: 1, LocalIP: "127.0.0.1", LocalPort: 60001, RemoteIP: "127.0.0.1", RemotePort: 60002},
		},
	}

	socks := Collect(context.Background(), snap, nil)

	var listen, egress *Socket
	for i := range socks {
		switch socks[i].PID {
		case 100:
			listen = &socks[i]
		case 200:
			egress = &socks[i]
		case 300:
			t.Error("non-AI sshd listener should not be classified")
		case 400:
			t.Error("loopback IPC should not be classified as egress")
		}
	}

	if listen == nil || listen.Category != "inference-api-local" || listen.Service != "ollama" || listen.LocalPort != 11434 {
		t.Errorf("ollama listener misclassified: %+v", listen)
	}
	if egress == nil || egress.Category != "ai-api-egress" {
		t.Errorf("aider egress misclassified: %+v", egress)
	}
}
