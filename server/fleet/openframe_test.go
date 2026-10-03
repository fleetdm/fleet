package fleet

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseOpenframeTeamID(t *testing.T) {
	cases := []struct {
		name   string
		raw    string
		wantID uint
		wantOK bool
	}{
		{name: "blank", raw: "", wantID: 0, wantOK: false},
		{name: "valid", raw: "5", wantID: 5, wantOK: true},
		{name: "zero is not a valid team", raw: "0", wantID: 0, wantOK: false},
		{name: "non-numeric", raw: "abc", wantID: 0, wantOK: false},
		{name: "negative", raw: "-1", wantID: 0, wantOK: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotID, gotOK := parseOpenframeTeamID(c.raw)
			if gotID != c.wantID || gotOK != c.wantOK {
				t.Fatalf("parseOpenframeTeamID(%q) = (%d, %v), want (%d, %v)", c.raw, gotID, gotOK, c.wantID, c.wantOK)
			}
		})
	}
}

func TestParseOpenframeEnabled(t *testing.T) {
	cases := []struct {
		raw  string
		want bool
	}{
		{"", false},
		{"true", true},
		{"TRUE", true},
		{"True", true},
		{"1", true},
		{" true ", true},
		{"false", false},
		{"0", false},
		{"yes", false},
		{"enabled", false},
	}
	for _, c := range cases {
		t.Run(c.raw, func(t *testing.T) {
			if got := parseOpenframeEnabled(c.raw); got != c.want {
				t.Fatalf("parseOpenframeEnabled(%q) = %v, want %v", c.raw, got, c.want)
			}
		})
	}
}

func TestOpenframeTeamIDFromEnvDecision(t *testing.T) {
	// Flag off ⇒ the env pin is inert (upstream behavior, even with a stray team id set).
	if _, ok := openframeTeamIDFromEnvDecision(false, "7"); ok {
		t.Fatal("flag off: FLEET_OPENFRAME_TEAM_ID must not pin the process")
	}
	// Flag on ⇒ the env pin applies.
	if id, ok := openframeTeamIDFromEnvDecision(true, "7"); !ok || id != 7 {
		t.Fatalf("flag on: env pin should apply, got (%d,%v)", id, ok)
	}
	// Flag on with no/bad value ⇒ no pin (shared mode; bad values are caught at startup by
	// ValidateOpenframeMultitenancy).
	if _, ok := openframeTeamIDFromEnvDecision(true, ""); ok {
		t.Fatal("flag on with empty team id must not pin")
	}
}

func TestOpenframeTeamIDContextOverridesEnv(t *testing.T) {
	// The context value must win regardless of the env-derived (cached) pin — and it must work
	// even with the multitenancy flag off, because tests and the (flag-gated) middleware are the
	// only writers of the ctx value.
	ctx := NewOpenframeTeamContext(context.Background(), 9)
	id, ok := OpenframeTeamID(ctx)
	if !ok || id != 9 {
		t.Fatalf("context team should be used: got (%d, %v), want (9, true)", id, ok)
	}
}

func TestOpenframeSharedModeDecision(t *testing.T) {
	cases := []struct {
		name                             string
		multitenancy, hasUUID, hasTeamID bool
		want                             bool
	}{
		{name: "off is never shared", multitenancy: false, want: false},
		{name: "off with uuid is never shared", multitenancy: false, hasUUID: true, want: false},
		{name: "on + uuid = pinned", multitenancy: true, hasUUID: true, want: false},
		{name: "on + team id = pinned", multitenancy: true, hasTeamID: true, want: false},
		{name: "on + no pin = shared", multitenancy: true, want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := openframeSharedModeDecision(c.multitenancy, c.hasUUID, c.hasTeamID); got != c.want {
				t.Fatalf("openframeSharedModeDecision(%v,%v,%v) = %v, want %v",
					c.multitenancy, c.hasUUID, c.hasTeamID, got, c.want)
			}
		})
	}
}

