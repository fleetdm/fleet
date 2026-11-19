package ai_agent_sessions

import (
	"bytes"
	"cmp"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
)

// account is who an agent is signed in as, from the agent's own config. Only
// the named values are decoded from each file: the same files hold tokens,
// API keys, prompt history and MCP server environments, none of which may
// reach a row or a log.
type account struct {
	// Identifier is the signed-in user, or how the agent authenticates when
	// there is no user: an API key or a third-party provider.
	Identifier string
}

// apiKeyAccount stands in for the user when an agent runs on an API key,
// which names no one.
const apiKeyAccount = "API KEY"

// flag is an env-style boolean: "1", "true", "yes", true, or a non-zero number.
type flag bool

func (f *flag) UnmarshalJSON(b []byte) error {
	var v any
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	switch v := v.(type) {
	case bool:
		*f = flag(v)
	case float64:
		*f = v != 0
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			*f = true
		}
	}
	return nil
}

// claudeProviderEnv decodes only the switches that route Claude Code to a
// third-party provider, never the rest of `env` (which holds API keys).
type claudeProviderEnv struct {
	Gateway         flag `json:"CLAUDE_CODE_USE_GATEWAY"`
	Bedrock         flag `json:"CLAUDE_CODE_USE_BEDROCK"`
	Foundry         flag `json:"CLAUDE_CODE_USE_FOUNDRY"`
	AnthropicAWS    flag `json:"CLAUDE_CODE_USE_ANTHROPIC_AWS"`
	AnthropicGoogle flag `json:"CLAUDE_CODE_USE_ANTHROPIC_GOOGLE_CLOUD"`
	Mantle          flag `json:"CLAUDE_CODE_USE_MANTLE"`
	Vertex          flag `json:"CLAUDE_CODE_USE_VERTEX"`
}

// provider follows Claude Code's own precedence.
func (e *claudeProviderEnv) provider() string {
	if e == nil {
		return ""
	}
	for _, p := range []struct {
		on   flag
		name string
	}{
		{e.Gateway, "gateway"}, {e.Bedrock, "bedrock"}, {e.Foundry, "foundry"},
		{e.AnthropicAWS, "anthropic_aws"}, {e.AnthropicGoogle, "anthropic_google_cloud"},
		{e.Mantle, "bedrock_mantle"}, {e.Vertex, "vertex"},
	} {
		if p.on {
			return p.name
		}
	}
	return ""
}

var (
	claudeSubscriptionOrgs = set{
		"claude_pro": {}, "claude_max": {}, "claude_team": {}, "claude_enterprise": {},
	}
	claudeSubscriptionBilling = set{
		"stripe_subscription": {}, "stripe_subscription_contracted": {},
		"stripe_subscription_enterprise_self_serve": {}, "aws_marketplace": {},
		"c4e_consumption_trial": {}, "apple_subscription": {}, "google_play_subscription": {},
	}
)

func claudeAccount(sc *scope) account {
	var cfg struct {
		OAuthAccount *struct {
			EmailAddress     string `json:"emailAddress"`
			OrganizationType string `json:"organizationType"`
			BillingType      string `json:"billingType"`
		} `json:"oauthAccount"`
		PrimaryAPIKey presence           `json:"primaryApiKey"`
		Env           *claudeProviderEnv `json:"env"`
	}
	readJSON(sc, filepath.Join(sc.home.Dir, ".claude.json"), &cfg)
	var settings struct {
		Env *claudeProviderEnv `json:"env"`
	}
	readJSON(sc, filepath.Join(sc.home.Dir, ".claude", "settings.json"), &settings)

	// Only a subscription names the user: on a provider or an API key the
	// stored OAuth account may be stale or unrelated.
	o := cfg.OAuthAccount
	switch provider := cmp.Or(settings.Env.provider(), cfg.Env.provider()); {
	case provider != "":
		return account{Identifier: provider}
	case o != nil && (claudeSubscriptionOrgs.has(o.OrganizationType) || claudeSubscriptionBilling.has(o.BillingType)):
		return account{Identifier: o.EmailAddress}
	case bool(cfg.PrimaryAPIKey):
		return account{Identifier: "console"}
	case o != nil:
		return account{Identifier: "anthropic"}
	}
	return account{}
}

func copilotAccount(sc *scope) account {
	type githubUser struct {
		Login string `json:"login"`
	}
	var cfg struct {
		LastLoggedInUser *githubUser  `json:"lastLoggedInUser"`
		LoggedInUsers    []githubUser `json:"loggedInUsers"`
	}
	readJSON(sc, filepath.Join(sc.home.Dir, ".copilot", "config.json"), &cfg)
	switch {
	case cfg.LastLoggedInUser != nil:
		return account{Identifier: cfg.LastLoggedInUser.Login}
	case len(cfg.LoggedInUsers) > 0:
		return account{Identifier: cfg.LoggedInUsers[0].Login}
	}
	return account{}
}

func codexAccount(sc *scope) account {
	var auth struct {
		AuthMode string   `json:"auth_mode"`
		APIKey   presence `json:"OPENAI_API_KEY"`
		Tokens   *struct {
			IDToken string `json:"id_token"`
		} `json:"tokens"`
	}
	if !readJSON(sc, filepath.Join(sc.home.Dir, ".codex", "auth.json"), &auth) {
		return account{}
	}
	switch {
	case auth.AuthMode == "apikey":
		return account{Identifier: apiKeyAccount}
	case auth.Tokens != nil && auth.Tokens.IDToken != "":
		var c struct {
			Email   string `json:"email"`
			Profile struct {
				Email string `json:"email"`
			} `json:"https://api.openai.com/profile"`
		}
		_ = json.Unmarshal(jwtClaims(auth.Tokens.IDToken), &c)
		return account{Identifier: cmp.Or(c.Email, c.Profile.Email)}
	case bool(auth.APIKey):
		return account{Identifier: apiKeyAccount}
	}
	return account{}
}

// jwtClaims decodes a JWT's payload without verifying it: the token is only
// a local record of who signed in, not a credential we act on.
func jwtClaims(token string) []byte {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload := strings.TrimRight(parts[1], "=")
	b, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil
	}
	return b
}

func geminiAccount(sc *scope) account {
	var active, authType string
	for _, root := range geminiRoots(sc.home) {
		var accts struct {
			Active string `json:"active"`
		}
		if active == "" && readJSON(sc, filepath.Join(root, "google_accounts.json"), &accts) {
			active = accts.Active
		}
		var settings struct {
			Security struct {
				Auth struct {
					SelectedType string `json:"selectedType"`
				} `json:"auth"`
			} `json:"security"`
			SelectedAuthType string `json:"selectedAuthType"`
		}
		if authType == "" && readJSON(sc, filepath.Join(root, "settings.json"), &settings) {
			authType = cmp.Or(settings.Security.Auth.SelectedType, settings.SelectedAuthType)
		}
	}
	// Signing in with Google is the default when no auth type is recorded.
	switch authType {
	case "", "oauth-personal":
		return account{Identifier: active}
	case "gemini-api-key":
		return account{Identifier: apiKeyAccount}
	case "vertex-ai":
		return account{Identifier: "vertex"}
	}
	return account{Identifier: authType}
}

func cursorAccount(sc *scope) account {
	var cfg struct {
		AuthInfo struct {
			Email string `json:"email"`
		} `json:"authInfo"`
	}
	readJSON(sc, filepath.Join(sc.home.Dir, ".cursor", "cli-config.json"), &cfg)
	return account{Identifier: cfg.AuthInfo.Email}
}

// unquote returns a JSON string's value, or "" when raw isn't one.
func unquote(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}
