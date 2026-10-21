require("dotenv").config();

const PLATFORMS = ["slack", "google-chat"];

const config = {
  platform: process.env.CHAT_PLATFORM || "slack",
  slack: {
    botToken: process.env.SLACK_BOT_TOKEN,
    appToken: process.env.SLACK_APP_TOKEN,
  },
  googleChat: {
    projectNumber: process.env.GOOGLE_CHAT_PROJECT_NUMBER,
    endpointUrl: process.env.GOOGLE_CHAT_ENDPOINT_URL,
    // Service account key JSON. When unset, Application Default Credentials are used.
    credentials: process.env.GOOGLE_CHAT_CREDENTIALS,
    allowedDomains: (process.env.GOOGLE_CHAT_ALLOWED_DOMAINS || "")
      .split(",")
      .map((d) => d.trim().toLowerCase())
      .filter(Boolean),
  },
  github: {
    token: process.env.GITHUB_TOKEN,
    repo: process.env.GITHUB_REPO || "fleetdm/fleet",
    baseBranch: process.env.GITHUB_BASE_BRANCH || "main",
    gitopsBasePath: process.env.GITOPS_BASE_PATH || "it-and-security",
    botUsername: process.env.GITHUB_BOT_USERNAME,
  },
  anthropic: {
    apiKey: process.env.ANTHROPIC_API_KEY,
    model: process.env.ANTHROPIC_MODEL || "claude-opus-4-6",
    maxToolCalls: parseInt(process.env.MAX_TOOL_CALLS || "100", 10),
  },
  webhook: {
    secret: process.env.GITHUB_WEBHOOK_SECRET,
    port: parseInt(process.env.PORT || "3000", 10),
  },
  ci: {
    checkName: process.env.GITOPS_CI_CHECK_NAME || "fleet-gitops",
    autoFix: process.env.CI_AUTO_FIX !== "false",
  },
  mcp: {
    url: process.env.FLEET_MCP_URL || "http://localhost:8181/sse",
    authToken: process.env.FLEET_MCP_AUTH_TOKEN,
  },
};

if (!PLATFORMS.includes(config.platform)) {
  console.error(`Invalid CHAT_PLATFORM "${config.platform}" (expected one of: ${PLATFORMS.join(", ")})`);
  process.exit(1);
}

const platformRequired = {
  slack: [
    ["SLACK_BOT_TOKEN", config.slack.botToken],
    ["SLACK_APP_TOKEN", config.slack.appToken],
  ],
  "google-chat": [
    ["GOOGLE_CHAT_PROJECT_NUMBER", config.googleChat.projectNumber],
    ["GOOGLE_CHAT_ENDPOINT_URL", config.googleChat.endpointUrl],
  ],
};

// Validate required env vars
const required = [
  ...platformRequired[config.platform],
  ["GITHUB_TOKEN", config.github.token],
  ["GITHUB_BOT_USERNAME", config.github.botUsername],
  ["ANTHROPIC_API_KEY", config.anthropic.apiKey],
  ["GITHUB_WEBHOOK_SECRET", config.webhook.secret],
  ["FLEET_MCP_AUTH_TOKEN", config.mcp.authToken],
];

for (const [name, value] of required) {
  if (!value) {
    console.error(`Missing required environment variable: ${name}`);
    process.exit(1);
  }
}

if (config.platform === "google-chat" && !/^\d+$/.test(config.googleChat.projectNumber)) {
  console.error("GOOGLE_CHAT_PROJECT_NUMBER must be the numeric project number, not the project ID");
  process.exit(1);
}

module.exports = config;
