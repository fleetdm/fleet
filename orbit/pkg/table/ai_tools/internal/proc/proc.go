// Package proc takes a single cross-platform snapshot of running processes and
// network connections via gopsutil. One snapshot is shared across the
// mcp_server correlation, agents/apps liveness checks, and the sockets
// collector (all types of the unified ai_tools table), and across the calls
// osquery makes for one query, so the extension enumerates the process table
// once.
//
// Connections and the costlier per-process fields are resolved on first use:
// on macOS enumerating connections shells out to lsof, and an inventory query
// that never asks for a port must not pay for it.
package proc

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

// Process is a slimmed-down view of a running process. Name and Cmdline are
// read for every process; read Exe, Username and CmdlineSlice through
// ExePath, User and Argv, which resolve them on first access for a process
// from Take. Tests may set those fields directly.
type Process struct {
	PID      int
	Name     string
	Exe      string
	Cmdline  string
	Username string

	// CmdlineSlice holds the true argv boundaries (one element per argument),
	// unlike Cmdline which gopsutil collapses into a single space-joined
	// string. A quoted argument containing its own spaces (e.g. the script
	// body of `node -e "<script>"`) is indistinguishable from several
	// separate arguments once joined — code that needs to identify argv[0],
	// argv[1], etc. (rather than just substring-match the whole command
	// line) must use this field, not re-split Cmdline on whitespace.
	CmdlineSlice []string

	lazy *lazyFields
}

// lazyFields is shared by every copy of a Process (Procs holds values), so a
// field resolved through one copy is resolved for all.
type lazyFields struct {
	reads *deferredReads
	p     *process.Process

	exeOnce, userOnce, argvOnce sync.Once
	exe, user                   string
	argv                        []string
}

// snapshotReadWindow is how long after Take a snapshot's deferred reads may
// run. The snapshot outlives the query that took it (osquery generates the
// table once per value of a `type IN (...)` list, and later calls reuse it for
// 10 seconds), so those reads can't use the query's cancellation. One deadline
// for all of them bounds the snapshot as a whole, however many reads hang; on
// macOS the connection listing shells out to lsof.
var snapshotReadWindow = time.Minute

// deferredReads is the context and deadline every deferred read of one
// snapshot shares.
type deferredReads struct {
	ctx      context.Context
	deadline time.Time
}

// start returns the context a deferred read runs under, or ok false once the
// deadline has passed, so a read that wouldn't honor the context isn't
// started at all.
func (r *deferredReads) start() (ctx context.Context, cancel context.CancelFunc, ok bool) {
	if !time.Now().Before(r.deadline) {
		return nil, nil, false
	}
	ctx, cancel = context.WithDeadline(r.ctx, r.deadline)
	return ctx, cancel, true
}

// ExePath returns the executable path.
func (p Process) ExePath() string {
	if p.lazy == nil {
		return p.Exe
	}
	p.lazy.exeOnce.Do(func() {
		ctx, cancel, ok := p.lazy.reads.start()
		if !ok {
			return
		}
		defer cancel()
		p.lazy.exe, _ = p.lazy.p.ExeWithContext(ctx)
	})
	return p.lazy.exe
}

// User returns the owning account name.
func (p Process) User() string {
	if p.lazy == nil {
		return p.Username
	}
	p.lazy.userOnce.Do(func() {
		ctx, cancel, ok := p.lazy.reads.start()
		if !ok {
			return
		}
		defer cancel()
		p.lazy.user, _ = p.lazy.p.UsernameWithContext(ctx)
	})
	return p.lazy.user
}

// Argv returns the true argv boundaries (see CmdlineSlice).
func (p Process) Argv() []string {
	if p.lazy == nil {
		return p.CmdlineSlice
	}
	p.lazy.argvOnce.Do(func() {
		ctx, cancel, ok := p.lazy.reads.start()
		if !ok {
			return
		}
		defer cancel()
		p.lazy.argv, _ = p.lazy.p.CmdlineSliceWithContext(ctx)
	})
	return p.lazy.argv
}