func TestOpenframeMultitenancyConfigError(t *testing.T) {
	cases := []struct {
		name          string
		multitenancy  bool
		teamIDRaw     string
		tenantUUIDRaw string
		err           bool
	}{
		{name: "off is always nil", multitenancy: false, err: false},
		{name: "off ignores a bad team id", multitenancy: false, teamIDRaw: "abc", err: false},
		{name: "off ignores a bad tenant uuid", multitenancy: false, tenantUUIDRaw: "nope", err: false},
		{name: "on + no pin = shared mode, valid", multitenancy: true, err: false},
		{name: "on + valid team id = pinned, valid", multitenancy: true, teamIDRaw: "5", err: false},
		{name: "on + unparsable team id errors", multitenancy: true, teamIDRaw: "abc", err: true},
		{name: "on + zero team id errors", multitenancy: true, teamIDRaw: "0", err: true},
		{name: "on + valid tenant uuid = pinned, valid", multitenancy: true, tenantUUIDRaw: "1877e27c-b3fa-488f-82b6-449b80c1cc97", err: false},
		{name: "on + valid tenant uuid with surrounding space, valid", multitenancy: true, tenantUUIDRaw: "  1877e27c-b3fa-488f-82b6-449b80c1cc97  ", err: false},
		{name: "on + malformed tenant uuid errors", multitenancy: true, tenantUUIDRaw: "openframe-junk", err: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotErr := openframeMultitenancyConfigError(c.multitenancy, c.teamIDRaw, c.tenantUUIDRaw) != nil
			if gotErr != c.err {
				t.Fatalf("openframeMultitenancyConfigError(%v,%q,%q) err=%v, want %v",
					c.multitenancy, c.teamIDRaw, c.tenantUUIDRaw, gotErr, c.err)
			}
		})
	}
}

func TestValidateOpenframeMultitenancy(t *testing.T) {
	// Off → no-op (the pre-migration per-tenant-database phase, where there is no team pin).
	t.Setenv("FLEET_OPENFRAME_MULTI_TENANCY_ENABLED", "")
	t.Setenv("FLEET_OPENFRAME_TEAM_ID", "")
	if err := ValidateOpenframeMultitenancy(); err != nil {
		t.Fatalf("multitenancy off must be a no-op: got %v", err)
	}

	// On with no pin → shared per-request mode, valid.
	t.Setenv("FLEET_OPENFRAME_MULTI_TENANCY_ENABLED", "true")
	if err := ValidateOpenframeMultitenancy(); err != nil {
		t.Fatalf("multitenancy on with no pin is shared mode and must validate: got %v", err)
	}

	// On with an unparsable team id → refuse to boot (silent fallback into the wrong mode).
	t.Setenv("FLEET_OPENFRAME_TEAM_ID", "not-a-team")
	if err := ValidateOpenframeMultitenancy(); err == nil {
		t.Fatal("multitenancy on with an unparsable FLEET_OPENFRAME_TEAM_ID must fail")
	}

	// On with a valid team id → pinned mode, valid.
	t.Setenv("FLEET_OPENFRAME_TEAM_ID", "5")
	if err := ValidateOpenframeMultitenancy(); err != nil {
		t.Fatalf("multitenancy on with a valid team id must validate: got %v", err)
	}
}

func TestOpenframeTenantUUID(t *testing.T) {
	t.Setenv("FLEET_OPENFRAME_TENANT_UUID", "  ")
	if _, ok := OpenframeTenantUUID(); ok {
		t.Fatal("blank/whitespace UUID must be treated as unset")
	}
	t.Setenv("FLEET_OPENFRAME_TENANT_UUID", "3f1a9b2c-0000-4d5e-8f00-000000000001")
	u, ok := OpenframeTenantUUID()
	if !ok || u != "3f1a9b2c-0000-4d5e-8f00-000000000001" {
		t.Fatalf("OpenframeTenantUUID() = (%q,%v), want the set UUID", u, ok)
	}
}

func TestSetOpenframeTeamIDPins(t *testing.T) {
	// Resolved pin takes precedence over the env fallback and is returned by OpenframeTeamID.
	// (The atomic pin is deliberately not gated on the flag inside OpenframeTeamID: its only
	// production writer, cmd/fleet/serve.go, is itself flag-gated.)
	openframePinnedTeamID.Store(0)
	t.Cleanup(func() { openframePinnedTeamID.Store(0) })

	SetOpenframeTeamID(0) // ignored
	if v := openframePinnedTeamID.Load(); v != 0 {
		t.Fatalf("zero pin must be ignored, got %d", v)
	}
	SetOpenframeTeamID(7)
	id, ok := OpenframeTeamID(context.Background())
	if !ok || id != 7 {
		t.Fatalf("resolved pin should be returned: got (%d,%v), want (7,true)", id, ok)
	}
}

