// Exercises the Teams routing (tenant gate, @mention gate, dedupe, fast ack +
// proactive continuation) and the handleRequest reply flow with stubbed
// Agents SDK, GitHub, and Claude pieces.
const { test } = require("node:test");
const assert = require("node:assert/strict");
const { Activity } = require("@microsoft/agents-activity");
const ClaudeClient = require("./claude-client");
const { registerHandlers } = require("./teams-handlers");

const TENANT = "11111111-1111-1111-1111-111111111111";
const BOT_ID = "28:00000000-0000-0000-0000-000000000000";
const PR_URL = "https://github.com/fleetdm/fleet/pull/7";

function makeConfig() {
  return {
    teams: { appId: "app-id", tenantId: TENANT.toUpperCase() },
    github: { gitopsBasePath: "it-and-security" },
  };
}

function makeActivity(overrides = {}) {
  return Activity.fromObject({
    type: "message",
    id: "msg-1",
    channelId: "msteams",
    serviceUrl: "https://smba.trafficmanager.net/amer/",
    from: { id: "29:user", name: "Jane Doe" },
    recipient: { id: BOT_ID, name: "Fleet" },
    conversation: { id: "a:1", conversationType: "personal", tenantId: TENANT },
    text: "How many hosts?",
    ...overrides,
  });
}

function mention(id, name) {
  return { type: "mention", text: `<at>${name}</at>`, mentioned: { id, name } };
}

// Stands in for the SDK's ActivityHandler: just captures the registered handlers.
function makeBot() {
  const handlers = {};
  return {
    handlers,
    onMessage(handler) {
      handlers.message = handler;
      return this;
    },
    onMembersAdded(handler) {
      handlers.membersAdded = handler;
      return this;
    },
  };
}

function makeContext(activity) {
  const sent = [];
  const updated = [];
  return {
    activity,
    identity: { aud: "app-id" },
    sent,
    updated,
    async sendActivity(activityOrText) {
      sent.push(typeof activityOrText === "string" ? { type: "message", text: activityOrText } : activityOrText);
      return { id: `sent-${sent.length}` };
    },
    async updateActivity(update) {
      updated.push(update);
    },
  };
}

// `gated: true` never runs the continuation until release() is called, which
// lets a test observe that onMessage returns before the slow work happens.
function makeAdapter({ gated = false } = {}) {
  const calls = [];
  let release;
  const gate = new Promise((resolve) => (release = resolve));
  return {
    calls,
    release,
    done: Promise.resolve(),
    async continueConversation(identity, reference, logic) {
      calls.push({ identity, reference });
      if (gated) {
        await gate;
        return;
      }
      const proactiveContext = makeContext(Activity.getContinuationActivity(reference));
      this.lastContext = proactiveContext;
      this.done = logic(proactiveContext);
      await this.done;
    },
  };
}

function makeGithub() {
  const calls = [];
  return {
    calls,
    async getRepoTreePaths() {
      return ["fleets/workstations.yml", "lib/macos/policies/update-slack.yml"];
    },
    async createBranch(branch) {
      calls.push(["createBranch", branch]);
    },
    async commitChanges(branch, changes, message) {
      calls.push(["commitChanges", branch, changes.map((c) => c.path), message]);
    },
    async createPullRequest(branch, title, body, opts) {
      calls.push(["createPullRequest", branch, title, opts]);
      return { url: PR_URL, number: 7 };
    },
  };
}

// A real ClaudeClient (for _parseResponse) with the API call stubbed out.
function makeClaude(responder) {
  const claude = new ClaudeClient({ apiKey: "test", model: "test", mcpClient: { getAnthropicTools: () => [] } });
  claude.userMessages = [];
  claude.runAgentLoop = async (userMessage, { onToolCall } = {}) => {
    claude.userMessages.push(userMessage);
    return responder({ onToolCall });
  };
  return claude;
}

