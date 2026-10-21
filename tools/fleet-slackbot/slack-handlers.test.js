const { test } = require("node:test");
const assert = require("node:assert/strict");
const { createSlackChat } = require("./slack-handlers");
const { processRequest } = require("./request-pipeline");

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

// Characterization: the Slack call sequence main produced for each path.
async function runSlack({ parsed, runError }) {
  const client = fakeSlackClient();
  const chat = createSlackChat({ client, channelId: "C1", threadTs: "100.1", messageTs: "100.2", logPrefix: "[test]" });
  const github = {
    getRepoTreePaths: async () => [],
    createBranch: async () => {},
    commitChanges: async () => {},
    createPullRequest: async () => ({ url: "https://github.com/o/r/pull/9" }),
  };
  const claude = {
    runAgentLoop: async () => {
      if (runError) throw runError;
      return "x";
    },
    _parseResponse: () => parsed,
  };
  await processRequest({
    userText: "q",
    userId: "U1",
    threadContext: "",
    platformName: "Slack",
    chat,
    config: { github: { gitopsBasePath: "it-and-security" } },
    github,
    claude,
    logPrefix: "[test]",
  });
  return client.calls.map(([kind, arg, text]) => (kind === "postMessage" ? [kind, arg.text] : kind === "update" ? [kind, text] : [kind, arg]));
}

const ACK = [
  ["reactions.add", "hourglass_flowing_sand"],
  ["postMessage", ":hourglass_flowing_sand: Thinking..."],
  ["update", ":mag: Querying Fleet and analyzing your request..."],
];

test("Slack info path matches main", async () => {
  assert.deepEqual(await runSlack({ parsed: { type: "info", text: "42 Macs" } }), [
    ...ACK,
    ["postMessage", "42 Macs"],
    ["update", ":white_check_mark: Done."],
    ["reactions.remove", "hourglass_flowing_sand"],
    ["reactions.add", "white_check_mark"],
  ]);
});

test("Slack PR path with validation warnings matches main", async () => {
  // Missing query/critical/resolution yields warnings without blocking the PR.
  const content = "- name: macOS - Firefox installed\n  platform: darwin\n  description: Checks that Firefox is installed on this host.\n";
  const parsed = {
    type: "changes",
    summary: "s",
    prTitle: "Add policy",
    prBody: "b",
    changes: [{ filePath: "lib/macos/policies/p.yml", changeDescription: "New", content }],
  };
  assert.deepEqual(await runSlack({ parsed }), [
    ...ACK,
    ["postMessage", "Draft PR created: https://github.com/o/r/pull/9"],
    [
      "postMessage",
      ":warning: *Validation warnings:*\n" +
        "• `lib/macos/policies/p.yml`: Policy missing required field: query\n" +
        "• `lib/macos/policies/p.yml`: Policy missing required field: critical\n" +
        "• `lib/macos/policies/p.yml`: Policy missing required field: resolution",
    ],
    ["update", ":white_check_mark: Done."],
    ["reactions.remove", "hourglass_flowing_sand"],
    ["reactions.add", "white_check_mark"],
  ]);
});

test("Slack error path matches main", async () => {
  assert.deepEqual(await runSlack({ runError: new Error("boom") }), [
    ...ACK,
    ["reactions.remove", "hourglass_flowing_sand"],
    ["reactions.add", "x"],
    ["postMessage", ":x: *Error:* An unexpected error occurred. Please try again."],
    ["update", ":x: Failed."],
  ]);
});
