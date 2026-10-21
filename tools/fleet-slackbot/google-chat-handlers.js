const crypto = require("crypto");
const { GoogleAuth } = require("google-auth-library");
const { processRequest } = require("./request-pipeline");
const { createRequestVerifier } = require("./google-chat-auth");
const { normalizeEvent, isDirectMessage, requestText } = require("./google-chat-events");

const CHAT_API = "https://chat.googleapis.com/v1";
const MAX_BODY_SIZE = 1024 * 1024;
// Chat caps a whole message (text + cards) at 32,000 bytes.
const MAX_MESSAGE_BYTES = 30000;
// Chat allows one write per second per space, shared by every app in it.
const SPACE_WRITE_INTERVAL_MS = 1100;
const MAX_CARD_FILES = 25;
const MAX_CARD_SUMMARY_CHARS = 2000;
const MAX_MEMORY_ENTRY_CHARS = 4000;

const STATUS_TEXT = {
  analyzing: () => "🔍 Querying Fleet and analyzing your request...",
  tool: (tool) => `⚙️ Calling Fleet tool: \`${tool}\`...`,
  creating_pr: () => "🛠️ Creating pull request...",
};

function truncateBytes(text, maxBytes, notice) {
  if (Buffer.byteLength(text) <= maxBytes) return text;
  // Cutting mid-character leaves a replacement character; drop it.
  const cut = Buffer.from(text).subarray(0, maxBytes - Buffer.byteLength(notice)).toString().replace(/\uFFFD$/, "");
  return cut + notice;
}

