import type { SlotUsage } from "../lib/ipc";

function basename(p: string): string {
  return p.replace(/\/+$/, "").split("/").pop() || p;
}

/// Text badges for a slot held by an agent: who reserved it and the running
/// job. Renders nothing for an unreserved slot.
export function SlotBadges({ usage }: { usage?: SlotUsage }) {
  if (!usage?.reserved_by && !usage?.job) return null;
  const chip = {
    fontSize: "var(--fs-xxx-small)",
    padding: "1px 6px",
    borderRadius: 999,
    border: "1px solid var(--app-border)",
    background: "var(--app-surface-2)",
    color: "var(--app-text-dim)",
    whiteSpace: "nowrap" as const,
  };
  return (
    <>
      {usage.reserved_by && (
        <span style={chip} title={usage.reserved_by}>
          Reserved by {basename(usage.reserved_by)}
        </span>
      )}
      {usage.job && (
        <span style={chip} title={`job ${usage.job.id}`} role="status">
          Agent: {usage.job.action}
        </span>
      )}
    </>
  );
}

/// One-line note beside the worktree picker; a reserved slot reverts GUI picks.
export function SlotReservedNote({ usage }: { usage?: SlotUsage }) {
  if (!usage?.reserved_by) return null;
  return (
    <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)" }}>
      Reserved by {basename(usage.reserved_by)}
      {usage.job ? ` (Agent: ${usage.job.action})` : ""}. Changes to this
      worktree are reverted while the reservation is held.
    </div>
  );
}
