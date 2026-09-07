// Package proc takes a single cross-platform snapshot of running processes and
// network connections via gopsutil. One snapshot is shared across the
// mcp_server correlation, agents/apps liveness checks, and the sockets
// collector (all types of the unified ai_tools table) so the extension
// enumerates the process/connection tables only once per query.
package proc

import (
	"context"
	"strings"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// Process is a slimmed-down view of a running process. Only fields actually
// consumed downstream are collected — keeping per-process syscalls minimal.
type Process struct {
	PID      int
	Name     string
	Exe      string
	Cmdline  string
	Username string
}

// Conn is a single network connection (listening or established).
type Conn struct {
	PID        int
	Status     string // LISTEN, ESTABLISHED, ...
	Type       uint32 // SOCK_STREAM=1 (tcp), SOCK_DGRAM=2 (udp)
	LocalIP    string
	LocalPort  int
	RemoteIP   string
	RemotePort int
}

// Snapshot is an immutable point-in-time view of processes and connections.
type Snapshot struct {
	Procs map[int]Process
	Conns []Conn
}

// Take collects the snapshot. It never returns nil; on enumeration failure the
// corresponding slice/map is simply empty (detection degrades, never panics).
func Take(ctx context.Context) *Snapshot {
	s := &Snapshot{Procs: map[int]Process{}}

	if ps, err := process.ProcessesWithContext(ctx); err == nil {
		for _, p := range ps {
			if ctx.Err() != nil {
				break
			}
			pr := Process{PID: int(p.Pid)}
			pr.Name, _ = p.NameWithContext(ctx)
			pr.Exe, _ = p.ExeWithContext(ctx)
			pr.Cmdline, _ = p.CmdlineWithContext(ctx)
			pr.Username, _ = p.UsernameWithContext(ctx)
			s.Procs[pr.PID] = pr
		}
	}

	if conns, err := gnet.ConnectionsWithContext(ctx, "all"); err == nil {
		for _, c := range conns {
			s.Conns = append(s.Conns, Conn{
				PID:        int(c.Pid),
				Status:     c.Status,
				Type:       c.Type,
				LocalIP:    c.Laddr.IP,
				LocalPort:  int(c.Laddr.Port),
				RemoteIP:   c.Raddr.IP,
				RemotePort: int(c.Raddr.Port),
			})
		}
	}
	return s
}

// ListenPort returns the first listening port owned by pid, or 0.
func (s *Snapshot) ListenPort(pid int) int {
	for _, c := range s.Conns {
		if c.PID == pid && strings.EqualFold(c.Status, "LISTEN") {
			return c.LocalPort
		}
	}
	return 0
}
