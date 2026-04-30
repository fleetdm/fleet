# Fleet MCP Server 🚀

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for the [Fleet](https://fleetdm.com) endpoint security platform.

**Transform how you interact with your endpoint data. Query OSQuery, check compliance, drill into per-host policy results, and investigate CVEs natively from Claude, Cursor, and any MCP-compatible AI agent.**

🔗 **GitHub Repo:** [https://github.com/karmine05/fleet-mcp](https://github.com/karmine05/fleet-mcp)
🔗 **Learn about MCP:** [https://modelcontextprotocol.io/](https://modelcontextprotocol.io/)
🔗 **Learn about Fleet:** [https://fleetdm.com/](https://fleetdm.com/)

---

## 📺 See it in Action

Watch the 1 hr walkthrough demonstrating how to use Claude Desktop to instantly write and run live OSQueries across your fleet.

[![Fleet MCP Walkthrough Demo](https://img.youtube.com/vi/8K77litllPk/maxresdefault.jpg)](https://www.youtube.com/watch?v=8K77litllPk)

## Overview

This server provides an MCP interface to Fleet, enabling AI systems (Claude Desktop, Claude Code, Cursor, and any MCP-compatible client) to natively interact with your Fleet deployment. Instead of raw API endpoints, it exposes typed **Tools** that AI agents can call directly — listing hosts with rich server-side filters, drilling into per-host policy compliance, finding hosts impacted by a CVE, running live OSQuery, and more.

Both **SSE** (Server-Sent Events) and **stdio** transports are supported. The same 18-tool surface is exposed identically on both.

## Tools

The server exposes 18 tools across three domains: **hosts**, **queries**, and **policies/vulnerabilities**.

### Hosts

| Tool | Description |
|------|-------------|
| `get_endpoints` | List hosts/endpoints enrolled in Fleet with rich server-side filters (`fleet`, `platform`, `status`, `query`, `label`, `policy_id`, `policy_response`, `per_page`). All filters compose in a single Fleet API call — narrow precisely instead of paginating client-side. The `query` parameter alone covers hostname / serial / primary IP / hardware model / user inventory (username, email, IdP group). |
| `get_host` | Get full details for a single host including labels, fleet, hardware serial, primary IP, and platform info. Accepts a numeric `host_id` (most precise — bypasses any hostname collisions) OR an `identifier` (exact hostname / UUID / serial / computer_name, OR a fuzzy substring). When the identifier matches multiple hosts (e.g. shared hostname), returns a candidate list with each host's id / hostname / display_name / serial / primary_ip / team for disambiguation. |
| `get_host_policies` | Get the compliance status of every policy applied to a single host (global + fleet-inherited). Returns each policy with its `response` field (`pass` / `fail` / `""` for not-yet-run) plus a summary block (`failing_count`, `passing_count`, `not_run_count`, `total`). Mirrors the Fleet UI's per-host Policies tab. Accepts `host_id` (preferred) or `identifier`, with the same disambiguation behavior as `get_host`. Supports an optional `response` filter to narrow to passing or failing only. |
| `get_total_system_count` | Total count of active enrolled systems |
| `get_aggregate_platforms` | System count broken down by OS platform (macOS / Windows / Linux / etc.) |
| `get_fleets` | List all fleets (teams) with their IDs and names |
| `get_labels` | List all endpoint labels |

### Queries

| Tool | Description |
|------|-------------|
| `get_queries` | List all saved Fleet queries (global + per-team) |
| `prepare_live_query` | Step 1 of 2: validate targets and return the OSQuery schema needed to author a valid SQL statement |
| `run_live_query` | Step 2 of 2: execute an OSQuery SQL statement against live Fleet devices. Targets resolved server-side via any combination of `hostnames`, `labels`, `platforms`, `fleets`. |
| `create_saved_query` | Create a new saved query in Fleet (with platform-aware SQL pre-validation) |
| `get_osquery_schema` | Get the hardcoded, accurate schema for the most important Fleet/Osquery tables, optionally filtered by platform |
| `get_vetted_queries` | Get a library of 100% vetted, production-safe CIS-8.1 policy queries for macOS, Windows, and Linux |

### Policies & Vulnerabilities

| Tool | Description |
|------|-------------|
| `get_policies` | List all policies (global + per-team) with their pass/fail host counts |
| `get_policy_compliance` | Get pass/fail counts for a specific policy. Defaults to global aggregate; pass `fleet` to scope to a single team (matches the per-fleet counts in the Fleet UI). |
| `get_policy_hosts` | List the hosts that pass or fail a given policy, optionally narrowed by `fleet`, `platform`, `label`, `status`, `query`. Use this to answer "which Linux hosts are failing policy 42?" — all filter dimensions compose server-side. |
| `get_vulnerability_impact` | Aggregate count of systems impacted by a CVE |
| `get_vulnerability_hosts` | List the specific hosts impacted by a CVE, optionally narrowed by `fleet`, `platform`, `label`, `status`, `query`. Composes a 3-step lookup (`/software/titles?vulnerable=true&query=CVE` → vulnerable version IDs → `/hosts?software_version_id=N`) and intersects client-side. Required because Fleet's `/hosts?cve=` and `/hosts?platform=` filters are silently ignored — see the Operational learnings section. |

### Filter dimensions at a glance

| Dimension | How to filter | Notes |
|---|---|---|
| **User / IP / hostname** | `query=` substring | One field — Fleet's substring matcher covers all of these plus serial / model. |
| **Display name** | not searchable via `query` | Use `host_id` directly (from a candidate list or prior call). The `/hosts/identifier/:id` endpoint also matches `computer_name` exactly, which often equals the display name. |
| **Team** | `fleet=<name>` | Resolved server-side to `team_id`. |
| **Label** | `label=<name>` | Resolved server-side to `label_id`. Single label only — Fleet's API doesn't accept multi-label intersection. |
| **Policy result** | `policy_id=<id>` + `policy_response=passing\|failing` | `policy_response` requires `policy_id`; otherwise rejected at the MCP layer. |
| **Platform / Status** | `platform=` / `status=` | Standard Fleet host filters. |

### Hostname collisions and host_id

Fleet allows multiple hosts to share a `hostname` (e.g. several Macs all reporting `hostname=mac`). Fleet's `/hosts/identifier/:id` endpoint silently returns one of them, which used to mean policy lookups could quietly target the wrong host. The current tools handle this with a **query-first** resolver:

1. If you pass `host_id` (numeric), it goes straight to `/hosts/:host_id` — exact, no collision possible.
2. Otherwise the tool does a substring search first. One match → fetch by ID. Multiple matches → return a candidate list with each host's `id`, `hostname`, `display_name`, `hardware_serial`, `primary_ip`, and `team_name`. Zero matches → fall back to `/hosts/identifier/:id` (catches UUIDs and `computer_name`-only matches).

If your AI agent gets a candidate list back, it should pick the right `id` and re-call with `host_id`. Display-name-only hosts (e.g. one named `USS Protostar` whose hostname is `mac`) are best fetched with `host_id` from the start.

## Configuration

Configure the server using environment variables or a `.env` file (in the same directory as the binary).

| Variable | Default | Description |
|----------|---------|-------------|
| `FLEET_BASE_URL` | *(required)* | Base URL of your Fleet instance, e.g. `https://dogfood.fleetdm.com` |
| `FLEET_API_KEY` | *(required)* | Fleet API token — see [Fleet docs](https://fleetdm.com/docs/using-fleet/rest-api#authentication). May alternatively be supplied via `FLEET_API_KEY_FILE`. |
| `FLEET_API_KEY_FILE` | *(optional)* | Path to a file containing the Fleet API token. Preferred over `FLEET_API_KEY` for production: keeps the admin token out of process env (`ps`), shell history, and `claude_desktop_config.json` (readable by your UID, lands in Time Machine backups). When both are set, `_FILE` wins. |
| `MCP_AUTH_TOKEN` | *(required)* | Bearer token for authenticating MCP clients. Generate with `openssl rand -hex 32`. **The server refuses to start without it on every transport (including stdio).** In SSE mode the server validates the token on every request and rate-limits each client IP to 20 requests/sec (burst 60); in stdio mode the token still must be set but is not checked at runtime (the client launches the binary as a local subprocess). May alternatively be supplied via `MCP_AUTH_TOKEN_FILE`. |
| `MCP_AUTH_TOKEN_FILE` | *(optional)* | Path to a file containing the MCP auth token. Same pattern as `FLEET_API_KEY_FILE`. |
| `PORT` | `8080` | HTTP port for SSE transport. Ignored in stdio mode. Render injects this automatically. |
| `LOG_LEVEL` | `info` | Log verbosity: `debug` / `info` / `warn` / `error`. Note: `debug` logs the route shape of every Fleet API call (path before query string only — no PII identifiers). Avoid `debug` in production deployments where logs are shipped to a centralized aggregator. |
| `FLEET_TLS_SKIP_VERIFY` | `false` | Skip TLS certificate verification. **Hard-gated to localhost — the server refuses to start with this set and a non-loopback `FLEET_BASE_URL`.** Conflicts with `FLEET_CA_FILE`. |
| `FLEET_CA_FILE` | *(optional)* | Path to a PEM CA certificate for self-signed Fleet instances |

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

4. **Fully quit and relaunch Claude Desktop** (`Cmd+Q`, not just close-window). The 18 Fleet tools will appear in your context.

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

Without these, Claude Desktop conservatively gates every tool behind destructive-action review and may collapse the surface to a single tool. The 16 read-only tools are annotated `readOnly=true, destructive=false, idempotent=true` so the AI agent can use them freely.

**Two tools are explicitly destructive and require user approval in MCP clients:**

| Tool | Annotations | Why destructive |
|---|---|---|
| `create_saved_query` | `readOnly=false, destructive=true, idempotent=false` | Writes a persistent saved query that can later be scheduled across every device. Resource creation IS destructive in the MCP threat model — auto-approval would let a prompt-injection chain create attacker-controlled queries silently. |
| `run_live_query` | `readOnly=false, destructive=true, idempotent=false` | Fires osquery against every targeted device, creates+deletes a transient saved query on the Fleet server, consumes device CPU, and shows up in EDR telemetry. Even SELECT-only SQL is operationally destructive at fleet scale. |

This means Claude Desktop will surface a confirmation prompt before either tool fires — required for any production deployment where the operator's Fleet API token is admin-scoped.

## Security model

The MCP holds the operator's `FLEET_API_KEY` which is admin-scoped on the target Fleet — compromise of the MCP gives an attacker full host inventory access plus arbitrary osquery against every enrolled device. Defenses:

- **TLS skip-verify hard-gated to localhost.** `FLEET_TLS_SKIP_VERIFY=true` paired with a non-loopback `FLEET_BASE_URL` makes the binary refuse to start (`logrus.Fatalf`) — copying a dev `.env` to a remote deploy can no longer expose the admin token to an on-path attacker.
- **Secret-from-file support.** `FLEET_API_KEY_FILE` and `MCP_AUTH_TOKEN_FILE` read the token from disk so it never appears in process env (`ps`), shell history, or `claude_desktop_config.json` (which is readable by your UID and lands in Time Machine backups). When both `_FILE` and direct env are set, `_FILE` wins. Recommended for production.
- **Per-IP rate limit on SSE transport.** Token-bucket limiter (default 20 req/sec, burst 60) defeats brute force against `MCP_AUTH_TOKEN` and amplification of authenticated requests into Fleet API quota. Stale visitor entries swept every minute, 10-minute TTL. Returns `429 Too Many Requests` with `Retry-After: 1` on overflow. Honors `X-Forwarded-For` first entry for Render-style deployments — direct exposure without a trusted proxy is not recommended.
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
   - `FLEET_API_KEY` — Fleet API token (or `FLEET_API_KEY_FILE` pointing at a Render Secret File)
   - `MCP_AUTH_TOKEN` — generate with `openssl rand -hex 32` (or `MCP_AUTH_TOKEN_FILE`)
5. `PORT` is injected automatically by Render — no action needed.

Render terminates TLS at its proxy and sets `X-Forwarded-For`, so the per-IP rate limiter sees real client IPs. If you deploy elsewhere without a trusted proxy, the `X-Forwarded-For` header is attacker-controlled and rate limiting can be bypassed — terminate TLS at a known proxy or accept the limitation.

## Development

### Project layout

```
tools/fleet-mcp/
  main.go                  # entrypoint, flag parsing, transport selection, http.Server timeouts, body-size cap
  config.go                # env-var loading + FLEET_API_KEY_FILE / MCP_AUTH_TOKEN_FILE secret resolution
  auth.go                  # bearer-auth middleware (SSE)
  rate_limit.go            # per-IP token-bucket throttle (SSE)
  route_guard.go           # SSE route allow-list
  fleet_integration.go     # FleetClient — wraps Fleet REST API. Every method takes ctx context.Context as first param.
  mcp_server.go            # SetupMCPServer orchestrator
  mcp_helpers.go           # getOptionalString, parseCSVArg, parsePerPageArg, validateCVEID, parsePositiveUintString, jsonResult
  mcp_tools_hosts.go       # host-domain MCP tools (7 tools)
  mcp_tools_queries.go     # query-domain MCP tools (6 tools)
  mcp_tools_policies.go    # policy/vuln MCP tools (5 tools)
  schema.go                # hardcoded osquery schema for AI
  vetted_queries.go        # vetted CIS-8.1 query library
  seed_fleet.go            # -seed mode
```

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
  - **Platform / label scoping** routes through `GET /labels/:id/hosts` (which DOES honor `team_id` and `query`, but ALSO ignores `software_version_id` and `policy_id`, so policy + label intersection is computed client-side by host ID).
  - **CVE → hosts** is a 3-step compose in `GetHostsForCVE`: `GET /software/titles?vulnerable=true&query=CVE-X` → per-title `GET /software/titles/:id` to harvest vulnerable version IDs → `GET /hosts?software_version_id=N` per ID → intersect with team / status / query / label-id client-side.
  - The single-call `GET /hosts?cve=` path is deliberately NOT used because it returns wrong results (e.g. CVE-2026-31431 yields 50 hosts via `?cve=`, but the correct answer is 1).
  - Future Fleet versions may fix these — revisit `GetEndpointsWithFilters` and `GetHostsForCVE` if/when that happens.
- **Team-scoped policy compliance** uses `/teams/:team_id/policies/:policy_id`, not the global path. `get_policy_compliance` routes to whichever based on whether `fleet` is set.
- **`/api/v1/fleet/host_summary` is the right endpoint for aggregate platform counts** — `GET /hosts` defaults to a 100-host page, so any client-side aggregation over `GetEndpoints(0)` is silently wrong on Fleets larger than 100 hosts. `get_aggregate_platforms` uses `host_summary` directly so totals match the Fleet UI at any inventory size.
- **`fetchHostsFromPath` paginates internally** with a hard cap (`fetchHostsHardCap = 10000`). Without this, a single call could buffer the full host inventory in memory (~2KB per Endpoint × 50k hosts ≈ 100MB) and OOM the MCP. When the cap fires a warning is logged so operators see truncation rather than silently getting a partial host set.
- **Per-team fan-out (`get_queries`, `get_policies`) is bounded-concurrent.** 8 in-flight goroutines, order-stable merge by team index. On enterprise Fleets with 50+ teams the sequential path was the dominant latency source; the bounded concurrency amortizes round-trip count without flooding Fleet with thousands of simultaneous requests.
- **CSV args drop empty segments.** `parseCSVArg("foo,,bar")` returns `["foo", "bar"]` — the legacy split-and-trim behavior leaked zero-value strings into filter logic, and a leading empty segment could silently disable filters that read `parts[0]`.
- **macOS Gatekeeper caches adhoc signatures** keyed to file identity. Replacing the binary at the same path silently invalidates the cached approval — re-run `codesign --force --sign -` after every rebuild before Claude Desktop will launch it.
- **Claude Desktop reads `env` from the JSON config**, not from a `.env` file. The `.env` template in this repo is for SSE/local development only.

## License

MIT
