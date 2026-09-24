# Fleet MCP — dev/test helpers. Run from cmd/fleet-mcp/.
#
# The MCP reads these (the binary auto-loads a .env in this dir, or export them):
#   FLEET_BASE_URL, FLEET_API_KEY, MCP_AUTH_TOKEN (>=32 chars)
#   FLEET_API_KEY must belong to an API-only Fleet user, else the binary refuses
#     to start (see README "API-only token required").
#   FLEET_TLS_SKIP_VERIFY=true     # only for a localhost dev Fleet with a self-signed cert
# Quickest setup:  cp .env.example .env  &&  edit it  (.env is gitignored).
#
# Usage:
#   make build
#   make tools                                   # list registered tools
#   make posture                                 # show the startup token-posture check (stderr)
#   make call TOOL=get_total_system_count
#   make call TOOL=get_endpoints ARGS='{"per_page":"5"}'
#   make call TOOL=run_live_query ARGS='{"sql":"SELECT version FROM os_version","host_ids":"2"}'
#   make sse PORT=8137                           # run the SSE server (foreground)
#
# Tool output is pretty-printed when `jq` is installed; otherwise raw JSON-RPC.
# No language runtime required beyond a POSIX shell + the built binary.

BIN  ?= ./fleet-mcp
TOOL ?=
ARGS ?= {}
PORT ?= 8080

# JSON-RPC handshake lines reused by call/tools.
INIT   := {"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"make","version":"1"}}}
INITED := {"jsonrpc":"2.0","method":"notifications/initialized"}

.PHONY: help build call tools posture sse

help:
	@grep -E '^#' Makefile | sed -e 's/^# \{0,1\}//'

build:
	@go build -o fleet-mcp .

# Drive a single tool over stdio. Clean output on success; prints the binary's
# stderr (e.g. a startup Fatalf) instead of going silent on failure.
call: build
	@test -n "$(TOOL)" || { echo "usage: make call TOOL=<tool> [ARGS='<json>']"; exit 2; }
	@err=$$(mktemp); \
	out=$$(printf '%s\n' \
	  '$(INIT)' \
	  '$(INITED)' \
	  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"$(TOOL)","arguments":$(ARGS)}}' \
	  | $(BIN) -transport stdio 2>$$err); \
	if [ -z "$$out" ]; then echo "no response — startup error:"; cat $$err; rm -f $$err; exit 1; fi; \
	rm -f $$err; \
	printf '%s\n' "$$out" | if command -v jq >/dev/null 2>&1; then jq -Rr 'fromjson? | select(.id==2) | (.result.content[0].text // .result // .)'; else grep '"id":2'; fi

# List the tools the server registers. Surfaces a startup error (e.g. a
# non-API-only token or unreachable Fleet) instead of printing nothing.
tools: build
	@err=$$(mktemp); \
	out=$$(printf '%s\n' '$(INIT)' '$(INITED)' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' \
	  | $(BIN) -transport stdio 2>$$err); \
	if [ -z "$$out" ]; then echo "no response — startup error:"; cat $$err; rm -f $$err; exit 1; fi; \
	rm -f $$err; \
	printf '%s\n' "$$out" | if command -v jq >/dev/null 2>&1; then jq -Rr 'fromjson? | select(.id==2) | .result.tools[].name'; else grep '"id":2'; fi

# Run the startup token check: the binary verifies FLEET_API_KEY is an API-only
# user (logs it, or refuses) on stderr, then exits on stdin EOF.
posture: build
	@$(BIN) -transport stdio </dev/null

# Run the SSE server in the foreground (Ctrl-C to stop).
sse: build
	PORT=$(PORT) $(BIN)
