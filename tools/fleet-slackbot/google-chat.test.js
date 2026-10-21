const { test } = require("node:test");
const assert = require("node:assert/strict");
const crypto = require("crypto");
const { Readable } = require("stream");
const { OAuth2Client } = require("google-auth-library");
const { normalizeEvent, isDirectMessage, requestText } = require("./google-chat-events");
const { createRequestVerifier } = require("./google-chat-auth");
const {
  createGoogleChatHandler,
  createSpacePacer,
  ConversationMemory,
  pullRequestMessage,
  senderAllowed,
  truncateBytes,
} = require("./google-chat-handlers");

const PROJECT_NUMBER = "123456789";
const ENDPOINT_URL = "https://bot.example.com/google-chat/events";

// ── Events ─────────────────────────────────────────────────────────────

const message = {
  name: "spaces/AAA/messages/M1",
  sender: { name: "users/1", email: "alice@example.com", type: "HUMAN" },
  text: "@Fleet how many macs?",
  argumentText: " how many macs?",
  thread: { name: "spaces/AAA/threads/T1" },
};
const namedSpace = { name: "spaces/AAA", spaceType: "SPACE" };

test("normalizeEvent reads classic and add-on message events the same way", () => {
  const classic = normalizeEvent({ type: "MESSAGE", message, space: namedSpace, user: message.sender });
  const addOn = normalizeEvent({ chat: { user: message.sender, messagePayload: { message, space: namedSpace } } });

  for (const event of [classic, addOn]) {
    assert.equal(event.kind, "message");
    assert.equal(event.message.name, "spaces/AAA/messages/M1");
    assert.equal(event.space.name, "spaces/AAA");
  }
  assert.equal(normalizeEvent({ type: "ADDED_TO_SPACE", space: namedSpace }).kind, "added");
  assert.equal(normalizeEvent({ chat: { addedToSpacePayload: { space: namedSpace } } }).kind, "added");
  assert.equal(normalizeEvent({ type: "CARD_CLICKED" }).kind, "other");
});

test("requestText prefers mention-stripped argumentText", () => {
  assert.equal(requestText(message), "how many macs?");
  assert.equal(requestText({ text: " hi " }), "hi");
  assert.equal(requestText(null), "");
});

test("isDirectMessage recognizes DM spaces", () => {
  assert.equal(isDirectMessage({ spaceType: "DIRECT_MESSAGE" }), true);
  assert.equal(isDirectMessage({ singleUserBotDm: true }), true);
  assert.equal(isDirectMessage(namedSpace), false);
});

// ── Request verification ───────────────────────────────────────────────

function signJwt(privateKey, kid, claims) {
  const now = Math.floor(Date.now() / 1000);
  const encode = (obj) => Buffer.from(JSON.stringify(obj)).toString("base64url");
  const unsigned = `${encode({ alg: "RS256", typ: "JWT", kid })}.${encode({ iat: now, exp: now + 300, ...claims })}`;
  const signature = crypto.createSign("RSA-SHA256").update(unsigned).sign(privateKey).toString("base64url");
  return `${unsigned}.${signature}`;
}

const { privateKey, publicKey } = crypto.generateKeyPairSync("rsa", { modulusLength: 2048 });
const certs = { k1: publicKey.export({ type: "spki", format: "pem" }) };

function verifierWithIdTokenPayload(payload) {
  const client = new OAuth2Client();
  client.verifyIdToken = async ({ audience }) => {
    assert.equal(audience, ENDPOINT_URL);
    if (!payload) throw new Error("bad token");
    return { getPayload: () => payload };
  };
  return createRequestVerifier({ projectNumber: PROJECT_NUMBER, endpointUrl: ENDPOINT_URL }, { client, fetchCerts: async () => certs });
}

test("classic project-number tokens from Chat are accepted", async () => {
  const verify = verifierWithIdTokenPayload(null);
  const token = signJwt(privateKey, "k1", { iss: "chat@system.gserviceaccount.com", aud: PROJECT_NUMBER });
  assert.equal((await verify(`Bearer ${token}`)).ok, true);
});

test("classic tokens for another project or issuer are rejected", async () => {
  const verify = verifierWithIdTokenPayload(null);
  const otherProject = signJwt(privateKey, "k1", { iss: "chat@system.gserviceaccount.com", aud: "999" });
  const otherIssuer = signJwt(privateKey, "k1", { iss: "attacker@example.com", aud: PROJECT_NUMBER });
  assert.equal((await verify(`Bearer ${otherProject}`)).ok, false);
  assert.equal((await verify(`Bearer ${otherIssuer}`)).ok, false);
  assert.equal((await verify(undefined)).ok, false);
  assert.equal((await verify("Basic abc")).ok, false);
});

