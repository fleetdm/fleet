const { processRequest, truncateText } = require("./request-pipeline");

// Slack messages have a ~40,000 char limit; truncate to be safe
const MAX_SLACK_TEXT = 39000;

const STATUS_TEXT = {
  analyzing: () => ":mag: Querying Fleet and analyzing your request...",
  tool: (tool) => `:gear: Calling Fleet tool: \`${tool}\`...`,
  creating_pr: () => ":hammer_and_wrench: Creating pull request...",
};

function formatActivity(lines) {
  return lines.length > 0 ? `:mag: *Tools used:*\n${lines.join("\n")}` : null;
}

/**
 * Build the Slack adapter for one request: replies go in the thread, and an
 * hourglass reaction on the user's message flips to a check or X when done.
 */
function createSlackChat({ client, channelId, threadTs, messageTs, logPrefix }) {
  const reactTs = messageTs || threadTs;
  let statusMsg;

  const post = (message) => client.chat.postMessage({ channel: channelId, thread_ts: threadTs, ...message });
  const setStatusText = (text) => client.chat.update({ channel: channelId, ts: statusMsg.ts, text });

  // Swap the hourglass reaction for a result emoji
  const setReaction = async (name) => {
    try {
      await client.reactions.remove({ channel: channelId, timestamp: reactTs, name: "hourglass_flowing_sand" });
    } catch { /* may already be removed */ }
    try {
      await client.reactions.add({ channel: channelId, timestamp: reactTs, name });
    } catch (err) {
      console.warn(`${logPrefix} Failed to set reaction: ${err.message}`);
    }
  };

  return {
    async acknowledge() {
      try {
        await client.reactions.add({ channel: channelId, timestamp: reactTs, name: "hourglass_flowing_sand" });
      } catch (err) {
        console.warn(`${logPrefix} Failed to add reaction: ${err.message}`);
      }
      // Status message that we'll update with progress
      statusMsg = await post({ text: ":hourglass_flowing_sand: Thinking..." });
    },

    async updateStatus({ phase, tool }) {
      await setStatusText(STATUS_TEXT[phase](tool));
    },

    async postAnswer(text) {
      // Post final answer as a new reply (triggers notification)
      await post({ text: truncateText(text, MAX_SLACK_TEXT, "\n\n_…response truncated due to length._", logPrefix) });
    },

    async postPullRequest({ url, title, summary, changes }) {
      const fileList = changes
        .map((c) => `• \`${c.filePath}\` — ${c.changeDescription}`)
        .join("\n");

      await post({
        blocks: [
          {
            type: "header",
            text: { type: "plain_text", text: "Draft PR Created" },
          },
          {
            type: "section",
            text: {
              type: "mrkdwn",
              text: `:white_check_mark: *<${url}|${title}>*\n\n${summary}\n\n*Files changed:*\n${fileList}`,
            },
          },
        ],
        text: `Draft PR created: ${url}`,
      });
    },

    async postWarnings(warnings) {
      await post({ text: `:warning: *Validation warnings:*\n${warnings.map((w) => `• ${w}`).join("\n")}` });
    },

    async complete(activityLines) {
      await setStatusText(formatActivity(activityLines) || ":white_check_mark: Done.");
      await setReaction("white_check_mark");
    },

    async fail(userMessage, activityLines) {
      await setReaction("x");
      try {
        // Post error as a new reply (triggers notification)
        await post({ text: `:x: *Error:* ${userMessage}` });
      } catch (err) {
        console.error(`${logPrefix} Failed to post error: ${err.message}`);
      }
      if (statusMsg) {
        await setStatusText(formatActivity(activityLines) || ":x: Failed.").catch(() => {});
      }
    },
  };
}

/**
 * Core request handler shared by the @mention and DM entry points.
 */
async function handleRequest({ userText, userId, channelId, threadTs, messageTs, threadContext, client, config, github, claude, logPrefix }) {
  const chat = createSlackChat({ client, channelId, threadTs, messageTs, logPrefix });
  await processRequest({ userText, userId, threadContext, platformName: "Slack", chat, config, github, claude, logPrefix });
}

