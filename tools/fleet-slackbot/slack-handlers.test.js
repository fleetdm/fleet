const { test } = require("node:test");
const assert = require("node:assert/strict");
const { ActivityLog } = require("./slack-handlers");

test("activity log fits in a chat.update message", () => {
  const log = new ActivityLog();
  const sql = "SELECT 'CIS check' AS chk, 1 AS pass UNION ALL ".repeat(50);
  for (let i = 0; i < 100; i++) {
    log.addToolCall("run_live_query", { host_ids: "1374", sql });
  }
  const text = log.format();
  assert.ok(text.length <= 4000, `got ${text.length} chars`);
  assert.match(text, /…and \d+ more$/);
});

test("short activity log is unchanged", () => {
  const log = new ActivityLog();
  log.addToolCall("get_host", { identifier: "mitch" });
  log.addToolCall("get_fleets", {});
  assert.equal(log.format(), ":mag: *Tools used:*\n• `get_host` — identifier=mitch\n• `get_fleets`");
});
