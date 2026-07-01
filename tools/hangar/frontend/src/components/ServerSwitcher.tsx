import type { Settings } from "../lib/ipc";
import { serverColorVar } from "../lib/servers";
import type { ServerHealth } from "../lib/useSystemHealth";

/// Top-bar control for switching between local Fleet servers and seeing each
/// one's status at a glance. One pill per configured server: name, accent dot,
/// and serve/docker state. The active pill is highlighted. "Manage" jumps to
/// the Servers settings section (add / remove / configure).
export function ServerSwitcher({
  settings,
  healthMap,
  onSwitch,
  onManage,
}: {
  settings: Settings;
  healthMap: Record<string, ServerHealth>;
  onSwitch: (id: string) => void;
  onManage: () => void;
}) {
  const activeId = settings.active_server_id;
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 8,
        padding: "5px var(--pad-medium)",
        background: "var(--app-surface)",
        borderBottom: "1px solid var(--app-border)",
        overflowX: "auto",
      }}
    >
      <span
        className="dim"
        style={{
          fontSize: "var(--fs-xxx-small)",
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          flexShrink: 0,
        }}
      >
        servers
      </span>
      <div style={{ display: "flex", gap: 6, flex: 1, minWidth: 0 }}>
        {settings.servers.map((s) => {
          const active = s.id === activeId;
          const health = healthMap[s.id];
          const accent = serverColorVar(s.color);
          const serveUp = health?.serve.up ?? false;
          const dockerUp = health?.docker.up ?? false;
          const configured = !!s.worktree_path;
          return (
            <button
              key={s.id}
              onClick={() => onSwitch(s.id)}
              title={
                configured
                  ? `${s.name} · serve ${serveUp ? "up" : "down"} · docker ${dockerUp ? "up" : "down"} · :${s.ports.server}`
                  : `${s.name} · no worktree configured`
              }
              style={{
                display: "flex",
                alignItems: "center",
                gap: 7,
                padding: "4px 10px",
                borderRadius: 999,
                border: `1px solid ${active ? accent : "var(--app-border)"}`,
                background: active ? `color-mix(in srgb, ${accent} 14%, transparent)` : "var(--app-surface-2)",
                color: active ? "var(--app-text)" : "var(--app-text-dim)",
                cursor: "pointer",
                whiteSpace: "nowrap",
                flexShrink: 0,
              }}
            >
              <span
                aria-hidden
                style={{
                  width: 8,
                  height: 8,
                  borderRadius: "50%",
                  background: accent,
                  opacity: active ? 1 : 0.55,
                  flexShrink: 0,
                }}
              />
              <span style={{ fontWeight: active ? 600 : 400 }}>{s.name}</span>
              {configured ? (
                <span style={{ display: "flex", gap: 3, alignItems: "center" }}>
                  <StatusDot up={serveUp} label="serve" />
                  <StatusDot up={dockerUp} label="docker" />
                </span>
              ) : (
                <span
                  className="dim"
                  style={{ fontSize: "var(--fs-xxx-small)", fontStyle: "italic" }}
                >
                  unconfigured
                </span>
              )}
            </button>
          );
        })}
      </div>
      <button
        onClick={onManage}
        title="Add, remove, and configure servers"
        style={{
          padding: "4px 10px",
          fontSize: "var(--fs-xx-small)",
          flexShrink: 0,
        }}
      >
        Manage servers
      </button>
    </div>
  );
}

/// Tiny serve/docker indicator: a dot in green when up, dim otherwise, with the
/// service name as a one-letter suffix so the two are distinguishable.
function StatusDot({ up, label }: { up: boolean; label: string }) {
  return (
    <span
      title={`${label} ${up ? "up" : "down"}`}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 2,
        fontSize: "var(--fs-xxx-small)",
        color: up ? "var(--core-fleet-green)" : "var(--app-text-dim)",
      }}
    >
      <span
        style={{
          width: 6,
          height: 6,
          borderRadius: "50%",
          background: up ? "var(--core-fleet-green)" : "var(--app-border)",
        }}
      />
      {label[0]}
    </span>
  );
}
