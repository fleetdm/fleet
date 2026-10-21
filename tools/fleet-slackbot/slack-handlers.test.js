const { test } = require("node:test");
const assert = require("node:assert/strict");
const { createSlackChat } = require("./slack-handlers");

function fakeSlackClient() {
  const calls = [];
  return {
    calls,
    reactions: {
      add: async (args) => { calls.push(["reactions.add", args.name]); },
      remove: async (args) => { calls.push(["reactions.remove", args.name]); },
    },
    chat: {
      postMessage: async (args) => { calls.push(["postMessage", args]); return { ts: "status-ts" }; },
      update: async (args) => { calls.push(["update", args.ts, args.text]); },
    },
  };
}

test("Slack adapter keeps the hourglass → check flow and threaded replies", async () => {
  const client = fakeSlackClient();
  const chat = createSlackChat({ client, channelId: "C1", threadTs: "100.1", messageTs: "100.2", logPrefix: "[test]" });

  await chat.acknowledge();
  await chat.updateStatus({ phase: "tool", tool: "get_endpoints" });
  await chat.postAnswer("42 Macs");
  await chat.complete(["• `get_endpoints`"]);

  assert.deepEqual(client.calls[0], ["reactions.add", "hourglass_flowing_sand"]);
  assert.equal(client.calls[1][1].text, ":hourglass_flowing_sand: Thinking...");
  assert.equal(client.calls[1][1].thread_ts, "100.1");
  assert.deepEqual(client.calls[2], ["update", "status-ts", ":gear: Calling Fleet tool: `get_endpoints`..."]);
  assert.equal(client.calls[3][1].text, "42 Macs");
  assert.deepEqual(client.calls[4], ["update", "status-ts", ":mag: *Tools used:*\n• `get_endpoints`"]);
  assert.deepEqual(client.calls.slice(5), [["reactions.remove", "hourglass_flowing_sand"], ["reactions.add", "white_check_mark"]]);
});

test("Slack adapter posts the Draft PR block", async () => {
  const client = fakeSlackClient();
  const chat = createSlackChat({ client, channelId: "C1", threadTs: "100.1", logPrefix: "[test]" });

  await chat.postPullRequest({
    url: "https://github.com/o/r/pull/1",
    title: "Add policy",
    summary: "Adds a policy",
    changes: [{ filePath: "lib/p.yml", changeDescription: "New" }],
  });

  const msg = client.calls[0][1];
  assert.equal(msg.text, "Draft PR created: https://github.com/o/r/pull/1");
  assert.equal(msg.blocks[0].text.text, "Draft PR Created");
  assert.equal(msg.blocks[1].text.text, ":white_check_mark: *<https://github.com/o/r/pull/1|Add policy>*\n\nAdds a policy\n\n*Files changed:*\n• `lib/p.yml` — New");
});

test("Slack adapter failure swaps to X, posts the error, and doesn't throw", async () => {
  const client = fakeSlackClient();
  const chat = createSlackChat({ client, channelId: "C1", threadTs: "100.1", logPrefix: "[test]" });
  await chat.acknowledge();
  client.calls.length = 0;
  client.chat.update = async () => { throw new Error("slack down"); };

  await chat.fail("Something broke", []);

  assert.deepEqual(client.calls[0], ["reactions.remove", "hourglass_flowing_sand"]);
  assert.deepEqual(client.calls[1], ["reactions.add", "x"]);
  assert.equal(client.calls[2][1].text, ":x: *Error:* Something broke");
});

test("Slack answers are truncated below Slack's message limit", async () => {
  const client = fakeSlackClient();
  const chat = createSlackChat({ client, channelId: "C1", threadTs: "100.1", logPrefix: "[test]" });
  await chat.postAnswer("x".repeat(50000));
  const text = client.calls[0][1].text;
  assert.ok(text.length < 40000);
  assert.ok(text.endsWith("_…response truncated due to length._"));
});
