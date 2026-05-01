const { test } = require("node:test");
const assert = require("node:assert/strict");
const ClaudeClient = require("./claude-client");

const { MAX_TOOL_ROUNDS } = ClaudeClient;

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
        // First MAX_TOOL_ROUNDS calls: keep returning tool_use so the loop drains its budget.
        if (calls.length <= MAX_TOOL_ROUNDS) {
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
    MAX_TOOL_ROUNDS + 1,
    `expected ${MAX_TOOL_ROUNDS} tool rounds plus one forced no-tools call`
  );

  // All in-budget calls must NOT disable tool use.
  for (let i = 0; i < MAX_TOOL_ROUNDS; i++) {
    assert.equal(
      calls[i].tool_choice,
      undefined,
      `call ${i + 1} should not pin tool_choice`
    );
  }

  // The fallback call must disable tool use so Claude is forced to produce text.
  assert.deepEqual(calls[MAX_TOOL_ROUNDS].tool_choice, { type: "none" });

  // The fallback call's last user message asks for a best-effort answer
  // with a non-leaky caveat — it must not mention tools, rounds, or limits.
  const finalMessages = calls[MAX_TOOL_ROUNDS].messages;
  const lastUser = finalMessages[finalMessages.length - 1];
  assert.equal(lastUser.role, "user");
  const lastUserText =
    typeof lastUser.content === "string"
      ? lastUser.content
      : JSON.stringify(lastUser.content);
  assert.match(lastUserText, /incomplete or unverified/);
  assert.doesNotMatch(lastUserText, /tool|round|limit/i);
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
