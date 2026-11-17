const http = require("node:http");
const { test } = require("node:test");
const assert = require("node:assert/strict");
const { StreamableHTTPError } = require("@modelcontextprotocol/sdk/client/streamableHttp.js");
const { McpServer } = require("@modelcontextprotocol/sdk/server/mcp.js");
const { StreamableHTTPServerTransport } = require("@modelcontextprotocol/sdk/server/streamableHttp.js");
const { ErrorCode, McpError } = require("@modelcontextprotocol/sdk/types.js");
const McpClient = require("./mcp-client");

function clientWithStub(callTool) {
  const mcp = new McpClient({ url: "http://localhost/mcp", toolTimeoutMs: 1234 });
  mcp._connected = true;
  mcp.client = { callTool };
  let reconnects = 0;
  mcp.connect = async () => {
    reconnects++;
    mcp._connected = true;
  };
  return { mcp, reconnects: () => reconnects };
}

test("passes the configured timeout to callTool", async () => {
  let seenOptions;
  const { mcp } = clientWithStub(async (_params, _schema, options) => {
    seenOptions = options;
    return { content: [{ type: "text", text: "ok" }] };
  });
  assert.equal(await mcp.callTool("get_fleets", {}), "ok");
  assert.deepEqual(seenOptions, { timeout: 1234 });
});

test("does not retry a timed-out call", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    throw new McpError(ErrorCode.RequestTimeout, "Request timed out");
  });
  await assert.rejects(mcp.callTool("run_live_query", { sql: "SELECT 1" }), /timed out/);
  assert.equal(calls, 1);
  assert.equal(reconnects(), 0);
});

test("does not retry errors that aren't connection loss", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    throw new McpError(ErrorCode.InternalError, "boom");
  });
  await assert.rejects(mcp.callTool("get_fleets", {}), /boom/);
  assert.equal(calls, 1);
  assert.equal(reconnects(), 0);
});

test("does not retry an ambiguous POST failure", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    throw new StreamableHTTPError(502, "Error POSTing to endpoint: Bad Gateway");
  });
  await assert.rejects(mcp.callTool("run_live_query", { sql: "SELECT 1" }), { code: 502 });
  assert.equal(calls, 1);
  assert.equal(reconnects(), 0);
});

test("does not retry when the connection drops mid-call, but reconnects next time", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    if (calls === 1) {
      mcp._connected = false; // the SDK's onclose fires before pending calls reject
      throw new McpError(ErrorCode.ConnectionClosed, "Connection closed");
    }
    return { content: [{ type: "text", text: "ok" }] };
  });
  await assert.rejects(mcp.callTool("run_live_query", { sql: "SELECT 1" }), /Connection closed/);
  assert.equal(calls, 1);
  assert.equal(await mcp.callTool("get_fleets", {}), "ok");
  assert.equal(reconnects(), 1);
});

test("retries when the client had no transport", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    if (calls === 1) throw new Error("Not connected");
    return { content: [{ type: "text", text: "ok" }] };
  });
  assert.equal(await mcp.callTool("get_fleets", {}), "ok");
  assert.equal(calls, 2);
  assert.equal(reconnects(), 1);
});

// Fleet's MCP server only speaks Streamable HTTP, without sessions, and
// authenticates each request with the API-only user's token.
test("connects over Streamable HTTP and sends the token on every request", async (t) => {
  const authHeaders = [];
  const httpServer = http.createServer(async (req, res) => {
    authHeaders.push(req.headers.authorization);
    if (req.method !== "POST") {
      res.writeHead(405).end();
      return;
    }
    const server = new McpServer({ name: "fleet", version: "1.0.0" });
    server.registerTool("get_total_system_count", { description: "Count hosts" }, async () => ({
      content: [{ type: "text", text: "Total Enrolled Systems: 3" }],
    }));
    const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
    await server.connect(transport);
    await transport.handleRequest(req, res);
  });
  await new Promise((resolve) => httpServer.listen(0, "127.0.0.1", resolve));

  const mcp = new McpClient({
    url: `http://127.0.0.1:${httpServer.address().port}/api/v1/fleet/mcp`,
    authToken: "api-only-user-token",
    toolTimeoutMs: 5000,
  });
  t.after(async () => {
    await mcp.disconnect();
    httpServer.closeAllConnections();
    httpServer.close();
  });

  await mcp.connect();
  assert.deepEqual(mcp.tools.map((tool) => tool.name), ["get_total_system_count"]);
  assert.equal(await mcp.callTool("get_total_system_count", {}), "Total Enrolled Systems: 3");
  assert.ok(authHeaders.length >= 3, `expected initialize, tools/list and tools/call, got ${authHeaders.length} requests`);
  assert.ok(authHeaders.every((h) => h === "Bearer api-only-user-token"));
});
