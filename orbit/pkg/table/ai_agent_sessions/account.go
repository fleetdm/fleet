package ai_agent_sessions

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// account is who an agent is signed in as, from the agent's own config. Only
// the named values are decoded from each file: the same files hold tokens,
// API keys, prompt history and MCP server environments, none of which may
// reach a row or a log.
type account struct {
	Username     string
	Type         string
	Organization string
	Plan         string
}

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
			OrganizationName string `json:"organizationName"`
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

	var a account
	if o := cfg.OAuthAccount; o != nil {
		a.Username, a.Organization, a.Plan = o.EmailAddress, o.OrganizationName, o.OrganizationType
	}
	switch provider := firstNonEmpty(settings.Env.provider(), cfg.Env.provider()); {
	case provider != "":
		a.Type, a.Organization = provider, ""
	case cfg.OAuthAccount != nil && (claudeSubscriptionOrgs.has(cfg.OAuthAccount.OrganizationType) ||
		claudeSubscriptionBilling.has(cfg.OAuthAccount.BillingType)):
		a.Type = "subscription"
	case bool(cfg.PrimaryAPIKey):
		a.Type = "console"
	case cfg.OAuthAccount != nil:
		a.Type = "anthropic"
	}
	return a
}

type githubUser struct {
	Host  string `json:"host"`
	Login string `json:"login"`
}

func copilotAccount(sc *scope) account {
	var cfg struct {
		LastLoggedInUser *githubUser     `json:"lastLoggedInUser"`
		LoggedInUsers    []githubUser    `json:"loggedInUsers"`
		CopilotUserCache json.RawMessage `json:"copilotUserCache"`
	}
	readJSON(sc, filepath.Join(sc.home.Dir, ".copilot", "config.json"), &cfg)

	u := cfg.LastLoggedInUser
	if u == nil && len(cfg.LoggedInUsers) > 0 {
		u = &cfg.LoggedInUsers[0]
	}
	if u == nil {
		return account{}
	}
	a := account{Username: u.Login}
	switch host := strings.TrimSuffix(u.Host, "/"); {
	case host == "https://github.com":
		a.Type = "github"
	case host != "":
		a.Type = "enterprise"
	}
	// The cache's nesting isn't documented, so look for the keys anywhere.
	var cache any
	if json.Unmarshal(cfg.CopilotUserCache, &cache) == nil {
		a.Plan, _ = findKey(cache, "copilot_plan", 4).(string)
		if orgs, ok := findKey(cache, "organization_list", 4).([]any); ok && len(orgs) > 0 {
			switch o := orgs[0].(type) {
			case string:
				a.Organization = o
			case map[string]any:
				a.Organization, _ = o["login"].(string)
				if a.Organization == "" {
					a.Organization, _ = o["name"].(string)
				}
			}
		}
	}
	return a
}

// findKey returns the first value under key in v, searching depth levels of
// nested objects.
func findKey(v any, key string, depth int) any {
	m, ok := v.(map[string]any)
	if !ok || depth == 0 {
		return nil
	}
	if found, ok := m[key]; ok {
		return found
	}
	keys := slices.Sorted(maps.Keys(m))
	for _, k := range keys {
		if found := findKey(m[k], key, depth-1); found != nil {
			return found
		}
	}
	return nil
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
	a := account{Type: auth.AuthMode}
	if auth.Tokens != nil && auth.Tokens.IDToken != "" {
		claims := jwtClaims(auth.Tokens.IDToken)
		var c struct {
			Email   string `json:"email"`
			Profile struct {
				Email string `json:"email"`
			} `json:"https://api.openai.com/profile"`
			Auth struct {
				Plan string `json:"chatgpt_plan_type"`
			} `json:"https://api.openai.com/auth"`
		}
		if json.Unmarshal(claims, &c) == nil {
			a.Username = firstNonEmpty(c.Email, c.Profile.Email)
			a.Plan = c.Auth.Plan
		}
		if a.Type == "" {
			a.Type = "chatgpt"
		}
	}
	if a.Type == "" && auth.APIKey {
		a.Type = "apikey"
	}
	return a
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

var geminiAuthTypes = map[string]string{
	"oauth-personal": "google",
	"gemini-api-key": "api_key",
	"vertex-ai":      "vertex",
}

func geminiAccount(sc *scope) account {
	var a account
	for _, root := range geminiRoots(sc.home) {
		var accts struct {
			Active string `json:"active"`
		}
		if a.Username == "" && readJSON(sc, filepath.Join(root, "google_accounts.json"), &accts) {
			a.Username = accts.Active
		}
		var settings struct {
			Security struct {
				Auth struct {
					SelectedType string `json:"selectedType"`
				} `json:"auth"`
			} `json:"security"`
			SelectedAuthType string `json:"selectedAuthType"`
		}
		if a.Type == "" && readJSON(sc, filepath.Join(root, "settings.json"), &settings) {
			t := firstNonEmpty(settings.Security.Auth.SelectedType, settings.SelectedAuthType)
			a.Type = firstNonEmpty(geminiAuthTypes[t], t)
		}
	}
	return a
}

func cursorAccount(sc *scope) account {
	var cfg struct {
		AuthInfo struct {
			Email    string `json:"email"`
			TeamName string `json:"teamName"` //nolint:apiparamcheck // Cursor's own config key
		} `json:"authInfo"`
	}
	readJSON(sc, filepath.Join(sc.home.Dir, ".cursor", "cli-config.json"), &cfg)
	return account{Username: cfg.AuthInfo.Email, Organization: cfg.AuthInfo.TeamName}
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
