import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  api,
  type ContextInfo,
  type ResolvedBinary,
  type Settings,
} from "../../lib/ipc";
import type { ServeStatus } from "../../lib/useSystemHealth";
import {
  CRONS,
  CRON_GROUP_SUBTITLE,
  CRON_GROUP_TITLE,
  type CronGroup,
  type CronInfo,
} from "../../lib/fleetctlCrons";
import { waitForExit } from "../../lib/orchestration";
import { activeServer } from "../../lib/servers";
import { noAutocorrect } from "../../lib/noAutocorrect";

type SubTab = "login" | "get" | "trigger" | "custom";

const SUB_TABS: { id: SubTab; label: string }[] = [
  { id: "login", label: "Login" },
  { id: "get", label: "Get" },
  { id: "trigger", label: "Trigger" },
  { id: "custom", label: "Custom" },
];

const TRIGGER_PROC_PREFIX = "fleetctl-trigger-";

interface GroupEntry {
  ts: number;
  cron: string;
  body: string;
  exitCode: number | null;
}

/// Output-buffer key. The user's *click source* drives output routing,
/// not the cron's home group — so a favorited cron triggered from
/// Favorites and from its home group produces separate histories.
type OutputSource = CronGroup | "favorites";

const MAX_GROUP_ENTRIES = 50;

export function FleetctlTab({
  settings,
  onSettingsChange,
  serve,
  goToSettings,
  goToServer,
  goToLogs,
}: {
  settings: Settings;
  onSettingsChange: (s: Settings) => void;
  serve: ServeStatus;
  goToSettings: () => void;
  goToServer: () => void;
  goToLogs: () => void;
}) {
  const repoPath = activeServer(settings).worktree_path;
  const favorites = useMemo(
    () => new Set(settings.favorite_crons),
    [settings.favorite_crons],
  );
  const toggleFavorite = useCallback(
    (name: string) => {
      const next_set = new Set(settings.favorite_crons);
      if (next_set.has(name)) next_set.delete(name);
      else next_set.add(name);
      const next: Settings = {
        ...settings,
        favorite_crons: Array.from(next_set),
      };
      // Apply locally first so the star fills/empties on the same
      // tick the user clicked; persistence happens in the background.
      onSettingsChange(next);
      api
        .saveSettings(next)
        .catch((e) => console.error("saveSettings(favorites) failed", e));
    },
    [settings, onSettingsChange],
  );
  const [sub, setSub] = useState<SubTab>("login");
  const [binary, setBinary] = useState<ResolvedBinary | null>(null);
  const [ctx, setCtx] = useState<ContextInfo | null>(null);
  // App-side state — fleetctl itself has no "current context"
  // pointer in the config file. Every invocation passes --context.
  // Persisting this across reloads isn't worth the complexity right
  // now; "default" is the right boot value.
  const [selectedContext, setSelectedContext] = useState<string>("default");
  const [error, setError] = useState<string | null>(null);

  const refreshBinary = useCallback(async () => {
    try {
      const b = await api.fleetctlResolveBinary(
        repoPath ?? null,
        settings.fleetctl_path ?? null,
      );
      setBinary(b);
    } catch (e) {
      setError(String(e));
    }
  }, [repoPath, settings.fleetctl_path]);

  const refreshCtx = useCallback(async () => {
    try {
      const c = await api.fleetctlReadContext();
      setCtx(c);
    } catch (e) {
      setError(String(e));
    }
  }, []);

  useEffect(() => {
    refreshBinary();
    refreshCtx();
  }, [refreshBinary, refreshCtx]);

  if (!repoPath) {
    return (
      <div
        style={{
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          color: "var(--app-text-dim)",
        }}
      >
        No Fleet repo configured · open Settings to pick one
      </div>
    );
  }

  const binaryReady = !!binary?.exists;
  const serverUp = serve.up;
  // We deliberately don't check login state up-front. The default
  // context usually has a valid token; if it doesn't, fleetctl will
  // tell the user with its own error message when they actually run
  // something. Saves a verification round-trip and avoids spurious
  // "not logged in" warnings while the API is just slow to start.
  const canAct = binaryReady && serverUp;

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      {/* Top strip — header card, no-contexts banner, error. Stays
          full-width above the sidebar+panel split so context is always
          visible no matter which sub-tab the user is on. */}
      <div
        style={{
          padding: "var(--pad-large) var(--pad-large) var(--pad-medium)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-medium)",
          flexShrink: 0,
        }}
      >
        <ContextHeader
          binary={binary}
          ctx={ctx}
          selectedContext={selectedContext}
          onContextChange={setSelectedContext}
          serverUp={serverUp}
          onRefresh={() => {
            refreshBinary();
            refreshCtx();
          }}
          onStartServer={goToServer}
        />

        {ctx && ctx.contexts.length === 0 && (
          <NoContextsBanner
            configExists={ctx.exists}
            onGoToSettings={goToSettings}
          />
        )}

        {error && (
          <div
            style={{
              color: "var(--ui-error)",
              fontSize: "var(--fs-xx-small)",
              padding: "6px 10px",
              background: "var(--tint-error-soft)",
              border: "1px solid var(--ui-error)",
              borderRadius: "var(--radius-md)",
            }}
          >
            {error}
          </div>
        )}
      </div>

      {canAct ? (
        <div
          style={{
            flex: 1,
            display: "flex",
            minHeight: 0,
            borderTop: "1px solid var(--app-border)",
          }}
        >
          <SubTabSidebar active={sub} onChange={setSub} />
          <div
            style={{
              flex: 1,
              minWidth: 0,
              padding: "var(--pad-large)",
              overflow: "auto",
            }}
          >
            {sub === "login" && (
              <LoginPanel
                binary={binary}
                ctx={ctx}
                canAct={canAct}
                serverUp={serverUp}
                repoPath={repoPath}
                selectedContext={selectedContext}
                onLoggedIn={refreshCtx}
                setError={setError}
              />
            )}

            {sub === "get" && (
              <GetPanel
                binary={binary}
                canAct={canAct}
                repoPath={repoPath}
                selectedContext={selectedContext}
                setError={setError}
              />
            )}

            {sub === "trigger" && (
              <TriggerPanel
                binary={binary}
                canAct={canAct}
                repoPath={repoPath}
                selectedContext={selectedContext}
                goToLogs={goToLogs}
                setError={setError}
                favorites={favorites}
                onToggleFavorite={toggleFavorite}
              />
            )}

            {sub === "custom" && (
              <CustomPanel
                binary={binary}
                canAct={canAct}
                repoPath={repoPath}
                selectedContext={selectedContext}
                setError={setError}
              />
            )}
          </div>
        </div>
      ) : (
        <div
          style={{
            flex: 1,
            padding: "0 var(--pad-large) var(--pad-large)",
            minHeight: 0,
            overflow: "auto",
          }}
        >
          <div
            className="dim"
            style={{
              fontSize: "var(--fs-xx-small)",
              textAlign: "center",
              padding: "var(--pad-large)",
              border: "1px dashed var(--app-border)",
              borderRadius: "var(--radius-md)",
            }}
          >
            {!serverUp
              ? "fleet serve is not running — start it from the Server tab. Login, Get, and Trigger all need the API up."
              : "fleetctl binary not found — check Settings."}
          </div>
        </div>
      )}
    </div>
  );
}

