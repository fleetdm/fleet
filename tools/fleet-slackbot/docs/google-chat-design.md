# Google Chat support: design

## Summary

The bot can now run on Google Chat as well as Slack. Each process serves one platform, picked with `CHAT_PLATFORM`. Everything that isn't chat I/O is shared by both platforms: the Claude agent loop, Fleet MCP tools, GitOps path and YAML guards, draft PR creation, and the GitHub webhook (PR-comment revisions and CI auto-fix).

## Requirements

**Functional (parity with Slack)**

- Answer questions about the Fleet environment when @mentioned in a space or messaged in a DM.
- Turn change requests into a draft GitOps PR and post a link card in the thread.
- Show progress while the agent works, then the tools it used.
- Use earlier messages in the thread as context for follow-ups.
- Keep PR-comment revisions and CI auto-fix working unchanged.

**Non-functional**

- Accept only events Google signed for our Google Cloud project.
- Acknowledge each event within Chat's 30-second deadline. Agent runs take minutes.
- Stay within Chat's write quota of 1 write per second per space.
- Slack behavior and the Slack system prompt stay byte-for-byte unchanged.
- No new admin-approved scopes: the bot uses only `chat.bot`.

## Design

```
Slack (Socket Mode) ──► slack-handlers.js ────────┐
                         (Slack adapter)          │
                                                  ▼
Google Chat (HTTPS) ──► google-chat-handlers.js ─► request-pipeline.js ─► ClaudeClient ─► Fleet MCP / read_gitops_file / web_search
  POST /google-chat/events  (Chat adapter)        │                                    │
                                                  └─► GitHubClient (branch, commit, draft PR)
GitHub webhook ────────► webhook-handler.js (unchanged)
```

`request-pipeline.js` holds the code that used to live in `slack-handlers.js`. It builds the untrusted-input prompt, runs the agent, validates paths and YAML before creating a branch, opens the draft PR, and maps errors to messages that are safe to show. A platform adapter supplies seven methods:

| Method | Slack | Google Chat |
|---|---|---|
| `acknowledge()` | ⏳ reaction plus a "Thinking..." status message | "⏳ Thinking..." status message |
| `updateStatus()` | `chat.update` on the status message | `PATCH` on the status message |
| `postAnswer()` | threaded reply, mrkdwn, under 39,000 chars | threaded reply, Markdown, under 30,000 bytes |
| `postPullRequest()` | Block Kit header and section | `cardsV2` card with an "Open pull request" button |
| `postWarnings()` | threaded reply | threaded reply |
| `complete()` | activity log in the status message, ⏳ → ✅ | activity log in the status message |
| `fail()` | ⏳ → ❌, error reply | error reply, status shows ❌ |

The pipeline throttles status edits to one every 2 seconds on both platforms.

### Google Chat event flow

1. Google POSTs the event with `Authorization: Bearer <token>`.
2. The bot verifies the token before reading the body. Then it reads at most 1 MB, parses it, and returns `200 {}` immediately.
3. Classic Chat events (top-level `type`) and add-on events (`chat.messagePayload`) are mapped to one shape.
4. The bot skips senders that are bots, applies the optional domain allowlist, and dedupes on `message.name`.
5. The pipeline runs. Replies go through the Chat REST API with app credentials. Creates carry a fresh `requestId`, so a retried create can't post twice.

## Security model

**Request authentication** (`google-chat-auth.js`). Two token types are accepted, and both are bound to our project:

| App type | Token check |
|---|---|
| Add-on (console default) | Google ID token, `aud` = `GOOGLE_CHAT_ENDPOINT_URL`, `email` = `service-<project number>@gcp-sa-gsuiteaddons.iam.gserviceaccount.com`, `email_verified` |
| Classic, "Project number" audience | JWT signed with `chat@system.gserviceaccount.com` certs, issuer is that account, `aud` = `GOOGLE_CHAT_PROJECT_NUMBER` |

Classic apps can also use the "HTTP endpoint URL" audience, which the bot rejects. Those tokens carry only the shared Chat service account and the URL. A Chat app in any other project, pointed at our URL, would pass that check and could run Fleet queries or open PRs. A test covers this.

**Who can use the bot.**

- The console's Visibility setting limits which people or groups can add the app.
- `GOOGLE_CHAT_ALLOWED_DOMAINS` restricts requests by sender email domain. If the sender has no email, the request is rejected. This is optional, as in Slack, which has no equivalent.

**Shared guards** (unchanged, now in one place for chat requests):

- User text and thread history are wrapped as untrusted. Google Chat, unlike Slack, doesn't escape `<` in message text, so users could close the `<user_input>` and `<thread_history>` tags themselves. Those tags are now neutralized inside user text.
- Changed paths must be `default.yml`, `fleets/`, or `lib/`, with no traversal.
- Placeholder content and the YAML schema are checked before a branch is created.
- PRs are always drafts.

**Output escaping.** Card text uses Chat's HTML subset, so the summary, file paths, and descriptions are HTML-escaped.

## Reliability and limits

