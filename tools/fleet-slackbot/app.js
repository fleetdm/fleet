const config = require("./config");
const GitHubClient = require("./github-client");
const ClaudeClient = require("./claude-client");
const McpClient = require("./mcp-client");
const SYSTEM_PROMPTS = require("./system-prompt");
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
  systemPrompt: SYSTEM_PROMPTS[config.platform],
});

const healthHandler = (_req, res) => {
  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ ok: true }));
};
const webhookHandler = createWebhookHandler(config, github, claude);

function createSlackApp() {
  const { App } = require("@slack/bolt");
  const { registerHandlers } = require("./slack-handlers");

  const app = new App({
    token: config.slack.botToken,
    socketMode: true,
    appToken: config.slack.appToken,
    processBeforeResponse: false,
    customRoutes: [
      {
        path: "/healthz",
        method: "GET",
        handler: healthHandler,
      },
      {
        path: "/github/webhook",
        method: "POST",
        handler: webhookHandler,
      },
    ],
    installerOptions: {
      port: config.webhook.port,
    },
  });

  registerHandlers(app, config, github, claude);
  return app;
}

function createGoogleChatServer() {
  const http = require("http");
  const { createChatApi, createGoogleChatHandler } = require("./google-chat-handlers");
  const api = createChatApi(config.googleChat);
  const routes = {
    "GET /healthz": healthHandler,
    "POST /github/webhook": webhookHandler,
    "POST /google-chat/events": createGoogleChatHandler(config, github, claude, { api }),
  };
  const server = http.createServer((req, res) => {
    let pathname;
    try {
      pathname = new URL(req.url, "http://localhost").pathname;
    } catch {
      // A malformed request line would otherwise throw here and crash the process.
      res.writeHead(400);
      res.end();
      return;
    }
    const route = routes[`${req.method} ${pathname}`];
    if (!route) {
      res.writeHead(404);
      res.end();
      return;
    }
    Promise.resolve(route(req, res)).catch((err) => {
      console.error("[http] %s %s failed:", req.method, pathname, err);
      if (!res.headersSent) res.writeHead(500);
      res.end();
    });
  });
  return {
    async start() {
      try {
        await api.checkCredentials();
      } catch (err) {
        console.error(`[gchat] Can't get Google Chat API credentials (set GOOGLE_CHAT_CREDENTIALS or Application Default Credentials): ${err.message}`);
        process.exit(1);
      }
      await new Promise((resolve) => server.listen(config.webhook.port, resolve));
    },
  };
}

const app = config.platform === "google-chat" ? createGoogleChatServer() : createSlackApp();

(async () => {
  // Connect to the Fleet MCP server before starting
  try {
    await mcpClient.connect();
  } catch (err) {
    console.warn(`[mcp] Warning: Could not connect to Fleet MCP server at ${config.mcp.url}: ${err.message}`);
    console.warn("[mcp] The bot will start but Fleet tool queries will not be available.");
  }

  await app.start();
  console.log(`Fleet is running! (platform: ${config.platform})`);
  console.log(`  Repo: ${config.github.repo}`);
  console.log(`  Branch: ${config.github.baseBranch}`);
  console.log(`  Path: ${config.github.gitopsBasePath}`);
  console.log(`  Model: ${config.anthropic.model}`);
  console.log(`  MCP: ${config.mcp.url}`);
  console.log(`  Webhook: /github/webhook (port ${config.webhook.port})`);
  console.log(`  CI auto-fix: ${config.ci.autoFix ? `enabled (check: ${config.ci.checkName})` : "disabled"}`);
  if (config.platform === "google-chat") {
    console.log(`  Google Chat events: /google-chat/events (port ${config.webhook.port})`);
  }
  console.log("\nListening for @mentions, DMs, and GitHub webhooks...");
})();