// ---------- header ----------

function ContextHeader({
  binary,
  ctx,
  selectedContext,
  onContextChange,
  serverUp,
  onRefresh,
  onStartServer,
}: {
  binary: ResolvedBinary | null;
  ctx: ContextInfo | null;
  selectedContext: string;
  onContextChange: (name: string) => void;
  serverUp: boolean;
  onRefresh: () => void;
  onStartServer: () => void;
}) {
  // We show the active context as info only — no judgement about whether
  // the stored token is still valid. fleetctl will tell the user when
  // they run something if it isn't.
  const known = ctx?.contexts ?? [];
  // The picker lists every context we found on disk plus the currently-
  // selected one (in case the user picked a name that doesn't exist yet —
  // e.g. they're about to log into a fresh context).
  const options = known.some((c) => c.name === selectedContext)
    ? known
    : [
        ...known,
        { name: selectedContext, address: null, email: null, has_token: false },
      ];
  const current = options.find((c) => c.name === selectedContext) ?? null;
  return (
    <div
      className="card"
      style={{
        display: "flex",
        alignItems: "center",
        gap: 18,
        padding: "12px 16px",
        flexWrap: "wrap",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span className={`dot ${serverUp ? "run" : "fail"}`} />
        <span style={{ fontWeight: 600 }}>
          fleet serve {serverUp ? "up" : "down"}
        </span>
        {!serverUp && (
          <button
            onClick={onStartServer}
            style={{ padding: "2px 8px", fontSize: "var(--fs-xxx-small)" }}
          >
            go to Server ↗
          </button>
        )}
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        binary ·{" "}
        <span
          className="mono"
          style={{
            color: binary?.exists ? "var(--app-text)" : "var(--ui-error)",
          }}
          title={binary?.path ?? ""}
        >
          {binary?.exists
            ? binary.source === "settings"
              ? "settings"
              : "build/fleetctl"
            : "missing"}
        </span>
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 6,
          fontSize: "var(--fs-xx-small)",
        }}
      >
        <span className="dim">context</span>
        <select
          value={selectedContext}
          onChange={(e) => onContextChange(e.target.value)}
          style={{
            background: "var(--app-surface-2)",
            color: "var(--app-text)",
            border: "1px solid var(--app-border)",
            borderRadius: 5,
            padding: "3px 6px",
            fontFamily: "var(--font-mono)",
            fontSize: "var(--fs-xx-small)",
          }}
        >
          {options.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name}
            </option>
          ))}
        </select>
      </div>
      {current?.address && (
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          <span
            className="mono"
            style={{ color: "var(--app-text)" }}
            title={current.address}
          >
            {current.address}
          </span>
          {current.email && (
            <>
              {" · "}
              <span style={{ color: "var(--app-text)" }}>
                {current.email}
              </span>
            </>
          )}
          {!current.has_token && (
            <>
              {" · "}
              <span style={{ color: "var(--ui-error)" }}>no token</span>
            </>
          )}
        </div>
      )}
      <div style={{ marginLeft: "auto", display: "flex", gap: 6 }}>
        <button
          onClick={onRefresh}
          style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
        >
          ↻ Refresh
        </button>
      </div>
    </div>
  );
}

function NoContextsBanner({
  configExists,
  onGoToSettings,
}: {
  configExists: boolean;
  onGoToSettings: () => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        gap: 10,
        alignItems: "center",
        padding: "10px 14px",
        border: "1px dashed var(--app-border)",
        borderRadius: "var(--radius-md)",
        background: "var(--app-surface-2)",
      }}
    >
      <div style={{ flex: 1 }}>
        <div className="card-title" style={{ marginBottom: 2 }}>
          No fleetctl contexts configured yet
        </div>
        <div
          className="dim"
          style={{ fontSize: "var(--fs-xx-small)", lineHeight: 1.4 }}
        >
          {configExists
            ? "Your ~/.fleet/config file exists but has no contexts."
            : "Your ~/.fleet/config doesn't exist yet."}{" "}
          You can still log in below with the <span className="mono">default</span>{" "}
          context — fleetctl will create the file on first login. Or edit
          the YAML directly from Settings to add named contexts (e.g.{" "}
          <span className="mono">staging</span>,{" "}
          <span className="mono">prod</span>).
        </div>
      </div>
      <button
        className="primary"
        onClick={onGoToSettings}
        style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
      >
        Set up in Settings ↗
      </button>
    </div>
  );
}

// ---------- sub-tab bar ----------

function SubTabSidebar({
  active,
  onChange,
}: {
  active: SubTab;
  onChange: (s: SubTab) => void;
}) {
  // Mirrors the Settings sidebar pattern — same width, same item
  // styling, same active treatment — so the two tabs feel like the
  // same UI shape rather than two ad-hoc designs. We only render this
  // when canAct is true, so we don't need to handle a disabled visual
  // state for the items themselves.
  return (
    <div
      style={{
        width: 200,
        flexShrink: 0,
        borderRight: "1px solid var(--app-border)",
        background: "var(--app-surface)",
        padding: "var(--pad-medium) 0",
        overflow: "auto",
      }}
    >
      {SUB_TABS.map((t) => {
        const isActive = t.id === active;
        return (
          <button
            key={t.id}
            onClick={() => onChange(t.id)}
            style={{
              display: "block",
              width: "100%",
              textAlign: "left",
              border: "none",
              borderRadius: 0,
              padding: "6px 12px",
              fontSize: "var(--fs-x-small)",
              color: isActive
                ? "var(--core-fleet-green)"
                : "var(--app-text)",
              background: isActive ? "var(--tint-success-soft)" : undefined,
              borderLeft: isActive
                ? "2px solid var(--core-fleet-green)"
                : "2px solid transparent",
            }}
          >
            {t.label}
          </button>
        );
      })}
    </div>
  );
}

// ---------- login ----------

type LoginMode = "credentials" | "token";

// Loopback addresses serve Fleet's self-signed dev cert, which the macOS
// platform verifier rejects ("x509: certificate is not standard
// compliant"). A tunneled/remote URL (ngrok etc.) presents a real CA cert
// and must keep verifying — so skip-verify auto-on is scoped to loopback
// only. Mirrors fleetctl preview, which sets TLSSkipVerify for local dev.
// Dev default used when a context has no stored address.
const DEFAULT_FLEET_ADDRESS = "https://localhost:8080";

function isLoopbackUrl(addr: string): boolean {
  try {
    const h = new URL(addr.trim()).hostname.toLowerCase();
    return (
      h === "localhost" ||
      h === "127.0.0.1" ||
      h === "::1" ||
      h.endsWith(".localhost")
    );
  } catch {
    const s = addr.toLowerCase();
    return s.includes("localhost") || s.includes("127.0.0.1");
  }
}