function setup({ gated = false, responder = async () => "There are **42** hosts." } = {}) {
  const bot = makeBot();
  const adapter = makeAdapter({ gated });
  const github = makeGithub();
  const claude = makeClaude(responder);
  registerHandlers(bot, adapter, makeConfig(), github, claude);
  return { bot, adapter, github, claude };
}

async function deliver(bot, activity) {
  const context = makeContext(activity);
  let nextCalled = false;
  await bot.handlers.message(context, async () => {
    nextCalled = true;
  });
  assert.equal(nextCalled, true, "handler must call next()");
  return context;
}

test("onMessage acknowledges immediately and hands the slow work to continueConversation", async () => {
  const { bot, adapter } = setup({ gated: true });

  await deliver(bot, makeActivity());

  assert.equal(adapter.calls.length, 1);
  assert.deepEqual(adapter.calls[0].identity, { aud: "app-id" });
  assert.equal(adapter.calls[0].reference.conversation.id, "a:1");
  assert.equal(adapter.calls[0].reference.activityId, "msg-1");

  // Teams redelivery of the same activity while it's in flight is ignored.
  await deliver(bot, makeActivity());
  assert.equal(adapter.calls.length, 1);

  adapter.release();
});

test("channel and group chat messages require an @mention; other mentions keep their names", async () => {
  const { bot, adapter, claude } = setup();
  const channel = { id: "19:abc@thread.tacv2;messageid=1700000000000", conversationType: "channel", tenantId: TENANT };

  await deliver(bot, makeActivity({ id: "msg-2", conversation: channel, text: "How many hosts?" }));
  assert.equal(adapter.calls.length, 0, "no @mention → ignored");

  await deliver(
    bot,
    makeActivity({
      id: "msg-3",
      conversation: channel,
      text: "<at>Fleet</at> which hosts does <at>Jane Doe</at> own?",
      entities: [mention(BOT_ID, "Fleet"), mention("29:jane", "Jane Doe")],
    })
  );
  await adapter.done;
  assert.equal(adapter.calls.length, 1);
  assert.match(claude.userMessages[0], /<user_input>\nwhich hosts does Jane Doe own\?\n<\/user_input>/);
});

test("messages from another tenant are ignored", async () => {
  const { bot, adapter } = setup();
  await deliver(
    bot,
    makeActivity({ conversation: { id: "a:2", conversationType: "personal", tenantId: "22222222-2222-2222-2222-222222222222" } })
  );
  assert.equal(adapter.calls.length, 0);
});

test("informational answers are posted with extended Markdown and the status shows the tools used", async () => {
  const { bot, adapter, claude } = setup({
    responder: async ({ onToolCall }) => {
      onToolCall("get_endpoints", { fleet: "Workstations" });
      return "There are **42** hosts.";
    },
  });

  await deliver(bot, makeActivity());
  await adapter.done;

  const { sent, updated } = adapter.lastContext;
  assert.deepEqual(
    sent.map((a) => a.type),
    ["typing", "message", "message"]
  );
  assert.deepEqual(sent[1], { type: "message", text: "⏳ Thinking...", textFormat: "extendedmarkdown" });
  assert.deepEqual(sent[2], { type: "message", text: "There are **42** hosts.", textFormat: "extendedmarkdown" });

  // The status message is edited in place, ending with the activity log.
  assert.ok(updated.every((u) => u.id === "sent-2"));
  assert.equal(updated.at(-1).text, "🔍 **Tools used:**\n\n- `get_endpoints` — fleet=Workstations");

  // The first message carries no history; a follow-up carries both sides of the exchange.
  assert.doesNotMatch(claude.userMessages[0], /<thread_history>/);
  await deliver(bot, makeActivity({ id: "msg-2", text: "And how many are Windows?" }));
  await adapter.done;
  assert.match(
    claude.userMessages[1],
    /<thread_history>\nuser: How many hosts\?\n\nassistant: There are \*\*42\*\* hosts\.\n<\/thread_history>/
  );
});