/**
 * Fetch recent thread context as a formatted string.
 */
async function getThreadContext(client, channelId, threadTs) {
  try {
    const replies = await client.conversations.replies({
      channel: channelId,
      ts: threadTs,
      limit: 20,
    });
    return (replies.messages || [])
      .slice(-10)
      .map((m) => {
        const role = m.bot_id ? "assistant" : "user";
        return `${role}: ${m.text}`;
      })
      .join("\n\n");
  } catch (err) {
    console.warn(`Could not fetch thread history: ${err.message}`);
    return "";
  }
}

/**
 * Register all Slack event handlers on the Bolt app.
 */
function registerHandlers(app, config, github, claude) {
  // Track messages already being processed to prevent duplicates (e.g., message edits)
  const processingMessages = new Set();

  /**
   * Check if a channel ID is a DM (starts with D).
   */
  function isDM(channelId) {
    return channelId.startsWith("D");
  }

  // ── @mentions in channels (starts new conversations) ──────────────────
  app.event("app_mention", async ({ event, client }) => {
    const channelId = event.channel;
    const threadTs = event.thread_ts || event.ts;
    const userText = (event.text || "").replace(/<@[A-Z0-9]+>/g, "").trim();

    if (!userText) return;

    // Deduplicate: skip if we're already processing this message (e.g., edit fired a second event)
    const dedupeKey = `${channelId}:${event.ts}`;
    if (processingMessages.has(dedupeKey)) {
      console.log(`[mention] Skipping duplicate event for ${dedupeKey}`);
      return;
    }
    processingMessages.add(dedupeKey);

    try {
      console.log(`[mention] @mention in ${channelId}: "${userText.slice(0, 100)}"`);

      const threadContext = event.thread_ts
        ? await getThreadContext(client, channelId, event.thread_ts)
        : "";

      await handleRequest({
        userText,
        userId: event.user,
        channelId,
        threadTs,
        messageTs: event.ts,
        threadContext,
        client,
        config,
        github,
        claude,
        logPrefix: "[mention]",
      });
    } finally {
      processingMessages.delete(dedupeKey);
    }
  });

  // ── Messages: DM conversations only ─────────────────────────────────
  // In channels, the bot only responds to @mentions (handled by app_mention above).
  // In DMs, the bot responds to every message — no @mention needed.
  app.message(async ({ message, client }) => {
    const event = message;

    // Skip bot messages, edits, deletions, etc. (but allow file_share so
    // messages with image attachments are still processed)
    if (event.subtype && event.subtype !== "file_share") return;
    if (event.bot_id) return;

    const channelId = event.channel;

    // Only auto-respond in DMs
    if (!isDM(channelId)) return;

    const userText = (event.text || "").replace(/<@[A-Z0-9]+>/g, "").trim();
    if (!userText) return;

    // Deduplicate: skip if we're already processing this message
    const dedupeKey = `${channelId}:${event.ts}`;
    if (processingMessages.has(dedupeKey)) {
      console.log(`[dm] Skipping duplicate event for ${dedupeKey}`);
      return;
    }
    processingMessages.add(dedupeKey);

    try {
      // Use thread if already in one, otherwise start a new thread from this message
      const threadTs = event.thread_ts || event.ts;

      console.log(`[dm] Message from ${event.user}: "${userText.slice(0, 100)}"`);

      const threadContext = event.thread_ts
        ? await getThreadContext(client, channelId, event.thread_ts)
        : "";

      await handleRequest({
        userText,
        userId: event.user,
        channelId,
        threadTs,
        messageTs: event.ts,
        threadContext,
        client,
        config,
        github,
        claude,
        logPrefix: "[dm]",
      });
    } finally {
      processingMessages.delete(dedupeKey);
    }
  });
}

module.exports = { registerHandlers, createSlackChat };
