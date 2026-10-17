// In-memory conversation history keyed by Teams conversation ID.
//
// The Slack bot rebuilds context from the thread through the Slack API. Teams
// has no bot API for reading messages (that needs Graph permissions), so the
// bot remembers the exchanges it took part in instead. Personal chats have no
// threads at all, so an idle timeout stands in for "replied in the thread".
const DEFAULT_MAX_ENTRIES = 10;
const DEFAULT_TTL_MS = 24 * 60 * 60 * 1000;
const DEFAULT_MAX_CONVERSATIONS = 1000;

class ConversationHistory {
  constructor({
    maxEntries = DEFAULT_MAX_ENTRIES,
    ttlMs = DEFAULT_TTL_MS,
    maxConversations = DEFAULT_MAX_CONVERSATIONS,
    now = Date.now,
  } = {}) {
    this.maxEntries = maxEntries;
    this.ttlMs = ttlMs;
    this.maxConversations = maxConversations;
    this.now = now;
    this.conversations = new Map(); // id → { entries: [{ role, text }], updatedAt }
  }

  append(conversationId, role, text) {
    this._evictExpired();
    const existing = this.conversations.get(conversationId);
    const entries = existing ? existing.entries : [];
    entries.push({ role, text });
    if (entries.length > this.maxEntries) {
      entries.splice(0, entries.length - this.maxEntries);
    }
    // Re-insert so Map iteration order doubles as least-recently-used order.
    this.conversations.delete(conversationId);
    this.conversations.set(conversationId, { entries, updatedAt: this.now() });
    while (this.conversations.size > this.maxConversations) {
      this.conversations.delete(this.conversations.keys().next().value);
    }
  }

  get(conversationId) {
    this._evictExpired();
    const conversation = this.conversations.get(conversationId);
    return conversation ? [...conversation.entries] : [];
  }

  /**
   * Prior exchanges formatted like the Slack bot's thread context, or "" if none.
   */
  format(conversationId) {
    return this.get(conversationId)
      .map((e) => `${e.role}: ${e.text}`)
      .join("\n\n");
  }

  _evictExpired() {
    const cutoff = this.now() - this.ttlMs;
    for (const [id, conversation] of this.conversations) {
      if (conversation.updatedAt < cutoff) {
        this.conversations.delete(id);
      }
    }
  }
}

ConversationHistory.DEFAULT_MAX_ENTRIES = DEFAULT_MAX_ENTRIES;
ConversationHistory.DEFAULT_TTL_MS = DEFAULT_TTL_MS;
ConversationHistory.DEFAULT_MAX_CONVERSATIONS = DEFAULT_MAX_CONVERSATIONS;
module.exports = ConversationHistory;