function escapeHtml(text) {
  return String(text).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

// Status and activity text avoids bold/italics: edits use the app-level
// markup default, and Google Chat's legacy syntax and Markdown disagree on `*`.
function formatActivity(lines) {
  return lines.length > 0 ? `🔍 Tools used:\n${lines.join("\n")}` : null;
}

/**
 * Card text uses Chat's HTML subset, so everything from the model is escaped.
 */
function pullRequestMessage({ url, title, summary, changes }) {
  const shown = changes.slice(0, MAX_CARD_FILES);
  const fileLines = shown.map((c) => `• <code>${escapeHtml(c.filePath)}</code> — ${escapeHtml(c.changeDescription)}`);
  if (changes.length > shown.length) {
    fileLines.push(`…and ${changes.length - shown.length} more`);
  }
  const shortSummary = summary.length > MAX_CARD_SUMMARY_CHARS ? `${summary.slice(0, MAX_CARD_SUMMARY_CHARS)}…` : summary;

  return {
    fallbackText: `Draft PR created: ${url}`,
    cardsV2: [
      {
        cardId: "draft-pr",
        card: {
          header: { title: "Draft PR created", subtitle: title },
          sections: [
            { widgets: [{ textParagraph: { text: escapeHtml(shortSummary) } }] },
            { header: "Files changed", widgets: [{ textParagraph: { text: fileLines.join("<br>") } }] },
            { widgets: [{ buttonList: { buttons: [{ text: "Open pull request", onClick: { openLink: { url } } }] } }] },
          ],
        },
      },
    ],
  };
}

/**
 * Serialize writes per space and keep at least `intervalMs` between them.
 * `resourceName` is a space, message, or thread name ("spaces/X/...").
 */
function createSpacePacer(intervalMs) {
  const lastWriteBySpace = new Map();
  const queueBySpace = new Map();

  return function paced(resourceName, fn) {
    const space = resourceName.split("/").slice(0, 2).join("/");
    const run = async () => {
      const wait = (lastWriteBySpace.get(space) || 0) + intervalMs - Date.now();
      if (wait > 0) await new Promise((resolve) => setTimeout(resolve, wait));
      try {
        return await fn();
      } finally {
        lastWriteBySpace.set(space, Date.now());
      }
    };
    const result = (queueBySpace.get(space) || Promise.resolve()).then(run);
    queueBySpace.set(space, result.catch(() => {}));
    return result;
  };
}

/**
 * Minimal Chat API client using app authentication (chat.bot scope). Writes
 * are paced per space, and 429/5xx responses are retried. Creates carry a
 * requestId so a retried create can't post twice.
 */
function parseCredentials(credentials) {
  try {
    return JSON.parse(credentials);
  } catch {
    // Don't echo the value: it's a private key.
    throw new Error("GOOGLE_CHAT_CREDENTIALS must be the service account key JSON itself, not a path (use GOOGLE_APPLICATION_CREDENTIALS for a key file)");
  }
}

function createChatApi({ credentials }) {
  const auth = new GoogleAuth({
    scopes: ["https://www.googleapis.com/auth/chat.bot"],
    ...(credentials ? { credentials: parseCredentials(credentials) } : {}),
  });
  const paced = createSpacePacer(SPACE_WRITE_INTERVAL_MS);

  const request = async (opts) => {
    const client = await auth.getClient();
    const res = await client.request({
      ...opts,
      retryConfig: {
        retry: 3,
        retryDelay: 1000,
        httpMethodsToRetry: ["POST", "PATCH"],
        statusCodesToRetry: [[429, 429], [500, 599]],
        onRetryAttempt: (err) => console.warn(`[gchat] Retrying ${opts.method} after HTTP ${err.response?.status ?? "error"}`),
      },
    });
    return res.data;
  };

  return {
    // Fail at startup, not on the first message: once Chat has our 200 ack,
    // the bot has no way left to tell the user its credentials are broken.
    checkCredentials: () => auth.getAccessToken(),
    createMessage: (spaceName, threadName, message) =>
      paced(spaceName, () =>
        request({
          url: `${CHAT_API}/${spaceName}/messages`,
          method: "POST",
          params: { messageReplyOption: "REPLY_MESSAGE_FALLBACK_TO_NEW_THREAD", requestId: crypto.randomUUID() },
          data: { ...message, ...(threadName ? { thread: { name: threadName } } : {}) },
        })
      ),
    updateMessageText: (messageName, text) =>
      paced(messageName, () =>
        request({
          url: `${CHAT_API}/${messageName}`,
          method: "PATCH",
          params: { updateMask: "text" },
          data: { text },
        })
      ),
  };
}

/**
 * Recent exchanges per conversation, kept in memory. Reading thread history
 * from the Chat API needs an admin-approved restricted scope
 * (chat.app.messages.readonly), so the bot only remembers what it saw. It
 * starts empty after a restart.
 */
class ConversationMemory {
  constructor({ maxEntries = 10, maxConversations = 500 } = {}) {
    this.maxEntries = maxEntries;
    this.maxConversations = maxConversations;
    this.conversations = new Map();
  }

  add(key, role, text) {
    const entries = this.conversations.get(key) || [];
    this.conversations.delete(key);
    entries.push({ role, text: text.length > MAX_MEMORY_ENTRY_CHARS ? `${text.slice(0, MAX_MEMORY_ENTRY_CHARS)}…` : text });
    this.conversations.set(key, entries.slice(-this.maxEntries));
    if (this.conversations.size > this.maxConversations) {
      this.conversations.delete(this.conversations.keys().next().value);
    }
  }

  context(key) {
    return (this.conversations.get(key) || []).map((e) => `${e.role}: ${e.text}`).join("\n\n");
  }
}

function emailDomain(email) {
  const at = (email || "").lastIndexOf("@");
  return at === -1 ? "" : email.slice(at + 1).toLowerCase();
}

function senderAllowed(email, allowedDomains) {
  return allowedDomains.length === 0 || allowedDomains.includes(emailDomain(email));
}

/**
 * Build the Google Chat adapter for one request. Chat apps can't add emoji
 * reactions with app auth, so progress and the final outcome live in the
 * status message.
 */
function createGoogleChatChat({ api, spaceName, threadName, memory, memoryKey, logPrefix }) {
  let statusName;

  const post = (message) => api.createMessage(spaceName, threadName, message);
  const postText = (text) => post({ text: truncateBytes(text, MAX_MESSAGE_BYTES, "\n\n_…response truncated due to length._"), markupSyntax: "MARKUP_SYNTAX_MARKDOWN" });
  const setStatusText = (text) => api.updateMessageText(statusName, truncateBytes(text, MAX_MESSAGE_BYTES, "\n…"));

  return {
    async acknowledge() {
      statusName = (await postText("⏳ Thinking...")).name;
    },

    async updateStatus({ phase, tool }) {
      await setStatusText(STATUS_TEXT[phase](tool));
    },

    async postAnswer(text) {
      await postText(text);
      memory.add(memoryKey, "assistant", text);
    },

    async postPullRequest(pr) {
      await post(pullRequestMessage(pr));
      memory.add(memoryKey, "assistant", `Opened draft PR "${pr.title}": ${pr.url}\n${pr.summary}`);
    },

    async postWarnings(warnings) {
      await postText(`⚠️ **Validation warnings:**\n${warnings.map((w) => `- ${w}`).join("\n")}`);
    },

    async complete(activityLines) {
      await setStatusText(formatActivity(activityLines) || "✅ Done.");
    },

    async fail(userMessage, activityLines) {
      try {
        await postText(`❌ **Error:** ${userMessage}`);
      } catch (err) {
        console.error(`${logPrefix} Failed to post error: ${err.message}`);
      }
      if (statusName) {
        await setStatusText(formatActivity(activityLines) || "❌ Failed.").catch((err) =>
          console.warn(`${logPrefix} Failed to set failure status: ${err.message}`)
        );
      }
    },
  };
}

async function readBody(req) {
  const buffers = [];
  let totalSize = 0;
  for await (const chunk of req) {
    totalSize += chunk.length;
    if (totalSize > MAX_BODY_SIZE) return null;
    buffers.push(chunk);
  }
  return Buffer.concat(buffers).toString("utf-8");
}

function sendJson(res, status, body) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(body));
}