test("configuration changes open a draft PR and announce it with an Adaptive Card", async () => {
  const content = [
    '- name: "macOS - Firefox installed"',
    "  query: \"SELECT 1 FROM apps WHERE bundle_identifier = 'org.mozilla.firefox';\"",
    "  critical: false",
    '  description: "Checks that Firefox is installed."',
    '  resolution: "Install Firefox from Self-service."',
    "  platform: darwin",
    "  calendar_events_enabled: false",
    "",
  ].join("\n");
  const { bot, adapter, github } = setup({
    responder: async () =>
      JSON.stringify({
        summary: "Adds a Firefox policy.",
        pr_title: "Add Firefox installed policy",
        pr_body: "Adds a policy.",
        changes: [
          {
            file_path: "lib/macos/policies/firefox-installed.yml",
            change_description: "New policy",
            content,
            is_new_file: true,
          },
        ],
      }),
  });

  await deliver(bot, makeActivity({ text: "Add a policy that checks Firefox is installed" }));
  await adapter.done;

  assert.deepEqual(
    github.calls.map((c) => c[0]),
    ["createBranch", "commitChanges", "createPullRequest"]
  );
  const branch = github.calls[0][1];
  assert.match(branch, /^fleet\/[0-9a-f]{12}$/);
  assert.deepEqual(github.calls[1], ["commitChanges", branch, ["it-and-security/lib/macos/policies/firefox-installed.yml"], "Add Firefox installed policy"]);
  assert.deepEqual(github.calls[2][3], { draft: true });

  const { sent, updated } = adapter.lastContext;
  const card = sent.find((a) => a.attachments);
  assert.ok(card, "expected an Adaptive Card message");
  assert.equal(sent.at(-1), card, "a valid policy must not trigger a validation-warning message");
  assert.equal(card.attachments[0].contentType, "application/vnd.microsoft.card.adaptive");
  assert.deepEqual(card.attachments[0].content.actions, [{ type: "Action.OpenUrl", title: "Open pull request", url: PR_URL }]);
  assert.equal(card.summary, `Draft PR created: ${PR_URL}`);
  assert.equal(updated.at(-1).text, "✅ Done.");
});

test("changes outside the GitOps structure are rejected before anything is committed", async () => {
  const { bot, adapter, github } = setup({
    responder: async () =>
      JSON.stringify({
        summary: "Oops",
        pr_title: "Escape",
        pr_body: "x",
        changes: [{ file_path: "../.github/workflows/evil.yml", change_description: "x", content: "x".repeat(80), is_new_file: true }],
      }),
  });

  await deliver(bot, makeActivity({ text: "do something sneaky" }));
  await adapter.done;

  assert.deepEqual(github.calls, []);
  const { sent, updated } = adapter.lastContext;
  assert.match(sent.at(-1).text, /^❌ \*\*Error:\*\* Invalid file path in response: Path traversal not allowed/);
  assert.equal(updated.at(-1).text, "❌ Failed.");
});

test("the welcome message is only sent when the bot is opened in a personal chat", async () => {
  const { bot } = setup();
  const membersAdded = [{ id: BOT_ID, name: "Fleet" }];

  const personal = makeContext(
    makeActivity({ type: "conversationUpdate", text: undefined, membersAdded })
  );
  await bot.handlers.membersAdded(personal, async () => {});
  assert.equal(personal.sent.length, 1);
  assert.match(personal.sent[0].text, /^👋 Hi, I'm \*\*Fleet\*\*/);

  const team = makeContext(
    makeActivity({
      type: "conversationUpdate",
      text: undefined,
      membersAdded,
      conversation: { id: "19:abc@thread.tacv2", conversationType: "channel", tenantId: TENANT },
    })
  );
  await bot.handlers.membersAdded(team, async () => {});
  assert.equal(team.sent.length, 0);
});
