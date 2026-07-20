import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  api,
  type ContextInfo,
  type ContextSummary,
  type DetectedProcess,
  type EnvVar,
  type GitopsDirScan,
  type NgrokYamlInfo,
  type ServerPorts,
  type ServerProfile,
  type Settings,
  type ThemePreference,
} from "../../lib/ipc";
import {
  activeServer,
  canAddServer,
  MAX_SERVERS,
  serverColorVar,
  updateActiveServer,
  updateServer,
} from "../../lib/servers";
import { staleNgrokTunnels } from "../../lib/orchestration";
import { noAutocorrect } from "../../lib/noAutocorrect";

export type SettingsSection =
  | "servers"
  | "paths"
  | "fleet-server"
  | "fleetctl"
  | "gitops"
  | "ngrok"
  | "python"
  | "troubleshoot";

export function SettingsTab({
  settings,
  onChange,
  section,
  onSectionChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
  section: SettingsSection;
  onSectionChange: (s: SettingsSection) => void;
}) {
  const setTheme = useCallback(
    (theme: ThemePreference) => {
      const next: Settings = { ...settings, theme };
      // Apply the in-memory change first so the body class flips on
      // the same tick the user clicked. The IPC save runs in the
      // background — if it fails the user just loses the preference
      // on the next launch.
      onChange(next);
      api.saveSettings(next).catch((e) => console.error("saveSettings(theme) failed", e));
    },
    [settings, onChange],
  );
  return (
    <div style={{ display: "flex", height: "100%", overflow: "hidden" }}>
      <Sidebar
        value={section}
        onChange={onSectionChange}
        theme={settings.theme}
        onThemeChange={setTheme}
      />
      <div
        style={{
          flex: 1,
          padding: "var(--pad-large)",
          overflow: "auto",
        }}
      >
        {section === "servers" && (
          <ServersSection settings={settings} onChange={onChange} />
        )}
        {section === "paths" && (
          <PathsSection settings={settings} onChange={onChange} />
        )}
        {section === "fleet-server" && (
          <FleetServerSection settings={settings} onChange={onChange} />
        )}
        {section === "fleetctl" && (
          <FleetctlContextsSection settings={settings} />
        )}
        {section === "gitops" && (
          <GitopsSection settings={settings} onChange={onChange} />
        )}
        {section === "ngrok" && (
          <NgrokSection settings={settings} onChange={onChange} />
        )}
        {section === "python" && (
          <PythonSection settings={settings} onChange={onChange} />
        )}
        {section === "troubleshoot" && (
          <TroubleshootSection settings={settings} />
        )}
      </div>
    </div>
  );
}

