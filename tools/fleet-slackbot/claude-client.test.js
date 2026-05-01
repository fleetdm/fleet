const { test } = require("node:test");
const assert = require("node:assert/strict");
const ClaudeClient = require("./claude-client");

// makeStreamingResponse builds a fake of the object returned by
// `client.messages.stream(...)` — a thenable-ish thing with `.on(...)` for
// "text" events and `.finalMessage()` returning a full message.
function makeStreamingResponse(stopReason, contentBlocks) {
  return {
    on() {},
    finalMessage() {
      return Promise.resolve({
        content: contentBlocks,
        stop_reason: stopReason,
        usage: { input_tokens: 0, output_tokens: 0 },
      });
    },
  };
}

test("runAgentLoop forces a final no-tools response when the budget is exhausted", async () => {
  const calls = [];
  const fakeClient = {
    messages: {
      stream(params) {
        calls.push(params);
        // First 12 calls: keep returning tool_use so the loop drains its budget.
        if (calls.length <= 12) {
          return makeStreamingResponse("tool_use", [
            {
              type: "tool_use",
              id: `tu_${calls.length}`,
              name: "stub_tool",
              input: {},
            },
          ]);
        }
        // 13th call (the budget-exhausted fallback): return a final text.
        return makeStreamingResponse("end_turn", [
          { type: "text", text: "Best-effort answer with partial data." },
        ]);
      },
    },
  };

  const fakeMcpClient = {
    getAnthropicTools() {
      return [
        {
          name: "stub_tool",
          description: "stub",
          input_schema: { type: "object" },
        },
      ];
    },
    callTool() {
      return Promise.resolve("stub result");
    },
  };

  const client = new ClaudeClient({
    apiKey: "test",
    model: "claude-test",
    mcpClient: fakeMcpClient,
  });
  client.client = fakeClient;

  const result = await client.runAgentLoop("Tell me something.");

  assert.equal(result, "Best-effort answer with partial data.");
  assert.equal(
    calls.length,
    13,
    "expected 12 tool rounds plus one forced no-tools call"
  );

  // The first 12 calls must NOT disable tool use.
  for (let i = 0; i < 12; i++) {
    assert.equal(
      calls[i].tool_choice,
      undefined,
      `call ${i + 1} should not pin tool_choice`
    );
  }

  // The 13th call must disable tool use so Claude is forced to produce text.
  assert.deepEqual(calls[12].tool_choice, { type: "none" });

  // The fallback call's last user message tells Claude the budget is gone.
  const finalMessages = calls[12].messages;
  const lastUser = finalMessages[finalMessages.length - 1];
  assert.equal(lastUser.role, "user");
  assert.match(
    typeof lastUser.content === "string"
      ? lastUser.content
      : JSON.stringify(lastUser.content),
    /maximum number of tool calls/
  );
});

test("runAgentLoop returns end_turn text without forcing the fallback when Claude finishes early", async () => {
  let callCount = 0;
  const fakeClient = {
    messages: {
      stream() {
        callCount++;
        return makeStreamingResponse("end_turn", [
          { type: "text", text: "Done." },
        ]);
      },
    },
  };

  const client = new ClaudeClient({
    apiKey: "test",
    model: "claude-test",
    mcpClient: { getAnthropicTools: () => [] },
  });
  client.client = fakeClient;

  const result = await client.runAgentLoop("Hello");
  assert.equal(result, "Done.");
  assert.equal(callCount, 1);
});