- **Startup checks.** The bot gets a Chat API access token before it starts listening, and exits with a clear message if it can't. Once Chat has the `200` ack, the bot has no way left to tell a user that its credentials are broken.
- **Failures after a PR opens.** If a chat post fails after the PR exists, the user is told the PR was opened, with its link, and that there's no need to retry. This applies on both platforms and prevents duplicate PRs. If only the final status edit fails, a delivered answer isn't reported as a failure.

- **Quota.** Writes are serialized per space with at least 1.1 seconds between them. 429 and 5xx responses are retried 3 times with backoff.
- **Size.** Chat limits a message to 32,000 bytes. Answers and status text are truncated at 30,000 bytes on a character boundary. The PR card shows at most 25 files and a 2,000-character summary.
- **Redelivery.** Chat may redeliver an event that timed out. The bot acks instantly, and events already being processed are deduped by message name.
- **Memory.** Conversation memory holds the last 10 entries for each of up to 500 conversations. It keys on the thread in spaces and on the whole DM, since DMs are often unthreaded. It lives in the process and is lost on restart.

## Trade-offs and alternatives

| Choice | Alternative | Why |
|---|---|---|
| Shared pipeline in this package | Standalone `fleet-google-chat-bot` copy | One copy of the security-critical guards. The cost is a refactor of `slack-handlers.js`, but it only moves code, and tests pin the Slack behavior. |
| One platform per process | Both platforms in one process | Keeps the Slack startup path identical and each deployment's config small. |
| HTTP endpoint | Pub/Sub | The bot already needs a public HTTPS endpoint for GitHub webhooks. Pub/Sub for add-on apps is still in Developer Preview. |
| `google-auth-library` with REST calls | `@googleapis/chat` or `@google-apps/chat` | One dependency covers token verification and service-account auth. The bot makes only two Chat API calls. |
| In-process memory for context | `spaces.messages.list` with `chat.app.messages.readonly` | That scope is restricted and needs one-time Workspace admin approval. The memory is lost on restart, and it can't see messages the bot wasn't mentioned in. |
| No emoji reactions | Domain-wide delegation to react as a user | The reaction would appear as a real person. The status message carries the same signal. |
| Markdown (`MARKUP_SYNTAX_MARKDOWN`) | Legacy Chat syntax | GA since August 2026. Model output is already Markdown-shaped. Status edits avoid `*` so they render the same under either syntax. |
| `google-auth-library` 10.9.1 | 11.x | 11.x requires Node 22. This package still targets Node 20 to match CI. |

## Acceptance criteria

Each criterion names the evidence that covers it.

- [x] Forged, other-project, other-issuer and URL-audience tokens get 401, and nothing is processed. The log gives the reason without the token. Evidence: `google-chat.test.js`, plus local `curl` against the running server.
- [x] A malformed request line gets 400 and the server stays up. Missing or invalid Google credentials stop startup with a message that names the setting. Evidence: local smoke test.
- [x] Our project's add-on tokens and classic tokens are accepted. Evidence: `google-chat.test.js`.
- [x] An @mention is acked with `200 {}` before any agent work. The answer lands in the event's thread with Markdown. Evidence: `google-chat.test.js`.
- [x] A follow-up DM includes the earlier exchange as untrusted thread history. Evidence: `google-chat.test.js`.
- [x] Bot senders are ignored, senders outside the allowlist get a refusal and no Fleet access, and a redelivered event runs once. Evidence: `google-chat.test.js`.
- [x] Card content is HTML-escaped, and output stays within Chat's byte limit even with multibyte text. Evidence: `google-chat.test.js`.
- [x] Path traversal is rejected before a branch exists, and draft PRs go on `fleet/<hash>` branches. Evidence: `request-pipeline.test.js`.
- [x] A chat failure after the PR exists reports the PR link instead of "try again". Only Anthropic errors are described as AI-service rate limits. Users can't close the untrusted-input tags. Evidence: `request-pipeline.test.js`.
- [x] The Slack adapter keeps its reactions, status text, Block Kit PR message, truncation, and error flow. Evidence: `slack-handlers.test.js`.
- [x] The Slack system prompt is byte-identical to the one before this change. Checked against `origin/main` during development.
- [ ] End to end against a real Google Chat app (needs a Workspace project): a DM question, an @mention in a space, a change request that opens a draft PR, and a follow-up in the thread.

## Revisit as this grows

- Slack mode has the same malformed-request-line crash, inside Bolt's `SocketModeReceiver` HTTP server. Fixing that belongs upstream in Bolt, or in a process-level handler, in a separate change.

- Move to Node 22 or later. Node 20 reached end of life in April 2026, and current Google client libraries require 22. This affects the Slack deployment and CI too.
- The default model (`claude-opus-4-6`) and the web search tool version are a generation behind. A model update is its own change.
- If thread history across restarts matters, ask a Workspace admin to approve `chat.app.messages.readonly` and read history with `spaces.messages.list`.
- If the package keeps growing platforms, rename `tools/fleet-slackbot` to something platform-neutral, and update the Render `rootDir`, the CI workflow, and Dependabot together.
