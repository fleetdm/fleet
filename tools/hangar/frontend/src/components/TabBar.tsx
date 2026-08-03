export type TabId =
  | "server"
  | "logs"
  | "database"
  | "git"
  | "fleetctl"
  | "gitops"
  | "osquery-perf"
  | "settings";

const TABS: { id: TabId; label: string }[] = [
  { id: "git", label: "Git" },
  { id: "server", label: "Server" },
  { id: "logs", label: "Logs" },
  { id: "database", label: "Database" },
  { id: "fleetctl", label: "fleetctl" },
  { id: "gitops", label: "GitOps" },
  { id: "osquery-perf", label: "osquery-perf" },
];

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
      <div style={{ display: "flex", flex: 1, minWidth: 0 }}>
        {TABS.map((t) => {
          const isActive = t.id === active;
          return (
            <button
              key={t.id}
              onClick={() => onChange(t.id)}
              className={`tab-btn${isActive ? " is-active" : ""}`}
            >
              {t.label}
            </button>
          );
        })}
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
