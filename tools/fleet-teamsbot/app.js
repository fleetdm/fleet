const express = require("express");
const { rateLimit } = require("express-rate-limit");
const { ActivityHandler, CloudAdapter } = require("@microsoft/agents-hosting");
const config = require("./config");
const GitHubClient = require("./github-client");
const ClaudeClient = require("./claude-client");
const McpClient = require("./mcp-client");
const { registerHandlers } = require("./teams-handlers");
const { createWebhookHandler } = require("./webhook-handler");

const github = new GitHubClient({
  token: config.github.token,
  repo: config.github.repo,
  baseBranch: config.github.baseBranch,
  gitopsBasePath: config.github.gitopsBasePath,
});

const mcpClient = new McpClient({
  url: config.mcp.url,
  authToken: config.mcp.authToken,
});

// Register local tool: read_gitops_file
// This lets Claude read any file from the GitOps repo via the GitHub API,
// without needing to add GitHub access to the MCP server.
mcpClient.addLocalTool(
  {
    name: "read_gitops_file",
    description:
      "Read the contents of a file from the GitOps repository (it-and-security/ directory). " +
      "Use this to inspect existing configuration files before proposing changes. " +
      "The path should be relative to the gitops root, e.g. 'fleets/workstations.yml' or 'lib/macos/policies/update-1password.yml'.",
    input_schema: {
      type: "object",
      properties: {
        path: {
          type: "string",
          description: "File path relative to the gitops root (e.g. 'fleets/workstations.yml')",
        },
      },
      required: ["path"],
    },
  },
  async (args) => {
    const normalized = require("path").posix.normalize(args.path);
    if (normalized.includes("..") || require("path").posix.isAbsolute(normalized) ||
        !(normalized === "default.yml" || normalized.startsWith("fleets/") || normalized.startsWith("lib/"))) {
      return `Error: Invalid path (must be under default.yml, fleets/, or lib/): ${args.path}`;
    }
    const filePath = `${config.github.gitopsBasePath}/${normalized}`;
    const content = await github.getFileContent(filePath);
    if (content === null) {
      return `Error: File not found: ${args.path}`;
    }
    return content;
  }
);

const claude = new ClaudeClient({
  apiKey: config.anthropic.apiKey,
  model: config.anthropic.model,
  maxToolCalls: config.anthropic.maxToolCalls,
  mcpClient,
});

// Both validations are off by default in the SDK for backward compatibility;
// Azure Bot Service tokens always carry the issuer and serviceurl claims they check.
const adapter = new CloudAdapter(
  {
    clientId: config.teams.appId,
    clientSecret: config.teams.appPassword,
    tenantId: config.teams.tenantId,
    validateIssuer: true,
  },
  undefined,
  undefined,
  { validateServiceUrl: true }
);
adapter.onTurnError = async (context, err) => {
  console.error("[teams] Unhandled turn error:", err);
  try {
    await context.sendActivity("❌ An unexpected error occurred. Please try again.");
  } catch {
    // best effort
  }
};

const bot = new ActivityHandler();
registerHandlers(bot, adapter, config, github, claude);

const server = express();
// The app runs behind the host's proxy; trust only the hop nearest the app so a
// client-supplied X-Forwarded-For can't choose its own rate-limit key.
server.set("trust proxy", 1);

// Throttles callers that fail authentication. Authenticated deliveries (2xx) are
// not counted, so bursts of GitHub check_run events never hit the limit.
const authFailureLimiter = rateLimit({
  windowMs: 60 * 1000,
  limit: 30,
  skipSuccessfulRequests: true,
  standardHeaders: "draft-8",
  legacyHeaders: false,
});

server.get("/healthz", (_req, res) => {
  res.json({ ok: true });
});

// Teams delivers activities here via Azure Bot Service. The JWT is verified before
// the body is parsed; Teams sends long messages twice (text plus an HTML copy), so
// the default 100 KB body limit is too small.
server.post(
  "/api/messages",
  authFailureLimiter,
  (req, res, next) => adapter.authorizeRequest(req, res, next),
  express.json({ limit: "1mb" }),
  (req, res) => adapter.process(req, res, (context) => bot.run(context))
);

// The webhook handler reads the raw request stream itself (the HMAC signature
// covers the exact bytes), so no body parser is mounted on this route.
server.post("/github/webhook", authFailureLimiter, createWebhookHandler(config, github, claude));

(async () => {
  // Connect to the Fleet MCP server before starting
  try {
    await mcpClient.connect();
  } catch (err) {
    console.warn(`[mcp] Warning: Could not connect to Fleet MCP server at ${config.mcp.url}: ${err.message}`);
    console.warn("[mcp] The bot will start but Fleet tool queries will not be available.");
  }

  server.listen(config.webhook.port, () => {
    console.log("Fleet is running!");
    console.log(`  Repo: ${config.github.repo}`);
    console.log(`  Branch: ${config.github.baseBranch}`);
    console.log(`  Path: ${config.github.gitopsBasePath}`);
    console.log(`  Model: ${config.anthropic.model}`);
    console.log(`  MCP: ${config.mcp.url}`);
    console.log(`  Teams: /api/messages (port ${config.webhook.port}, tenant ${config.teams.tenantId})`);
    console.log(`  Webhook: /github/webhook (port ${config.webhook.port})`);
    console.log(`  CI auto-fix: ${config.ci.autoFix ? `enabled (check: ${config.ci.checkName})` : "disabled"}`);
    console.log("\nListening for Teams messages, @mentions, and GitHub webhooks...");
  });
})();