func TestOpenframeDefaultAppConfigServesAgentOptions(t *testing.T) {
	appConfig := OpenframeDefaultAppConfig()
	if appConfig.AgentOptions == nil {
		t.Fatal("agent options must be seeded for shared-mode tenants")
	}

	var top map[string]json.RawMessage
	if err := json.Unmarshal(*appConfig.AgentOptions, &top); err != nil {
		t.Fatalf("agent options must be valid JSON: %v", err)
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(top["config"], &cfg); err != nil {
		t.Fatalf("config must be a JSON object: %v", err)
	}
	var opts map[string]json.RawMessage
	if err := json.Unmarshal(cfg["options"], &opts); err != nil {
		t.Fatalf("options must be a JSON object: %v", err)
	}
	for _, key := range []string{"distributed_interval", "logger_tls_period", "distributed_tls_max_attempts", "pack_delimiter"} {
		if _, ok := opts[key]; !ok {
			t.Fatalf("behavioral option %q must be seeded", key)
		}
	}
	for _, key := range openframeUnsafeAgentOptions {
		if _, ok := opts[key]; ok {
			t.Fatalf("unsafe option %q must be trimmed at the source", key)
		}
	}
	if _, ok := cfg["decorators"]; !ok {
		t.Fatal("decorators must be kept")
	}
	if !appConfig.Features.EnableSoftwareInventory {
		t.Fatal("software inventory must stay enabled")
	}
}

func TestOpenframeTrimAgentOptions(t *testing.T) {
	full := json.RawMessage(`{"options": {"pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT 1;"]}}`)
	trimmed := openframeTrimAgentOptions(full)

	var top map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &top); err != nil {
		t.Fatalf("trimmed payload must stay valid JSON: %v", err)
	}
	var opts map[string]json.RawMessage
	if err := json.Unmarshal(top["options"], &opts); err != nil {
		t.Fatalf("options must stay a JSON object: %v", err)
	}
	for _, key := range openframeUnsafeAgentOptions {
		if _, ok := opts[key]; ok {
			t.Fatalf("unsafe key %q must be stripped", key)
		}
	}
	for _, key := range []string{"distributed_interval", "logger_tls_period", "distributed_tls_max_attempts", "pack_delimiter", "disable_distributed"} {
		if _, ok := opts[key]; !ok {
			t.Fatalf("behavioral key %q must be kept", key)
		}
	}
	if _, ok := top["decorators"]; !ok {
		t.Fatal("sibling keys must be untouched")
	}

	for _, raw := range []json.RawMessage{nil, json.RawMessage(`not-json`), json.RawMessage(`{"decorators": {}}`)} {
		if got := openframeTrimAgentOptions(raw); string(got) != string(raw) {
			t.Fatalf("payload without trimmable options must pass through unchanged: %q -> %q", raw, got)
		}
	}
}

// OPENFRAME(mysql-multitenancy): the research-superuser allowlist. Exercised through the pure
// parse/decide pair so the cached env read is never involved.
func TestParseOpenframeSuperuserEmails(t *testing.T) {
	allowlist := parseOpenframeSuperuserEmails(" Research@Flamingo.CX , ops@flamingo.cx ,, ")

	// Trimmed, lowercased, blanks dropped — an operator pasting a spaced list still gets what
	// they meant, and a trailing comma does not create a "" entry that matches a missing e-mail.
	if len(allowlist) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(allowlist), allowlist)
	}
	for _, email := range []string{"research@flamingo.cx", "RESEARCH@FLAMINGO.CX", " ops@flamingo.cx "} {
		if !isOpenframeSuperuser(allowlist, email) {
			t.Errorf("expected %q to be recognised", email)
		}
	}
	for _, email := range []string{"", "   ", "someone.else@flamingo.cx", "admin@openframe.local"} {
		if isOpenframeSuperuser(allowlist, email) {
			t.Errorf("expected %q NOT to be recognised", email)
		}
	}
}

// The default must be "no such caller exists", so a deployment that never names a superuser keeps
// shared mode fail closed exactly as before this feature.
func TestOpenframeSuperuserAllowlistEmptyByDefault(t *testing.T) {
	for _, raw := range []string{"", "   ", ",", " , "} {
		allowlist := parseOpenframeSuperuserEmails(raw)
		if len(allowlist) != 0 {
			t.Errorf("raw %q: expected empty allowlist, got %v", raw, allowlist)
		}
		if isOpenframeSuperuser(allowlist, "research@flamingo.cx") {
			t.Errorf("raw %q: recognised a caller from an empty allowlist", raw)
		}
	}
}