// MatchesBin reports whether the process is the given binary: its exact name,
// a path token in its command line, or its executable's base name. Never a
// name suffix, so a short binary like "q" doesn't match "icq".
func (p Process) MatchesBin(bin string) bool {
	return matchesBin(p.Name, p.ExePath, p.Cmdline, bin)
}

// matchesBin consults exe last because resolving it costs a syscall per
// process.
func matchesBin(name string, exe func() string, cmdline, bin string) bool {
	if bin == "" {
		return false
	}
	if strings.EqualFold(name, bin) || strings.EqualFold(name, bin+".exe") {
		return true
	}
	bin = strings.ToLower(bin)
	cmd := strings.ToLower(cmdline)
	// Path token: .../bin or .../bin <args> (also Windows backslash).
	for _, sep := range []string{"/", "\\"} {
		tok := sep + bin
		if strings.HasSuffix(cmd, tok) || strings.Contains(cmd, tok+" ") ||
			strings.HasSuffix(cmd, tok+".exe") || strings.Contains(cmd, tok+".exe ") {
			return true
		}
	}
	base := filepath.Base(exe())
	return strings.EqualFold(base, bin) || strings.EqualFold(base, bin+".exe")
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

// Snapshot is a point-in-time view of processes, with connections taken on
// first use.
type Snapshot struct {
	Procs map[int]Process
	conns func() []Conn // nil for a snapshot with no connections
}

// NewSnapshot returns a snapshot whose connections come from loadConns the
// first time they are needed.
func NewSnapshot(procs map[int]Process, loadConns func() []Conn) *Snapshot {
	s := &Snapshot{Procs: procs}
	if loadConns != nil {
		s.conns = sync.OnceValue(loadConns)
	}
	return s
}

// Take collects the process list. It never returns nil; on enumeration failure
// the map is simply empty (detection degrades, never panics). Its deferred
// reads carry ctx's values but not its cancellation, and all stop at
// snapshotReadWindow after Take.
func Take(ctx context.Context) *Snapshot {
	procs := map[int]Process{}
	reads := &deferredReads{ctx: context.WithoutCancel(ctx), deadline: time.Now().Add(snapshotReadWindow)}
	if ps, err := process.ProcessesWithContext(ctx); err == nil {
		for _, p := range ps {
			if ctx.Err() != nil {
				break
			}
			pr := Process{PID: int(p.Pid), lazy: &lazyFields{reads: reads, p: p}}
			pr.Name, _ = p.NameWithContext(ctx)
			pr.Cmdline, _ = p.CmdlineWithContext(ctx)
			procs[pr.PID] = pr
		}
	}
	return NewSnapshot(procs, func() []Conn {
		ctx, cancel, ok := reads.start()
		if !ok {
			return nil
		}
		defer cancel()
		return connections(ctx)
	})
}

func connections(ctx context.Context) []Conn {
	conns, err := gnet.ConnectionsWithContext(ctx, "all")
	if err != nil {
		return nil
	}
	out := make([]Conn, 0, len(conns))
	for _, c := range conns {
		out = append(out, Conn{
			PID:        int(c.Pid),
			Status:     c.Status,
			Type:       c.Type,
			LocalIP:    c.Laddr.IP,
			LocalPort:  int(c.Laddr.Port),
			RemoteIP:   c.Raddr.IP,
			RemotePort: int(c.Raddr.Port),
		})
	}
	return out
}

// Connections returns the host's network connections, enumerating them on the
// first call.
func (s *Snapshot) Connections() []Conn {
	if s.conns == nil {
		return nil
	}
	return s.conns()
}

// ListenPort returns the first listening port owned by pid, or 0.
func (s *Snapshot) ListenPort(pid int) int {
	for _, c := range s.Connections() {
		if c.PID == pid && strings.EqualFold(c.Status, "LISTEN") {
			return c.LocalPort
		}
	}
	return 0
}
