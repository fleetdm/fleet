// Package tables exposes a single, unified osquery table — ai_tools —
// covering every AI-tool type (MCP servers, IDE plugins, AI agent
// CLIs, AI desktop apps, live AI/MCP sockets, agent instruction files, and browser extensions)
// through one schema with a `type` discriminator, security `risk_flags` and
// `sha256` columns, and a JSON `detail` column for type-specific fields.
//
// Every row surfaced is AI-related by construction — collectors only emit
// AI/agent artifacts — so there is no `is_ai` column; presence in the table
// is the signal.
//
// It is optimized for a lightweight footprint:
//   - constraint pushdown: a query with `WHERE type = '...'` (or `type IN (...)`)
//     only runs the collectors it needs;
//   - one process/connection snapshot per query (shared across mcp/agents/apps/
//     sockets), skipped entirely when only ide_plugins is requested;
//   - one home-directory enumeration and one MCP-config scan, shared between the
//     mcp_server and sockets collectors.
package ai_tools

import (
	"context"
	"encoding/json"
	"maps"
	"net"
	"sort"
	"strconv"
	"strings"

	"github.com/osquery/osquery-go/plugin/table"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/agents"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/apps"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/browserext"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/ide"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/instructions"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/mcp"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/netsock"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/proc"
)

// allTypes is the set of values the `type` column can take.
var allTypes = []string{"mcp_server", "ide_plugins", "agents", "apps", "sockets", "agent_instruction", "browser_extension"}

// columns is the unified schema. Common fields are first-class; everything
// type-specific lives in `detail` (compact JSON, empty fields omitted).
var columns = []string{
	"type",       // mcp_server | ide_plugins | agents | apps | sockets | agent_instruction | browser_extension
	"name",       // server/plugin/agent/app/process/instruction-file name
	"identifier", // plugin_id | bundle_id | mcp server name | agent binary | socket service
	"category",   // classification bucket (coding-assistant, agent-runtime, ai-api-egress, ...)
	"location",   // local | remote
	"source",     // provenance: client | editor | install_method | platform_source | direction | tool
	"version",
	"path",     // config/install/binary/app/process/instruction-file path
	"endpoint", // remote MCP url or socket remote addr:port
	"running",  // 0/1
	"pid",
	"port",       // listening_port | api_port | local_port
	"risk_flags", // comma-separated security risk tokens ("" = none)
	"sha256",     // content hash of the primary artifact (diffable identity / threat-intel match)
	"uid",
	"username",
	"detail", // JSON: type-specific extras
}

// All returns the single table plugin exposed by the extension.
func All() []*table.Plugin {
	return []*table.Plugin{
		table.NewPlugin("ai_tools", columnDefs(), generate),
	}
}

func columnDefs() []table.ColumnDefinition {
	defs := make([]table.ColumnDefinition, 0, len(columns))
	for _, c := range columns {
		switch c {
		case "running", "pid", "port":
			defs = append(defs, table.IntegerColumn(c))
		default:
			defs = append(defs, table.TextColumn(c))
		}
	}
	return defs
}

func generate(ctx context.Context, qc table.QueryContext) ([]map[string]string, error) {
	types := requestedTypes(qc)
	has := func(t string) bool { _, ok := types[t]; return ok }

	hs := homes.All()

	needProc := has("mcp_server") || has("agents") || has("apps") || has("sockets")
	var snap *proc.Snapshot
	if needProc {
		snap = proc.Take(ctx)
	}

	// MCP config scan feeds only the mcp_server rows (the sockets collector no
	// longer consumes MCP hostnames — it attributes egress by owning process).
	var servers []mcp.Server
	if has("mcp_server") {
		for _, h := range hs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			servers = append(servers, mcp.ScanConfigs(h)...)
		}
	}

	rows := make([]map[string]string, 0, 128)

	if has("sockets") {
		for _, s := range netsock.Collect(snap) {
			rows = append(rows, socketRow(s))
		}
	}
	if has("mcp_server") {
		for _, s := range mcp.Correlate(servers, snap) {
			rows = append(rows, mcpRow(s))
		}
	}
	if has("ide_plugins") {
		for _, h := range hs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, p := range ide.Scan(h) {
				rows = append(rows, ideRow(p))
			}
		}
	}
	if has("agents") {
		for _, h := range hs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, a := range agents.Scan(h, snap) {
				rows = append(rows, agentRow(a))
			}
		}
	}
	if has("apps") {
		for _, a := range apps.Scan(hs, snap) {
			rows = append(rows, appRow(a))
		}
	}
	if has("agent_instruction") {
		for _, h := range hs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, in := range instructions.Scan(h) {
				rows = append(rows, instructionRow(in))
			}
		}
	}
	if has("browser_extension") {
		for _, h := range hs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			for _, e := range browserext.Scan(h) {
				rows = append(rows, browserExtRow(e))
			}
		}
	}
	return rows, nil
}