// Heuristic match for the self-signed-cert failure so we can nudge the
// user toward skip-verify. Covers the macOS verifier message and the
// generic x509 path.
function looksLikeTlsCertError(msg: string): boolean {
  const m = msg.toLowerCase();
  return (
    m.includes("x509") ||
    m.includes("certificate is not standard compliant") ||
    (m.includes("tls") && m.includes("certificate"))
  );
}

function LoginPanel({
  binary,
  ctx,
  canAct,
  serverUp,
  repoPath,
  selectedContext,
  onLoggedIn,
  setError,
}: {
  binary: ResolvedBinary | null;
  ctx: ContextInfo | null;
  canAct: boolean;
  serverUp: boolean;
  repoPath: string;
  selectedContext: string;
  onLoggedIn: () => Promise<void>;
  setError: (e: string | null) => void;
}) {
  const [mode, setMode] = useState<LoginMode>("credentials");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [token, setToken] = useState("");
  // Look up the selected context's stored address as the initial value.
  // Fall back to the dev default. We re-sync this whenever the user
  // switches contexts in the header (below).
  const selectedAddress =
    ctx?.contexts.find((c) => c.name === selectedContext)?.address ?? null;
  const [address, setAddress] = useState(
    selectedAddress ?? DEFAULT_FLEET_ADDRESS,
  );
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  // fleetctl persists tls-skip-verify per-context and defaults it to
  // false, so a self-signed local instance otherwise fails login with
  // "x509: certificate is not standard compliant". Default it ON for
  // loopback URLs (the self-signed dev case) and OFF otherwise.
  const [skipTlsVerify, setSkipTlsVerify] = useState(() =>
    isLoopbackUrl(selectedAddress ?? DEFAULT_FLEET_ADDRESS),
  );
  // Mirror in a ref so the "Enable & retry" action can flip it on and
  // immediately re-run login without waiting for a state-update render.
  const skipRef = useRef(skipTlsVerify);
  skipRef.current = skipTlsVerify;
  // Shown after a login fails with a cert error while skip-verify is off.
  const [showTlsHint, setShowTlsHint] = useState(false);

  // Re-seed the address field when the user picks a different context
  // (or when refreshCtx brings new data in). The user's manual edits
  // for the *current* selection are preserved via userEditedAddress —
  // we reset that flag on selection change so a context switch always
  // pulls the stored value.
  const userEditedAddress = useRef(false);
  // Likewise: once the user ticks/unticks the box themselves, stop
  // auto-deriving it from the address. Reset on context switch so a fresh
  // context re-derives from its stored address.
  const userToggledSkipVerify = useRef(false);
  useEffect(() => {
    userEditedAddress.current = false;
    userToggledSkipVerify.current = false;
  }, [selectedContext]);
  useEffect(() => {
    // Keep skip-verify in sync with whether the target is loopback, until
    // the user overrides it manually.
    if (!userToggledSkipVerify.current) {
      setSkipTlsVerify(isLoopbackUrl(address));
    }
  }, [address]);
  useEffect(() => {
    // On a context switch, pull the stored address — or fall back to the dev
    // default when the new context has none, so a stale URL from the previous
    // context isn't carried over (and then written into the fresh context on
    // token login).
    if (!userEditedAddress.current) {
      setAddress(selectedAddress ?? DEFAULT_FLEET_ADDRESS);
    }
  }, [selectedAddress]);

  // Persist tls-skip-verify on the context before login / config-set.
  // Order matters in the credentials flow: `fleetctl login` reads the
  // stored context, so the flag has to be set first. It's idempotent, so
  // running it first in the token flow too is harmless. Returns false (and
  // sets the error) on failure so callers can bail early.
  async function applySkipTlsVerify(): Promise<boolean> {
    if (!binary) return false;
    // Always persist the explicit value: passing it only when enabled would
    // leave a previously-enabled context skipping verification after the user
    // unchecks the box. `config set` is idempotent, so writing it every login
    // is harmless.
    const run = await api.fleetctlRunCapture({
      program: binary.path,
      cwd: repoPath,
      args: [
        "config",
        "set",
        "--context",
        selectedContext,
        `--tls-skip-verify=${skipRef.current ? "true" : "false"}`,
      ],
      timeoutMs: 15_000,
    });
    if (run.exit_code !== 0) {
      setError(
        (run.stderr || run.stdout).trim() || "failed to set tls-skip-verify",
      );
      return false;
    }
    return true;
  }

  // Surface a failure and, if it looks like the self-signed-cert error and
  // skip-verify isn't already on, raise the hint nudging the user to it.
  function reportLoginFailure(msg: string) {
    setError(msg);
    if (!skipRef.current && looksLikeTlsCertError(msg)) setShowTlsHint(true);
  }

  // One-click recovery from the hint: flip skip-verify on (and remember it
  // as a manual choice) and immediately re-run the current login flow. We
  // set the ref synchronously so applySkipTlsVerify sees it this run.
  function enableSkipAndRetry() {
    userToggledSkipVerify.current = true;
    skipRef.current = true;
    setSkipTlsVerify(true);
    setShowTlsHint(false);
    if (mode === "credentials") void doCredentialsLogin();
    else void doTokenLogin();
  }

  async function doCredentialsLogin() {
    if (!binary?.exists) return;
    setBusy(true);
    setError(null);
    setResult(null);
    setShowTlsHint(false);
    try {
      // Set tls-skip-verify first — login reads the stored context.
      if (!(await applySkipTlsVerify())) {
        setBusy(false);
        return;
      }
      // Pass the password via env var rather than --password so it
      // doesn't show up in `ps`. EMAIL is non-sensitive but symmetric.
      const run = await api.fleetctlRunCapture({
        program: binary.path,
        cwd: repoPath,
        // --context is a per-subcommand flag in fleetctl (urfave/cli);
        // it must appear after the verb, not before.
        args: ["login", "--context", selectedContext],
        env: { EMAIL: email, PASSWORD: password },
        timeoutMs: 30_000,
      });
      if (run.exit_code === 0) {
        setResult(run.stdout.trim() || "Logged in.");
        setPassword("");
        await onLoggedIn();
      } else {
        const msg = (run.stderr || run.stdout).trim();
        reportLoginFailure(msg || `fleetctl login exited ${run.exit_code}`);
      }
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  async function doTokenLogin() {
    if (!binary?.exists) return;
    setBusy(true);
    setError(null);
    setResult(null);
    setShowTlsHint(false);
    try {
      // tls-skip-verify first (idempotent), then address, then token.
      if (!(await applySkipTlsVerify())) {
        setBusy(false);
        return;
      }
      // Two steps: address first (idempotent), then token. Doing it as
      // two separate calls keeps the error messages precise — a typo'd
      // address shouldn't make the user think their token is bad.
      const setAddr = await api.fleetctlRunCapture({
        program: binary.path,
        cwd: repoPath,
        args: [
          "config",
          "set",
          "--context",
          selectedContext,
          "--address",
          address,
        ],
        timeoutMs: 15_000,
      });
      if (setAddr.exit_code !== 0) {
        reportLoginFailure(
          (setAddr.stderr || setAddr.stdout).trim() || "failed to set address",
        );
        setBusy(false);
        return;
      }
      const setTok = await api.fleetctlRunCapture({
        program: binary.path,
        cwd: repoPath,
        args: [
          "config",
          "set",
          "--context",
          selectedContext,
          "--token",
          token,
        ],
        timeoutMs: 15_000,
      });
      if (setTok.exit_code !== 0) {
        reportLoginFailure(
          (setTok.stderr || setTok.stdout).trim() || "failed to set token",
        );
        setBusy(false);
        return;
      }
      setResult("Token saved. Verifying…");
      // Light verification: try a cheap get. Failure here is informational.
      const verify = await api.fleetctlRunCapture({
        program: binary.path,
        cwd: repoPath,
        args: ["get", "hosts", "--context", selectedContext],
        timeoutMs: 15_000,
      });
      if (verify.exit_code === 0) {
        setResult("Token saved and verified.");
      } else {
        const raw = (verify.stderr || verify.stdout).trim();
        setResult(`Token saved, but verification failed: ${raw.slice(-300) || "see Logs"}`);
        if (!skipRef.current && looksLikeTlsCertError(raw)) setShowTlsHint(true);
      }
      setToken("");
      await onLoggedIn();
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  const disabled =
    !canAct ||
    busy ||
    (mode === "credentials"
      ? !email.trim() || !password
      : !token.trim() || !address.trim());

  return (
    <div className="card" style={{ padding: 14, maxWidth: 560 }}>
      <div style={{ display: "flex", gap: 6, marginBottom: 12 }}>
        <ModeChip
          active={mode === "credentials"}
          label="Email + password"
          onClick={() => setMode("credentials")}
        />
        <ModeChip
          active={mode === "token"}
          label="Paste token (SSO/MFA)"
          onClick={() => setMode("token")}
        />
      </div>

      {mode === "credentials" ? (
        <>
          <Field label="email">
            <input
              type="email"
              autoComplete="username"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@example.com"
              {...noAutocorrect}
              style={fieldStyle}
            />
          </Field>
          <Field label="password">
            <input
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              {...noAutocorrect}
              style={fieldStyle}
            />
          </Field>
          <div className="dim" style={{ fontSize: "var(--fs-xxx-small)", marginBottom: 12 }}>
            Sent via env vars (EMAIL/PASSWORD) so the password doesn't
            land in the process list.
          </div>
        </>
      ) : (
        <>
          <Field label="server URL">
            <input
              value={address}
              onChange={(e) => {
                userEditedAddress.current = true;
                setAddress(e.target.value);
              }}
              placeholder="https://localhost:8080"
              {...noAutocorrect}
              style={fieldStyle}
            />
          </Field>
          <Field label="API token">
            <input
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="paste from Fleet UI → My account"
              {...noAutocorrect}
              style={fieldStyle}
            />
          </Field>
          <div className="dim" style={{ fontSize: "var(--fs-xxx-small)", marginBottom: 12 }}>
            For SSO/MFA users: grab the token from the Fleet UI &gt; My
            account. We'll run{" "}
            <span className="mono">fleetctl config set</span> for you.
          </div>
        </>
      )}

      <label
        style={{
          display: "flex",
          alignItems: "flex-start",
          gap: 8,
          marginBottom: 12,
          cursor: "pointer",
        }}
      >
        <input
          type="checkbox"
          checked={skipTlsVerify}
          onChange={(e) => {
            userToggledSkipVerify.current = true;
            setSkipTlsVerify(e.target.checked);
            if (e.target.checked) setShowTlsHint(false);
          }}
          style={{ marginTop: 1, flexShrink: 0 }}
        />
        <span style={{ fontSize: "var(--fs-xxx-small)", lineHeight: 1.4 }}>
          <span style={{ color: "var(--app-text)" }}>
            Skip TLS verification
          </span>{" "}
          <span style={{ color: "var(--ui-error)" }}>(insecure)</span>
          <span className="dim">
            {" "}
            — for self-signed local instances. Runs{" "}
            <span className="mono">config set --tls-skip-verify</span> on this
            context first.
          </span>
        </span>
      </label>

      {showTlsHint && !skipTlsVerify && (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 10,
            marginBottom: 12,
            padding: "8px 10px",
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
            background: "var(--app-surface-2)",
          }}
        >
          <span style={{ fontSize: "var(--fs-xxx-small)", flex: 1 }}>
            That looks like a self-signed certificate. Enable{" "}
            <span style={{ color: "var(--app-text)" }}>
              Skip TLS verification
            </span>{" "}
            and try again.
          </span>
          <button
            onClick={enableSkipAndRetry}
            disabled={busy}
            style={{ fontSize: "var(--fs-xxx-small)", flexShrink: 0 }}
          >
            Enable &amp; retry
          </button>
        </div>
      )}

      <div
        style={{
          display: "flex",
          justifyContent: "flex-end",
          alignItems: "center",
          gap: 8,
        }}
      >
        {!serverUp && (
          <span
            className="dim"
            style={{
              fontSize: "var(--fs-xxx-small)",
              marginRight: "auto",
            }}
          >
            fleet serve is down
          </span>
        )}
        <button
          className="primary"
          disabled={disabled}
          onClick={mode === "credentials" ? doCredentialsLogin : doTokenLogin}
          style={{ padding: "6px 14px" }}
        >
          {busy ? "…" : mode === "credentials" ? "Log in" : "Save token"}
        </button>
      </div>

      {result && (
        <div
          className="dim"
          style={{
            fontSize: "var(--fs-xx-small)",
            marginTop: 10,
            color: "var(--core-fleet-green)",
          }}
        >
          {result}
        </div>
      )}
    </div>
  );
}

function ModeChip({
  active,
  label,
  onClick,
}: {
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: "4px 12px",
        fontSize: "var(--fs-xxx-small)",
        background: active ? "var(--tint-success-soft)" : "transparent",
        border: `1px solid ${active ? "var(--core-fleet-green)" : "var(--app-border)"}`,
        color: active ? "var(--core-fleet-green)" : "var(--app-text-dim)",
        borderRadius: 999,
      }}
    >
      {label}
    </button>
  );
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div style={{ marginBottom: 10 }}>
      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)", marginBottom: 4 }}
      >
        {label}
      </div>
      {children}
    </div>
  );
}

