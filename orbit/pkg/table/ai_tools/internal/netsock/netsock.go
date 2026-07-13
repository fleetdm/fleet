// Package netsock turns the process/connection snapshot into classified
// agentic network sockets: locally-bound AI/MCP/inference listeners and
// outbound connections to AI/MCP endpoints. This is the runtime,
// config-independent detection vector — it catches servers and agents that are
// live right now even when nothing on disk declares them.
package netsock

import (
	"context"
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

type hostInfo struct {
	service  string
	category string
}

// Collect classifies the snapshot's connections. remoteMCPHosts maps a hostname
// parsed from a remote MCP config to a label; those hosts are resolved to IPs so
// established connections can be attributed even without per-connection reverse
// DNS.
func Collect(ctx context.Context, snap *proc.Snapshot, remoteMCPHosts map[string]string) []Socket {
	if snap == nil {
		return nil
	}
	hosts := buildHostSet(remoteMCPHosts)
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
			classifyEstablished(&sock, hosts)
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

func classifyEstablished(s *Socket, hosts map[string]hostInfo) {
	// Loopback "connections" are local IPC (e.g. Electron helper processes), not
	// network egress. Only surface them when the client is talking to a known
	// local inference port — i.e. an app using a local LLM server.
	if isLoopback(s.RemoteAddress) {
		if svc, ok := classify.LocalPortService(s.RemotePort); ok {
			s.Service, s.Category = svc, "inference-api-local"
		}
		return
	}
	// Process-classification first: any outbound connection owned by an AI/agent
	// process is AI traffic, no DNS required.
	if ok, cat := classify.Cmdline(s.Cmdline); ok {
		if cat == "mcp-server" {
			s.Category = "agent-runtime"
		} else {
			s.Category = "ai-api-egress"
		}
		return
	}
	// Otherwise attribute by destination if it resolves to a known AI/MCP host.
	if hi, ok := hosts[strings.ToLower(s.RemoteAddress)]; ok {
		s.Category, s.Service, s.RemoteHost = hi.category, hi.service, hi.service
	}
}

func isLoopback(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

// buildHostSet maps user-declared remote MCP hostnames to their labels.
//
// It deliberately performs NO name resolution. The hostnames come from MCP
// config files that any unprivileged user can write, and this collector runs in
// the root/SYSTEM daemon; resolving them would emit attacker-chosen outbound DNS
// on every query, turning a passive inventory table into a network beacon /
// DNS-exfil channel. Egress is attributed by the owning process instead
// (classifyEstablished), which is robust and DNS-free — so dropping resolution
// costs only IP-based attribution of a configured remote MCP host, a secondary
// path already dominated by process attribution.
func buildHostSet(remoteMCP map[string]string) map[string]hostInfo {
	set := map[string]hostInfo{}
	for host, label := range remoteMCP {
		host = strings.ToLower(strings.TrimSpace(host))
		if host == "" {
			continue
		}
		set[host] = hostInfo{label, "mcp-remote-egress"}
	}
	return set
}

func protoOf(sockType uint32) string {
	if sockType == 2 { // SOCK_DGRAM
		return "udp"
	}
	return "tcp"
}
