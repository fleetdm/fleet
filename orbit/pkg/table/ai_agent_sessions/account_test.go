package ai_agent_sessions

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func accountCols(row map[string]string) []string {
	return []string{row["account_username"], row["account_type"], row["account_organization"], row["account_plan"]}
}

func TestClaudeCodeAccount(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   string
		settings string
		want     []string
	}{
		{
			name:   "subscription",
			config: `{"oauthAccount":{"emailAddress":"alice@example.com","organizationName":"Acme","organizationType":"claude_team","billingType":"stripe_subscription"},"primaryApiKey":"sk-ant-FAKE-KEY"}`,
			want:   []string{"alice@example.com", "subscription", "Acme", "claude_team"},
		},
		{
			name:   "subscription by billing type",
			config: `{"oauthAccount":{"emailAddress":"alice@example.com","billingType":"apple_subscription"}}`,
			want:   []string{"alice@example.com", "subscription", "", ""},
		},
		{
			name:   "console key",
			config: `{"primaryApiKey":"sk-ant-FAKE-KEY","projects":{"/x":{"history":["secret prompt"]}}}`,
			want:   []string{"", "console", "", ""},
		},
		{
			name:   "other Anthropic account",
			config: `{"oauthAccount":{"emailAddress":"alice@example.com","organizationName":"Acme","billingType":"prepaid"}}`,
			want:   []string{"alice@example.com", "anthropic", "Acme", ""},
		},
		{
			name:     "Bedrock wins over the stored account",
			config:   `{"oauthAccount":{"emailAddress":"alice@example.com","organizationName":"Acme","organizationType":"claude_team"}}`,
			settings: `{"env":{"CLAUDE_CODE_USE_BEDROCK":"1","AWS_SECRET_ACCESS_KEY":"FAKE-AWS-SECRET"}}`,
			want:     []string{"alice@example.com", "bedrock", "", "claude_team"},
		},
		{
			name:   "provider in the legacy env block",
			config: `{"env":{"CLAUDE_CODE_USE_VERTEX":"true","CLAUDE_CODE_USE_FOUNDRY":"0"}}`,
			want:   []string{"", "vertex", "", ""},
		},
		{
			name: "nothing signed in",
			want: []string{"", "", "", ""},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := setup(t)
			writeClaudePrompts(t, home, "s", "hi")
			if tc.config != "" {
				writeFile(t, filepath.Join(home, ".claude.json"), tc.config)
			}
			if tc.settings != "" {
				writeFile(t, filepath.Join(home, ".claude", "settings.json"), tc.settings)
			}
			rows := sessionRows(t)
			require.Equal(t, tc.want, accountCols(rows[0]))
			for _, secret := range []string{"FAKE-KEY", "FAKE-AWS-SECRET", "secret prompt"} {
				requireAbsent(t, rows, secret)
			}
		})
	}
}

func TestCopilotCLIAccount(t *testing.T) {
	home := setup(t)
	writeFile(t, filepath.Join(home, ".copilot", "session-state", "s", "events.jsonl"), copilotEvents)
	writeFile(t, filepath.Join(home, ".copilot", "config.json"), `{
	  "lastLoggedInUser":{"host":"https://github.com","login":"alice-gh"},
	  "loggedInUsers":[{"host":"https://github.com","login":"alice-gh"}],
	  "authTokens":{"https://github.com:alice-gh":"gho_FAKE_TOKEN"},
	  "copilotTokens":{"x":"FAKE_COPILOT_TOKEN"},
	  "copilotUserCache":{"https://github.com:alice-gh":{"copilot_plan":"business","organization_list":[{"login":"acme"}]}}
	}`)
	rows := sessionRows(t)
	require.Equal(t, []string{"alice-gh", "github", "acme", "business"}, accountCols(rows[0]))
	requireAbsent(t, rows, "FAKE_")

	writeFile(t, filepath.Join(home, ".copilot", "config.json"), `{"loggedInUsers":[{"host":"https://acme.ghe.com","login":"alice"}]}`)
	require.Equal(t, []string{"alice", "enterprise", "", ""}, accountCols(sessionRows(t)[0]))

	writeFile(t, filepath.Join(home, ".copilot", "config.json"), `{"loggedInUsers":[{"login":"alice"}]}`)
	require.Equal(t, []string{"alice", "", "", ""}, accountCols(sessionRows(t)[0]), "no host, no type")
}