const fieldStyle: React.CSSProperties = {
  width: "100%",
  boxSizing: "border-box",
  background: "var(--app-surface-2)",
  border: "1px solid var(--app-border)",
  borderRadius: 5,
  padding: "6px 10px",
  fontFamily: "var(--font-mono)",
  fontSize: "var(--fs-xx-small)",
  color: "var(--app-text)",
};

// ---------- get ----------

type GetEntity = "hosts" | "host" | "reports" | "labels" | "fleets";

const GET_ENTITIES: { id: GetEntity; label: string; needsId?: boolean }[] = [
  { id: "hosts", label: "hosts (list)" },
  { id: "host", label: "host by id", needsId: true },
  { id: "reports", label: "reports" },
  { id: "labels", label: "labels" },
  { id: "fleets", label: "fleets" },
];

function GetPanel({
  binary,
  canAct,
  repoPath,
  selectedContext,
  setError,
}: {
  binary: ResolvedBinary | null;
  canAct: boolean;
  repoPath: string;
  selectedContext: string;
  setError: (e: string | null) => void;
}) {
  const [entity, setEntity] = useState<GetEntity>("hosts");
  const [hostId, setHostId] = useState("");
  const [busy, setBusy] = useState(false);
  // Per-entity output so the user can cross-reference between tabs
  // without losing what they just ran. Keyed by GetEntity; "host"
  // intentionally keeps only the latest result regardless of which
  // id was queried — that's the lone moving-target tab.
  const [outputs, setOutputs] = useState<Partial<Record<GetEntity, string>>>(
    {},
  );
  const [exitCodes, setExitCodes] = useState<
    Partial<Record<GetEntity, number | null>>
  >({});

  const output = outputs[entity] ?? "";
  const exitCode = exitCodes[entity] ?? null;

  const argsPreview = useMemo(() => {
    const tail = entity === "host" ? `host ${hostId || "<id>"}` : entity;
    const ctxFragment =
      selectedContext === "default" ? "" : ` --context ${selectedContext}`;
    return `fleetctl get ${tail}${ctxFragment}`;
  }, [entity, hostId, selectedContext]);

  async function run() {
    if (!binary?.exists) return;
    const meta = GET_ENTITIES.find((e) => e.id === entity);
    if (meta?.needsId && !hostId.trim()) return;
    const runEntity = entity;
    setBusy(true);
    setError(null);
    setOutputs((prev) => ({ ...prev, [runEntity]: "" }));
    setExitCodes((prev) => ({ ...prev, [runEntity]: null }));
    try {
      // --context is a per-subcommand flag in fleetctl; it goes after
      // the verb (here: after "get <entity>"), not before. urfave/cli
      // rejects it if placed before the subcommand. We let fleetctl
      // pick its default output format — table for list commands,
      // YAML for `get host <id>` — both of which read better than JSON.
      const args =
        runEntity === "host"
          ? ["get", "host", hostId.trim(), "--context", selectedContext]
          : ["get", runEntity, "--context", selectedContext];
      const r = await api.fleetctlRunCapture({
        program: binary.path,
        cwd: repoPath,
        args,
        timeoutMs: 60_000,
      });
      // Write back keyed by the entity we *started* with, not the
      // current selection — the user may have switched tabs while the
      // request was in flight, and we don't want results to land on
      // the wrong tab.
      setExitCodes((prev) => ({ ...prev, [runEntity]: r.exit_code }));
      // Some `get` commands emit progress on stderr even when they succeed.
      // Surface both, with stderr appended if it has content.
      const body = r.stdout + (r.stderr ? `\n--- stderr ---\n${r.stderr}` : "");
      setOutputs((prev) => ({
        ...prev,
        [runEntity]: body || "(no output)",
      }));
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  const disabled =
    !canAct ||
    busy ||
    (GET_ENTITIES.find((e) => e.id === entity)?.needsId && !hostId.trim());

  return (
    <div
      className="card"
      style={{ padding: 14, display: "flex", flexDirection: "column", gap: 10, minHeight: 0 }}
    >
      <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
        {GET_ENTITIES.map((e) => (
          <button
            key={e.id}
            onClick={() => setEntity(e.id)}
            className={entity === e.id ? "primary" : ""}
            style={{
              padding: "4px 12px",
              fontSize: "var(--fs-xx-small)",
              opacity: !canAct ? 0.5 : 1,
            }}
            disabled={!canAct}
          >
            {e.label}
          </button>
        ))}
      </div>

      {entity === "host" && (
        <input
          value={hostId}
          onChange={(e) => setHostId(e.target.value)}
          placeholder="host id (numeric) or identifier"
          {...noAutocorrect}
          style={{ ...fieldStyle, maxWidth: 320 }}
          disabled={!canAct}
        />
      )}

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 8,
        }}
      >
        <span
          className="mono dim"
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          {argsPreview}
        </span>
        <div style={{ display: "flex", gap: 6 }}>
          {output && (
            <>
              <button
                onClick={() => navigator.clipboard.writeText(output)}
                style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
              >
                Copy
              </button>
              <button
                onClick={() =>
                  setOutputs((prev) => ({ ...prev, [entity]: "" }))
                }
                style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
              >
                Clear
              </button>
            </>
          )}
          <button
            className="primary"
            disabled={disabled}
            onClick={run}
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            {busy ? "running…" : "▶ Run"}
          </button>
        </div>
      </div>

      {exitCode != null && exitCode !== 0 && (
        <div
          style={{
            fontSize: "var(--fs-xxx-small)",
            color: "var(--ui-error)",
          }}
        >
          fleetctl exited {exitCode}
        </div>
      )}

      <pre
        style={{
          margin: 0,
          padding: "10px 12px",
          background: "var(--log-bg, var(--app-surface-2))",
          color: "var(--app-text)",
          fontFamily: "var(--font-mono)",
          fontSize: "var(--fs-xxx-small)",
          borderRadius: "var(--radius-md)",
          maxHeight: 480,
          overflow: "auto",
          // pre (not pre-wrap) so fleetctl's ASCII tables don't get
          // line-wrapped — the container scrolls horizontally instead.
          whiteSpace: "pre",
          border: "1px solid var(--app-border)",
          flex: 1,
          minHeight: 200,
        }}
      >
        {output || (busy ? "running…" : "")}
      </pre>
    </div>
  );
}

