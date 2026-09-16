import type { BranchStatus, ProcInfo } from "../lib/ipc";

function branchState(s: BranchStatus | null): {
  dot: string;
  label: string;
} {
  if (!s) return { dot: "idle", label: "no repo" };
  if (s.behind > 0) return { dot: "warn", label: `behind ${s.behind}` };
  return { dot: "ok", label: "up to date" };
}

export function StatusRail({
  branchStatus,
  procs,
  dockerUp,
}: {
  branchStatus: BranchStatus | null;
  procs: ProcInfo[];
  dockerUp: boolean;
}) {
  const bs = branchState(branchStatus);

  return (
    <footer
      style={{
        height: 26,
        background: "var(--app-surface)",
        borderTop: "1px solid var(--app-border)",
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        padding: "0 var(--pad-medium)",
        fontSize: "var(--fs-xx-small)",
        color: "var(--app-text-dim)",
        gap: "var(--pad-medium)",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <span className={`dot ${bs.dot}`} />
        <span className="mono" style={{ color: "var(--app-text)" }}>
          {branchStatus?.branch ?? "—"}
        </span>
        <span className="dim">· {bs.label}</span>
      </div>

      {/* Services indicator collapsed to the right side. docker is
          stitched in alongside the managed procs because docker compose
          up -d exits quickly (so it doesn't appear in procs) but the
          containers persist — health probe tells the real story. */}
      <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
        <ProcSummary procs={procs} dockerUp={dockerUp} />
      </div>
    </footer>
  );
}

function ProcSummary({
  procs,
  dockerUp,
}: {
  procs: ProcInfo[];
  dockerUp: boolean;
}) {
  const running = procs.filter(
    (p) => p.state === "running" || p.state === "stopping",
  );
  // Docker compose's spawn (id: docker-compose-up) goes "done" almost
  // immediately because `-d` returns once containers are launched. We
  // surface it as a synthetic chip driven by the health probe so the
  // user sees the running stack even though we don't own a live spawn
  // for it. Also filter out the docker-compose-up proc itself when
  // surfaced this way to avoid showing it twice.
  const ownChips = running.filter((p) => !p.id.endsWith("docker-compose-up"));

  const totalChips = ownChips.length + (dockerUp ? 1 : 0);
  if (totalChips === 0) {
    return <span className="dim">no processes</span>;
  }
  if (totalChips >= 5) {
    return (
      <span
        style={{ display: "flex", alignItems: "center", gap: 4 }}
        title={[
          ...ownChips.map((p) => p.label),
          ...(dockerUp ? ["docker compose"] : []),
        ].join(", ")}
      >
        <span className="dot run" />
        <span>{totalChips} services</span>
      </span>
    );
  }
  // 1–4: individual chips with shorter labels above 2.
  const compact = totalChips >= 3;
  return (
    <>
      {ownChips.map((p) => (
        <span
          key={p.id}
          style={{ display: "flex", alignItems: "center", gap: 4 }}
          title={p.command}
        >
          <span className={`dot ${p.state}`} />
          <span>{compact ? shortLabel(p.label) : p.label}</span>
        </span>
      ))}
      {dockerUp && (
        <span
          style={{ display: "flex", alignItems: "center", gap: 4 }}
          title="docker compose stack"
        >
          <span className="dot run" />
          <span>{compact ? "docker" : "docker compose"}</span>
        </span>
      )}
    </>
  );
}

function shortLabel(label: string): string {
  // Trim verbose labels for the compact 3–4 procs view.
  if (label.startsWith("fleet serve")) return "serve";
  if (label.startsWith("docker compose")) return "docker";
  if (label === "python http.server") return "http";
  return label;
}