// requestedTypes reads `type` equality/IN constraints so we only run the
// collectors the query asks for. Any non-equality predicate (!=, LIKE) falls
// back to all types (safe superset).
func requestedTypes(qc table.QueryContext) map[string]struct{} {
	cl, ok := qc.Constraints["type"]
	if !ok {
		return allSet()
	}
	want := map[string]struct{}{}
	for _, c := range cl.Constraints {
		if c.Operator == table.OperatorEquals {
			want[c.Expression] = struct{}{}
		}
	}
	if len(want) == 0 {
		return allSet()
	}
	// Keep only valid types; if the filter excluded everything valid, the query
	// legitimately wants nothing.
	out := map[string]struct{}{}
	for _, k := range allTypes {
		if _, ok := want[k]; ok {
			out[k] = struct{}{}
		}
	}
	return out
}

func allSet() map[string]struct{} {
	m := make(map[string]struct{}, len(allTypes))
	for _, k := range allTypes {
		m[k] = struct{}{}
	}
	return m
}

// ---- row mappers ----

func mcpRow(s mcp.Server) map[string]string {
	return row(map[string]string{
		"type":       "mcp_server",
		"name":       s.ServerName,
		"identifier": s.ServerName,
		"category":   "mcp-server",
		"location":   s.Location,
		"source":     s.Client,
		"path":       s.ConfigPath,
		"endpoint":   s.URL,
		"running":    itoa(s.Running),
		"pid":        itoa(s.PID),
		"port":       itoa(s.ListeningPort),
		"risk_flags": s.RiskFlags,
		"sha256":     s.SHA256,
		"uid":        s.UID,
		"username":   s.Username,
	}, map[string]string{
		"transport":    s.Transport,
		"command":      s.Command,
		"args":         s.Args,
		"env_keys":     s.EnvKeys,
		"scope":        s.Scope,
		"source_type":  s.Source,
		"enabled":      itoa(s.Enabled),
		"capabilities": s.Capabilities,
		"launch_hash":  s.LaunchHash,
	})
}

func ideRow(p ide.Plugin) map[string]string {
	return row(map[string]string{
		"type":       "ide_plugins",
		"name":       p.Name,
		"identifier": p.PluginID,
		"category":   p.AICategory,
		"location":   "local",
		"source":     p.Editor,
		"version":    p.Version,
		"path":       p.InstallPath,
		"uid":        p.UID,
		"username":   p.Username,
	}, map[string]string{
		"editor_family": p.EditorFamily,
		"publisher":     p.Publisher,
		"manifest_path": p.ManifestPath,
	})
}

func agentRow(a agents.Agent) map[string]string {
	return row(map[string]string{
		"type":       "agents",
		"name":       a.Name,
		"identifier": a.Binary,
		"location":   "local",
		"source":     a.InstallMethod,
		"version":    a.Version,
		"path":       a.Path,
		"running":    itoa(a.Running),
		"pid":        itoa(a.PID),
		"risk_flags": a.RiskFlags,
		"sha256":     a.SHA256,
		"uid":        a.UID,
		"username":   a.Username,
	}, map[string]string{
		"runtime":         a.Runtime,
		"binary":          a.Binary,
		"binary_path":     a.BinaryPath,
		"permission_mode": a.PermissionMode,
	})
}