// ---------- trigger ----------

function TriggerPanel({
  binary,
  canAct,
  repoPath,
  selectedContext,
  goToLogs,
  setError,
  favorites,
  onToggleFavorite,
}: {
  binary: ResolvedBinary | null;
  canAct: boolean;
  repoPath: string;
  selectedContext: string;
  goToLogs: () => void;
  setError: (e: string | null) => void;
  favorites: Set<string>;
  onToggleFavorite: (name: string) => void;
}) {
  const [lastTriggered, setLastTriggered] = useState<Record<string, number>>(
    {},
  );
  // Per-section output log: each trigger run appends an entry to the
  // buffer of the section the user *clicked from* (NOT cron.group), so
  // triggering the same cron from Favorites and from its home group
  // produces isolated history streams. Newest first; capped at
  // MAX_GROUP_ENTRIES so the history doesn't grow without bound.
  // Streaming still lands in the Logs tab via startProcess; this is the
  // final-snapshot history pinned next to the buttons that produced it.
  const [groupOutputs, setGroupOutputs] = useState<
    Partial<Record<OutputSource, GroupEntry[]>>
  >({});
  const [busyCron, setBusyCron] = useState<string | null>(null);
  const [showFast, setShowFast] = useState(false);
  const [showMigration, setShowMigration] = useState(false);
  const [now, setNow] = useState(Date.now());

  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 15_000);
    return () => window.clearInterval(id);
  }, []);

  async function trigger(cron: CronInfo, source: OutputSource) {
    if (!binary?.exists) return;
    setBusyCron(cron.name);
    setError(null);
    try {
      // Use the managed-process pipeline so streaming output also lands
      // in the Logs tab — vulnerabilities can take minutes and seeing
      // progress live is useful. We separately capture the recent_log
      // snapshot after exit for inline display below the card.
      const id = `${TRIGGER_PROC_PREFIX}${cron.name}`;
      await api.startProcess({
        id,
        label: `trigger ${cron.name}`,
        cwd: repoPath,
        program: binary.path,
        args: [
          "trigger",
          "--context",
          selectedContext,
          "--name",
          cron.name,
        ],
      });
      const ok = await waitForExit(id);
      // Pull the captured output regardless of success — failures'
      // error text is the whole point of showing this inline. Append
      // (newest first, capped) to the source section's buffer so output
      // is scoped to wherever the user clicked from.
      const procs = await api.listProcesses();
      const proc = procs.find((p) => p.id === id);
      const body = (proc?.recent_log ?? []).join("\n").trim();
      const entry: GroupEntry = {
        ts: Date.now(),
        cron: cron.name,
        body: body || (ok ? "(no output)" : "(no output before exit)"),
        exitCode: proc?.exit_code ?? null,
      };
      setGroupOutputs((prev) => {
        const current = prev[source] ?? [];
        return {
          ...prev,
          [source]: [entry, ...current].slice(0, MAX_GROUP_ENTRIES),
        };
      });
      if (ok) {
        setLastTriggered((prev) => ({ ...prev, [cron.name]: Date.now() }));
      }
    } catch (e) {
      setError(String(e));
    }
    setBusyCron(null);
  }

  const groups: CronGroup[] = [
    "featured",
    "mdm",
    "maintenance",
    "fast",
    "migration",
  ];

  // Resolve favorites in CRONS order so the section is deterministic
  // regardless of click sequence. A starred name that no longer exists
  // in the registry is silently dropped — keeps the UI from rendering
  // ghost rows after a Fleet upgrade renames a cron.
  const favoriteCrons = CRONS.filter((c) => favorites.has(c.name));

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: "var(--pad-medium)",
        minHeight: 0,
      }}
    >
      {favoriteCrons.length > 0 && (
        <div className="card" style={{ padding: 14 }}>
          <div style={{ marginBottom: 10 }}>
            <div className="section-title" style={{ margin: 0 }}>
              Favorites{" "}
              <span
                className="dim"
                style={{
                  fontSize: "var(--fs-xxx-small)",
                  fontWeight: 400,
                  textTransform: "none",
                  letterSpacing: 0,
                }}
              >
                · {favoriteCrons.length}
              </span>
            </div>
            <div
              className="dim"
              style={{ fontSize: "var(--fs-xxx-small)" }}
            >
              Your starred crons. Triggering here keeps output isolated
              from the home group.
            </div>
          </div>
          <div
            style={{
              display: "grid",
              gridTemplateColumns:
                "repeat(auto-fill, minmax(260px, 1fr))",
              gap: 8,
            }}
          >
            {favoriteCrons.map((c) => (
              <CronRow
                key={`fav-${c.name}`}
                cron={c}
                busy={busyCron === c.name}
                disabled={!canAct || busyCron != null}
                lastTriggered={lastTriggered[c.name] ?? null}
                now={now}
                onTrigger={() => trigger(c, "favorites")}
                onLogs={goToLogs}
                favorited
                onToggleFavorite={() => onToggleFavorite(c.name)}
              />
            ))}
          </div>
          <GroupOutput
            entries={groupOutputs["favorites"] ?? []}
            onClear={() =>
              setGroupOutputs((prev) => {
                const next = { ...prev };
                delete next["favorites"];
                return next;
              })
            }
          />
        </div>
      )}
      {groups.map((g) => {
        const items = CRONS.filter((c) => c.group === g);
        if (items.length === 0) return null;
        const collapsible = g === "fast" || g === "migration";
        const open =
          (g === "fast" && showFast) ||
          (g === "migration" && showMigration) ||
          !collapsible;
        return (
          <div key={g} className="card" style={{ padding: 14 }}>
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: open ? 10 : 0,
                cursor: collapsible ? "pointer" : "default",
              }}
              onClick={() => {
                if (g === "fast") setShowFast((v) => !v);
                if (g === "migration") setShowMigration((v) => !v);
              }}
            >
              <div>
                <div className="section-title" style={{ margin: 0 }}>
                  {CRON_GROUP_TITLE[g]}{" "}
                  <span
                    className="dim"
                    style={{
                      fontSize: "var(--fs-xxx-small)",
                      fontWeight: 400,
                      textTransform: "none",
                      letterSpacing: 0,
                    }}
                  >
                    · {items.length}
                  </span>
                </div>
                <div
                  className="dim"
                  style={{ fontSize: "var(--fs-xxx-small)" }}
                >
                  {CRON_GROUP_SUBTITLE[g]}
                </div>
              </div>
              {collapsible && (
                <button
                  style={{
                    padding: "2px 10px",
                    fontSize: "var(--fs-xxx-small)",
                  }}
                >
                  {open ? "Hide" : "Show"}
                </button>
              )}
            </div>
            {open && (
              <>
                <div
                  style={{
                    display: "grid",
                    gridTemplateColumns:
                      "repeat(auto-fill, minmax(260px, 1fr))",
                    gap: 8,
                  }}
                >
                  {items.map((c) => (
                    <CronRow
                      key={c.name}
                      cron={c}
                      busy={busyCron === c.name}
                      disabled={!canAct || busyCron != null}
                      lastTriggered={lastTriggered[c.name] ?? null}
                      now={now}
                      onTrigger={() => trigger(c, g)}
                      onLogs={goToLogs}
                      favorited={favorites.has(c.name)}
                      onToggleFavorite={() => onToggleFavorite(c.name)}
                    />
                  ))}
                </div>
                <GroupOutput
                  entries={groupOutputs[g] ?? []}
                  onClear={() =>
                    setGroupOutputs((prev) => {
                      const next = { ...prev };
                      delete next[g];
                      return next;
                    })
                  }
                />
              </>
            )}
          </div>
        );
      })}
    </div>
  );
}