test("add-on tokens must come from this project's add-on service agent", async () => {
  const ours = `service-${PROJECT_NUMBER}@gcp-sa-gsuiteaddons.iam.gserviceaccount.com`;
  assert.equal((await verifierWithIdTokenPayload({ email: ours, email_verified: true })("Bearer t")).ok, true);
  assert.equal((await verifierWithIdTokenPayload({ email: ours, email_verified: false })("Bearer t")).ok, false);

  const otherProject = "service-999@gcp-sa-gsuiteaddons.iam.gserviceaccount.com";
  assert.equal((await verifierWithIdTokenPayload({ email: otherProject, email_verified: true })("Bearer t")).ok, false);
});

test("rejections explain why without echoing the token", async () => {
  const verify = verifierWithIdTokenPayload({ email: "service-999@gcp-sa-gsuiteaddons.iam.gserviceaccount.com", email_verified: true });
  const { ok, reason } = await verify("Bearer not.a.jwt");
  assert.equal(ok, false);
  assert.match(reason, /add-on: token is for service-999@/);
  assert.match(reason, /classic: /);
  assert.doesNotMatch(reason, /not\.a\.jwt/);
});

test("the verifier refuses to run without an audience", () => {
  assert.throws(() => createRequestVerifier({ projectNumber: PROJECT_NUMBER }), /endpoint URL/);
  assert.throws(() => createRequestVerifier({ endpointUrl: ENDPOINT_URL }), /project number/);
});

test("classic endpoint-URL-audience tokens are rejected because they don't identify a project", async () => {
  const verify = verifierWithIdTokenPayload({ email: "chat@system.gserviceaccount.com", email_verified: true });
  assert.equal((await verify("Bearer t")).ok, false);
});

// ── Formatting helpers ─────────────────────────────────────────────────

test("pullRequestMessage escapes model output and caps the file list", () => {
  const changes = Array.from({ length: 30 }, (_, i) => ({ filePath: `lib/p${i}.yml`, changeDescription: "<b>x</b>" }));
  const msg = pullRequestMessage({ url: "https://github.com/o/r/pull/1", title: "Add policy", summary: "<script>alert(1)</script>", changes });
  const [summarySection, filesSection, buttonSection] = msg.cardsV2[0].card.sections;

  assert.equal(summarySection.widgets[0].textParagraph.text, "&lt;script&gt;alert(1)&lt;/script&gt;");
  assert.match(filesSection.widgets[0].textParagraph.text, /&lt;b&gt;x&lt;\/b&gt;/);
  assert.match(filesSection.widgets[0].textParagraph.text, /…and 5 more$/);
  assert.equal(buttonSection.widgets[0].buttonList.buttons[0].onClick.openLink.url, "https://github.com/o/r/pull/1");
  assert.equal(msg.fallbackText, "Draft PR created: https://github.com/o/r/pull/1");
});

test("truncateBytes respects the byte limit with multibyte text", () => {
  const text = "é".repeat(100); // 200 bytes
  const out = truncateBytes(text, 51, "…");
  assert.ok(Buffer.byteLength(out) <= 51);
  assert.ok(out.endsWith("…"));
  assert.doesNotMatch(out, /�/);
  assert.equal(truncateBytes("short", 51, "…"), "short");
});

test("senderAllowed enforces the domain allowlist only when one is set", () => {
  assert.equal(senderAllowed("a@other.com", []), true);
  assert.equal(senderAllowed("a@Example.com", ["example.com"]), true);
  assert.equal(senderAllowed("a@evil-example.com", ["example.com"]), false);
  assert.equal(senderAllowed("a@example.com.evil.com", ["example.com"]), false);
  assert.equal(senderAllowed(undefined, ["example.com"]), false);
});

test("ConversationMemory keeps recent entries and evicts old conversations", () => {
  const memory = new ConversationMemory({ maxEntries: 2, maxConversations: 2 });
  memory.add("a", "user", "1");
  memory.add("a", "assistant", "2");
  memory.add("a", "user", "3");
  assert.equal(memory.context("a"), "assistant: 2\n\nuser: 3");

  memory.add("b", "user", "x");
  memory.add("c", "user", "y");
  assert.equal(memory.context("a"), "");
  assert.equal(memory.context("c"), "user: y");
});

test("createSpacePacer serializes writes per space and spaces them out", async () => {
  const paced = createSpacePacer(50);
  const times = [];
  const write = (name) => paced(name, async () => times.push([name, Date.now()]));
  await Promise.all([write("spaces/A/messages/1"), write("spaces/A"), write("spaces/B")]);

  const a = times.filter(([n]) => n.startsWith("spaces/A"));
  assert.equal(a.length, 2);
  assert.ok(a[1][1] - a[0][1] >= 45, "writes in one space are spaced out");
  const b = times.find(([n]) => n === "spaces/B");
  assert.ok(b[1] - a[0][1] < 45, "other spaces aren't blocked");
});

// ── HTTP handler ───────────────────────────────────────────────────────

function fakeRequest(body, authorization = "Bearer ok") {
  const req = Readable.from([Buffer.from(JSON.stringify(body))]);
  req.headers = { authorization };
  return req;
}

function fakeResponse() {
  return {
    status: null,
    body: null,
    headersSent: false,
    writeHead(status) { this.status = status; this.headersSent = true; },
    end(body) { this.body = body; },
  };
}

