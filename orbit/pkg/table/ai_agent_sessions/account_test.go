package ai_agent_sessions

import (
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeJWT builds an unsigned token; its signature carries FAKE so a leaked
// token is caught by the same check as every other secret.
func fakeJWT(claims string) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(claims)) + ".FAKE-SIG"
}

func TestAccountIdentifier(t *testing.T) {
	aliceToken := fakeJWT(`{"email":"alice@example.com"}`)

	// Every credential in the files contains FAKE, and none may reach a row.
	for _, tc := range []struct {
		name, agent string
		files       map[string]string // relative to the home directory
		want        string
	}{
		{
			name: "subscription", agent: agentClaudeCode,
			files: map[string]string{".claude.json": `{"oauthAccount":{"emailAddress":"alice@example.com","organizationType":"claude_team","billingType":"stripe_subscription"},"primaryApiKey":"sk-ant-FAKE-KEY"}`},
			want:  "alice@example.com",
		},
		{
			name: "subscription by billing type", agent: agentClaudeCode,
			files: map[string]string{".claude.json": `{"oauthAccount":{"emailAddress":"alice@example.com","billingType":"apple_subscription"}}`},
			want:  "alice@example.com",
		},
		{
			name: "console key", agent: agentClaudeCode,
			files: map[string]string{".claude.json": `{"primaryApiKey":"sk-ant-FAKE-KEY","projects":{"/x":{"history":["FAKE prompt"]}}}`},
			want:  "console",
		},
		{
			name: "Anthropic account without a subscription", agent: agentClaudeCode,
			files: map[string]string{".claude.json": `{"oauthAccount":{"emailAddress":"alice@example.com","billingType":"prepaid"}}`},
			want:  "anthropic",
		},
		{
			name: "provider in settings wins over the stored account", agent: agentClaudeCode,
			files: map[string]string{
				".claude.json":          `{"oauthAccount":{"emailAddress":"alice@example.com","organizationType":"claude_team"}}`,
				".claude/settings.json": `{"env":{"CLAUDE_CODE_USE_BEDROCK":"1","AWS_SECRET_ACCESS_KEY":"FAKE-AWS-SECRET"}}`,
			},
			want: "bedrock",
		},
		{
			name: "provider in the legacy env block", agent: agentClaudeCode,
			files: map[string]string{".claude.json": `{"env":{"CLAUDE_CODE_USE_VERTEX":"true","CLAUDE_CODE_USE_FOUNDRY":"0"}}`},
			want:  "vertex",
		},
		{name: "nothing signed in", agent: agentClaudeCode, want: ""},
		{
			name: "last logged-in user", agent: agentCopilotCLI,
			files: map[string]string{".copilot/config.json": `{
				"lastLoggedInUser":{"login":"alice-gh"},
				"loggedInUsers":[{"login":"bob-gh"},{"login":"alice-gh"}],
				"authTokens":{"https://github.com:alice-gh":"gho_FAKE_TOKEN"},
				"copilotTokens":{"x":"FAKE_COPILOT_TOKEN"}
			}`},
			want: "alice-gh",
		},
		{
			name: "first logged-in user", agent: agentCopilotCLI,
			files: map[string]string{".copilot/config.json": `{"loggedInUsers":[{"login":"alice"}]}`},
			want:  "alice",
		},
		{
			name: "ChatGPT sign-in", agent: agentCodexCLI,
			files: map[string]string{".codex/auth.json": `{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{"id_token":"` + aliceToken + `","access_token":"FAKE-ACCESS","refresh_token":"FAKE-REFRESH"}}`},
			want:  "alice@example.com",
		},
		{
			name: "email in the profile claim", agent: agentCodexCLI,
			files: map[string]string{".codex/auth.json": `{"tokens":{"id_token":"` + fakeJWT(`{"https://api.openai.com/profile":{"email":"bob@example.com"}}`) + `"}}`},
			want:  "bob@example.com",
		},
		{
			name: "API key", agent: agentCodexCLI,
			files: map[string]string{".codex/auth.json": `{"OPENAI_API_KEY":"sk-FAKE"}`},
			want:  apiKeyAccount,
		},
		{
			name: "API key auth mode wins over stored tokens", agent: agentCodexCLI,
			files: map[string]string{".codex/auth.json": `{"auth_mode":"apikey","OPENAI_API_KEY":"sk-FAKE","tokens":{"id_token":"` + aliceToken + `"}}`},
			want:  apiKeyAccount,
		},
		{
			name: "model provider wins over the sign-in", agent: agentCodexCLI,
			files: map[string]string{
				".codex/auth.json": `{"tokens":{"id_token":"` + aliceToken + `"}}`,
				".codex/sessions/2026/08/01/rollout-2026-08-01T12-00-00-" + codexID + ".jsonl": `{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"` + codexID + `","cwd":"/w","model_provider":"amazon-bedrock"}}` + "\n",
			},
			want: "amazon-bedrock",
		},
		{
			name: "Google sign-in", agent: agentGeminiCLI,
			files: map[string]string{
				".gemini/google_accounts.json": `{"active":"alice@example.com","old":["x@example.com"]}`,
				".gemini/settings.json":        `{"security":{"auth":{"selectedType":"oauth-personal"}}}`,
				".gemini/oauth_creds.json":     `{"access_token":"FAKE-GOOGLE"}`,
			},
			want: "alice@example.com",
		},
		{
			name: "no auth type recorded means Google", agent: agentGeminiCLI,
			files: map[string]string{".gemini/google_accounts.json": `{"active":"alice@example.com"}`},
			want:  "alice@example.com",
		},
		{
			name: "Gemini API key", agent: agentGeminiCLI,
			files: map[string]string{
				".gemini/google_accounts.json": `{"active":"alice@example.com"}`,
				".gemini/settings.json":        `{"security":{"auth":{"selectedType":"gemini-api-key"}}}`,
			},
			want: apiKeyAccount,
		},
		{
			name: "Vertex AI", agent: agentGeminiCLI,
			files: map[string]string{".gemini/settings.json": `{"selectedAuthType":"vertex-ai"}`},
			want:  "vertex",
		},
		{
			name: "other auth type", agent: agentGeminiCLI,
			files: map[string]string{".gemini/settings.json": `{"selectedAuthType":"cloud-shell"}`},
			want:  "cloud-shell",
		},
		{
			name: "Cursor sign-in", agent: agentCursor,
			files: map[string]string{
				".cursor/cli-config.json":  `{"authInfo":{"email":"alice@example.com","authId":"auth0|FAKE-ID"}}`,
				".config/cursor/auth.json": `{"accessToken":"FAKE-CURSOR"}`,
			},
			want: "alice@example.com",
		},
	} {
		t.Run(tc.agent+"/"+tc.name, func(t *testing.T) {
			home := setup(t)
			sessionFixtures[tc.agent](t, home)
			for path, content := range tc.files {
				writeFile(t, filepath.Join(home, filepath.FromSlash(path)), content)
			}
			rows := sessionRows(t, eq("agent", tc.agent))
			require.NotEmpty(t, rows)
			for _, r := range rows {
				require.Equal(t, tc.want, r["account_identifier"])
			}
			requireAbsent(t, rows, "FAKE")
		})
	}
}