func appRow(a apps.App) map[string]string {
	return row(map[string]string{
		"type":       "apps",
		"name":       a.Name,
		"identifier": a.BundleID,
		"location":   "local",
		"source":     a.PlatformSource,
		"version":    a.Version,
		"path":       a.Path,
		"running":    itoa(a.Running),
		"pid":        itoa(a.PID),
		"port":       itoa(a.APIPort),
		"sha256":     a.SHA256,
	}, map[string]string{
		"vendor":           a.Vendor,
		"bundle_id":        a.BundleID,
		"scope":            a.Scope,
		"serves_local_api": itoa(a.ServesLocalAPI),
	})
}

func instructionRow(in instructions.Instruction) map[string]string {
	return row(map[string]string{
		"type":       "agent_instruction",
		"name":       in.Name,
		"identifier": in.Tool,
		"category":   "agent-instruction",
		"location":   "local",
		"source":     in.Tool,
		"path":       in.Path,
		"risk_flags": in.RiskFlags,
		"sha256":     in.SHA256,
		"uid":        in.UID,
		"username":   in.Username,
	}, map[string]string{
		"scope":   in.Scope,
		"size":    itoa(int(in.Size)),
		"markers": in.Markers,
	})
}

func browserExtRow(e browserext.Extension) map[string]string {
	return row(map[string]string{
		"type":       "browser_extension",
		"name":       e.Name,
		"identifier": e.ID,
		"category":   e.Category,
		"location":   "local",
		"source":     e.Browser,
		"version":    e.Version,
		"path":       e.Path,
		"risk_flags": e.RiskFlags,
		"sha256":     e.SHA256,
		"uid":        e.UID,
		"username":   e.Username,
	}, map[string]string{
		"browser":          e.Browser,
		"profile":          e.Profile,
		"engine":           e.Engine,
		"manifest_version": itoa(e.ManifestVer),
		"scope":            e.Scope,
		"host_perms":       strings.Join(e.HostPerms, ","),
		"from_webstore":    fromWebstoreStr(e.FromWebstore),
		"signed_state":     signedStateStr(e.SignedState),
	})
}

// fromWebstoreStr renders the tri-state webstore flag as a label so a "false"
// value survives compactJSON (which drops "" and "0").
func fromWebstoreStr(v int) string {
	switch v {
	case 1:
		return "true"
	case 0:
		return "false"
	default:
		return "" // unknown
	}
}

// signedStateStr renders a Gecko signedState int as a readable label (and ""
// for the unknown sentinel), avoiding compactJSON dropping the meaningful 0.
func signedStateStr(s int) string {
	switch s {
	case 2:
		return "privileged"
	case 1:
		return "signed"
	case 0:
		return "missing"
	case -1:
		return "unknown"
	case -2:
		return "broken"
	default:
		return ""
	}
}

func socketRow(s netsock.Socket) map[string]string {
	loc := "local"
	endpoint := ""
	if s.Direction == "established" {
		loc = "remote"
		if s.RemoteAddress != "" {
			endpoint = net.JoinHostPort(s.RemoteAddress, strconv.Itoa(s.RemotePort))
		}
	}
	return row(map[string]string{
		"type":       "sockets",
		"name":       s.ProcessName,
		"identifier": s.Service,
		"category":   s.Category,
		"location":   loc,
		"source":     s.Direction,
		"path":       s.ProcessPath,
		"endpoint":   endpoint,
		"running":    "1",
		"pid":        itoa(s.PID),
		"port":       itoa(s.LocalPort),
		"username":   s.Username,
	}, map[string]string{
		"protocol":      s.Protocol,
		"local_address": s.LocalAddress,
		"remote_host":   s.RemoteHost,
		"service":       s.Service,
		"cmdline":       s.Cmdline,
	})
}

// ---- helpers ----

// row fills a complete column map from a set of populated fields plus a
// detail map (empty detail entries are dropped before JSON-encoding).
func row(fields, detail map[string]string) map[string]string {
	m := make(map[string]string, len(columns))
	for _, c := range columns {
		m[c] = ""
	}
	maps.Copy(m, fields)
	m["detail"] = compactJSON(detail)
	return m
}

func compactJSON(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v != "" && v != "0" {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		out[k] = m[k]
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}

func itoa(i int) string { return strconv.Itoa(i) }
