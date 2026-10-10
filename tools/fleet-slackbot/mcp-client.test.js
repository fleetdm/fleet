const { test } = require("node:test");
const assert = require("node:assert/strict");
const { ErrorCode, McpError } = require("@modelcontextprotocol/sdk/types.js");
const McpClient = require("./mcp-client");

function clientWithStub(callTool) {
  const mcp = new McpClient({ url: "http://localhost/sse", toolTimeoutMs: 1234 });
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

test("retries when the server rejects the POST (stale session)", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    if (calls === 1) throw new Error('Error POSTing to endpoint (HTTP 400): {"jsonrpc":"2.0","id":null,"error":{"code":-32602,"message":"Invalid session ID"}}');
    return { content: [{ type: "text", text: "ok" }] };
  });
  assert.equal(await mcp.callTool("get_fleets", {}), "ok");
  assert.equal(reconnects(), 1);
});

test("does not retry an ambiguous POST failure", async () => {
  let calls = 0;
  const { mcp, reconnects } = clientWithStub(async () => {
    calls++;
    throw new Error("Error POSTing to endpoint (HTTP 502): Bad Gateway");
  });
  await assert.rejects(mcp.callTool("run_live_query", { sql: "SELECT 1" }), /502/);
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
