// Package netsock turns the process/connection snapshot into classified
// agentic network sockets: locally-bound AI/MCP/inference listeners and
// outbound connections to AI/MCP endpoints. This is the runtime,
// config-independent detection vector — it catches servers and agents that are
// live right now even when nothing on disk declares them.
package netsock

import (
	"net"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/classify"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

// Socket is one classified network endpoint owned by a process.
type Socket struct {
	PID                                         int
	ProcessName, ProcessPath, Cmdline, Username string
	Direction                                   string // listen | established
	Protocol                                    string // tcp | udp
	LocalAddress                                string
	LocalPort                                   int
	RemoteAddress                               string
	RemotePort                                  int
	RemoteHost                                  string
	Service                                     string
	Category                                    string // mcp-server-local | inference-api-local | mcp-remote-egress | ai-api-egress | agent-runtime
}

// Collect classifies the snapshot's connections into AI/MCP/inference sockets.
// Attribution is entirely by owning process (classifyEstablished/classifyListen)
// and known local ports; it performs NO name resolution, so it never emits
// outbound DNS from the root daemon for a hostname taken from an untrusted MCP
// config.
func Collect(snap *proc.Snapshot) []Socket {
	if snap == nil {
		return nil
	}
	var out []Socket

	for _, c := range snap.Conns {
		p := snap.Procs[c.PID]
		sock := Socket{
			PID:           c.PID,
			ProcessName:   p.Name,
			ProcessPath:   p.Exe,
			Cmdline:       p.Cmdline,
			Username:      p.Username,
			Protocol:      protoOf(c.Type),
			LocalAddress:  c.LocalIP,
			LocalPort:     c.LocalPort,
			RemoteAddress: c.RemoteIP,
			RemotePort:    c.RemotePort,
		}

		switch {
		case strings.EqualFold(c.Status, "LISTEN"):
			sock.Direction = "listen"
			classifyListen(&sock)
		case strings.EqualFold(c.Status, "ESTABLISHED") || (c.RemotePort != 0 && c.RemoteIP != ""):
			sock.Direction = "established"
			classifyEstablished(&sock)
		default:
			continue // ignore transient states (TIME_WAIT, CLOSE_WAIT, ...)
		}

		if sock.Category != "" {
			if sock.Service == "" {
				sock.Service = "unknown"
			}
			out = append(out, sock)
		}
	}
	return out
}

func classifyListen(s *Socket) {
	if svc, ok := classify.LocalPortService(s.LocalPort); ok {
		s.Service, s.Category = svc, "inference-api-local"
		return
	}
	if ok, cat := classify.Cmdline(s.Cmdline); ok {
		switch cat {
		case "mcp-server":
			s.Category = "mcp-server-local"
		case "inference-api-local":
			s.Category = "inference-api-local"
		default:
			s.Category = "agent-runtime"
		}
	}
}

func classifyEstablished(s *Socket) {
	// Loopback "connections" are local IPC (e.g. Electron helper processes), not
	// network egress. Only surface them when the client is talking to a known
	// local inference port — i.e. an app using a local LLM server.
	if isLoopback(s.RemoteAddress) {
		if svc, ok := classify.LocalPortService(s.RemotePort); ok {
			s.Service, s.Category = svc, "inference-api-local"
		}
		return
	}
	// Attribution is by owning process: any outbound connection owned by an
	// AI/agent process is AI traffic. This is DNS-free — we deliberately do not
	// resolve or match against hostnames from MCP configs (see Collect).
	if ok, cat := classify.Cmdline(s.Cmdline); ok {
		if cat == "mcp-server" {
			s.Category = "agent-runtime"
		} else {
			s.Category = "ai-api-egress"
		}
	}
}

func isLoopback(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

func protoOf(sockType uint32) string {
	if sockType == 2 { // SOCK_DGRAM
		return "udp"
	}
	return "tcp"
}
