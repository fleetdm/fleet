require("dotenv").config();

const config = {
  slack: {
    botToken: process.env.SLACK_BOT_TOKEN,
    appToken: process.env.SLACK_APP_TOKEN,
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

// Validate required env vars
const required = [
  ["SLACK_BOT_TOKEN", config.slack.botToken],
  ["SLACK_APP_TOKEN", config.slack.appToken],
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

module.exports = config;
