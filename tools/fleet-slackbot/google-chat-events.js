/**
 * Normalize a Google Chat event into one shape. Chat apps configured as
 * Google Workspace add-ons nest the payload under `chat.*`; classic Chat
 * apps put `type`, `message`, and `space` at the top level.
 *
 * Returns { kind, message, space, user, addOn } where kind is
 * "message" | "added" | "removed" | "other".
 */
function normalizeEvent(body) {
  if (body && body.chat) {
    const chat = body.chat;
    if (chat.messagePayload) {
      return { kind: "message", message: chat.messagePayload.message, space: chat.messagePayload.space, user: chat.user, addOn: true };
    }
    if (chat.addedToSpacePayload) {
      return { kind: "added", message: null, space: chat.addedToSpacePayload.space, user: chat.user, addOn: true };
    }
    if (chat.removedFromSpacePayload) {
      return { kind: "removed", message: null, space: chat.removedFromSpacePayload.space, user: chat.user, addOn: true };
    }
    return { kind: "other", message: null, space: null, user: chat.user, addOn: true };
  }

  const kinds = { MESSAGE: "message", ADDED_TO_SPACE: "added", REMOVED_FROM_SPACE: "removed" };
  return {
    kind: kinds[body && body.type] || "other",
    message: (body && body.message) || null,
    space: (body && body.space) || null,
    user: (body && body.user) || null,
    addOn: false,
  };
}

function isDirectMessage(space) {
  return Boolean(space) && (space.spaceType === "DIRECT_MESSAGE" || space.type === "DM" || space.singleUserBotDm === true);
}

/**
 * Text of the user's request without the app @mention. `argumentText` is
 * Chat's mention-stripped text; fall back to `text` (DMs have no mention).
 */
function requestText(message) {
  return ((message && (message.argumentText ?? message.text)) || "").trim();
}

module.exports = { normalizeEvent, isDirectMessage, requestText };