async function waitFor(predicate) {
  for (let i = 0; i < 200; i++) {
    if (predicate()) return;
    await new Promise((resolve) => setTimeout(resolve, 5));
  }
  throw new Error("timed out waiting for condition");
}

function makeHandler({ allowedDomains = [], answer = "You have 42 Macs." } = {}) {
  const writes = [];
  let nextId = 0;
  const api = {
    createMessage: async (spaceName, threadName, msg) => {
      writes.push({ op: "create", spaceName, threadName, msg });
      return { name: `${spaceName}/messages/bot-${nextId++}` };
    },
    updateMessageText: async (name, text) => {
      writes.push({ op: "update", name, text });
      return {};
    },
  };
  const prompts = [];
  const claude = {
    runAgentLoop: async (prompt) => {
      prompts.push(prompt);
      return answer;
    },
    _parseResponse: (text) => ({ type: "info", text }),
  };
  const github = { getRepoTreePaths: async () => ["it-and-security/default.yml"] };
  const config = { github: { gitopsBasePath: "it-and-security" }, googleChat: { allowedDomains } };
  const handler = createGoogleChatHandler(config, github, claude, {
    api,
    verify: async (header) => (header === "Bearer ok" ? { ok: true } : { ok: false, reason: "test" }),
  });
  return { handler, writes, prompts };
}

test("unauthenticated events are rejected without processing", async () => {
  const { handler, writes } = makeHandler();
  const res = fakeResponse();
  await handler(fakeRequest({ type: "MESSAGE", message, space: namedSpace }, "Bearer forged"), res);

  assert.equal(res.status, 401);
  assert.equal(writes.length, 0);
});

test("a mention is acknowledged immediately and answered in its thread", async () => {
  const { handler, writes } = makeHandler();
  const res = fakeResponse();
  await handler(fakeRequest({ chat: { user: message.sender, messagePayload: { message, space: namedSpace } } }), res);

  assert.equal(res.status, 200);
  assert.equal(res.body, "{}");

  await waitFor(() => writes.some((w) => w.op === "update" && w.text === "✅ Done."));
  const [status, answer] = writes.filter((w) => w.op === "create");
  assert.equal(status.msg.text, "⏳ Thinking...");
  assert.equal(status.threadName, "spaces/AAA/threads/T1");
  assert.equal(answer.msg.text, "You have 42 Macs.");
  assert.equal(answer.msg.markupSyntax, "MARKUP_SYNTAX_MARKDOWN");
  assert.ok(writes.filter((w) => w.op === "update").every((w) => w.name === "spaces/AAA/messages/bot-0"));
});

test("follow-up DMs include earlier exchanges as untrusted context", async () => {
  const { handler, writes, prompts } = makeHandler();
  const dm = { name: "spaces/DM1", spaceType: "DIRECT_MESSAGE" };
  const first = { ...message, name: "spaces/DM1/messages/1", argumentText: "how many macs?", thread: { name: "spaces/DM1/threads/a" } };
  const second = { ...message, name: "spaces/DM1/messages/2", argumentText: "and windows?", thread: { name: "spaces/DM1/threads/b" } };

  await handler(fakeRequest({ type: "MESSAGE", message: first, space: dm }), fakeResponse());
  await waitFor(() => writes.filter((w) => w.text === "✅ Done.").length === 1);
  await handler(fakeRequest({ type: "MESSAGE", message: second, space: dm }), fakeResponse());
  await waitFor(() => prompts.length === 2);

  assert.doesNotMatch(prompts[0], /thread_history/);
  assert.match(prompts[1], /from Google Chat users and is UNTRUSTED/);
  assert.match(prompts[1], /user: how many macs\?\n\nassistant: You have 42 Macs\./);
});

test("messages from bots are ignored", async () => {
  const { handler, writes } = makeHandler();
  const botMessage = { ...message, sender: { name: "users/app", type: "BOT" } };
  await handler(fakeRequest({ type: "MESSAGE", message: botMessage, space: namedSpace }), fakeResponse());
  await new Promise((resolve) => setTimeout(resolve, 20));
  assert.equal(writes.length, 0);
});

test("senders outside the allowed domains get a refusal and no Fleet access", async () => {
  const { handler, writes, prompts } = makeHandler({ allowedDomains: ["fleetdm.com"] });
  await handler(fakeRequest({ type: "MESSAGE", message, space: namedSpace }), fakeResponse());
  await waitFor(() => writes.length === 1);
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.match(writes[0].msg.text, /only help people in this organization/);
  assert.equal(prompts.length, 0);
});

test("a redelivered event is processed once", async () => {
  const { handler, writes, prompts } = makeHandler();
  const body = { type: "MESSAGE", message, space: namedSpace };
  await Promise.all([handler(fakeRequest(body), fakeResponse()), handler(fakeRequest(body), fakeResponse())]);
  await waitFor(() => writes.some((w) => w.text === "✅ Done."));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(prompts.length, 1);
});