function CronRow({
  cron,
  busy,
  disabled,
  lastTriggered,
  now,
  onTrigger,
  onLogs,
  favorited,
  onToggleFavorite,
}: {
  cron: CronInfo;
  busy: boolean;
  disabled: boolean;
  lastTriggered: number | null;
  now: number;
  onTrigger: () => void;
  onLogs: () => void;
  favorited: boolean;
  onToggleFavorite: () => void;
}) {
  return (
    <div
      style={{
        background: "var(--app-surface-2)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "8px 10px",
        display: "flex",
        flexDirection: "column",
        gap: 4,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          gap: 8,
        }}
      >
        <span
          className="mono"
          style={{
            fontSize: "var(--fs-xx-small)",
            color: "var(--app-text)",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
            flex: 1,
          }}
          title={cron.name}
        >
          {cron.name}
        </span>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 6,
            flexShrink: 0,
          }}
        >
          <button
            onClick={onToggleFavorite}
            title={favorited ? "Unfavorite" : "Add to favorites"}
            aria-label={favorited ? "Unfavorite" : "Add to favorites"}
            aria-pressed={favorited}
            style={{
              padding: 0,
              border: "none",
              background: "transparent",
              lineHeight: 1,
              fontSize: "var(--fs-x-small)",
              color: favorited
                ? "var(--ui-warning)"
                : "var(--app-text-dim)",
            }}
          >
            {favorited ? "★" : "☆"}
          </button>
          <span
            className="dim"
            style={{
              fontSize: "var(--fs-xxx-small)",
              background: "var(--app-surface)",
              padding: "1px 6px",
              borderRadius: 999,
              border: "1px solid var(--app-border)",
            }}
          >
            {cron.interval}
          </span>
        </div>
      </div>
      {cron.note && (
        <div
          className="dim"
          style={{
            fontSize: "var(--fs-xxx-small)",
            lineHeight: 1.4,
            display: "-webkit-box",
            WebkitLineClamp: 2,
            WebkitBoxOrient: "vertical",
            overflow: "hidden",
          }}
          title={cron.note}
        >
          {cron.note}
        </div>
      )}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 6,
          marginTop: 2,
        }}
      >
        <span
          className="dim"
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          {busy
            ? "triggering…"
            : lastTriggered
              ? `triggered ${humanAgo(now - lastTriggered)}`
              : ""}
        </span>
        <div style={{ display: "flex", gap: 4 }}>
          {busy && (
            <button
              onClick={onLogs}
              style={{
                padding: "2px 8px",
                fontSize: "var(--fs-xxx-small)",
              }}
            >
              logs ↗
            </button>
          )}
          <button
            disabled={disabled}
            onClick={onTrigger}
            className="primary"
            style={{
              padding: "2px 10px",
              fontSize: "var(--fs-xxx-small)",
            }}
          >
            {busy ? "…" : "▶ Trigger"}
          </button>
        </div>
      </div>
    </div>
  );
}

