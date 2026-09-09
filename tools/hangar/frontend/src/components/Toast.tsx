// Toast is the small transient confirmation/error used by the SCEP, MDM assets,
// and TUF tabs. Errors use the danger color; everything else sits on a surface.
export function Toast({ kind, msg }: { kind: "ok" | "err"; msg: string }) {
  return (
    <div
      style={{
        position: "fixed",
        bottom: 16,
        left: "50%",
        transform: "translateX(-50%)",
        background: kind === "err" ? "var(--core-vibrant-red)" : "var(--app-surface)",
        color: kind === "err" ? "var(--core-fleet-white)" : "var(--app-text)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "var(--pad-small) var(--pad-medium)",
        fontSize: "var(--fs-x-small)",
        boxShadow: "var(--shadow-popover)",
        zIndex: 1200,
      }}
    >
      {msg}
    </div>
  );
}