function Sidebar({
  value,
  onChange,
  theme,
  onThemeChange,
}: {
  value: SettingsSection;
  onChange: (s: SettingsSection) => void;
  theme: ThemePreference;
  onThemeChange: (t: ThemePreference) => void;
}) {
  const Group = ({ title, children }: { title: string; children: React.ReactNode }) => (
    <div style={{ marginBottom: 18 }}>
      <div
        style={{
          fontSize: "var(--fs-xxx-small)",
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          color: "var(--app-text-dim)",
          padding: "4px 12px",
          marginBottom: 4,
        }}
      >
        {title}
      </div>
      {children}
    </div>
  );
  const Item = ({ id, label }: { id: SettingsSection; label: string }) => {
    const active = value === id;
    return (
      <button
        onClick={() => onChange(id)}
        style={{
          display: "block",
          width: "100%",
          textAlign: "left",
          border: "none",
          borderRadius: 0,
          padding: "6px 12px",
          fontSize: "var(--fs-x-small)",
          color: active ? "var(--core-fleet-green)" : "var(--app-text)",
          background: active ? "var(--tint-success-soft)" : undefined,
          borderLeft: active
            ? "2px solid var(--core-fleet-green)"
            : "2px solid transparent",
        }}
      >
        {label}
      </button>
    );
  };
  return (
    <div
      style={{
        width: 220,
        flexShrink: 0,
        borderRight: "1px solid var(--app-border)",
        background: "var(--app-surface)",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <div style={{ flex: 1, overflow: "auto", padding: "var(--pad-medium) 0" }}>
        <Group title="Setup">
          <Item id="servers" label="Servers" />
          <Item id="paths" label="Paths" />
          <Item id="fleet-server" label="Fleet server (active)" />
          <Item id="fleetctl" label="fleetctl contexts" />
          <Item id="gitops" label="GitOps directory" />
        </Group>
        <Group title="Optional services">
          <Item id="ngrok" label="ngrok" />
          <Item id="python" label="python http.server" />
        </Group>
        <Group title="Diagnostics">
          <Item id="troubleshoot" label="Troubleshoot" />
        </Group>
      </div>
      <ThemeToggle value={theme} onChange={onThemeChange} />
    </div>
  );
}

/// Three-segment Light / Auto / Dark control pinned to the bottom of
/// the Settings sidebar. "Auto" follows the OS via prefers-color-scheme.
/// The active segment uses the same success-tint treatment as a selected
/// sidebar item so the visual vocabulary stays consistent.
function ThemeToggle({
  value,
  onChange,
}: {
  value: ThemePreference;
  onChange: (t: ThemePreference) => void;
}) {
  const options: { id: ThemePreference; label: string; glyph: string }[] = [
    { id: "light", label: "Light", glyph: "☀" },
    { id: "system", label: "Auto", glyph: "◐" },
    { id: "dark", label: "Dark", glyph: "☾" },
  ];
  return (
    <div
      style={{
        borderTop: "1px solid var(--app-border)",
        padding: "var(--pad-smedium) var(--pad-medium)",
      }}
    >
      <div
        style={{
          fontSize: "var(--fs-xxx-small)",
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          color: "var(--app-text-dim)",
          marginBottom: 6,
        }}
      >
        Appearance
      </div>
      <div
        role="radiogroup"
        aria-label="Theme"
        style={{
          display: "flex",
          background: "var(--app-surface-2)",
          border: "1px solid var(--app-border)",
          borderRadius: "var(--radius-md)",
          padding: 2,
          gap: 2,
        }}
      >
        {options.map((opt) => {
          const active = opt.id === value;
          return (
            <button
              key={opt.id}
              role="radio"
              aria-checked={active}
              onClick={() => onChange(opt.id)}
              title={opt.label}
              style={{
                flex: 1,
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                gap: 4,
                padding: "4px 6px",
                fontSize: "var(--fs-xx-small)",
                border: "none",
                borderRadius: "var(--radius-sm)",
                background: active ? "var(--tint-success-soft)" : "transparent",
                color: active ? "var(--core-fleet-green)" : "var(--app-text-dim)",
                fontWeight: active ? 600 : 400,
              }}
            >
              <span aria-hidden style={{ fontSize: "var(--fs-x-small)" }}>
                {opt.glyph}
              </span>
              <span>{opt.label}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function EnableToggle({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 10,
        padding: "10px 14px",
        background: checked ? "var(--tint-success-soft)" : "var(--app-surface)",
        border: checked
          ? "1px solid var(--core-fleet-green)"
          : "1px solid var(--app-border)",
        borderRadius: "var(--radius-lg)",
        cursor: "pointer",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        style={{ accentColor: "var(--core-fleet-green)", marginTop: 2 }}
      />
      <div style={{ minWidth: 0 }}>
        <div
          className="card-title"
          style={{
            color: checked ? "var(--core-fleet-green)" : "var(--app-text)",
          }}
        >
          {label}
        </div>
        <div
          className="dim"
          style={{ fontSize: "var(--fs-xx-small)", marginTop: 2 }}
        >
          {description}
        </div>
      </div>
    </label>
  );
}

function PageHeading({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        fontSize: "var(--fs-medium)",
        fontWeight: 600,
        marginBottom: "var(--pad-medium)",
      }}
    >
      {children}
    </div>
  );
}

function PathField({
  label,
  value,
  placeholder,
  onPick,
  onClear,
  busy,
  hint,
}: {
  label: string;
  value: string | null;
  placeholder: string;
  onPick?: () => void;
  onClear?: () => void;
  busy?: boolean;
  hint?: string;
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)" }}>
        {label}
      </div>
      <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
        <div
          className="mono"
          style={{
            flex: 1,
            background: "var(--app-surface-2)",
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
            padding: "6px 10px",
            color: value ? "var(--app-text)" : "var(--app-text-dim)",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {value ?? placeholder}
        </div>
        {onClear && value && (
          <button onClick={onClear} disabled={busy} title="revert to auto-detect">
            Clear
          </button>
        )}
        {onPick && (
          <button onClick={onPick} disabled={busy}>
            📁 Change
          </button>
        )}
      </div>
      {hint && (
        <div className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
          {hint}
        </div>
      )}
    </div>
  );
}

/* ----- Servers section ----- */

// Lightweight path helpers for deriving a sibling worktree dir (string-only;
// the backend validates the real path on `git worktree add`).
function pathDirname(p: string): string {
  const t = p.replace(/\/+$/, "");
  const i = t.lastIndexOf("/");
  return i <= 0 ? "/" : t.slice(0, i);
}
function pathBasename(p: string): string {
  const t = p.replace(/\/+$/, "");
  const i = t.lastIndexOf("/");
  return i < 0 ? t : t.slice(i + 1);
}

const PORT_FIELDS: { key: keyof ServerPorts; label: string }[] = [
  { key: "server", label: "fleet serve" },
  { key: "mysql", label: "MySQL" },
  { key: "redis", label: "Redis" },
  { key: "s3", label: "S3" },
  { key: "s3_console", label: "S3 console" },
];

const COLOR_KEYS = ["green", "purple", "blue"];

/// Finds host ports used by more than one server. Each entry is "two servers
/// configured with the same number" — the most common pre-launch mistake.
function portConflicts(servers: ServerProfile[]): Map<number, string[]> {
  const byPort = new Map<number, string[]>();
  for (const s of servers) {
    for (const f of PORT_FIELDS) {
      const port = s.ports[f.key];
      const arr = byPort.get(port) ?? [];
      arr.push(`${s.name} ${f.label}`);
      byPort.set(port, arr);
    }
  }
  const conflicts = new Map<number, string[]>();
  for (const [port, who] of byPort) {
    if (who.length > 1) conflicts.set(port, who);
  }
  return conflicts;
}

function ServersSection({
  settings,
  onChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
}) {
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const commit = useCallback(
    async (next: Settings) => {
      setError(null);
      try {
        await api.saveSettings(next);
        onChange(next);
      } catch (e) {
        setError(String(e));
      }
    },
    [onChange],
  );

  async function addServer() {
    setBusy(true);
    setError(null);
    try {
      const profile = await api.newServerProfile();
      await commit({
        ...settings,
        servers: [...settings.servers, profile],
        active_server_id: profile.id,
      });
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  function removeServer(id: string) {
    if (settings.servers.length <= 1) return;
    const servers = settings.servers.filter((s) => s.id !== id);
    const active_server_id =
      settings.active_server_id === id
        ? servers[0].id
        : settings.active_server_id;
    commit({ ...settings, servers, active_server_id });
  }

  const conflicts = portConflicts(settings.servers);
  const primaryRepo = settings.servers[0]?.worktree_path ?? null;

  return (
    <div style={{ maxWidth: 820, display: "flex", flexDirection: "column", gap: "var(--pad-medium)" }}>
      <div>
        <PageHeading>Servers</PageHeading>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)", lineHeight: 1.5 }}>
          Run up to {MAX_SERVERS} independent local Fleet servers in parallel — each on its
          own git worktree (so it can build/run a different branch), ports, and
          docker compose project. Server 1 keeps the canonical dev ports; others
          use offset blocks. Switch the active server from the top bar.
        </div>
      </div>

      {conflicts.size > 0 && (
        <div
          style={{
            background: "var(--tint-warning-soft)",
            border: "1px solid var(--ui-warning)",
            borderRadius: "var(--radius-md)",
            padding: "8px 12px",
            fontSize: "var(--fs-xx-small)",
            color: "var(--ui-on-warning)",
          }}
        >
          ⚠ Port conflict: {Array.from(conflicts.entries()).map(([port, who]) => `${port} (${who.join(" + ")})`).join("; ")}.
          Give each server a distinct port or the second stack will fail to bind.
        </div>
      )}

      {settings.servers.map((s, i) => (
        <ServerCard
          key={s.id}
          server={s}
          index={i}
          isActive={s.id === settings.active_server_id}
          canRemove={settings.servers.length > 1}
          primaryRepo={i === 0 ? null : primaryRepo}
          onSetActive={() => commit({ ...settings, active_server_id: s.id })}
          onUpdate={(updater) => commit(updateServer(settings, s.id, updater))}
          onRemove={() => removeServer(s.id)}
          setError={setError}
        />
      ))}

      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <button
          className="primary"
          onClick={addServer}
          disabled={busy || !canAddServer(settings)}
          style={{ padding: "6px 14px" }}
          title={
            canAddServer(settings)
              ? "Add another local server"
              : `Maximum of ${MAX_SERVERS} servers`
          }
        >
          {busy ? "adding…" : "+ Add server"}
        </button>
        {!canAddServer(settings) && (
          <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
            Maximum of {MAX_SERVERS} servers reached.
          </span>
        )}
      </div>

      {error && (
        <div style={{ color: "var(--ui-error)", fontSize: "var(--fs-xx-small)" }}>
          {error}
        </div>
      )}
    </div>
  );
}

function ServerCard({
  server,
  index,
  isActive,
  canRemove,
  primaryRepo,
  onSetActive,
  onUpdate,
  onRemove,
  setError,
}: {
  server: ServerProfile;
  index: number;
  isActive: boolean;
  canRemove: boolean;
  // The main clone to base new worktrees on (null for server 1, which IS the
  // base / configures its worktree by picking a clone directly).
  primaryRepo: string | null;
  onSetActive: () => void;
  onUpdate: (updater: (s: ServerProfile) => ServerProfile) => void;
  onRemove: () => void;
  setError: (e: string | null) => void;
}) {
  const accent = serverColorVar(server.color);
  const [ref, setRef] = useState("");
  const [creating, setCreating] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);

  async function pickExisting() {
    const result = await api.pickFolder();
    if (!result || typeof result !== "string") return;
    const probed = await api.probeFleetRepo(result);
    if (!probed[0]?.valid) {
      setError(probed[0]?.reason ?? "not a valid fleet repo");
      return;
    }
    setError(null);
    const detected = await api.detectFleetConfig(probed[0].path);
    onUpdate((s) => ({
      ...s,
      worktree_path: probed[0].path,
      fleet_serve: { ...s.fleet_serve, config_path: detected },
    }));
  }

  async function createWorktree() {
    if (!primaryRepo || !ref.trim()) return;
    // Sibling of the main clone, named <clone>-<serverID> (e.g. fleet-s2).
    const dest = `${pathDirname(primaryRepo)}/${pathBasename(primaryRepo)}-${server.id}`;
    setCreating(true);
    setError(null);
    try {
      await api.gitAddWorktree(primaryRepo, dest, ref.trim());
      const detected = await api.detectFleetConfig(dest);
      onUpdate((s) => ({
        ...s,
        worktree_path: dest,
        branch: ref.trim(),
        fleet_serve: { ...s.fleet_serve, config_path: detected },
      }));
      setRef("");
    } catch (e) {
      setError(String(e));
    }
    setCreating(false);
  }

  return (
    <div
      className="card"
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 12,
        borderLeft: `3px solid ${accent}`,
      }}
    >
      {/* header */}
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <span
          aria-hidden
          style={{ width: 10, height: 10, borderRadius: "50%", background: accent, flexShrink: 0 }}
        />
        <input
          value={server.name}
          onChange={(e) => onUpdate((s) => ({ ...s, name: e.target.value }))}
          {...noAutocorrect}
          aria-label="Server name"
          style={{ fontWeight: 600, flex: 1, minWidth: 0, maxWidth: 220 }}
        />
        <span className="mono dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
          {server.compose_project}
        </span>
        <div style={{ flex: 1 }} />
        <ColorPicker
          value={server.color}
          onChange={(color) => onUpdate((s) => ({ ...s, color }))}
        />
        {isActive ? (
          <span
            style={{
              fontSize: "var(--fs-xxx-small)",
              textTransform: "uppercase",
              letterSpacing: "0.06em",
              color: accent,
              border: `1px solid ${accent}`,
              borderRadius: 999,
              padding: "2px 8px",
            }}
          >
            active
          </span>
        ) : (
          <button
            onClick={onSetActive}
            style={{ padding: "3px 10px", fontSize: "var(--fs-xx-small)" }}
          >
            Set active
          </button>
        )}
        {canRemove && (
          <button
            className={confirmRemove ? "danger" : undefined}
            onClick={() => {
              if (!confirmRemove) {
                setConfirmRemove(true);
                return;
              }
              onRemove();
            }}
            onBlur={() => setConfirmRemove(false)}
            title="Remove this server profile (leaves the worktree on disk)"
            style={{ padding: "3px 10px", fontSize: "var(--fs-xx-small)" }}
          >
            {confirmRemove ? "Confirm" : "Remove"}
          </button>
        )}
      </div>

      {/* worktree */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)" }}>
          {index === 0 ? "Fleet clone (worktree base)" : "Git worktree"}
        </div>
        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <div
            className="mono"
            style={{
              flex: 1,
              minWidth: 0,
              background: "var(--app-surface-2)",
              border: "1px solid var(--app-border)",
              borderRadius: "var(--radius-md)",
              padding: "6px 10px",
              color: server.worktree_path ? "var(--app-text)" : "var(--app-text-dim)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
            title={server.worktree_path ?? ""}
          >
            {server.worktree_path ?? "not set"}
          </div>
          <button onClick={pickExisting}>📁 Pick existing</button>
        </div>

        {index > 0 && (
          <div
            style={{
              display: "flex",
              gap: 8,
              alignItems: "center",
              marginTop: 4,
            }}
          >
            <input
              value={ref}
              onChange={(e) => setRef(e.target.value)}
              placeholder="branch / tag / commit (e.g. rc-minor-fleet-v4.86.0)"
              {...noAutocorrect}
              className="mono"
              style={{ flex: 1, minWidth: 0, fontSize: "var(--fs-xx-small)" }}
              disabled={!primaryRepo}
            />
            <button
              onClick={createWorktree}
              disabled={!primaryRepo || !ref.trim() || creating}
              title={
                primaryRepo
                  ? `git worktree add ${pathBasename(primaryRepo)}-${server.id} ${ref || "<ref>"}`
                  : "Configure server 1's clone first"
              }
              style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
            >
              {creating ? "creating…" : "Create worktree"}
            </button>
          </div>
        )}
        <div className="dim" style={{ fontSize: "var(--fs-xxx-small)", marginTop: 2 }}>
          {index === 0
            ? "The main clone. Other servers add git worktrees from it."
            : primaryRepo
              ? `Create checks out the ref into a sibling of the main clone (${pathBasename(primaryRepo)}-${server.id}), or pick an existing clone/worktree.`
              : "Set server 1's clone first to enable worktree creation."}
        </div>
      </div>

      {/* ports */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)" }}>
          Host ports
        </div>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap" }}>
          {PORT_FIELDS.map((f) => (
            <PortField
              key={f.key}
              label={f.label}
              value={server.ports[f.key]}
              onCommit={(n) =>
                onUpdate((s) => ({ ...s, ports: { ...s.ports, [f.key]: n } }))
              }
            />
          ))}
        </div>
      </div>
    </div>
  );
}

function ColorPicker({
  value,
  onChange,
}: {
  value: string;
  onChange: (c: string) => void;
}) {
  return (
    <div style={{ display: "flex", gap: 4 }} role="group" aria-label="Accent color">
      {COLOR_KEYS.map((c) => {
        const selected = c === value;
        return (
          <button
            key={c}
            onClick={() => onChange(c)}
            aria-label={c}
            aria-pressed={selected}
            style={{
              width: 18,
              height: 18,
              padding: 0,
              borderRadius: "50%",
              background: serverColorVar(c),
              border: selected ? "2px solid var(--app-text)" : "2px solid transparent",
              cursor: "pointer",
            }}
          />
        );
      })}
    </div>
  );
}

/// Numeric port input committing on blur (or Enter). Keeps local draft so
/// typing doesn't fire a save per keystroke; reverts an out-of-range value.
function PortField({
  label,
  value,
  onCommit,
}: {
  label: string;
  value: number;
  onCommit: (n: number) => void;
}) {
  const [draft, setDraft] = useState(String(value));
  useEffect(() => {
    setDraft(String(value));
  }, [value]);
  function commit() {
    const n = Number(draft);
    if (!Number.isInteger(n) || n < 1 || n > 65535) {
      setDraft(String(value));
      return;
    }
    if (n !== value) onCommit(n);
  }
  return (
    <label style={{ display: "flex", flexDirection: "column", gap: 2 }}>
      <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
        {label}
      </span>
      <input
        type="number"
        value={draft}
        min={1}
        max={65535}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={commit}
        onKeyDown={(e) => {
          if (e.key === "Enter") (e.target as HTMLInputElement).blur();
        }}
        className="mono"
        style={{ width: 86, fontSize: "var(--fs-xx-small)" }}
      />
    </label>
  );
}

/* ----- Paths section ----- */

function PathsSection({
  settings,
  onChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
}) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function pickFleetctl() {
    // No file-type filter — fleetctl is a bare binary with no extension
    // on Unix. The OS picker only returns existing files anyway, so
    // we trust the path and let fleetctl tab surface "binary missing"
    // if the user later moves it.
    const result = await api.pickFile();
    if (!result || typeof result !== "string") return;
    setError(null);
    await save({ ...settings, fleetctl_path: result });
  }
  async function clearFleetctl() {
    setError(null);
    await save({ ...settings, fleetctl_path: null });
  }
  async function save(next: Settings) {
    setBusy(true);
    try {
      await api.saveSettings(next);
      onChange(next);
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  return (
    <div style={{ maxWidth: 720 }}>
      <PageHeading>Paths</PageHeading>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)", marginBottom: "var(--pad-medium)" }}>
        Fleet repositories / worktrees are configured per server in the{" "}
        <span style={{ color: "var(--app-text)" }}>Servers</span> section. This
        is for the shared fleetctl binary.
      </div>
      <div
        className="card"
        style={{ display: "flex", flexDirection: "column", gap: 16 }}
      >
        <PathField
          label="fleetctl binary"
          value={settings.fleetctl_path}
          placeholder="auto-detect · <repo>/build/fleetctl"
          onPick={pickFleetctl}
          onClear={clearFleetctl}
          busy={busy}
          hint={
            settings.fleetctl_path
              ? "Using the picked binary. Click clear to revert to the active server's build/fleetctl."
              : "Falls back to the active server's <worktree>/build/fleetctl. Pick a different binary to point at a release build elsewhere."
          }
        />
        {error && (
          <div style={{ color: "var(--ui-error)", fontSize: "var(--fs-xx-small)" }}>
            {error}
          </div>
        )}
      </div>
    </div>
  );
}


/* ----- GitOps directory section ----- */

function GitopsSection({
  settings,
  onChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
}) {
  const [scan, setScan] = useState<GitopsDirScan | null>(null);
  const [scanError, setScanError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const rescan = useCallback(async () => {
    const dir = settings.gitops_dir;
    if (!dir) {
      setScan(null);
      setScanError(null);
      return;
    }
    setScanError(null);
    try {
      const s = await api.gitopsListRepos(dir);
      setScan(s);
    } catch (e) {
      setScan(null);
      setScanError(String(e));
    }
  }, [settings.gitops_dir]);

  useEffect(() => {
    rescan();
  }, [rescan]);

  async function pickDir() {
    const result = await api.pickFolder();
    if (!result || typeof result !== "string") return;
    setBusy(true);
    try {
      const next: Settings = { ...settings, gitops_dir: result };
      await api.saveSettings(next);
      onChange(next);
    } finally {
      setBusy(false);
    }
  }

  async function clearDir() {
    setBusy(true);
    try {
      const next: Settings = { ...settings, gitops_dir: null };
      await api.saveSettings(next);
      onChange(next);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div style={{ maxWidth: 720, display: "flex", flexDirection: "column", gap: "var(--pad-medium)" }}>
      <div>
        <PageHeading>GitOps directory</PageHeading>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          Folder containing one or more gitops repos (each with a{" "}
          <span className="mono">default.yml</span>). If the folder itself
          contains <span className="mono">default.yml</span>, the GitOps tab
          treats it as a single repo and hides the repo list.
        </div>
      </div>

      <div
        className="card"
        style={{ display: "flex", flexDirection: "column", gap: 12 }}
      >
        <PathField
          label="Root directory"
          value={settings.gitops_dir}
          placeholder="not set"
          onPick={pickDir}
          onClear={settings.gitops_dir ? clearDir : undefined}
          busy={busy}
          hint={
            settings.gitops_dir
              ? scan?.single_repo_mode
                ? "Detected as a single repo — default.yml lives directly in this folder."
                : `Detected ${scan?.repos.length ?? 0} repo${(scan?.repos.length ?? 0) === 1 ? "" : "s"}.`
              : "Pick a folder. Common layouts: ~/repositories/gitops (many repos) or ~/repositories/fleet-gitops (single repo)."
          }
        />

        <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
          <button
            onClick={rescan}
            disabled={!settings.gitops_dir || busy}
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            ↺ Rescan
          </button>
          {settings.gitops_dir && (
            <button
              onClick={() =>
                api.openPath(settings.gitops_dir!).catch(console.error)
              }
              style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
            >
              Reveal in Finder
            </button>
          )}
        </div>

        {scanError && (
          <div
            style={{
              color: "var(--ui-error)",
              fontSize: "var(--fs-xx-small)",
            }}
          >
            {scanError}
          </div>
        )}

        {scan && (
          <div
            style={{
              border: "1px solid var(--app-border)",
              borderRadius: "var(--radius-md)",
              background: "var(--app-surface-2)",
              padding: "8px 10px",
              fontSize: "var(--fs-xx-small)",
              display: "flex",
              flexDirection: "column",
              gap: 6,
            }}
          >
            {scan.single_repo_mode ? (
              <span style={{ color: "var(--core-fleet-purple)" }}>
                single-repo mode · default.yml in root
              </span>
            ) : scan.repos.length === 0 ? (
              <span className="dim">
                No repos detected. Add a folder with a{" "}
                <span className="mono">default.yml</span> inside.
              </span>
            ) : (
              <>
                <span className="dim" style={{ textTransform: "uppercase", letterSpacing: "0.05em", fontSize: "var(--fs-xxx-small)" }}>
                  detected repos
                </span>
                <div style={{ display: "flex", flexDirection: "column", gap: 3 }}>
                  {scan.repos.map((r) => (
                    <div
                      key={r.name}
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        gap: 8,
                      }}
                    >
                      <span className="mono">{r.name}</span>
                      <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
                        {r.team_files.length} team{r.team_files.length === 1 ? "" : "s"}
                      </span>
                    </div>
                  ))}
                </div>
                {scan.ignored.length > 0 && (
                  <details style={{ fontSize: "var(--fs-xxx-small)", color: "var(--app-text-dim)" }}>
                    <summary style={{ cursor: "pointer" }}>
                      {scan.ignored.length} folder(s) ignored (no default.yml)
                    </summary>
                    <div style={{ paddingTop: 4 }}>
                      {scan.ignored.map((n) => (
                        <div key={n} className="mono">
                          · {n}
                        </div>
                      ))}
                    </div>
                  </details>
                )}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

/* ----- Fleet server section ----- */

function FleetServerSection({
  settings,
  onChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
}) {
  // Edits the ACTIVE server's serve config (switch servers from the top bar).
  const server = activeServer(settings);
  const cfg = server.fleet_serve;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function save(updates: Partial<typeof cfg>) {
    setBusy(true);
    setError(null);
    try {
      const next = updateActiveServer(settings, (s) => ({
        ...s,
        fleet_serve: { ...s.fleet_serve, ...updates },
      }));
      await api.saveSettings(next);
      onChange(next);
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  async function pickConfig() {
    const result = await api.pickFileWithFilter("YAML", "*.yml;*.yaml");
    if (!result || typeof result !== "string") return;
    await save({ config_path: result });
  }

  async function clearConfig() {
    await save({ config_path: null });
  }

  // Env-var rows. Tracked as local state so a draft row with an empty
  // key isn't persisted until the user types a key. Persisting onChange
  // saves on every keystroke after the key exists, matching how python
  // port saves.
  const envRows = cfg.env;

  async function updateEnvRow(
    index: number,
    updates: Partial<EnvVar>,
  ) {
    const next = envRows.map((row, i) =>
      i === index ? { ...row, ...updates } : row,
    );
    await save({ env: next });
  }

  async function addEnvRow() {
    await save({ env: [...envRows, { key: "", value: "", enabled: true }] });
  }

  async function removeEnvRow(index: number) {
    await save({ env: envRows.filter((_, i) => i !== index) });
  }

  return (
    <div
      style={{
        maxWidth: 720,
        display: "flex",
        flexDirection: "column",
        gap: "var(--pad-medium)",
      }}
    >
      <div>
        <PageHeading>
          Fleet server ·{" "}
          <span style={{ color: serverColorVar(server.color) }}>{server.name}</span>
        </PageHeading>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          Controls what flags and env vars{" "}
          <span className="mono">./build/fleet serve --dev</span> spawns with
          for the <strong>active</strong> server ({server.name}). Switch servers
          from the top bar. Premium/free lives on the Server tab next to the
          chain row so you can flip it without leaving the run flow.
        </div>
      </div>

      <div
        className="card"
        style={{ display: "flex", flexDirection: "column", gap: 16 }}
      >
        <PathField
          label="Config file (--config)"
          value={cfg.config_path}
          placeholder="(none — flag omitted, fleet falls back to env / built-in defaults)"
          onPick={pickConfig}
          onClear={cfg.config_path ? clearConfig : undefined}
          busy={busy}
          hint={
            cfg.config_path
              ? "Spawn passes --config with this path. Relative paths resolve against the repo."
              : "Clear → the --config flag is dropped entirely. Useful if you drive fleet via env vars or want to test the defaults."
          }
        />

        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 6,
          }}
        >
          <CheckboxRow
            label="--debug"
            description="Verbose request/response logging."
            checked={cfg.debug}
            onChange={(v) => save({ debug: v })}
          />
          <CheckboxRow
            label="--logging_debug"
            description="Debug-level logging from the logger subsystem (very chatty)."
            checked={cfg.logging_debug}
            onChange={(v) => save({ logging_debug: v })}
          />
        </div>

        {error && (
          <div style={{ color: "var(--ui-error)", fontSize: "var(--fs-xx-small)" }}>
            {error}
          </div>
        )}
      </div>

      <div
        className="card"
        style={{ display: "flex", flexDirection: "column", gap: 10 }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <div>
            <div className="card-title">Environment variables</div>
            <div
              className="dim"
              style={{ fontSize: "var(--fs-xxx-small)", marginTop: 2 }}
            >
              Applied on top of the inherited environment. Uncheck a row
              to keep it around but skip it on the next spawn. Empty
              value means <span className="mono">KEY=</span> (set but
              blank); empty-key rows are ignored.
            </div>
          </div>
          <button
            onClick={addEnvRow}
            disabled={busy}
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            + Add
          </button>
        </div>

        {envRows.length === 0 ? (
          <div
            className="dim"
            style={{
              fontSize: "var(--fs-xxx-small)",
              textAlign: "center",
              padding: 14,
              border: "1px dashed var(--app-border)",
              borderRadius: "var(--radius-md)",
            }}
          >
            No env vars configured.
          </div>
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
            {envRows.map((row, i) => (
              <EnvRow
                key={i}
                row={row}
                onChange={(updates) => updateEnvRow(i, updates)}
                onRemove={() => removeEnvRow(i)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function CheckboxRow({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label
      style={{
        display: "flex",
        alignItems: "flex-start",
        gap: 10,
        padding: "8px 10px",
        background: "var(--app-surface-2)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        cursor: "pointer",
      }}
    >
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        style={{ accentColor: "var(--core-fleet-green)", marginTop: 2 }}
      />
      <div style={{ minWidth: 0 }}>
        <div className="mono" style={{ color: "var(--app-text)" }}>
          {label}
        </div>
        <div
          className="dim"
          style={{ fontSize: "var(--fs-xxx-small)", marginTop: 2 }}
        >
          {description}
        </div>
      </div>
    </label>
  );
}

function EnvRow({
  row,
  onChange,
  onRemove,
}: {
  row: EnvVar;
  onChange: (updates: Partial<EnvVar>) => void;
  onRemove: () => void;
}) {
  const disabled = !row.enabled;
  return (
    <div
      style={{
        display: "grid",
        gridTemplateColumns:
          "auto minmax(140px, 1fr) minmax(180px, 2fr) auto",
        gap: 8,
        alignItems: "center",
      }}
    >
      <input
        type="checkbox"
        checked={row.enabled}
        onChange={(e) => onChange({ enabled: e.target.checked })}
        title={row.enabled ? "Applied on next spawn" : "Skipped"}
        style={{ accentColor: "var(--core-fleet-green)" }}
      />
      <input
        type="text"
        value={row.key}
        onChange={(e) => onChange({ key: e.target.value })}
        placeholder="KEY"
        {...noAutocorrect}
        className="mono"
        style={{
          width: "100%",
          opacity: disabled ? 0.45 : 1,
          textDecoration: disabled ? "line-through" : "none",
        }}
      />
      <input
        type="text"
        value={row.value}
        onChange={(e) => onChange({ value: e.target.value })}
        placeholder="value"
        {...noAutocorrect}
        className="mono"
        style={{ width: "100%", opacity: disabled ? 0.45 : 1 }}
      />
      <button
        onClick={onRemove}
        title="Remove"
        style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
      >
        ✕
      </button>
    </div>
  );
}

/* ----- fleetctl contexts section ----- */

function FleetctlContextsSection({
  settings: _settings,
}: {
  settings: Settings;
}) {
  const [info, setInfo] = useState<ContextInfo | null>(null);
  const [configPath, setConfigPath] = useState<string>("~/.fleet/config");
  const [configExists, setConfigExists] = useState<boolean>(false);
  const [diskContents, setDiskContents] = useState<string>("");
  const [draft, setDraft] = useState<string>("");
  const [selectedContext, setSelectedContext] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // justSaved tracks whether the most recent on-disk write came from
  // us. Used to swap the status pill from "unsaved" → "saved" without
  // a separate success banner. Edits naturally hide it because the
  // "unsaved" branch takes priority when dirty.
  const [justSaved, setJustSaved] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  const refresh = useCallback(async () => {
    setError(null);
    try {
      const [ctx, raw] = await Promise.all([
        api.fleetctlReadContext(),
        api.fleetctlReadConfigRaw(),
      ]);
      setInfo(ctx);
      setConfigPath(raw.path);
      setConfigExists(raw.exists);
      setDiskContents(raw.contents);
      setDraft(raw.contents);
    } catch (e) {
      setError(String(e));
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const dirty = draft !== diskContents;
  const contexts = info?.contexts ?? [];

  // Discard unsaved changes if the user navigates away. Without this the
  // next visit would still show the stale draft and the dirty pill.
  useEffect(() => {
    return () => {
      // Effect cleanup runs on unmount — nothing to do, draft is local
      // state and will be re-fetched on next mount. Keep as a hook anchor
      // in case we add an unsaved-changes confirm later.
    };
  }, []);

  function selectRow(name: string) {
    setSelectedContext(name);
    const ta = textareaRef.current;
    if (!ta) return;
    // Find the line starting the matching block. We accept the common
    // shapes: `- name: foo` (sequence) and `foo:` (mapping under
    // `contexts:`). The latter handles fleetctl's actual file layout;
    // the former handles the wf-style YAML in the design.
    const lines = draft.split("\n");
    const seqRe = new RegExp(
      `^\\s*-\\s*name:\\s*${escapeRegex(name)}\\s*$`,
    );
    const mapRe = new RegExp(`^\\s*${escapeRegex(name)}:\\s*$`);
    let startLine = -1;
    for (let i = 0; i < lines.length; i++) {
      if (seqRe.test(lines[i]) || mapRe.test(lines[i])) {
        startLine = i;
        break;
      }
    }
    if (startLine < 0) return;
    let charIndex = 0;
    for (let i = 0; i < startLine; i++) charIndex += lines[i].length + 1;
    const lineLen = lines[startLine].length;
    // Focusing + setSelectionRange triggers the browser's auto-scroll
    // for the caret — the textarea will scroll the selection into view.
    ta.focus();
    ta.setSelectionRange(charIndex, charIndex + lineLen);
  }

  async function save() {
    setBusy(true);
    setError(null);
    try {
      await api.fleetctlSaveConfig(draft);
      await refresh();
      setJustSaved(true);
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  function discard() {
    setDraft(diskContents);
    setError(null);
  }

  function insertStarterTemplate() {
    if (draft.trim().length > 0) return;
    setDraft(STARTER_YAML);
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <PageHeading>fleetctl contexts</PageHeading>
      <div
        className="dim"
        style={{ fontSize: "var(--fs-xx-small)", lineHeight: 1.5 }}
      >
        Contexts live in{" "}
        <span className="mono">{configPath}</span>. Edit the file directly
        — saving validates the YAML and writes to disk. The list on the
        left mirrors what's currently parsed.
      </div>

      <div
        style={{
          display: "grid",
          gridTemplateColumns: "260px 1fr",
          gap: 14,
          minHeight: 360,
        }}
      >
        {/* ---- left: parsed list ---- */}
        <div
          className="card"
          style={{
            padding: 10,
            display: "flex",
            flexDirection: "column",
            gap: 6,
            minHeight: 0,
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "baseline",
              padding: "2px 4px",
            }}
          >
            <div className="card-title">Contexts</div>
            <div
              className="dim"
              style={{ fontSize: "var(--fs-xxx-small)" }}
            >
              {contexts.length === 0
                ? configExists
                  ? "none parsed"
                  : "no file yet"
                : `${contexts.length}`}
            </div>
          </div>

          {contexts.length === 0 ? (
            <div
              className="dim"
              style={{
                fontSize: "var(--fs-xxx-small)",
                textAlign: "center",
                padding: 16,
                border: "1px dashed var(--app-border)",
                borderRadius: "var(--radius-md)",
              }}
            >
              {configExists
                ? "Config exists but contexts couldn't be parsed. Edit the YAML on the right and save."
                : "No contexts yet."}
            </div>
          ) : (
            <div
              style={{
                display: "flex",
                flexDirection: "column",
                gap: 4,
                overflow: "auto",
              }}
            >
              {contexts.map((c) => (
                <ParsedRow
                  key={c.name}
                  ctx={c}
                  selected={selectedContext === c.name}
                  onClick={() => selectRow(c.name)}
                />
              ))}
            </div>
          )}
        </div>

        {/* ---- right: YAML editor ---- */}
        <div
          className="card"
          style={{
            padding: 10,
            display: "flex",
            flexDirection: "column",
            gap: 6,
            minHeight: 0,
          }}
        >
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              padding: "2px 4px",
              gap: 8,
            }}
          >
            <div
              className="card-title"
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                minWidth: 0,
              }}
            >
              <span
                style={{
                  whiteSpace: "nowrap",
                  overflow: "hidden",
                  textOverflow: "ellipsis",
                }}
              >
                {configPath}
              </span>
              {dirty ? (
                <StatusPill kind="unsaved">unsaved</StatusPill>
              ) : justSaved ? (
                <StatusPill kind="saved">saved</StatusPill>
              ) : !configExists ? (
                <span
                  className="dim"
                  style={{ fontSize: "var(--fs-xxx-small)" }}
                >
                  (file will be created on save)
                </span>
              ) : null}
            </div>
            <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
              <button
                onClick={refresh}
                disabled={busy}
                title="re-read from disk (will lose unsaved changes)"
                style={{
                  padding: "4px 10px",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                ↻ Reload
              </button>
              <button
                onClick={discard}
                disabled={busy || !dirty}
                style={{
                  padding: "4px 12px",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                Discard
              </button>
              <button
                className="primary"
                onClick={save}
                disabled={busy || !dirty}
                style={{
                  padding: "4px 14px",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                {busy ? "saving…" : "Save"}
              </button>
            </div>
          </div>

          <textarea
            ref={textareaRef}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            {...noAutocorrect}
            placeholder={
              configExists
                ? "Config file is empty — paste or generate a starter template below."
                : "Config file doesn't exist yet — paste or generate a starter template below."
            }
            style={{
              flex: 1,
              minHeight: 320,
              boxSizing: "border-box",
              background: "var(--app-surface-2)",
              color: "var(--app-text)",
              border: "1px solid var(--app-border)",
              borderRadius: "var(--radius-md)",
              padding: "10px 12px",
              fontFamily: "var(--font-mono)",
              fontSize: "var(--fs-xx-small)",
              lineHeight: 1.55,
              resize: "vertical",
              whiteSpace: "pre",
              tabSize: 2,
            }}
          />

          {draft.trim().length === 0 && (
            <div style={{ display: "flex", justifyContent: "flex-start" }}>
              <button
                onClick={insertStarterTemplate}
                style={{
                  padding: "4px 10px",
                  fontSize: "var(--fs-xxx-small)",
                }}
              >
                insert starter template
              </button>
            </div>
          )}

          {error && (
            <div
              style={{
                color: "var(--ui-error)",
                fontSize: "var(--fs-xx-small)",
                padding: "4px 4px",
              }}
            >
              {error}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ParsedRow({
  ctx,
  selected,
  onClick,
}: {
  ctx: ContextSummary;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClick();
        }
      }}
      style={{
        padding: "8px 10px",
        borderRadius: "var(--radius-md)",
        cursor: "pointer",
        background: selected
          ? "var(--tint-success-soft)"
          : "var(--app-surface-2)",
        border: `1px solid ${selected ? "var(--core-fleet-green)" : "transparent"}`,
        display: "flex",
        flexDirection: "column",
        gap: 2,
        minWidth: 0,
      }}
    >
      {/* Top row: name on the left, token status on the right. Two
          slots so the token indicator never gets eaten by a long
          address ellipsis. */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          gap: 8,
          minWidth: 0,
        }}
      >
        <span
          className="mono"
          style={{
            fontSize: "var(--fs-x-small)",
            color: "var(--app-text)",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
            minWidth: 0,
          }}
          title={ctx.name}
        >
          {ctx.name}
        </span>
        <span
          style={{
            fontSize: "var(--fs-xxx-small)",
            color: ctx.has_token
              ? "var(--core-fleet-green)"
              : "var(--ui-error)",
            flexShrink: 0,
            textTransform: "uppercase",
            letterSpacing: "0.06em",
          }}
        >
          {ctx.has_token ? "token" : "no token"}
        </span>
      </div>
      <div
        className="dim"
        style={{
          fontSize: "var(--fs-xxx-small)",
          whiteSpace: "nowrap",
          overflow: "hidden",
          textOverflow: "ellipsis",
        }}
        title={ctx.address ?? ""}
      >
        {ctx.address ?? "no address"}
      </div>
    </div>
  );
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function StatusPill({
  kind,
  children,
}: {
  kind: "unsaved" | "saved";
  children: React.ReactNode;
}) {
  // Both pills use fleet-green to feel like one slot that changes
  // label. "unsaved" gets a filled tint to draw the eye toward the
  // pending save action; "saved" is an outline so it reads as a
  // passive ack and doesn't compete with the buttons.
  const filled = kind === "unsaved";
  return (
    <span
      style={{
        fontSize: "var(--fs-xxx-small)",
        color: "var(--core-fleet-green)",
        background: filled ? "var(--tint-success-soft)" : "transparent",
        border: `1px solid ${filled ? "transparent" : "var(--core-fleet-green)"}`,
        padding: "1px 6px",
        borderRadius: 999,
        whiteSpace: "nowrap",
      }}
    >
      {children}
    </span>
  );
}

// Starter shown when the user clicks "insert starter template" on an
// empty config. Mirrors fleetctl's own writeConfig output: contexts is
// a mapping keyed by name. The default context is created with just an
// address; the user fills in email/token via fleetctl login or by
// editing here.
const STARTER_YAML = `contexts:
  default:
    address: https://localhost:8080
`;

/* ----- ngrok section ----- */

function NgrokSection({
  settings,
  onChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
}) {
  const [info, setInfo] = useState<NgrokYamlInfo | null>(null);
  const [busy, setBusy] = useState(false);
  const [yaml, setYaml] = useState<string>("");
  const [yamlDirty, setYamlDirty] = useState(false);
  const [yamlError, setYamlError] = useState<string | null>(null);
  const [savedToast, setSavedToast] = useState(false);
  const cfg = settings.ngrok;

  async function reparse() {
    setBusy(true);
    try {
      const result = await api.parseNgrokYml(cfg.yml_path);
      setInfo(result);
      // Self-heal: drop any selected tunnel that no longer exists in the yml
      // (renamed/removed) so the start command can't reference a phantom.
      const stale = staleNgrokTunnels(settings, result);
      if (stale.length > 0) {
        updateDefaults({
          default_tunnels: cfg.default_tunnels.filter(
            (n) => !stale.includes(n),
          ),
        });
      }
      if (result.valid && result.resolved_path) {
        try {
          const txt = await api.readTextFile(result.resolved_path);
          setYaml(txt);
          setYamlDirty(false);
          setYamlError(null);
        } catch (e) {
          // Don't blow away an in-progress edit on a transient read
          // failure — surface the error and keep whatever the user has
          // typed so they can recover.
          setYamlError(
            `Could not read ${result.resolved_path}: ${String(e)}`,
          );
        }
      } else {
        // Only clear when we have no resolved file to read at all
        // (e.g. user just set yml_path to an empty path). In that case
        // there is no edit to preserve.
        if (!yamlDirty) setYaml("");
      }
    } catch (e) {
      console.error(e);
    }
    setBusy(false);
  }

  useEffect(() => {
    reparse();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cfg.yml_path]);

  async function saveYaml() {
    if (!info?.resolved_path) return;
    setYamlError(null);
    try {
      await api.writeTextFile(info.resolved_path, yaml);
      setYamlDirty(false);
      setSavedToast(true);
      window.setTimeout(() => setSavedToast(false), 2000);
      reparse();
    } catch (e) {
      setYamlError(String(e));
    }
  }

  async function openInEditor() {
    if (!info?.resolved_path) return;
    try {
      await api.openPath(info.resolved_path);
    } catch (e) {
      console.error(e);
    }
  }
  async function revealInFinder() {
    if (!info?.resolved_path) return;
    try {
      await api.openPath(info.resolved_path, true);
    } catch (e) {
      console.error(e);
    }
  }

  async function pickPath() {
    const result = await api.pickFileWithFilter("YAML", "*.yml;*.yaml");
    if (!result || typeof result !== "string") return;
    const next = { ...settings, ngrok: { ...cfg, yml_path: result } };
    await api.saveSettings(next);
    onChange(next);
  }

  async function updateDefaults(updates: Partial<typeof cfg>) {
    const next = { ...settings, ngrok: { ...cfg, ...updates } };
    await api.saveSettings(next);
    onChange(next);
  }


  return (
    <div style={{ maxWidth: 720, display: "flex", flexDirection: "column", gap: "var(--pad-medium)" }}>
      <div>
        <PageHeading>ngrok</PageHeading>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          We read tunnels from your ngrok.yml. We never write to it — edit it
          yourself to add/change tunnels, then reparse.
        </div>
      </div>

      <EnableToggle
        label="Enable ngrok"
        description="Show ngrok in the Active processes panel on the Server tab."
        checked={cfg.enabled}
        onChange={(v) => {
          // Enabling with nothing selected leaves ngrok unlaunchable and
          // it's easy to miss — pre-select the first tunnel from the
          // parsed ngrok.yml so it's ready to start.
          if (
            v &&
            !cfg.start_all &&
            cfg.default_tunnels.length === 0 &&
            info?.tunnels.length
          ) {
            updateDefaults({
              enabled: true,
              default_tunnels: [info.tunnels[0].name],
            });
          } else {
            updateDefaults({ enabled: v });
          }
        }}
      />

      <div
        className="card"
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 16,
          opacity: cfg.enabled ? 1 : 0.45,
          pointerEvents: cfg.enabled ? "auto" : "none",
        }}
      >
        <PathField
          label="ngrok.yml path"
          value={info?.resolved_path ?? cfg.yml_path}
          placeholder="~/Library/Application Support/ngrok/ngrok.yml"
          onPick={pickPath}
          busy={busy}
          hint={cfg.yml_path ? undefined : "default location — change to use a custom file"}
        />
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <ValidationLine info={info} busy={busy} />
          <button onClick={reparse} disabled={busy} style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}>
            ↻ Reparse
          </button>
        </div>
      </div>

      <div
        className="card"
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 12,
          opacity: cfg.enabled ? 1 : 0.45,
          pointerEvents: cfg.enabled ? "auto" : "none",
        }}
      >
        <label
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
          }}
        >
          <input
            type="checkbox"
            checked={cfg.start_all}
            onChange={(e) => updateDefaults({ start_all: e.target.checked })}
            style={{ accentColor: "var(--core-fleet-green)" }}
          />
          <span style={{ fontSize: "var(--fs-xx-small)" }}>
            Start all tunnels (uses{" "}
            <span className="mono">ngrok start --all</span>)
          </span>
        </label>
        <div className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
          When unchecked, pick tunnels by clicking their chips in the Active
          processes panel on the Server tab.
        </div>
      </div>

      <div
        className="card"
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 10,
          opacity: cfg.enabled ? 1 : 0.45,
          pointerEvents: cfg.enabled ? "auto" : "none",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
          }}
        >
          <div className="card-title">ngrok.yml contents</div>
          <div style={{ display: "flex", gap: 6 }}>
            <button
              onClick={openInEditor}
              disabled={!info?.resolved_path}
              style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
              title="Open with the system default editor"
            >
              Open in editor
            </button>
            <button
              onClick={revealInFinder}
              disabled={!info?.resolved_path}
              style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
              title="Reveal in Finder"
            >
              Reveal
            </button>
          </div>
        </div>
        <textarea
          value={yaml}
          onChange={(e) => {
            setYaml(e.target.value);
            setYamlDirty(true);
          }}
          {...noAutocorrect}
          className="mono"
          placeholder="(empty — pick a valid ngrok.yml path above)"
          style={{
            width: "100%",
            minHeight: 220,
            background: "var(--log-bg)",
            color: "var(--app-text)",
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
            padding: "8px 10px",
            fontSize: "var(--fs-xx-small)",
            lineHeight: 1.5,
            resize: "vertical",
            whiteSpace: "pre",
            overflow: "auto",
          }}
        />
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            fontSize: "var(--fs-xx-small)",
          }}
        >
          <div className="dim">
            Edits write to{" "}
            <span className="mono">
              {info?.resolved_path ?? "—"}
            </span>{" "}
            and re-parse.
          </div>
          <div style={{ display: "flex", gap: 6, alignItems: "center" }}>
            {savedToast && (
              <span style={{ color: "var(--core-fleet-green)" }}>
                ✓ saved
              </span>
            )}
            {yamlError && (
              <span style={{ color: "var(--ui-error)" }}>{yamlError}</span>
            )}
            <button
              onClick={() => reparse()}
              disabled={!yamlDirty || busy}
              style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
            >
              Discard
            </button>
            <button
              className="primary"
              onClick={saveYaml}
              disabled={!yamlDirty || !info?.resolved_path}
              style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
            >
              Save
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function ValidationLine({
  info,
  busy,
}: {
  info: NgrokYamlInfo | null;
  busy: boolean;
}) {
  if (busy) return <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>parsing…</span>;
  if (!info)
    return <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>—</span>;
  if (!info.valid)
    return (
      <span style={{ color: "var(--ui-error)", fontSize: "var(--fs-xx-small)" }}>
        ✗ {info.error}
      </span>
    );
  if (!info.has_authtoken)
    return (
      <span style={{ color: "var(--ui-warning)", fontSize: "var(--fs-xx-small)" }}>
        ⚠ valid · {info.tunnels.length} tunnels · no authtoken in file
      </span>
    );
  return (
    <span style={{ color: "var(--core-fleet-green)", fontSize: "var(--fs-xx-small)" }}>
      ✓ valid · {info.tunnels.length} tunnels · authtoken present
    </span>
  );
}

/* ----- Python section ----- */

function PythonSection({
  settings,
  onChange,
}: {
  settings: Settings;
  onChange: (next: Settings) => void;
}) {
  const cfg = settings.python_server;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // Local mirror of the port input so each keystroke doesn't trigger a
  // saveSettings IPC + disk write. Persists on blur (commitPort) or
  // 600ms after the user stops typing.
  const [portDraft, setPortDraft] = useState<string>(String(cfg.port));
  useEffect(() => {
    setPortDraft(String(cfg.port));
  }, [cfg.port]);

  async function save(updates: Partial<typeof cfg>) {
    setBusy(true);
    try {
      const next: Settings = {
        ...settings,
        python_server: { ...cfg, ...updates },
      };
      await api.saveSettings(next);
      onChange(next);
    } catch (e) {
      setError(String(e));
    }
    setBusy(false);
  }

  function commitPort() {
    const n = Number(portDraft);
    if (!Number.isFinite(n) || n < 1 || n > 65535) {
      // Invalid input — revert to last-saved value.
      setPortDraft(String(cfg.port));
      return;
    }
    if (n !== cfg.port) save({ port: n });
  }

  // Debounce port commits so typing "8080" doesn't fire four saves.
  useEffect(() => {
    const n = Number(portDraft);
    if (!Number.isFinite(n) || n < 1 || n > 65535 || n === cfg.port) {
      return;
    }
    const t = window.setTimeout(() => {
      save({ port: n });
    }, 600);
    return () => window.clearTimeout(t);
    // save / cfg.port are intentionally not in deps: this effect drives
    // off the user's typing rhythm only.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [portDraft]);

  async function pickDir() {
    const result = await api.pickFolder();
    if (!result || typeof result !== "string") return;
    save({ directory: result });
  }

  return (
    <div style={{ maxWidth: 720, display: "flex", flexDirection: "column", gap: "var(--pad-medium)" }}>
      <div>
        <PageHeading>python http.server</PageHeading>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          Quick way to share a directory over HTTP. Uses{" "}
          <span className="mono">python3 -m http.server</span>.
        </div>
      </div>

      <EnableToggle
        label="Enable python http.server"
        description="Show python http.server in the Active processes panel on the Server tab."
        checked={cfg.enabled}
        onChange={(v) => save({ enabled: v })}
      />

      <div
        className="card"
        style={{
          display: "flex",
          flexDirection: "column",
          gap: 16,
          opacity: cfg.enabled ? 1 : 0.45,
          pointerEvents: cfg.enabled ? "auto" : "none",
        }}
      >
        <PathField
          label="Default directory"
          value={cfg.directory}
          placeholder="repo root"
          onPick={pickDir}
          busy={busy}
          hint="Relative paths resolve against the fleet repo. Empty = repo root."
        />
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)" }}>
            Default port
          </div>
          <input
            type="number"
            value={portDraft}
            min={1}
            max={65535}
            onChange={(e) => setPortDraft(e.target.value)}
            onBlur={commitPort}
            className="mono"
            style={{ width: 120 }}
          />
        </div>
        {error && (
          <div style={{ color: "var(--ui-error)", fontSize: "var(--fs-xx-small)" }}>
            {error}
          </div>
        )}
      </div>
    </div>
  );
}

/* ----- Troubleshoot section ----- */

type ScanMode =
  | { kind: "port"; port: number }
  | { kind: "pattern"; pattern: string };

interface TroubleshootCard {
  id: string;
  title: string;
  subtitle: string;
  mode: ScanMode;
}

function TroubleshootSection({ settings }: { settings: Settings }) {
  const pythonPort = settings.python_server.port;
  // Memoizing the cards array (and therefore each card.mode reference)
  // is what keeps the per-card useCallback/useEffect stable across
  // parent re-renders. Without this, App.tsx's frequent re-renders
  // (proc poll, health probe) would cascade into a fresh `cards`
  // array, fresh `mode` objects, fresh `scan` callbacks, and the
  // auto-scan effect would re-fire on every tick.
  const cards = useMemo<TroubleshootCard[]>(
    () => [
      {
        id: "ngrok",
        title: "ngrok",
        subtitle: "any ngrok process (matches command line)",
        mode: { kind: "pattern", pattern: "^ngrok " },
      },
      {
        // Port-based for python because pattern matching is unreliable:
        // `pgrep -f http.server` misses macOS framework Python (the bin
        // is "Python", not "python3"), and bare `python` matches too
        // much. `lsof :<port>` is the authoritative "who's bound here".
        id: "python",
        title: `python http.server (port ${pythonPort})`,
        subtitle: `whoever is listening on port ${pythonPort}`,
        mode: { kind: "port", port: pythonPort },
      },
      {
        // We run perf as `go run ./agent.go --server_url … --os_templates …`
        // from cmd/osquery-perf. That spawns TWO processes — the `go run`
        // wrapper and the compiled binary in ~/Library/Caches/go-build/…/agent
        // — and NEITHER command line contains the string "osquery-perf"
        // (the cwd does, but pgrep -f matches the command line, not cwd).
        // The `--os_templates` flag is the distinctive token present in
        // both, and every run passes it (Start is gated on ≥1 OS), so it
        // reliably catches the wrapper and the worker without matching
        // the dozens of macOS "*Agent" system processes that a bare
        // "agent" pattern would.
        id: "osquery-perf",
        title: "osquery-perf",
        subtitle: "perf agents (matches --os_templates on the command line)",
        mode: { kind: "pattern", pattern: "os_templates" },
      },
      {
        // Catches both Hangar's cached <app-data>/bin/scepserver and any
        // external scepserver-<os>-<arch> binary run by hand.
        id: "scep",
        title: "SCEP servers",
        subtitle: "any scepserver process (matches the command line)",
        mode: { kind: "pattern", pattern: "scepserver" },
      },
      {
        // The local TUF file-server the tools/tuf/test scripts leave running.
        id: "tuf",
        title: "TUF server (port 8081)",
        subtitle: "whoever is listening on the TUF port",
        mode: { kind: "port", port: 8081 },
      },
    ],
    [pythonPort],
  );

  return (
    <div style={{ maxWidth: 820 }}>
      <PageHeading>Troubleshoot</PageHeading>
      <div
        className="dim"
        style={{
          fontSize: "var(--fs-xx-small)",
          lineHeight: 1.5,
          marginBottom: "var(--pad-medium)",
        }}
      >
        Hunt for orphan processes the app's tracking lost touch with —
        usually the result of a crash, an HMR reload during dev, or an
        external invocation. Scanning queries the OS directly
        (<span className="mono">pgrep</span> /{" "}
        <span className="mono">lsof</span>), so what shows up here is
        what's actually there. fleet serve and docker aren't included;
        manage those via the Server tab or Docker Desktop.
      </div>

      <div
        style={{
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-medium)",
        }}
      >
        {cards.map((c) => (
          <TroubleshootCardView key={c.id} card={c} />
        ))}
        <TufAssetsCard />
      </div>
    </div>
  );
}

// TufAssetsCard removes the generated local TUF repo (<repo>/test_tuf) — the
// asset cleanup that pairs with killing the TUF server above.
function TufAssetsCard() {
  const [phase, setPhase] = useState<"idle" | "armed" | "deleting">("idle");
  const [msg, setMsg] = useState<string | null>(null);

  const del = async () => {
    setPhase("deleting");
    setMsg(null);
    try {
      await api.tufDeleteAssets();
      setMsg("Deleted test_tuf");
    } catch (e) {
      setMsg(e instanceof Error ? e.message : String(e));
    } finally {
      setPhase("idle");
    }
  };

  return (
    <div className="card" style={{ padding: "var(--pad-medium)", display: "flex", alignItems: "center", gap: 8 }}>
      <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
        <span style={{ fontWeight: 600, fontSize: "var(--fs-x-small)" }}>TUF assets</span>
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          delete the generated repo (<span className="mono">test_tuf</span>) in the Fleet repo root
        </span>
        {msg && (
          <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>{msg}</span>
        )}
      </div>
      <div style={{ flex: 1 }} />
      {phase === "armed" ? (
        <>
          <button onClick={del} className="danger" style={{ padding: "5px 12px" }}>
            Confirm delete
          </button>
          <button onClick={() => setPhase("idle")} style={{ padding: "5px 12px" }}>
            Cancel
          </button>
        </>
      ) : (
        <button onClick={() => setPhase("armed")} disabled={phase === "deleting"} style={{ padding: "5px 12px" }}>
          {phase === "deleting" ? "Deleting…" : "Delete assets"}
        </button>
      )}
    </div>
  );
}

function TroubleshootCardView({ card }: { card: TroubleshootCard }) {
  const [found, setFound] = useState<DetectedProcess[] | null>(null);
  const [scanning, setScanning] = useState(false);
  const [busyPid, setBusyPid] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);

  const count = found?.length ?? 0;

  const scan = useCallback(async () => {
    setScanning(true);
    setError(null);
    setNote(null);
    try {
      if (card.mode.kind === "port") {
        const r = await api.troubleshootScanPort(card.mode.port);
        setFound(r);
      } else {
        const r = await api.troubleshootScanPattern(card.mode.pattern);
        setFound(r);
      }
    } catch (e) {
      setError(String(e));
    }
    setScanning(false);
  }, [card.mode]);

  // Auto-scan on first mount. Cheap commands; the user opening the
  // section almost certainly wants to see state right away.
  useEffect(() => {
    scan();
  }, [scan]);

  async function killOne(pid: number) {
    setBusyPid(pid);
    setError(null);
    setNote(null);
    try {
      const r = await api.troubleshootKillPid(pid);
      if (r.gone) {
        setNote(
          r.used_kill
            ? `pid ${pid} stopped (needed SIGKILL)`
            : `pid ${pid} stopped`,
        );
      } else {
        setError(
          r.error ? `pid ${pid}: ${r.error}` : `pid ${pid} still alive`,
        );
      }
      await scan();
    } catch (e) {
      setError(String(e));
    }
    setBusyPid(null);
  }

  async function killAll() {
    if (!found || found.length === 0) return;
    setError(null);
    setNote(null);
    let stopped = 0;
    let needed_kill = 0;
    let failed = 0;
    for (const p of found) {
      setBusyPid(p.pid);
      try {
        const r = await api.troubleshootKillPid(p.pid);
        if (r.gone) {
          stopped++;
          if (r.used_kill) needed_kill++;
        } else {
          failed++;
        }
      } catch {
        failed++;
      }
    }
    setBusyPid(null);
    setNote(
      `${stopped} stopped${needed_kill ? ` (${needed_kill} via SIGKILL)` : ""}${
        failed ? ` · ${failed} failed` : ""
      }`,
    );
    await scan();
  }

  return (
    <div className="card" style={{ padding: 14 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: 8,
          marginBottom: 10,
        }}
      >
        <div style={{ minWidth: 0 }}>
          <div
            className="card-title"
            style={{
              display: "flex",
              alignItems: "center",
              gap: 8,
            }}
          >
            {card.title}
            <CountBadge count={count} scanning={scanning} />
          </div>
          <div
            className="dim"
            style={{ fontSize: "var(--fs-xxx-small)", marginTop: 2 }}
          >
            {card.subtitle}
          </div>
        </div>
        <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
          {(found?.length ?? 0) > 1 && (
            <button
              className="danger"
              disabled={busyPid != null || scanning}
              onClick={killAll}
              style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
            >
              Kill all
            </button>
          )}
          <button
            onClick={scan}
            disabled={scanning || busyPid != null}
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            {scanning ? "scanning…" : "↻ Scan"}
          </button>
        </div>
      </div>

      <ProcessFindings
        found={found}
        scanning={scanning}
        busyPid={busyPid}
        onKill={killOne}
      />

      {note && (
        <div
          style={{
            marginTop: 8,
            fontSize: "var(--fs-xxx-small)",
            color: "var(--core-fleet-green)",
          }}
        >
          {note}
        </div>
      )}
      {error && (
        <div
          style={{
            marginTop: 8,
            fontSize: "var(--fs-xxx-small)",
            color: "var(--ui-error)",
          }}
        >
          {error}
        </div>
      )}
    </div>
  );
}

function CountBadge({
  count,
  scanning,
}: {
  count: number;
  scanning: boolean;
}) {
  if (scanning && count === 0) {
    return (
      <span
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)" }}
      >
        scanning…
      </span>
    );
  }
  const clean = count === 0;
  return (
    <span
      style={{
        fontSize: "var(--fs-xxx-small)",
        color: clean ? "var(--core-fleet-green)" : "var(--ui-error)",
        background: clean
          ? "var(--tint-success-soft)"
          : "var(--tint-error-strong)",
        padding: "1px 8px",
        borderRadius: 999,
        fontWeight: 600,
        letterSpacing: "0.03em",
      }}
    >
      {clean ? "clean" : `${count} found`}
    </span>
  );
}

function ProcessFindings({
  found,
  scanning,
  busyPid,
  onKill,
}: {
  found: DetectedProcess[] | null;
  scanning: boolean;
  busyPid: number | null;
  onKill: (pid: number) => void;
}) {
  if (found == null && scanning) {
    return (
      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)", padding: "4px 0" }}
      >
        scanning…
      </div>
    );
  }
  if (!found || found.length === 0) {
    return (
      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)", padding: "4px 0" }}
      >
        nothing detected
      </div>
    );
  }
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      {found.map((p) => (
        <div
          key={p.pid}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            padding: "6px 10px",
            background: "var(--app-surface-2)",
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
          }}
        >
          <span
            className="mono"
            style={{
              fontSize: "var(--fs-xxx-small)",
              color: "var(--app-text-dim)",
              minWidth: 56,
            }}
          >
            pid {p.pid}
          </span>
          <span
            className="mono"
            style={{
              flex: 1,
              minWidth: 0,
              fontSize: "var(--fs-xxx-small)",
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
            title={p.command}
          >
            {p.command}
          </span>
          <button
            className="danger"
            disabled={busyPid != null}
            onClick={() => onKill(p.pid)}
            style={{
              padding: "2px 10px",
              fontSize: "var(--fs-xxx-small)",
              flexShrink: 0,
            }}
          >
            {busyPid === p.pid ? "…" : "Kill"}
          </button>
        </div>
      ))}
    </div>
  );
}