function GroupOutput({
  entries,
  onClear,
}: {
  entries: GroupEntry[];
  onClear: () => void;
}) {
  return (
    <div
      style={{
        marginTop: 12,
        background: "var(--app-surface-2)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "8px 10px",
        display: "flex",
        flexDirection: "column",
        gap: 6,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          fontSize: "var(--fs-xxx-small)",
        }}
      >
        <span
          className="dim"
          style={{
            textTransform: "uppercase",
            letterSpacing: "0.06em",
          }}
        >
          output{entries.length > 0 ? ` · ${entries.length}` : ""}
        </span>
        {entries.length > 0 && (
          <button
            onClick={onClear}
            style={{
              padding: "2px 8px",
              fontSize: "var(--fs-xxx-small)",
            }}
          >
            Clear
          </button>
        )}
      </div>
      <div
        style={{
          // Capped height keeps the section from pushing groups
          // offscreen when the user has fired a bunch of triggers.
          // Scrolls internally — newest entries are at the top so the
          // most recent run is always immediately visible.
          maxHeight: 260,
          overflow: "auto",
          background: "var(--log-bg, var(--app-surface))",
          border: "1px solid var(--app-border)",
          borderRadius: "var(--radius-md)",
          padding: entries.length === 0 ? "10px 12px" : "6px 0",
        }}
      >
        {entries.length === 0 ? (
          <div
            className="dim"
            style={{ fontSize: "var(--fs-xxx-small)" }}
          >
            no triggers run yet · output appears here
          </div>
        ) : (
          entries.map((e, i) => (
            <GroupOutputEntry key={`${e.ts}-${i}`} entry={e} />
          ))
        )}
      </div>
    </div>
  );
}

function GroupOutputEntry({ entry }: { entry: GroupEntry }) {
  const failed = entry.exitCode != null && entry.exitCode !== 0;
  return (
    <div
      style={{
        padding: "6px 12px",
        borderTop: "1px solid var(--app-border)",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "baseline",
          gap: 8,
          fontSize: "var(--fs-xxx-small)",
          marginBottom: 2,
        }}
      >
        <span className="dim mono">{formatClock(entry.ts)}</span>
        <span
          className="mono"
          style={{ color: "var(--app-text)", fontWeight: 600 }}
        >
          {entry.cron}
        </span>
        <span
          style={{
            color: failed
              ? "var(--ui-error)"
              : "var(--core-fleet-green)",
            textTransform: "uppercase",
            letterSpacing: "0.06em",
          }}
        >
          exit {entry.exitCode ?? "?"}
        </span>
      </div>
      <pre
        className="mono"
        style={{
          margin: 0,
          fontSize: "var(--fs-xxx-small)",
          color: failed ? "var(--ui-error)" : "var(--app-text)",
          background: "transparent",
          // pre preserves log alignment; container scrolls horizontally
          // if a line is wider than the box.
          whiteSpace: "pre",
          lineHeight: 1.4,
          overflowX: "auto",
        }}
      >
        {entry.body}
      </pre>
    </div>
  );
}

function formatClock(ms: number): string {
  const d = new Date(ms);
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  const ss = String(d.getSeconds()).padStart(2, "0");
  return `${hh}:${mm}:${ss}`;
}

