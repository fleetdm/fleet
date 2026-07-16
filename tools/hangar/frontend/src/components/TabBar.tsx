export type TabId =
  | "server"
  | "logs"
  | "database"
  | "git"
  | "fleetctl"
  | "gitops"
  | "osquery-perf"
  | "settings";

// Tabs whose content is scoped to the active server — the top server pills
// drive these.
const SERVER_TABS: { id: TabId; label: string }[] = [
  { id: "git", label: "Git" },
  { id: "server", label: "Server" },
  { id: "logs", label: "Logs" },
  { id: "database", label: "Database" },
];

// Global tabs: shared services / tools that don't depend on the active server.
const GLOBAL_TABS: { id: TabId; label: string }[] = [
  { id: "fleetctl", label: "fleetctl" },
  { id: "gitops", label: "GitOps" },
  { id: "osquery-perf", label: "osquery-perf" },
];

const SERVER_TAB_IDS = new Set<TabId>(SERVER_TABS.map((t) => t.id));

// isServerScopedTab reports whether a tab's content depends on the active
// server (so callers can, e.g., dim the server switcher on global tabs).
export function isServerScopedTab(id: TabId): boolean {
  return SERVER_TAB_IDS.has(id);
}

export function TabBar({
  active,
  onChange,
}: {
  active: TabId;
  onChange: (id: TabId) => void;
}) {
  const settingsActive = active === "settings";
  return (
    <nav
      style={{
        display: "flex",
        alignItems: "stretch",
        background: "var(--app-bg)",
        borderBottom: "1px solid var(--app-border)",
        padding: "0 var(--pad-medium)",
      }}
    >
      <div
        style={{ display: "flex", flex: 1, minWidth: 0, alignItems: "stretch" }}
      >
        {SERVER_TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => onChange(t.id)}
            className={`tab-btn${t.id === active ? " is-active" : ""}`}
          >
            {t.label}
          </button>
        ))}
        <div
          aria-hidden
          style={{
            width: 1,
            alignSelf: "center",
            height: "50%",
            margin: "0 8px",
            background: "var(--app-border)",
          }}
        />
        {GLOBAL_TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => onChange(t.id)}
            className={`tab-btn${t.id === active ? " is-active" : ""}`}
          >
            {t.label}
          </button>
        ))}
      </div>
      <button
        onClick={() => onChange("settings")}
        title="Settings"
        aria-label="Settings"
        className={`tab-gear${settingsActive ? " is-active" : ""}`}
      >
        ⚙
      </button>
    </nav>
  );
}
