const { test } = require("node:test");
const assert = require("node:assert/strict");
const { stripBotMention, isBotMentioned, buildPrCard, textMessage } = require("./teams-handlers");

const BOT_ID = "28:00000000-0000-0000-0000-000000000000";

function mentionEntity(id, name) {
  return { type: "mention", text: `<at>${name}</at>`, mentioned: { id, name } };
}

test("stripBotMention removes the bot mention but keeps other users' names", () => {
  const activity = {
    text: "<at>Fleet</at> how many hosts does <at>Jane Doe</at> own?",
    recipient: { id: BOT_ID, name: "Fleet" },
    entities: [mentionEntity(BOT_ID, "Fleet"), mentionEntity("29:user", "Jane Doe")],
  };
  assert.equal(stripBotMention(activity), "how many hosts does Jane Doe own?");
});

test("stripBotMention tolerates missing text, entities, and stray <at> tags", () => {
  assert.equal(stripBotMention({}), "");
  assert.equal(stripBotMention({ text: "  <at>Someone</at> hello  " }), "Someone hello");
  assert.equal(stripBotMention({ text: "line one\nline two", entities: [] }), "line one\nline two");
});

test("isBotMentioned only matches a mention of the recipient", () => {
  const base = { recipient: { id: BOT_ID } };
  assert.equal(isBotMentioned({ ...base, entities: [mentionEntity(BOT_ID, "Fleet")] }), true);
  assert.equal(isBotMentioned({ ...base, entities: [mentionEntity("29:user", "Jane")] }), false);
  assert.equal(isBotMentioned({ ...base }), false);
  assert.equal(isBotMentioned({ entities: [mentionEntity(BOT_ID, "Fleet")] }), false);
});

test("textMessage opts into extended Markdown", () => {
  assert.deepEqual(textMessage("**hi**"), { type: "message", text: "**hi**", textFormat: "extendedmarkdown" });
});

test("buildPrCard links to the PR and lists the changed files", () => {
  const pr = { url: "https://github.com/fleetdm/fleet/pull/1", number: 1 };
  const result = {
    prTitle: "Add Firefox policy",
    summary: "Adds a policy checking that Firefox is installed on workstations.",
    changes: [
      { filePath: "lib/macos/policies/firefox-installed.yml", changeDescription: "New policy" },
      { filePath: "fleets/workstations.yml", changeDescription: "Reference the new policy" },
    ],
  };

  const card = buildPrCard(pr, result);
  assert.equal(card.type, "AdaptiveCard");
  assert.deepEqual(card.actions, [{ type: "Action.OpenUrl", title: "Open pull request", url: pr.url }]);

  const text = card.body.map((b) => b.text).join("\n");
  assert.match(text, /\[Add Firefox policy\]\(https:\/\/github\.com\/fleetdm\/fleet\/pull\/1\)/);
  assert.match(text, /- lib\/macos\/policies\/firefox-installed\.yml — New policy/);
  assert.match(text, /- fleets\/workstations\.yml — Reference the new policy/);
  assert.ok(card.body.every((b) => b.wrap === true), "every TextBlock should wrap");
});