function humanAgo(ms: number): string {
  const sec = Math.floor(ms / 1000);
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s ago`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
  return `${Math.floor(sec / 86400)}d ago`;
}

// ---------- custom ----------

function CustomPanel({
  binary,
  canAct,
  repoPath,
  selectedContext,
  setError,
}: {
  binary: ResolvedBinary | null;
  canAct: boolean;
  repoPath: string;
  selectedContext: string;
  setError: (e: string | null) => void;
}) {
  const [input, setInput] = useState("");
  const [busy, setBusy] = useState(false);
  const [output, setOutput] = useState("");
  const [exitCode, setExitCode] = useState<number | null>(null);
  const [tokenizeError, setTokenizeError] = useState<string | null>(null);

  // Parse the input as soon as it changes so the preview and the run
  // button can react. We don't want to surface "unmatched quote" as a
  // hard error until the user actually clicks run — incomplete input
  // is normal while typing.
  const parsed = useMemo(() => {
    try {
      const tokens = tokenizeArgs(input);
      const hasContext = tokens.some(
        (t) => t === "--context" || t.startsWith("--context="),
      );
      const finalArgs = hasContext
        ? tokens
        : [...tokens, "--context", selectedContext];
      return { tokens, finalArgs, hasContext, error: null as string | null };
    } catch (e) {
      return {
        tokens: [] as string[],
        finalArgs: [] as string[],
        hasContext: false,
        error: e instanceof Error ? e.message : String(e),
      };
    }
  }, [input, selectedContext]);

  const argsPreview = useMemo(() => {
    if (parsed.error) return "(invalid input)";
    if (parsed.finalArgs.length === 0) return "fleetctl";
    // Show with shell-style quoting so the preview is something the
    // user could actually paste into a terminal.
    return `fleetctl ${parsed.finalArgs.map(shellQuote).join(" ")}`;
  }, [parsed]);

  const disabled =
    !canAct || busy || parsed.tokens.length === 0 || parsed.error != null;

  async function run() {
    if (disabled || !binary?.exists) return;
    setBusy(true);
    setError(null);
    setTokenizeError(null);
    setOutput("");
    setExitCode(null);
    try {
      const r = await api.fleetctlRunCapture({
        program: binary.path,
        cwd: repoPath,
        args: parsed.finalArgs,
        timeoutMs: 120_000,
      });
      setExitCode(r.exit_code);
      const body =
        r.stdout + (r.stderr ? `\n--- stderr ---\n${r.stderr}` : "");
      setOutput(body || "(no output)");
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  return (
    <div
      className="card"
      style={{
        padding: 14,
        display: "flex",
        flexDirection: "column",
        gap: 10,
        minHeight: 0,
      }}
    >
      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)", lineHeight: 1.5 }}
      >
        Type whatever you'd type after <span className="mono">fleetctl</span>{" "}
        on the command line. We supply the binary path and append{" "}
        <span className="mono">--context {selectedContext}</span> if you
        don't include one yourself.
      </div>

      <textarea
        value={input}
        onChange={(e) => {
          setInput(e.target.value);
          setTokenizeError(null);
        }}
        onKeyDown={(e) => {
          // Cmd+Enter (mac) / Ctrl+Enter (other) to run. Shift+Enter
          // still inserts a newline for multi-line commands.
          if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
            e.preventDefault();
            run();
          }
        }}
        placeholder="e.g.  get hosts  ·  version  ·  trigger --name vulnerabilities"
        {...noAutocorrect}
        rows={2}
        style={{
          width: "100%",
          boxSizing: "border-box",
          background: "var(--app-surface-2)",
          color: "var(--app-text)",
          border: `1px solid ${parsed.error ? "var(--ui-error)" : "var(--app-border)"}`,
          borderRadius: "var(--radius-md)",
          padding: "8px 10px",
          fontFamily: "var(--font-mono)",
          fontSize: "var(--fs-xx-small)",
          lineHeight: 1.5,
          resize: "vertical",
          minHeight: 56,
        }}
        disabled={!canAct}
      />

      {parsed.error && (
        <div
          style={{
            color: "var(--ui-error)",
            fontSize: "var(--fs-xxx-small)",
          }}
        >
          {parsed.error}
        </div>
      )}

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 8,
        }}
      >
        <span
          className="mono dim"
          style={{
            fontSize: "var(--fs-xxx-small)",
            minWidth: 0,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
            flex: 1,
          }}
          title={argsPreview}
        >
          {argsPreview}
        </span>
        <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
          {output && (
            <>
              <button
                onClick={() => navigator.clipboard.writeText(output)}
                style={{
                  padding: "4px 10px",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                Copy
              </button>
              <button
                onClick={() => setOutput("")}
                style={{
                  padding: "4px 10px",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                Clear
              </button>
            </>
          )}
          <button
            className="primary"
            disabled={disabled}
            onClick={run}
            title="⌘↵ to run"
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            {busy ? "running…" : "▶ Run"}
          </button>
        </div>
      </div>

      {tokenizeError && (
        <div
          style={{
            color: "var(--ui-error)",
            fontSize: "var(--fs-xxx-small)",
          }}
        >
          {tokenizeError}
        </div>
      )}

      {exitCode != null && exitCode !== 0 && (
        <div
          style={{ fontSize: "var(--fs-xxx-small)", color: "var(--ui-error)" }}
        >
          fleetctl exited {exitCode}
        </div>
      )}

      <pre
        style={{
          margin: 0,
          padding: "10px 12px",
          background: "var(--log-bg, var(--app-surface-2))",
          color: "var(--app-text)",
          fontFamily: "var(--font-mono)",
          fontSize: "var(--fs-xxx-small)",
          borderRadius: "var(--radius-md)",
          maxHeight: 480,
          overflow: "auto",
          whiteSpace: "pre",
          border: "1px solid var(--app-border)",
          flex: 1,
          minHeight: 200,
        }}
      >
        {output || (busy ? "running…" : "")}
      </pre>
    </div>
  );
}

/// Minimal POSIX-ish tokenizer: handles single quotes (literal), double
/// quotes (with \\ and \" escapes), and backslash escapes outside
/// quotes. Throws on unmatched quotes. Enough for the kinds of commands
/// people type into fleetctl — not a full shell parser; no $vars, no
/// backticks, no pipes.
function tokenizeArgs(input: string): string[] {
  const tokens: string[] = [];
  let current = "";
  let started = false;
  let i = 0;
  let inSingle = false;
  let inDouble = false;
  while (i < input.length) {
    const c = input[i];
    if (inSingle) {
      if (c === "'") {
        inSingle = false;
        i++;
        continue;
      }
      current += c;
      i++;
    } else if (inDouble) {
      if (c === '"') {
        inDouble = false;
        i++;
        continue;
      }
      if (c === "\\" && i + 1 < input.length) {
        const next = input[i + 1];
        if (next === '"' || next === "\\" || next === "$" || next === "`") {
          current += next;
          i += 2;
          continue;
        }
      }
      current += c;
      i++;
    } else {
      if (c === "'") {
        inSingle = true;
        started = true;
        i++;
        continue;
      }
      if (c === '"') {
        inDouble = true;
        started = true;
        i++;
        continue;
      }
      if (c === "\\" && i + 1 < input.length) {
        current += input[i + 1];
        started = true;
        i += 2;
        continue;
      }
      if (/\s/.test(c)) {
        if (started) {
          tokens.push(current);
          current = "";
          started = false;
        }
        i++;
        continue;
      }
      current += c;
      started = true;
      i++;
    }
  }
  if (inSingle || inDouble) {
    throw new Error("unmatched quote");
  }
  if (started) tokens.push(current);
  return tokens;
}

/// Re-quote a token for the preview line. Goal: produce something the
/// user could paste into a terminal and get the same effect. Simple
/// rule — single-quote anything containing whitespace or shell-special
/// characters; escape any single quotes inside via the standard
/// '\''  dance.
function shellQuote(s: string): string {
  if (s === "") return "''";
  if (/^[A-Za-z0-9_+,.\/:=@%-]+$/.test(s)) return s;
  return `'${s.replace(/'/g, `'\\''`)}'`;
}
