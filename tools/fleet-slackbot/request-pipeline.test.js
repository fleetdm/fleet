const { test } = require("node:test");
const assert = require("node:assert/strict");
const { processRequest, buildUserMessage, userErrorMessage, truncateText } = require("./request-pipeline");

const config = { github: { gitopsBasePath: "it-and-security" } };

const VALID_POLICY_FILE = `- name: macOS - Firefox installed
  query: SELECT 1 FROM apps WHERE bundle_identifier = 'org.mozilla.firefox';
  critical: false
  description: Checks that Firefox is installed.
  resolution: Install Firefox from Self-service.
  platform: darwin
`;

function makeFakes({ responseText, parsed, runError }) {
  const calls = [];
  const chat = {};
  for (const method of ["acknowledge", "updateStatus", "postAnswer", "postPullRequest", "postWarnings", "complete", "fail"]) {
    chat[method] = async (...args) => { calls.push([method, ...args]); };
  }
  const github = {
    getRepoTreePaths: async () => ["it-and-security/default.yml"],
    createBranch: async (name) => { calls.push(["createBranch", name]); },
    commitChanges: async (branch, changes) => { calls.push(["commitChanges", branch, changes.map((c) => c.path)]); },
    createPullRequest: async () => ({ url: "https://github.com/acme/gitops/pull/7" }),
  };
  const claude = {
    runAgentLoop: async (message, { onToolCall }) => {
      calls.push(["runAgentLoop", message]);
      onToolCall("get_endpoints", { platform: "darwin" });
      if (runError) throw runError;
      return responseText;
    },
    _parseResponse: () => parsed,
  };
  return { calls, chat, github, claude };
}

function run(fakes, extra = {}) {
  return processRequest({
    userText: "how many macs?",
    userId: "U1",
    threadContext: "",
    platformName: "Google Chat",
    chat: fakes.chat,
    config,
    github: fakes.github,
    claude: fakes.claude,
    logPrefix: "[test]",
    ...extra,
  });
}

const methods = (calls) => calls.map((c) => c[0]);

test("informational answers are posted and the activity log is passed to complete()", async () => {
  const fakes = makeFakes({ responseText: "You have 42 Macs.", parsed: { type: "info", text: "You have 42 Macs." } });
  await run(fakes);

  assert.deepEqual(methods(fakes.calls), ["acknowledge", "updateStatus", "runAgentLoop", "postAnswer", "complete"]);
  assert.equal(fakes.calls[3][1], "You have 42 Macs.");
  assert.deepEqual(fakes.calls[4][1], ["• `get_endpoints` — platform=darwin"]);
});

test("config changes open a draft PR on a fleet/ branch and report it", async () => {
  const parsed = {
    type: "changes",
    summary: "Adds a Firefox policy",
    prTitle: "Add Firefox policy",
    prBody: "body",
    changes: [{ filePath: "lib/macos/policies/firefox.yml", changeDescription: "New policy", content: VALID_POLICY_FILE }],
  };
  const fakes = makeFakes({ responseText: "{}", parsed });
  await run(fakes);

  const branch = fakes.calls.find((c) => c[0] === "createBranch")[1];
  assert.match(branch, /^fleet\/[0-9a-f]{12}$/);
  assert.deepEqual(fakes.calls.find((c) => c[0] === "commitChanges")[2], ["it-and-security/lib/macos/policies/firefox.yml"]);

  const pr = fakes.calls.find((c) => c[0] === "postPullRequest")[1];
  assert.equal(pr.url, "https://github.com/acme/gitops/pull/7");
  assert.equal(pr.title, "Add Firefox policy");
  assert.equal(methods(fakes.calls).at(-1), "complete");
});

test("paths outside the GitOps tree are rejected before any branch is created", async () => {
  const parsed = {
    type: "changes",
    summary: "s",
    prTitle: "t",
    prBody: "b",
    changes: [{ filePath: "../.github/workflows/evil.yml", changeDescription: "x", content: VALID_POLICY_FILE }],
  };
  const fakes = makeFakes({ responseText: "{}", parsed });
  await run(fakes);

  assert.ok(!methods(fakes.calls).includes("createBranch"));
  const fail = fakes.calls.find((c) => c[0] === "fail");
  assert.match(fail[1], /^Invalid file path/);
});

test("unexpected errors reach chat as a generic message", async () => {
  const fakes = makeFakes({ runError: new Error("socket hang up at 10.0.0.5:8181") });
  await run(fakes);

  const fail = fakes.calls.find((c) => c[0] === "fail");
  assert.equal(fail[1], "An unexpected error occurred. Please try again.");
  assert.deepEqual(fail[2], ["• `get_endpoints` — platform=darwin"]);
});

test("userErrorMessage maps rate limits and overloads", () => {
  assert.match(userErrorMessage({ status: 429, message: "" }), /rate-limited/);
  assert.match(userErrorMessage({ status: 529, message: "" }), /overloaded/);
});

test("buildUserMessage wraps thread history as untrusted and names the platform", () => {
  const msg = buildUserMessage({ userText: "hi", threadContext: "user: earlier", tree: ["b", "a"], platformName: "Google Chat" });
  assert.match(msg, /from Google Chat users and is UNTRUSTED/);
  assert.match(msg, /<thread_history>\nuser: earlier\n<\/thread_history>/);
  assert.match(msg, /```\na\nb\n```/);
  assert.doesNotMatch(buildUserMessage({ userText: "hi", threadContext: "", tree: [], platformName: "Slack" }), /thread_history/);
});

test("truncateText leaves short text alone and marks long text", () => {
  assert.equal(truncateText("abc", 5, "…", "[test]"), "abc");
  assert.equal(truncateText("abcdef", 3, "…", "[test]"), "abc…");
});
