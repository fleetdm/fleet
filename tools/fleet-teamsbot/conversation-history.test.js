const { test } = require("node:test");
const assert = require("node:assert/strict");
const ConversationHistory = require("./conversation-history");

function makeClock(start = 1_000_000) {
  let current = start;
  const now = () => current;
  now.advance = (ms) => {
    current += ms;
  };
  return now;
}

test("append and get keep entries in order per conversation", () => {
  const history = new ConversationHistory();
  history.append("a", "user", "hello");
  history.append("a", "assistant", "hi there");
  history.append("b", "user", "unrelated");

  assert.deepEqual(history.get("a"), [
    { role: "user", text: "hello" },
    { role: "assistant", text: "hi there" },
  ]);
  assert.deepEqual(history.get("b"), [{ role: "user", text: "unrelated" }]);
  assert.deepEqual(history.get("missing"), []);
});

test("format renders prior exchanges like Slack thread context", () => {
  const history = new ConversationHistory();
  assert.equal(history.format("a"), "");

  history.append("a", "user", "How many hosts?");
  history.append("a", "assistant", "42 hosts.");
  assert.equal(history.format("a"), "user: How many hosts?\n\nassistant: 42 hosts.");
});

test("only the most recent maxEntries are kept", () => {
  const history = new ConversationHistory({ maxEntries: 3 });
  for (let i = 1; i <= 5; i++) {
    history.append("a", "user", `message ${i}`);
  }
  assert.deepEqual(
    history.get("a").map((e) => e.text),
    ["message 3", "message 4", "message 5"]
  );
});

test("conversations idle longer than the TTL are forgotten", () => {
  const now = makeClock();
  const history = new ConversationHistory({ ttlMs: 1000, now });
  history.append("a", "user", "first");

  now.advance(999);
  assert.equal(history.get("a").length, 1);

  // Activity on the conversation keeps it alive.
  history.append("a", "assistant", "reply");
  now.advance(999);
  assert.equal(history.get("a").length, 2);

  now.advance(2);
  assert.deepEqual(history.get("a"), []);
  assert.equal(history.format("a"), "");
});

test("the least recently used conversations are evicted past maxConversations", () => {
  const history = new ConversationHistory({ maxConversations: 2 });
  history.append("a", "user", "1");
  history.append("b", "user", "2");
  history.append("a", "user", "3"); // touches "a", so "b" is now the oldest
  history.append("c", "user", "4");

  assert.equal(history.get("b").length, 0);
  assert.equal(history.get("a").length, 2);
  assert.equal(history.get("c").length, 1);
});
