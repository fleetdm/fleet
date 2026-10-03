package fleet

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// IsOpenframeMode returns true when FLEET_OPENFRAME_MODE=1 is set.
// All hosts_include_any (policy_hosts / query_hosts) logic is gated behind this flag.
func IsOpenframeMode() bool {
	return os.Getenv("FLEET_OPENFRAME_MODE") == "1"
}

// IsOpenframeMultitenancy reports whether OpenFrame shared-database multitenancy
// is enabled via FLEET_OPENFRAME_MULTI_TENANCY_ENABLED — the Fleet-side mapping of the platform
// property `openframe.fleet.multi-tenancy.enabled`. It is the master switch for every
// multi-tenancy behavior added by this feature: when off, Fleet behaves exactly as the fork did
// before this feature (all tenant fences are inert, even if a stray team-pin env var is set;
// pre-existing fork features — FLEET_OPENFRAME_MODE host assignments, the Redis key prefix,
// agent OpenFrame mode, query-results TTL cleanup — are independent of this flag and unchanged).
//
// Two modes exist when the flag is on (see IsOpenframeSharedMode):
//   - pinned: FLEET_OPENFRAME_TENANT_UUID (preferred) or FLEET_OPENFRAME_TEAM_ID pins the whole
//     process to one tenant (the transitional one-Fleet-per-tenant deployment);
//   - shared: no process pin — one Fleet serves many tenants and every request is pinned
//     individually (X-Tenant-Id middleware, host team, enroll secret team).
//
// This is deliberately separate from IsOpenframeMode and from the pre-migration phase, in which
// tenants still have their own databases and run with no team pin.
func IsOpenframeMultitenancy() bool {
	return parseOpenframeEnabled(os.Getenv("FLEET_OPENFRAME_MULTI_TENANCY_ENABLED"))
}

// parseOpenframeEnabled parses the master-flag value ("true"/"1"/etc per strconv.ParseBool,
// whitespace-tolerant; anything unparsable is off). Pure (no env) so it can be unit-tested
// directly.
func parseOpenframeEnabled(raw string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && b
}

// openframeSharedMode caches the mode decision: multitenancy on with no process-level tenant pin
// means this process serves many tenants and must pin each request individually. Cached because
// it is consulted on per-request paths (endpoint middleware).
var openframeSharedMode = sync.OnceValue(func() bool {
	_, hasUUID := OpenframeTenantUUID()
	_, hasTeamID := parseOpenframeTeamID(os.Getenv("FLEET_OPENFRAME_TEAM_ID"))
	return openframeSharedModeDecision(IsOpenframeMultitenancy(), hasUUID, hasTeamID)
})

// IsOpenframeSharedMode reports whether this process runs in shared multi-tenant mode: the
// multitenancy flag is on and no process-level pin (tenant UUID or team id) is configured. In
// shared mode a request that cannot be resolved to a tenant must be rejected (fail closed) —
// see the tenant middleware and the host/enroll pin helpers.
func IsOpenframeSharedMode() bool {
	return openframeSharedMode()
}

// openframeSharedModeDecision is the pure decision behind IsOpenframeSharedMode. Separated so it
// can be unit-tested without mutating process env / the cached value.
func openframeSharedModeDecision(multitenancyEnabled, hasUUID, hasTeamID bool) bool {
	return multitenancyEnabled && !hasUUID && !hasTeamID
}

type openframeTeamIDCtxKey struct{}

// NewOpenframeTeamContext returns a context carrying the OpenFrame tenant team id. It lets a
// request (or test) scope datastore access to a team without relying on the process-global
// FLEET_OPENFRAME_TEAM_ID env var. In shared mode this is how every request gets its tenant
// (set by the X-Tenant-Id middleware / host-auth / enroll pin); it is also required for tests,
// since the MySQL test harness runs in parallel and mutating process env there is unsafe.
func NewOpenframeTeamContext(ctx context.Context, teamID uint) context.Context {
	return context.WithValue(ctx, openframeTeamIDCtxKey{}, teamID)
}

// openframeTeamIDFromEnv parses FLEET_OPENFRAME_TEAM_ID exactly once. The team pin is a
// process-level constant, so the read+parse is cached rather than repeated on every datastore
// call (OpenframeTeamID is on hot enrollment/query paths). The master multitenancy flag is part
// of the cached decision: with the flag off a stray FLEET_OPENFRAME_TEAM_ID must NOT pin the
// process (flag off ⇒ pre-feature fork behavior, a bit for bit).
var openframeTeamIDFromEnv = sync.OnceValues(func() (uint, bool) {
	return openframeTeamIDFromEnvDecision(IsOpenframeMultitenancy(), os.Getenv("FLEET_OPENFRAME_TEAM_ID"))
})

