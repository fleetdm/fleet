const { test } = require("node:test");
const assert = require("node:assert/strict");
const ClaudeClient = require("./claude-client");

const { MAX_TOOL_CALLS } = ClaudeClient;

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
        // First MAX_TOOL_CALLS turns each emit a single tool_use, so the
        // tool-call counter increments by 1 per turn and trips the cap.
        if (calls.length <= MAX_TOOL_CALLS) {
          return makeStreamingResponse("tool_use", [
            {
              type: "tool_use",
              id: `tu_${calls.length}`,
              name: "stub_tool",
              input: {},
            },
          ]);
        }
        // Final call (the budget-exhausted fallback): return a final text.
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
    MAX_TOOL_CALLS + 1,
    `expected ${MAX_TOOL_CALLS} tool-call turns plus one forced no-tools call`
  );

  // All in-budget calls must NOT disable tool use.
  for (let i = 0; i < MAX_TOOL_CALLS; i++) {
    assert.equal(
      calls[i].tool_choice,
      undefined,
      `call ${i + 1} should not pin tool_choice`
    );
  }

  // The fallback call must disable tool use so Claude is forced to produce text.
  assert.deepEqual(calls[MAX_TOOL_CALLS].tool_choice, { type: "none" });

  // The fallback call's last user message asks for a best-effort answer
  // with a non-leaky caveat — it must not mention tools, rounds, or limits.
  const finalMessages = calls[MAX_TOOL_CALLS].messages;
  const lastUser = finalMessages[finalMessages.length - 1];
  assert.equal(lastUser.role, "user");
  const lastUserText =
    typeof lastUser.content === "string"
      ? lastUser.content
      : JSON.stringify(lastUser.content);
  assert.match(lastUserText, /incomplete or unverified/);
  assert.doesNotMatch(lastUserText, /tool|round|limit/i);
});

test("runAgentLoop caps tool calls within a single fanned-out turn", async () => {
  // One turn emits MAX_TOOL_CALLS + 5 parallel tool_use blocks. Only the
  // first MAX_TOOL_CALLS may execute; the rest must be answered with a
  // synthetic "skipped" tool_result so the message history stays valid,
  // and the loop must immediately move to the no-tools fallback.
  const calls = [];
  const executedToolIds = [];
  const fanOutCount = MAX_TOOL_CALLS + 5;
  const fakeClient = {
    messages: {
      stream(params) {
        calls.push(params);
        if (calls.length === 1) {
          const blocks = [];
          for (let i = 0; i < fanOutCount; i++) {
            blocks.push({
              type: "tool_use",
              id: `tu_${i}`,
              name: "stub_tool",
              input: {},
            });
          }
          return makeStreamingResponse("tool_use", blocks);
        }
        return makeStreamingResponse("end_turn", [
          { type: "text", text: "Capped fan-out answer." },
        ]);
      },
    },
  };

  const fakeMcpClient = {
    getAnthropicTools: () => [
      { name: "stub_tool", description: "stub", input_schema: { type: "object" } },
    ],
    callTool(_name, _input) {
      return Promise.resolve("stub result");
    },
  };

  const client = new ClaudeClient({
    apiKey: "test",
    model: "claude-test",
    mcpClient: fakeMcpClient,
  });
  client.client = fakeClient;
  // Wrap callTool to record executions.
  const originalCallTool = fakeMcpClient.callTool.bind(fakeMcpClient);
  client.mcpClient.callTool = async (name, input) => {
    executedToolIds.push(name);
    return originalCallTool(name, input);
  };

  const result = await client.runAgentLoop("Fan out.");

  assert.equal(result, "Capped fan-out answer.");
  assert.equal(executedToolIds.length, MAX_TOOL_CALLS);
  // Two stream calls total: the fan-out turn, then the no-tools fallback.
  assert.equal(calls.length, 2);
  assert.deepEqual(calls[1].tool_choice, { type: "none" });

  // The tool_result message that goes back must include a tool_result for
  // every tool_use block (executed + skipped) so Anthropic's API contract
  // stays satisfied.
  const toolResultMessage = calls[1].messages.find(
    (m) =>
      m.role === "user" &&
      Array.isArray(m.content) &&
      m.content.some((b) => b.type === "tool_result")
  );
  assert.ok(toolResultMessage, "expected a tool_result user message");
  const results = toolResultMessage.content.filter((b) => b.type === "tool_result");
  assert.equal(results.length, fanOutCount);
  const skippedResults = results.filter((r) => r.is_error === true);
  assert.equal(skippedResults.length, fanOutCount - MAX_TOOL_CALLS);
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
