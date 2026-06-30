# Fleet MCP Server 🚀

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for the [Fleet](https://fleetdm.com) endpoint security platform.

**Transform how you interact with your endpoint data. Query osquery, check compliance, drill into per-host policy results, and investigate CVEs natively from Claude, Cursor, and any MCP-compatible AI agent.**

🔗 **Learn about MCP:** [https://modelcontextprotocol.io/](https://modelcontextprotocol.io/)
🔗 **Learn about Fleet:** [https://fleetdm.com/](https://fleetdm.com/)

---

## 📺 See it in Action

Watch the 1 hr walkthrough demonstrating how to use Claude Desktop to instantly write and run live OSQueries across your fleet.

[![Fleet MCP Walkthrough Demo](https://img.youtube.com/vi/8K77litllPk/maxresdefault.jpg)](https://www.youtube.com/watch?v=8K77litllPk)

## Overview

This server provides an MCP interface to Fleet, enabling AI systems (Claude Desktop, Claude Code, Cursor, and any MCP-compatible client) to natively interact with your Fleet deployment. Instead of raw API endpoints, it exposes typed **Tools** that AI agents can call directly — listing hosts with rich server-side filters, drilling into per-host policy compliance, finding hosts impacted by a CVE, running live osquery, and more.

Both **SSE** (Server-Sent Events) and **stdio** transports are supported. The same 18-tool surface is exposed identically on both.

## Tools

The server exposes tools across three domains: **hosts**, **queries**, and **policies/vulnerabilities**. One of them (`run_live_query`) runs arbitrary osquery on devices, so scope the Fleet API token accordingly (see [Security model](#security-model)).

### Hosts

| Tool | Description |
|------|-------------|
| `get_endpoints` | List hosts/endpoints enrolled in Fleet with rich server-side filters (`fleet`, `platform`, `status`, `query`, `label`, `policy_id`, `policy_response`, `per_page`). All filters compose in a single Fleet API call — narrow precisely instead of paginating client-side. The `query` parameter alone covers hostname / serial / primary IP / hardware model / user inventory (username, email, IdP group). |
| `get_host` | Get full details for a single host including labels, fleet, hardware serial, primary IP, and platform info. Accepts a numeric `host_id` (most precise — bypasses any hostname collisions) OR an `identifier` (exact hostname / UUID / serial / computer_name, OR a fuzzy substring). When the identifier matches multiple hosts (e.g. shared hostname), returns a candidate list with each host's id / hostname / display_name / serial / primary_ip / fleet for disambiguation. |
| `get_host_policies` | Get the compliance status of every policy applied to a single host (global + fleet-inherited). Returns each policy with its `response` field (`pass` / `fail` / `""` for not-yet-run) plus a summary block (`failing_count`, `passing_count`, `not_run_count`, `total`). Mirrors the Fleet UI's per-host Policies tab. Accepts `host_id` (preferred) or `identifier`, with the same disambiguation behavior as `get_host`. Supports an optional `response` filter to narrow to passing or failing only. |
| `get_total_system_count` | Total count of active enrolled systems |
| `get_aggregate_platforms` | System count broken down by OS platform (macOS / Windows / Linux / etc.) |
| `get_fleets` | List all fleets with their IDs and names |
| `get_labels` | List all endpoint labels |

### Queries

| Tool | Description |
|------|-------------|
| `get_queries` | List all saved Fleet queries (global + per-fleet) |
| `prepare_live_query` | Step 1 of 2: validate targets and return the osquery schema needed to author a valid SQL statement |
| `run_live_query` | Step 2 of 2: execute an osquery SQL statement against live Fleet devices. **Schema-first contract**: callers must call `get_osquery_schema` (or `prepare_live_query`) first; SQL is pre-validated against canonical column types — TEXT-vs-bare-integer comparisons are rejected. Targets resolve server-side via direct selectors like `hostnames` / `host_ids` and intersecting filters including `fleet`, `platform`, `label`, `status`, `query`, `policy_id`, `policy_response`, and `cve_id`. **Fleet-scoped**: when `fleet` is set, only hosts in that fleet are targeted. |
| `get_osquery_schema` | Returns the canonical, source-of-truth schema for Fleet/osquery tables. Sourced from the Fleet monorepo `schema/osquery_fleet_schema.json` (also rendered at <https://fleetdm.com/tables>) and refreshed in the background — column TYPES are always accurate. Defaults to a curated short list filtered by `platform`; pass `tables` (comma-separated) for full canonical coverage of any of the 360+ tables. |
| `refresh_osquery_schema` | Force-refresh the in-memory schema from <https://raw.githubusercontent.com/fleetdm/fleet/main/schema/osquery_fleet_schema.json>. Use when `get_osquery_schema` returns data that conflicts with the live Fleet docs. Background refresh handles routine drift; this tool is for the rare manual override. |
| `get_vetted_queries` | Get a library of 100% vetted, production-safe CIS-8.1 policy queries for macOS, Windows, and Linux |

### Policies & Vulnerabilities

| Tool | Description |
|------|-------------|
| `get_policies` | List all policies (global + per-fleet) with their pass/fail host counts |
| `get_policy_compliance` | Get pass/fail counts for a specific policy. Defaults to global aggregate; pass `fleet` to scope to a single fleet (matches the per-fleet counts in the Fleet UI). |
| `get_policy_hosts` | List the hosts that pass or fail a given policy, optionally narrowed by `fleet`, `platform`, `label`, `status`, `query`. Use this to answer "which Linux hosts are failing policy 42?" — all filter dimensions compose server-side. |
| `get_vulnerability_impact` | Aggregate count of systems impacted by a CVE |
| `get_vulnerability_hosts` | List the specific hosts impacted by a CVE, optionally narrowed by `fleet`, `platform`, `label`, `status`, `query`. Composes a 3-step lookup (`/software/titles?vulnerable=true&query=CVE` → vulnerable version IDs → `/hosts?software_version_id=N`) and intersects client-side. Required because Fleet's `/hosts?cve=` and `/hosts?platform=` filters are silently ignored — see the Operational learnings section. |

### Filter dimensions at a glance

| Dimension | How to filter | Notes |
|---|---|---|
| **User / IP / hostname** | `query=` substring | One field — Fleet's substring matcher covers all of these plus serial / model. |
| **Display name** | not searchable via `query` | Use `host_id` directly (from a candidate list or prior call). The `/hosts/identifier/:id` endpoint also matches `computer_name` exactly, which often equals the display name. |
| **Fleet** | `fleet=<name>` | Resolved server-side to `fleet_id`. |
| **Label** | `label=<name>` | Resolved server-side to `label_id`. Single label only — Fleet's API doesn't accept multi-label intersection. |
| **Policy result** | `policy_id=<id>` + `policy_response=passing\|failing` | `policy_response` requires `policy_id`; otherwise rejected at the MCP layer. |
| **Platform / Status** | `platform=` / `status=` | Standard Fleet host filters. |

### Hostname collisions and host_id

Fleet allows multiple hosts to share a `hostname` (e.g. several Macs all reporting `hostname=mac`). Fleet's `/hosts/identifier/:id` endpoint silently returns one of them, which used to mean policy lookups could quietly target the wrong host. The current tools handle this with a **query-first** resolver:

1. If you pass `host_id` (numeric), it goes straight to `/hosts/:host_id` — exact, no collision possible.
2. Otherwise the tool does a substring search first. One match → fetch by ID. Multiple matches → return a candidate list with each host's `id`, `hostname`, `display_name`, `hardware_serial`, `primary_ip`, and `fleet_name`. Zero matches → fall back to `/hosts/identifier/:id` (catches UUIDs and `computer_name`-only matches).

If your AI agent gets a candidate list back, it should pick the right `id` and re-call with `host_id`. Display-name-only hosts (e.g. one named `USS Protostar` whose hostname is `mac`) are best fetched with `host_id` from the start.

## Configuration

Configure the server using environment variables or a `.env` file (in the same directory as the binary).

| Variable | Default | Description |
|----------|---------|-------------|
| `FLEET_BASE_URL` | *(required)* | Base URL of your Fleet instance, e.g. `https://dogfood.fleetdm.com` |
| `FLEET_API_KEY` | *(required)* | Fleet API token — see [Fleet docs](https://fleetdm.com/docs/using-fleet/rest-api#authentication). **Use the least-privileged Fleet role that covers your tools:** an **observer** / **observer-plus** token is enough for all tools - **admin is not required** |
| `MCP_AUTH_TOKEN` | *(required)* | Bearer token for authenticating MCP clients. Generate with `openssl rand -hex 32` (**min 32 chars — the server refuses a weaker one**). **Required on every transport (including stdio); the server refuses to start without it.** In SSE mode the server validates it on every request; in stdio mode it must be set but is not checked at runtime (the client launches the binary as a local subprocess). |
| `PORT` | `8080` | HTTP port for SSE transport. Ignored in stdio mode. Render injects this automatically. |
| `LOG_LEVEL` | `info` | Log verbosity: `debug` / `info` / `warn` / `error`. Note: `debug` logs the route shape of every Fleet API call (path before query string only — no PII identifiers). Avoid `debug` in production deployments where logs are shipped to a centralized aggregator. |
| `FLEET_TLS_SKIP_VERIFY` | `false` | Skip TLS certificate verification. **Hard-gated to localhost — the server refuses to start with this set and a non-loopback `FLEET_BASE_URL`.** Conflicts with `FLEET_CA_FILE`. |
| `FLEET_CA_FILE` | *(optional)* | Path to a PEM CA certificate for self-signed Fleet instances |
| `FLEET_LIVE_QUERY_REST_PERIOD` | `25s` | How long `run_live_query` waits for hosts to report before returning. Accepts any Go duration string (e.g. `25s`, `1m`). Multi-host runs stop early once every online host has responded; this is the upper bound for the wait. Mirrors the same variable on the Fleet server — keep this ≥ the server's value so the MCP doesn't give up before the server finishes the campaign. |

Copy the provided template:

```bash
cp .env.example .env
# Edit .env with your Fleet URL, Fleet API key, and a freshly generated MCP_AUTH_TOKEN
```

> **Note for Claude Desktop (stdio):** Claude Desktop reads environment variables from the `env` block of `claude_desktop_config.json`, **not** from a `.env` file. See the [Stdio Transport](#stdio-transport-claude-desktop) section.

## Installation

### Prerequisites

- Go 1.25.7+
- A running [Fleet](https://fleetdm.com) instance
- A Fleet API token with appropriate read permissions

### Build

```bash
git clone https://github.com/fleetdm/fleet
cd fleet/tools/fleet-mcp
go mod tidy
go build -o fleet-mcp .
```

### Generate an MCP auth token

```bash
openssl rand -hex 32
```

Use the output as `MCP_AUTH_TOKEN` in your `.env` (SSE) or your Claude Desktop config (stdio).

## Usage

### SSE Transport (Claude Code, Cursor, web clients)

Start the server — it will listen for SSE connections:

```bash
./fleet-mcp
# transport: SSE — listening on :8080
```

Configure your MCP client to connect to `http://localhost:8080/sse` and include the bearer token. For **Claude Code**, add to your project's `.mcp.json` or your global MCP config:

```json
{
  "mcpServers": {
    "fleet": {
      "type": "sse",
      "url": "http://localhost:8080/sse",
      "headers": {
        "Authorization": "Bearer <your-MCP_AUTH_TOKEN>"
      }
    }
  }
}
```

For a remote deployment (e.g. Render):

```json
{
  "mcpServers": {
    "fleet": {
      "type": "sse",
      "url": "https://your-fleet-mcp.onrender.com/sse",
      "headers": {
        "Authorization": "Bearer <your-MCP_AUTH_TOKEN>"
      }
    }
  }
}
```

### Stdio Transport (Claude Desktop)

Stdio mode runs the binary directly as a subprocess — no network port, no TLS to worry about, all communication over stdin/stdout JSON-RPC.

1. **Build the binary:**

   ```bash
   go build -o fleet-mcp .
   ```

2. **(macOS only) Adhoc-sign the binary** so Gatekeeper doesn't kill it after replacement. **Required after every rebuild on Apple Silicon** — without this, copying a freshly built binary over an existing one at the same path can result in silent crashes or `exit 137`:

   ```bash
   codesign --force --sign - ./fleet-mcp
   ```

3. **Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS):**

   ```json
   {
     "mcpServers": {
       "fleet-mcp": {
         "command": "/absolute/path/to/fleet-mcp",
         "args": ["-transport", "stdio"],
         "env": {
           "FLEET_BASE_URL": "https://your-fleet.example.com",
           "FLEET_API_KEY": "YOUR_FLEET_API_KEY",
           "MCP_AUTH_TOKEN": "YOUR_MCP_AUTH_TOKEN",
           "LOG_LEVEL": "info"
         }
       }
     }
   }
   ```

   Use an **absolute path** for `command`. Relative paths and `~` are not expanded.

4. **Fully quit and relaunch Claude Desktop** (`Cmd+Q`, not just close-window). All the tools will appear in your context.

### Smoke-test stdio mode without Claude Desktop

You can drive the binary directly via stdio JSON-RPC for debugging:

```bash
export FLEET_BASE_URL="https://your-fleet.example.com"
export FLEET_API_KEY="..."
export MCP_AUTH_TOKEN="..."

cat <<'EOF' | ./fleet-mcp -transport stdio
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"1.0"}}}
{"jsonrpc":"2.0","method":"notifications/initialized"}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_total_system_count","arguments":{}}}
EOF
```

Replace the `tools/call` line to exercise any tool, e.g. `get_host_policies` with `host_id`.

### `-seed` flag

The binary also supports a one-shot seed mode that loads Fleet with the standard set of saved queries shipped with this repo, then exits:

```bash
./fleet-mcp -seed
```

This is a developer convenience — skip it for normal MCP server use.

## Tool annotations and Claude Desktop

Every tool here ships with explicit MCP annotations:

- `readOnlyHint` — does the tool only read, or can it write?
- `destructiveHint` — can it mutate or remove data?
- `idempotentHint` — does repeating the call have the same effect?
- `openWorldHint` — does it talk to a remote system, or only consult in-binary data?

Without these, Claude Desktop conservatively gates every tool behind destructive-action review and may collapse the surface to a single tool. The 17 read-only tools are annotated `readOnly=true, destructive=false, idempotent=true` so the AI agent can use them freely. **Note:** these annotations are advisory hints honored by well-behaved clients (e.g. Claude Desktop prompts before the one destructive tool) — they are not a server-side control. The real control is the privilege of the `FLEET_API_KEY` role (use a least-privilege/observer token; see [Security model](#security-model)).

**One tool is explicitly destructive and requires user approval in MCP clients:**

| Tool | Annotations | Why destructive |
|---|---|---|
| `run_live_query` | `readOnly=false, destructive=true, idempotent=false` | Fires osquery against every targeted device via an ad-hoc campaign, consumes device CPU, and shows up in EDR telemetry. Even SELECT-only SQL is operationally destructive at fleet scale. |

This means Claude Desktop will surface a confirmation prompt before it fires — required for any production deployment where the operator's Fleet API token can run live queries (observer-plus or higher).

## Security model

The MCP holds the operator's `FLEET_API_KEY` and acts with exactly that token's Fleet role — with a role that can run live queries (observer-plus or higher), compromise of the MCP gives an attacker full host inventory access plus arbitrary osquery against every enrolled device. Defenses:

- **Terminate TLS in front (the SSE listener is plain HTTP).** The server speaks HTTP on `PORT`, so the `MCP_AUTH_TOKEN` bearer and all traffic are **cleartext on the wire**. For any non-loopback deployment, run it behind a TLS-terminating layer: **Render's edge already does this**; for a self-hosted / private-network deployment, put a reverse proxy (nginx / Caddy / Cloudflare) in front and only expose the HTTPS endpoint. The `stdio` transport is unaffected (no network hop).
- **TLS skip-verify hard-gated to localhost.** `FLEET_TLS_SKIP_VERIFY=true` paired with a non-loopback `FLEET_BASE_URL` makes the binary refuse to start (`logrus.Fatalf`) — copying a dev `.env` to a remote deploy can no longer expose the Fleet token to an on-path attacker.
- **Strong `MCP_AUTH_TOKEN` enforced.** The server refuses to start if `MCP_AUTH_TOKEN` is shorter than 32 characters — a high-entropy token is the real defense against bearer brute force (`openssl rand -hex 32` satisfies it).
- **Rate limiting / DoS protection belongs upstream.** The MCP runs no limiter of its own — Fleet typically runs behind a reverse proxy / edge that terminates TLS (above) and throttles requests, which is where per-client rate limiting belongs (Render's edge; or a reverse proxy / WAF such as nginx / Caddy / Cloudflare). The strong `MCP_AUTH_TOKEN` is the brute-force defense, and failed-auth requests are rejected by the MCP before they ever reach Fleet.
- **API-only token required.** At startup the MCP calls `GET /api/v1/fleet/me` and **refuses to start unless `FLEET_API_KEY` belongs to an API-only Fleet user**. An API-only user has no UI session, its own audit identity, and — via Fleet's per-user role/team scoping — can be locked down to exactly the endpoints (and teams/fleets) the MCP needs, adding a Fleet-side authorization boundary on top of the MCP's bearer auth. Fails closed: if `/me` can't confirm the principal (Fleet unreachable or token invalid) the MCP won't start. To create one with the right access, see Fleet's docs: [Create an API-only user](https://fleetdm.com/docs/rest-api/rest-api#create-api-only-user), [the endpoints an API-only user can reach](https://fleetdm.com/docs/rest-api/rest-api#list-api-endpoints-for-api-only-user-permissions), and [Using `fleetctl` with an API-only user](https://fleetdm.com/guides/fleetctl#using-fleetctl-with-an-api-only-user).
- **Read vs run = the token's Fleet role.** The MCP acts with exactly the role of `FLEET_API_KEY` (it has no mode of its own), and Fleet enforces it. The read-only tools work with an **observer** token; the one mutating tool, `run_live_query`, needs **observer_plus** (or admin / maintainer / technician) per Fleet's RBAC — an observer token gets a `403`. **No maintainer or admin role is required.** Recommended: an API-only user with the **lowest role** your tools need — **observer** for a read-only deployment, **observer_plus** if you use `run_live_query`. The MCP doesn't infer or log these rights — it only enforces the API-only requirement above.
- **Live queries run arbitrary osquery (scope the token).** `run_live_query` dispatches arbitrary osquery to devices — exactly like Fleet's own UI / `fleetctl` / live-query REST API, which the MCP proxies. Tables such as `curl` / `curl_certificate` / `carves` can make outbound requests or exfiltrate file content from a managed device (e.g. cloud-metadata credentials). **This is a Fleet/osquery capability, not specific to the MCP** — the MCP does not (and should not) special-case it. The MCP-layer control is a **least-privilege `FLEET_API_KEY`** (an observer token → Fleet rejects live queries entirely). To remove the capability everywhere (UI, `fleetctl`, scheduled, MCP), disable the tables on the osquery agent: `--disable_tables=curl,curl_certificate,carves,yara,yara_events` via fleetd/orbit agent options.
- **Body size cap on SSE transport.** `http.MaxBytesReader` caps every incoming request body at 1 MiB. Hostile clients cannot OOM the MCP via oversized JSON-RPC payloads.
- **HTTP server timeouts.** `ReadHeaderTimeout=10s`, `ReadTimeout=30s`, `IdleTimeout=120s` defeat Slowloris-style header/body starvation.
- **Saved-query sweeper at startup.** Any `fleet-mcp-temp-*` saved queries left over from previous runs (whose deferred DELETE failed during a crash or 5xx) are deleted on the next MCP boot. Temp query names use `crypto/rand` suffixes so concurrent invocations cannot collide.
- **PII-safe debug logs.** The Fleet API call log was changed to log only the route shape (path before any `?` query string) — host serials, user emails passed via `?query=`, and CVE IDs no longer leak to debug logs.
- **CVE / policy / per_page input validation at the MCP layer.** `cve_id` must match `^CVE-\d{4}-\d{4,}$`, `policy_id` must be a positive integer, `per_page` is clamped to 200. Malformed inputs get a usable error message before any Fleet API call.
- **Context propagation end-to-end.** Every FleetClient method takes `ctx context.Context`; MCP handler cancellation propagates through to in-flight Fleet API calls, including between iterations of fan-out paths (CVE compose, label intersection). A cancelled MCP request stops the whole fan-out instead of running every remaining HTTP call to completion.

## Deploying to Render

`tools/fleet-mcp/render.yaml` is a standalone Render Blueprint, separate from the root `render.yaml` used by the main Fleet service.

1. Push `tools/fleet-mcp/render.yaml` to your repo.
2. In the Render dashboard go to **New → Blueprint**.
3. Connect your repo and set the **Blueprint file path** to `tools/fleet-mcp/render.yaml`.
4. During setup, fill in the following environment variables:
   - `FLEET_BASE_URL` — URL of your Fleet instance
   - `FLEET_API_KEY` — Fleet API token
   - `MCP_AUTH_TOKEN` — generate with `openssl rand -hex 32`
5. `PORT` is injected automatically by Render — no action needed.
6. Health check: point Render's (or any orchestrator's) probe at **`GET /healthz`** — it returns `200 ok` unauthenticated, so it stays green under load.


## Development

### Project layout

```
tools/fleet-mcp/
  main.go                  # entrypoint, flag parsing, transport selection, http.Server timeouts, body-size cap
  config.go                # env-var loading
  auth.go                  # bearer-auth middleware (SSE)
  route_guard.go           # SSE route allow-list
  fleet_integration.go     # FleetClient — wraps Fleet REST API. Every method takes ctx context.Context as first param.
  mcp_server.go            # SetupMCPServer orchestrator
  mcp_helpers.go           # getOptionalString, parseCSVArg, parsePerPageArg, validateCVEID, parsePositiveUintString, jsonResult
  mcp_tools_hosts.go       # host-domain MCP tools (7 tools)
  mcp_tools_queries.go     # query-domain MCP tools (6 tools)
  mcp_tools_policies.go    # policy/vuln MCP tools (5 tools)
  schema.go                # canonical osquery schema (embedded fallback + live HTTP refresh from raw.githubusercontent.com/fleetdm/fleet/main/schema/osquery_fleet_schema.json) and ValidateSQLForPlatforms (table-vs-platform + TEXT-column type sniff)
  osquery_fleet_schema.json # vendored canonical snapshot (//go:embed source-of-truth fallback). Refresh via `go generate ./tools/fleet-mcp/...`.
  vetted_queries.go        # vetted CIS-8.1 query library
  seed_fleet.go            # -seed mode
```

Tunables (env vars) for the schema layer:

- `FLEET_MCP_SCHEMA_REFRESH_INTERVAL` — refresh cadence for the background goroutine, accepts any `time.Duration` string (e.g. `6h`, `30m`). Default `24h`.
- `FLEET_MCP_SCHEMA_REFRESH_DISABLE` — when set, the live refresh goroutine is not started and the binary uses the embedded snapshot only. Useful for air-gapped environments.

### Adding a new tool

1. Add a method to `FleetClient` in `fleet_integration.go` that wraps the Fleet API call.
2. Pick the right domain file (`mcp_tools_hosts.go`, `mcp_tools_queries.go`, or `mcp_tools_policies.go`) and add a `register<ToolName>` function.
3. Wire the new register function into the matching `register<Domain>Tools` orchestrator at the top of the same file.
4. Always set `readOnly` / `destructive` / `idempotent` annotations so Claude Desktop can advertise it.
5. Build and run the smoke test from the [Smoke-test stdio mode](#smoke-test-stdio-mode-without-claude-desktop) section.

### Operational learnings

A few non-obvious behaviors discovered while building this:

- **`?query=` substring matching covers hostname, serial, primary IP, hardware model, AND host_users (username/email/IdP groups)** — but **not** display_name. Use `host_id` for display-name-only lookups.
- **`/hosts/identifier/:id` matches more than the docs claim:** in addition to hostname / UUID / serial, it also matches `computer_name` exactly. That's why an identifier like `"USS Protostar"` resolves even though `?query=` doesn't match it.
- **Hostname collisions are real** in any sizeable fleet. Always prefer `host_id` when you have it. The substring resolver returns up to 50 candidates with `display_name` / `serial` / `primary_ip` for disambiguation.
- **`policy_response` requires `policy_id`** at the API level. The MCP layer rejects the orphan combination upfront with a clean error rather than letting Fleet return a vague 400.
- **Fleet's `/hosts` endpoint silently ignores several filter params we tested.** As of Fleet 4.85, passing `cve=CVE-X`, `platform=linux`, or `label_id=N` to `GET /hosts` is accepted without error but returns the unfiltered host list — the MCP cannot rely on these. Workarounds shipped in this repo:
  - **Platform / label scoping** routes through `GET /labels/:id/hosts` (which DOES honor `fleet_id` and `query`, but ALSO ignores `software_version_id` and `policy_id`, so policy + label intersection is computed client-side by host ID).
  - **CVE → hosts** is a 3-step compose in `GetHostsForCVE`: `GET /software/titles?vulnerable=true&query=CVE-X` → per-title `GET /software/titles/:id` to harvest vulnerable version IDs → `GET /hosts?software_version_id=N` per ID → intersect with fleet / status / query / label-id client-side.
  - The single-call `GET /hosts?cve=` path is deliberately NOT used because it returns wrong results (e.g. CVE-2026-31431 yields 50 hosts via `?cve=`, but the correct answer is 1).
  - Future Fleet versions may fix these — revisit `GetEndpointsWithFilters` and `GetHostsForCVE` if/when that happens.
- **Fleet-scoped policy compliance** uses `/fleets/:fleet_id/policies/:policy_id`, not the global path. `get_policy_compliance` routes to whichever based on whether `fleet` is set.
- **`/api/v1/fleet/host_summary` is the right endpoint for aggregate platform counts** — `GET /hosts` defaults to a 100-host page, so any client-side aggregation over `GetEndpoints(0)` is silently wrong on Fleets larger than 100 hosts. `get_aggregate_platforms` uses `host_summary` directly so totals match the Fleet UI at any inventory size.
- **`fetchHostsFromPath` paginates internally** with a hard cap (`fetchHostsHardCap = 10000`). Without this, a single call could buffer the full host inventory in memory (~2KB per Endpoint × 50k hosts ≈ 100MB) and OOM the MCP. When the cap fires a warning is logged so operators see truncation rather than silently getting a partial host set.
- **Per-fleet fan-out (`get_queries`, `get_policies`) is bounded-concurrent.** 8 in-flight goroutines, order-stable merge by fleet index. On enterprise Fleet instances with 50+ fleets the sequential path was the dominant latency source; the bounded concurrency amortizes round-trip count without flooding Fleet with thousands of simultaneous requests.
- **CSV args drop empty segments.** `parseCSVArg("foo,,bar")` returns `["foo", "bar"]` — the legacy split-and-trim behavior leaked zero-value strings into filter logic, and a leading empty segment could silently disable filters that read `parts[0]`.
- **macOS Gatekeeper caches adhoc signatures** keyed to file identity. Replacing the binary at the same path silently invalidates the cached approval — re-run `codesign --force --sign -` after every rebuild before Claude Desktop will launch it.
- **Claude Desktop reads `env` from the JSON config**, not from a `.env` file. The `.env` template in this repo is for SSE/local development only.

## License

MIT
