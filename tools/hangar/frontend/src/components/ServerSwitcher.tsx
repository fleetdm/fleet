import type { Settings } from "../lib/ipc";
import { serverColorVar } from "../lib/servers";
import type { ServerHealth } from "../lib/useSystemHealth";

/// Top-bar control for switching between local Fleet servers and seeing each
/// one's status at a glance. One pill per server: a single status dot (green
/// and pulsing when anything is running on that server, grey when off — the
/// same dot the Server-tab process cards use), the name, and an accent border
/// when active. "Manage" jumps to the Servers settings section.
export function ServerSwitcher({
  settings,
  healthMap,
  onSwitch,
  onManage,
  dimmed = false,
}: {
  settings: Settings;
  healthMap: Record<string, ServerHealth>;
  onSwitch: (id: string) => void;
  onManage: () => void;
  // Faded when the active tab is global (server selection doesn't apply there).
  dimmed?: boolean;
}) {
  const activeId = settings.active_server_id;
  return (
    <div
      title={
        dimmed
          ? "Server selection applies to the Git / Server / Logs / Database tabs"
          : undefined
      }
      style={{
        display: "flex",
        alignItems: "center",
        gap: 8,
        padding: "5px var(--pad-medium)",
        background: "var(--app-surface)",
        borderBottom: "1px solid var(--app-border)",
        overflowX: "auto",
        opacity: dimmed ? 0.45 : 1,
        transition: "opacity 0.15s ease",
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
          // Anything running on this server -> one green (pulsing) dot; else grey.
          const running = configured && (serveUp || dockerUp);
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
              <span aria-hidden className={`dot ${running ? "run" : "idle"}`} />
              <span style={{ fontWeight: active ? 600 : 400 }}>{s.name}</span>
              {!configured && (
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