func fakeJWT(claims string) string {
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"none"}`)) + "." + enc([]byte(claims)) + ".sig"
}

func TestCodexCLIAccount(t *testing.T) {
	home := setup(t)
	writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"), codexLegacy)
	idToken := fakeJWT(`{"email":"alice@example.com","https://api.openai.com/auth":{"chatgpt_plan_type":"team"}}`)
	writeFile(t, filepath.Join(home, ".codex", "auth.json"),
		`{"auth_mode":"chatgpt","OPENAI_API_KEY":null,"tokens":{"id_token":"`+idToken+`","access_token":"FAKE-ACCESS","refresh_token":"FAKE-REFRESH","account_id":"acct"}}`)
	rows := sessionRows(t)
	require.Equal(t, []string{"alice@example.com", "chatgpt", "", "team"}, accountCols(rows[0]))
	requireAbsent(t, rows, "FAKE-")
	requireAbsent(t, rows, idToken)

	t.Run("profile claim and API key fallback", func(t *testing.T) {
		resetState()
		idToken := fakeJWT(`{"https://api.openai.com/profile":{"email":"bob@example.com"}}`)
		writeFile(t, filepath.Join(home, ".codex", "auth.json"), `{"tokens":{"id_token":"`+idToken+`"}}`)
		require.Equal(t, []string{"bob@example.com", "chatgpt", "", ""}, accountCols(sessionRows(t)[0]))

		writeFile(t, filepath.Join(home, ".codex", "auth.json"), `{"OPENAI_API_KEY":"sk-FAKE"}`)
		rows := sessionRows(t)
		require.Equal(t, []string{"", "apikey", "", ""}, accountCols(rows[0]))
		requireAbsent(t, rows, "sk-FAKE")
	})

	t.Run("model provider overrides the sign-in", func(t *testing.T) {
		resetState()
		writeFile(t, codexPath(home, "rollout-2026-08-01T12-00-00-"+codexID+".jsonl"),
			`{"timestamp":"2026-08-01T12:00:00.000Z","type":"session_meta","payload":{"id":"`+codexID+`","cwd":"/w","model_provider":"amazon-bedrock"}}`+"\n")
		require.Equal(t, "amazon-bedrock", sessionRows(t)[0]["account_type"])
	})
}

func TestGeminiCLIAccount(t *testing.T) {
	home := setup(t)
	writeGeminiFixture(t, home)
	writeFile(t, filepath.Join(home, ".gemini", "google_accounts.json"), `{"active":"alice@example.com","old":["x@example.com"]}`)
	writeFile(t, filepath.Join(home, ".gemini", "settings.json"), `{"security":{"auth":{"selectedType":"oauth-personal"}}}`)
	writeFile(t, filepath.Join(home, ".gemini", "oauth_creds.json"), `{"access_token":"FAKE-GOOGLE"}`)
	rows := sessionRows(t, eq("agent", agentGeminiCLI))
	for _, r := range rows {
		require.Equal(t, []string{"alice@example.com", "google", "", ""}, accountCols(r))
	}
	requireAbsent(t, rows, "FAKE-GOOGLE")

	writeFile(t, filepath.Join(home, ".gemini", "settings.json"), `{"selectedAuthType":"vertex-ai"}`)
	require.Equal(t, "vertex", sessionRows(t, eq("agent", agentGeminiCLI))[0]["account_type"])
}

func TestCursorAccount(t *testing.T) {
	home := setup(t)
	copyCursorCapture(t, home)
	writeFile(t, filepath.Join(home, ".cursor", "cli-config.json"),
		`{"authInfo":{"email":"alice@example.com","displayName":"Alice","userId":1,"authId":"auth0|FAKE-ID","teamName":"Acme"}}`)
	writeFile(t, filepath.Join(home, ".config", "cursor", "auth.json"), `{"accessToken":"FAKE-CURSOR"}`)
	rows := sessionRows(t, eq("agent", agentCursor))
	require.Equal(t, []string{"alice@example.com", "", "Acme", ""}, accountCols(rows[0]))
	requireAbsent(t, rows, "FAKE-")
}

func TestGenericTarget(t *testing.T) {
	for _, tc := range []struct {
		tool, input, action, target string
	}{
		{"SomeNewTool", `{"query":"q","path":"p"}`, "other", "p"},
		{"SomeNewTool", `{"other":1}`, "other", "SomeNewTool"},
		{"mcp__claude_ai_Gmail__search__threads", `{"query":"from:bob"}`, "mcp", "claude_ai_Gmail: search__threads from:bob"},
		{"Bash", `{"command":"echo ` + strings.Repeat("é", 200) + `"}`, "execute", "echo " + strings.Repeat("é", 125)},
	} {
		action, target := classify(agentClaudeCode, tc.tool, []byte(tc.input))
		require.Equal(t, tc.action, action, tc.tool)
		require.Equal(t, tc.target, target, tc.tool)
		require.LessOrEqual(t, len(target), maxTargetBytes)
	}
}