/**
 * HTTP handler for Google Chat events. It acknowledges within Chat's
 * 30-second deadline with an empty response, then replies asynchronously
 * through the Chat API.
 */
function createGoogleChatHandler(config, github, claude, deps = {}) {
  const verify = deps.verify || createRequestVerifier(config.googleChat);
  const api = deps.api || createChatApi(config.googleChat);
  const memory = deps.memory || new ConversationMemory();
  const processing = new Set();

  async function handleMessage(event) {
    const message = event.message;
    const sender = message.sender || event.user;
    if (!sender || sender.type === "BOT") return;

    const space = event.space || message.space;
    const threadName = message.thread && message.thread.name;
    const logPrefix = isDirectMessage(space) ? "[gchat-dm]" : "[gchat-mention]";

    const email = sender.email || (event.user && event.user.email);
    if (!senderAllowed(email, config.googleChat.allowedDomains)) {
      console.log(`${logPrefix} Refusing sender outside GOOGLE_CHAT_ALLOWED_DOMAINS (domain: ${emailDomain(email) || "no email on event"})`);
      await api.createMessage(space.name, threadName, { text: "Sorry, I can only help people in this organization's Google Workspace." });
      return;
    }

    const userText = requestText(message);
    if (!userText) return;

    // Chat may redeliver an event it thinks timed out
    if (processing.has(message.name)) {
      console.log(`${logPrefix} Skipping duplicate event for ${message.name}`);
      return;
    }
    processing.add(message.name);

    try {
      console.log(`${logPrefix} Message in ${space.name}: "${userText.slice(0, 100)}"`);

      // DMs often aren't threaded, so remember the whole DM; elsewhere, the thread.
      const memoryKey = isDirectMessage(space) ? space.name : threadName || space.name;
      const threadContext = memory.context(memoryKey);
      memory.add(memoryKey, "user", userText);

      const chat = createGoogleChatChat({ api, spaceName: space.name, threadName, memory, memoryKey, logPrefix });
      await processRequest({
        userText,
        userId: sender.name,
        threadContext,
        platformName: "Google Chat",
        chat,
        config,
        github,
        claude,
        logPrefix,
      });
    } finally {
      processing.delete(message.name);
    }
  }

  return async function handleGoogleChatEvent(req, res) {
    // The token is in a header, so reject before buffering the body.
    const verdict = await verify(req.headers.authorization);
    if (!verdict.ok) {
      console.log(`[gchat] Rejecting request: ${verdict.reason}. Check GOOGLE_CHAT_PROJECT_NUMBER and GOOGLE_CHAT_ENDPOINT_URL.`);
      res.writeHead(401);
      res.end("Unauthorized");
      return;
    }

    const rawBody = await readBody(req);
    if (rawBody === null) {
      console.log("[gchat] Rejecting request body over 1MB");
      res.writeHead(413);
      res.end("Request body too large");
      return;
    }

    let event;
    try {
      event = normalizeEvent(JSON.parse(rawBody));
    } catch (err) {
      console.error("[gchat] Invalid JSON payload:", err.message);
      sendJson(res, 400, { ok: false });
      return;
    }

    sendJson(res, 200, {});

    if (event.kind !== "message" || !event.message) return;
    handleMessage(event).catch((err) => {
      // The Chat API error body says why, e.g. API disabled or app not in the space.
      const detail = err.response?.data?.error?.message || err.message;
      console.error(`[gchat] Failed handling ${event.message.name}: ${err.response?.status ?? ""} ${detail}`);
    });
  };
}

module.exports = {
  createChatApi,
  createGoogleChatHandler,
  createGoogleChatChat,
  createSpacePacer,
  ConversationMemory,
  pullRequestMessage,
  senderAllowed,
  truncateBytes,
};