// openframeTeamIDFromEnvDecision is the pure decision behind the cached env fallback.
// Separated so the "flag off ⇒ a stray FLEET_OPENFRAME_TEAM_ID must NOT pin" guarantee can be
// unit-tested without mutating process env / the cached value.
func openframeTeamIDFromEnvDecision(multitenancyEnabled bool, raw string) (uint, bool) {
	if !multitenancyEnabled {
		return 0, false
	}
	return parseOpenframeTeamID(raw)
}

// parseOpenframeTeamID parses a team id from its string form, returning ok=false for blank,
// non-numeric, or non-positive values. Pure (no env) so it can be unit-tested directly.
func parseOpenframeTeamID(raw string) (uint, bool) {
	if raw == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// OpenframeTenantUUID returns the Flamingo tenant UUID this process is pinned to
// (FLEET_OPENFRAME_TENANT_UUID), if set. This is the platform's stable, UUID-format tenant
// identity; it is resolved to Fleet's integer team_id at startup via EnsureOpenframeTeamID (the
// teams.openframe_tenant_uuid bridge).
func OpenframeTenantUUID() (string, bool) {
	u := strings.TrimSpace(os.Getenv("FLEET_OPENFRAME_TENANT_UUID"))
	if u == "" {
		return "", false
	}
	return u, true
}

// openframePinnedTeamID holds the team id resolved from the tenant UUID at startup
// (0 = not resolved). It takes precedence over the FLEET_OPENFRAME_TEAM_ID env fallback.
// It is only ever set from the flag-gated startup path in cmd/fleet/serve.go (and tests).
var openframePinnedTeamID atomic.Uint64

// SetOpenframeTeamID pins this process to the given Fleet team id. Called once at startup after the
// tenant UUID is resolved to its team (EnsureOpenframeTeamID). A zero id is ignored.
func SetOpenframeTeamID(teamID uint) {
	if teamID != 0 {
		openframePinnedTeamID.Store(uint64(teamID))
	}
}

// OpenframeTeamID returns the tenant team this request/process is pinned to under OpenFrame
// shared-database multitenancy, in precedence order: the context value if present
// (per-request pin — shared mode middleware, or tests); else the team resolved from
// FLEET_OPENFRAME_TENANT_UUID at startup (SetOpenframeTeamID); else the FLEET_OPENFRAME_TEAM_ID
// env fallback (inert unless FLEET_OPENFRAME_MULTI_TENANCY_ENABLED is on). ok is false when none
// yields a valid (non-zero) team, in which case callers must not assume a tenant scope.
func OpenframeTeamID(ctx context.Context) (uint, bool) {
	if ctx != nil {
		if v, ok := ctx.Value(openframeTeamIDCtxKey{}).(uint); ok && v != 0 {
			return v, true
		}
	}
	if v := openframePinnedTeamID.Load(); v != 0 {
		return uint(v), true
	}
	return openframeTeamIDFromEnv()
}

// ValidateOpenframeMultitenancy validates the multitenancy configuration at startup. When
// FLEET_OPENFRAME_MULTI_TENANCY_ENABLED is off it is a no-op. When on, both a pinned process
// (FLEET_OPENFRAME_TENANT_UUID or FLEET_OPENFRAME_TEAM_ID set — one Fleet per tenant) and an
// unpinned process (shared mode — every request is pinned individually, fail closed) are valid.
// A set-but-malformed pin is rejected — an unparsable FLEET_OPENFRAME_TEAM_ID (silently ignoring
// it would boot the wrong mode), or a non-UUID FLEET_OPENFRAME_TENANT_UUID (which would otherwise
// reach EnsureOpenframeTeamID and create a garbage `openframe-<junk>` team + seed a secret; the
// shared-mode X-Tenant-Id path already rejects non-UUIDs, so this keeps pinned mode symmetric).
func ValidateOpenframeMultitenancy() error {
	return openframeMultitenancyConfigError(
		IsOpenframeMultitenancy(),
		os.Getenv("FLEET_OPENFRAME_TEAM_ID"),
		os.Getenv("FLEET_OPENFRAME_TENANT_UUID"),
	)
}

// openframeMultitenancyConfigError is the pure decision behind ValidateOpenframeMultitenancy.
// Separated so it can be unit-tested without mutating process env / the cached pin.
func openframeMultitenancyConfigError(multitenancyEnabled bool, teamIDRaw, tenantUUIDRaw string) error {
	if !multitenancyEnabled {
		return nil
	}
	if teamIDRaw != "" {
		if _, ok := parseOpenframeTeamID(teamIDRaw); !ok {
			return fmt.Errorf(
				"invalid FLEET_OPENFRAME_TEAM_ID %q: must be a positive integer team id (or unset — with FLEET_OPENFRAME_TENANT_UUID for a pinned process, or neither for shared per-request mode)",
				teamIDRaw,
			)
		}
	}
	if tenantUUID := strings.TrimSpace(tenantUUIDRaw); tenantUUID != "" {
		if _, err := uuid.Parse(tenantUUID); err != nil {
			return fmt.Errorf(
				"invalid FLEET_OPENFRAME_TENANT_UUID %q: must be a valid UUID (or unset — with FLEET_OPENFRAME_TEAM_ID for a pinned process, or neither for shared per-request mode)",
				tenantUUID,
			)
		}
	}
	return nil
}

// openframeSuperuserEmails caches the research-superuser allowlist
// (FLEET_OPENFRAME_SUPERUSER_EMAILS, comma-separated, case-insensitive). Empty — the default —
// means the escape hatch does not exist and shared mode stays fail-closed for every caller, so
// naming a superuser is an explicit operator decision per deployment.
var openframeSuperuserEmails = sync.OnceValue(func() map[string]struct{} {
	return parseOpenframeSuperuserEmails(os.Getenv("FLEET_OPENFRAME_SUPERUSER_EMAILS"))
})

// parseOpenframeSuperuserEmails parses the allowlist into a lookup set. Pure (no env) so it can
// be unit-tested directly.
func parseOpenframeSuperuserEmails(raw string) map[string]struct{} {
	emails := make(map[string]struct{})
	for _, entry := range strings.Split(raw, ",") {
		email := normalizeOpenframeEmail(entry)
		if email != "" {
			emails[email] = struct{}{}
		}
	}
	return emails
}

// IsOpenframeSuperuser reports whether this user may run a shared-mode request UNPINNED, i.e.
// with every tenant fence inert, which is what reaching all tenants' devices from one live query
// requires. It is consulted only on the two shared-mode gates that would otherwise fail closed
// (the tenant middleware and the campaign stream), and only for a caller that already holds a
// valid Fleet session — so it widens what an authenticated operator may see, never who may
// authenticate.
func IsOpenframeSuperuser(email string) bool {
	return isOpenframeSuperuser(openframeSuperuserEmails(), email)
}

// isOpenframeSuperuser is the pure decision behind IsOpenframeSuperuser. Separated so the
// allowlist semantics can be unit-tested without mutating process env / the cached set.
func isOpenframeSuperuser(allowlist map[string]struct{}, email string) bool {
	normalized := normalizeOpenframeEmail(email)
	if normalized == "" {
		return false
	}
	_, ok := allowlist[normalized]
	return ok
}

func normalizeOpenframeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// openframeUnsafeAgentOptions are osquery options that replace endpoint/plugin flag values on the
// agent. Serving them rewrote logger_tls_endpoint on openframe agents and silently dropped every
// scheduled result and status log - behavioral options (intervals etc.) are safe.
var openframeUnsafeAgentOptions = []string{
	"config_plugin",
	"config_tls_endpoint",
	"distributed_plugin",
	"distributed_tls_read_endpoint",
	"distributed_tls_write_endpoint",
	"logger_plugin",
	"logger_tls_endpoint",
}

func openframeTrimAgentOptions(config json.RawMessage) json.RawMessage {
	if len(config) == 0 {
		return config
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(config, &top); err != nil {
		return config
	}
	var opts map[string]json.RawMessage
	if err := json.Unmarshal(top["options"], &opts); err != nil {
		return config
	}
	trimmed := false
	for _, key := range openframeUnsafeAgentOptions {
		if _, ok := opts[key]; ok {
			delete(opts, key)
			trimmed = true
		}
	}
	if !trimmed {
		return config
	}
	optsRaw, err := json.Marshal(opts)
	if err != nil {
		return config
	}
	top["options"] = optsRaw
	out, err := json.Marshal(top)
	if err != nil {
		return config
	}
	return out
}

// OpenframeDefaultAppConfig is the config a tenant is born with in shared mode — the equivalent
// of what POST /setup persists on a dedicated Fleet: upstream new-install defaults with the
// agent options already trimmed of endpoint/plugin keys. Trimmed at the source (not only at
// serve time) so no seeded or fallback config can ever carry an endpoint override, even when
// read by a server predating the serve-time trim — openframe/docs/agent-options.md.
func OpenframeDefaultAppConfig() *AppConfig {
	appConfig := &AppConfig{}
	appConfig.ApplyDefaultsForNewInstalls()
	appConfig.AgentOptions = openframeTrimStoredAgentOptions(appConfig.AgentOptions)
	return appConfig
}

// openframeTrimStoredAgentOptions trims the stored agent-options shape
// ({"config": {"options": ...}, "overrides": ...}). Fails safe: on any parse error the
// options are dropped entirely rather than stored untrimmed.
func openframeTrimStoredAgentOptions(raw *json.RawMessage) *json.RawMessage {
	if raw == nil {
		return nil
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(*raw, &top); err != nil {
		return nil
	}
	top["config"] = openframeTrimAgentOptions(top["config"])
	out, err := json.Marshal(top)
	if err != nil {
		return nil
	}
	trimmed := json.RawMessage(out)
	return &trimmed
}
