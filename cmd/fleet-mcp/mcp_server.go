package main

import (
	"github.com/mark3labs/mcp-go/server"
)

const defaultEndpointsPerPage = 50

// fleetMCPInstructions is the server-level system prompt advertised to MCP
// clients (Claude Desktop, Cursor, etc.) via the `initialize` response. It
// mandates the schema-first workflow that prevents the most common class of
// silent-zero-row bug (assuming column types when writing osquery SQL) and
// directs the client to confirm write operations with the operator first.
//
// NOTE: these are advisory instructions to a cooperating LLM client, not a
// server-enforced control. A prompt-injected or non-cooperating client (or any
// caller driving the tools over raw JSON-RPC) can ignore them — the real bounds
// on writes are the FLEET_API_KEY's Fleet role (an observer token => Fleet 403s
// the writes) and agent-side osquery `--disable_tables`.
const fleetMCPInstructions = `Fleet MCP — host management and live osquery on managed devices.

CRITICAL WORKFLOW for any tool that takes a 'sql' argument (run_live_query):

1. BEFORE writing SQL, call get_osquery_schema(platform=<target>) to fetch the curated table list for that platform.
2. For any table you reference, verify column NAMES and TYPES against the schema response. If a needed table is not in the curated list, call get_osquery_schema(tables="table1,table2") for full canonical coverage.
3. Pay attention to column TYPE in the schema response. Many osquery columns are 'text' even when their values look numeric (e.g. windows_update_history.result_code is text with values like 'Succeeded' / 'Failed', NOT integer codes). Comparing a text column against an unquoted integer literal silently returns zero rows.
4. prepare_live_query already returns the schema for the inferred platform — use it as a single 'preview targets + schema' call, then pass the same filter args to run_live_query.

Schema freshness: the in-memory schema is refreshed periodically from https://raw.githubusercontent.com/fleetdm/fleet/main/schema/osquery_fleet_schema.json (the JSON behind https://fleetdm.com/tables). If you suspect a schema mismatch — e.g. fleet docs show a column the response is missing — call refresh_osquery_schema and try again.

Team (Fleet) scoping: when the user names a team in the conversation (e.g. "Workstations", "Servers"), pass it as the 'fleet' argument to run_live_query to restrict the targeted hosts to that team. Only omit 'fleet' when the user explicitly wants all teams.

CONFIRM BEFORE RUNNING run_live_query: show the operator the exact SQL and the resolved target scope (the host_ids / label / 'fleet', or "all hosts" if unscoped), then wait for explicit confirmation. Never auto-approve.

Skipping step 1 produces queries that parse and run but return wrong or empty results. Always verify before emitting SQL.`

// SetupMCPServer creates and configures the MCP server with all available tools.
// Tool registrations are split by domain across mcp_tools_*.go files.
func SetupMCPServer(config *Config, fleetClient *FleetClient) *server.MCPServer {
	s := server.NewMCPServer(
		"fleet-mcp", "1.0.0",
		server.WithLogging(),
		server.WithInstructions(fleetMCPInstructions),
	)

	// Kick off background refresh of the osquery schema from the canonical
	// fleetdm/fleet source. Reads the embedded snapshot synchronously at
	// init() so this is purely best-effort freshness.
	StartSchemaRefresh(0)

	registerHostTools(s, fleetClient)
	registerQueryTools(s, fleetClient)
	registerPolicyTools(s, fleetClient)
	registerInventoryTools(s, fleetClient)

	return s
}
