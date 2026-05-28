import { useEffect, useMemo, useRef, useState } from "react";
import { listen } from "@tauri-apps/api/event";
import {
  api,
  type LogLine,
  type PerfTemplate,
  type ProcEvent,
  type ProcInfo,
  type Settings,
} from "../../lib/tauri";
import { noAutocorrect } from "../../lib/noAutocorrect";

/// Local-only form shape — v1 keeps these in React state and re-renders
/// from defaults each app launch. We deliberately don't persist them so
/// the spawn config stays simple and there's no migration story when
/// flags change in the agent.
export interface PerfFormConfig {
  server_url: string;
  enroll_secret: string;
  /// Per-template host counts, keyed by template id. A key present here
  /// means the template is selected. The total host count is the SUM of
  /// these values — the agent fatals unless the per-template counts sum
  /// exactly to --host_count, so deriving the total from the sum is the
  /// only model that can't produce an invalid run (see perfArgsFor).
  os_counts: Record<string, number>;
  mdm_enabled: boolean;
  mdm_prob: number;
  mdm_scep_challenge: string;
  start_period: string;
  query_interval: string;
  config_interval: string;
}

/// Seed total used when selecting the first OS (or re-splitting from an
/// empty selection). Matches the old default host count.
const DEFAULT_TOTAL_HOSTS = 200;

const DEFAULT_PERF_FORM: PerfFormConfig = {
  server_url: "https://localhost:8080",
  enroll_secret: "",
  os_counts: { "macos_14.1.2": DEFAULT_TOTAL_HOSTS },
  mdm_enabled: false,
  mdm_prob: 1,
  mdm_scep_challenge: "",
  start_period: "30s",
  query_interval: "10s",
  config_interval: "1m",
};

/// Even-split `total` across `ids` (in the given order), giving the
/// leftover remainder to the leading templates so the parts sum exactly
/// to `total`. Used by OS toggle + quick-pick so the per-template counts
/// always sum to the displayed total.
function evenSplit(total: number, ids: string[]): Record<string, number> {
  const out: Record<string, number> = {};
  const n = ids.length;
  if (n === 0) return out;
  const base = Math.floor(total / n);
  const remainder = total - base * n;
  ids.forEach((id, i) => {
    out[id] = base + (i < remainder ? 1 : 0);
  });
  return out;
}

function totalHosts(counts: Record<string, number>): number {
  return Object.values(counts).reduce((a, b) => a + b, 0);
}

/// Hard cap on concurrent perf runs. The agent can simulate thousands
/// of hosts per process; the cap is here so the app doesn't let
/// you accidentally bury the machine. Enforced both server-side
/// (start button disabled) and at the spawn (a 5th will fail the
/// `pids` check in spawn_managed).
const MAX_PERF_RUNS = 4;
/// All perf runs share this id/channel prefix so we can pick them out
/// of the global proc list and the log writer namespace.
const PERF_PREFIX = "perf-";
/// In-memory tail size for the run-card mini-log. Persistent storage
/// lives in the existing log channel — this is just the live preview.
const PERF_TAIL_LINES = 8;

type PerfLogLine = {
  ts_ms: number;
  message: string;
  stream: "stdout" | "stderr";
};

export function OsqueryPerfTab({
  settings,
  procs,
}: {
  settings: Settings;
  procs: ProcInfo[];
}) {
  const repoPath = settings.repo_path;
  const [templates, setTemplates] = useState<PerfTemplate[]>([]);
  const [error, setError] = useState<string | null>(null);
  // Form lives in React state only — no settings.json round-trip in v1.
  // Each app launch starts from DEFAULT_PERF_FORM; the user re-enters
  // anything custom. Trade-off accepted in exchange for not needing a
  // migration story when agent flags change.
  const [form, setForm] = useState<PerfFormConfig>(DEFAULT_PERF_FORM);
  // Per-run rolling tails, keyed by proc id. Filled by the
  // proc:log/proc:state event listeners (one subscription, not per
  // card — keeps the listener count small even at the cap).
  const [tails, setTails] = useState<Record<string, PerfLogLine[]>>({});
  // Failed runs the user explicitly dismissed via the card's ✕ button.
  // `done` runs are auto-dismissed (filtered below) so they don't need
  // entries here. We never remove ids from this set — the proc itself
  // ages out of the parent procs array soon enough.
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());

  useEffect(() => {
    api
      .perfListTemplates()
      .then(setTemplates)
      .catch((e) => setError(String(e)));
  }, []);

  // Live log subscription. Filters in JS rather than the backend so we
  // share the same `proc:log` channel that fleet serve uses — no new
  // event types to maintain.
  //
  // The `cancelled` flag closes the React-18-StrictMode race where the
  // effect runs → registers a listener (promise pending) → cleanup
  // runs before the promise resolves → effect runs again. Without it,
  // each line lands in the tail twice.
  useEffect(() => {
    let cancelled = false;
    const unlistens: Array<() => void> = [];
    const register = async <T,>(
      event: string,
      handler: (e: { payload: T }) => void,
    ) => {
      const u = await listen<T>(event, handler);
      if (cancelled) u();
      else unlistens.push(u);
    };

    register<LogLine>("proc:log", (e) => {
      const id = e.payload.proc_id;
      if (!id.startsWith(PERF_PREFIX)) return;
      setTails((prev) => {
        const buf = prev[id] ?? [];
        const next = [
          ...buf,
          {
            ts_ms: e.payload.ts_ms,
            message: e.payload.line,
            stream: e.payload.stream,
          },
        ];
        if (next.length > PERF_TAIL_LINES) {
          next.splice(0, next.length - PERF_TAIL_LINES);
        }
        return { ...prev, [id]: next };
      });
    });
    register<ProcEvent>("proc:state", (_e) => {
      // When a run ends (done/failed/stopped), keep its tail visible
      // until the user dismisses the card — useful for inspecting why
      // it died. Trim is the only state change worth handling here.
    });

    return () => {
      cancelled = true;
      unlistens.forEach((u) => u());
    };
  }, []);

  const perfProcs = useMemo(
    () =>
      procs.filter((p) => {
        if (!p.id.startsWith(PERF_PREFIX)) return false;
        // Auto-clean clean exits: when the user (or osquery-perf
        // itself) ends a run normally we drop the card right away.
        // Failed runs stick around so the error is readable, until
        // the user clicks Dismiss.
        if (p.state === "done") return false;
        if (p.state === "failed" && dismissed.has(p.id)) return false;
        return true;
      }),
    [procs, dismissed],
  );
  const activeCount = perfProcs.filter(
    (p) => p.state === "running" || p.state === "stopping",
  ).length;
  const totalHosts = perfProcs
    .filter((p) => p.state === "running" || p.state === "stopping")
    .reduce((sum, p) => sum + extractHostCount(p), 0);

  async function startRun(form: PerfFormConfig, name: string) {
    if (!repoPath) {
      setError("Set the Fleet repo path in Settings first.");
      return;
    }
    if (activeCount >= MAX_PERF_RUNS) {
      setError(`All ${MAX_PERF_RUNS} slots are in use.`);
      return;
    }
    const id = `${PERF_PREFIX}${Date.now().toString(36)}`;
    const args = perfArgsFor(form);
    // Spawn from the agent's directory so `software-library/software.db`
    // (resolved relative to cwd inside the agent) is found. Running
    // from the repo root used to "go run ./cmd/osquery-perf/agent.go"
    // but that breaks the agent's path resolution.
    const perfDir = `${repoPath.replace(/\/$/, "")}/cmd/osquery-perf`;
    try {
      await api.startProcess({
        id,
        label: name,
        cwd: perfDir,
        program: "go",
        args: ["run", "./agent.go", ...args],
        // No log_channel: perf output is ephemeral. The mini-log on the
        // run card subscribes to the `proc:log` event (always emitted)
        // and the 60-line recent_log tail still populates for failure
        // diagnostics — we just skip the disk write + channel ring.
      });
      setError(null);
    } catch (e) {
      setError(String(e));
    }
  }

  function dismissRun(id: string) {
    // Hide immediately via local state for instant feedback…
    setDismissed((prev) => {
      const next = new Set(prev);
      next.add(id);
      return next;
    });
    // …and forget it on the backend so it's gone from list_processes
    // for good. Without this the entry lingers in the proc map and the
    // card reappears when the tab remounts (the local `dismissed` set
    // doesn't survive unmount).
    api.forgetProcess(id).catch((e) => console.error("forgetProcess", e));
  }

  async function killRun(id: string) {
    try {
      await api.stopProcess(id);
    } catch (e) {
      setError(String(e));
    }
  }

  async function killAll() {
    for (const p of perfProcs) {
      if (p.state === "running" || p.state === "stopping") {
        try {
          await api.stopProcess(p.id);
        } catch (e) {
          console.error("kill perf run failed", p.id, e);
        }
      }
    }
  }

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

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <div
        style={{
          padding: "var(--pad-large) var(--pad-large) var(--pad-medium)",
          flexShrink: 0,
        }}
      >
        <StatusStrip
          activeCount={activeCount}
          totalHosts={totalHosts}
          targetUrl={form.server_url}
          onKillAll={killAll}
          hasRunning={activeCount > 0}
        />
      </div>

      {error && (
        <div
          style={{
            margin: "var(--pad-small) var(--pad-medium)",
            padding: "6px 10px",
            background: "rgba(224,120,136,0.08)",
            border: "1px solid var(--ui-error)",
            color: "var(--ui-error)",
            borderRadius: "var(--radius-md)",
            fontSize: "var(--fs-xx-small)",
          }}
        >
          {error}
        </div>
      )}

      <div
        style={{
          flex: 1,
          minHeight: 0,
          minWidth: 0,
          display: "grid",
          gridTemplateColumns: "minmax(0, 1fr) 480px",
          gap: "var(--pad-medium)",
          padding: "var(--pad-medium)",
          overflow: "hidden",
        }}
      >
        <ActiveRunsPanel
          procs={perfProcs}
          tails={tails}
          activeCount={activeCount}
          onKill={killRun}
          onDismiss={dismissRun}
        />
        <NewRunPanel
          form={form}
          onChange={setForm}
          templates={templates}
          onStart={startRun}
          canStart={activeCount < MAX_PERF_RUNS && templates.length > 0}
          activeCount={activeCount}
        />
      </div>
    </div>
  );
}

/* --------------- Status strip --------------- */

function StatusStrip({
  activeCount,
  totalHosts,
  targetUrl,
  onKillAll,
  hasRunning,
}: {
  activeCount: number;
  totalHosts: number;
  targetUrl: string;
  onKillAll: () => void;
  hasRunning: boolean;
}) {
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
      <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span className={`dot ${activeCount > 0 ? "run" : "idle"}`} />
        <span style={{ fontWeight: 600, fontSize: "var(--fs-x-small)" }}>
          {activeCount} / {MAX_PERF_RUNS} runs
        </span>
        {totalHosts > 0 && (
          <span
            className="dim"
            style={{ fontSize: "var(--fs-xx-small)" }}
          >
            · {totalHosts.toLocaleString()} hosts simulated
          </span>
        )}
      </span>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <span
        className="dim"
        style={{ fontSize: "var(--fs-xx-small)" }}
      >
        target ·{" "}
        <span className="mono" style={{ color: "var(--app-text)" }}>
          {targetUrl}
        </span>
      </span>
      <span style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
        <button
          onClick={onKillAll}
          className="danger"
          disabled={!hasRunning}
          style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
        >
          ■ Kill all runs
        </button>
      </span>
    </div>
  );
}

/* --------------- Active runs panel --------------- */

function ActiveRunsPanel({
  procs,
  tails,
  activeCount,
  onKill,
  onDismiss,
}: {
  procs: ProcInfo[];
  tails: Record<string, PerfLogLine[]>;
  activeCount: number;
  onKill: (id: string) => void;
  onDismiss: (id: string) => void;
}) {
  // Show most-recent-first so a freshly-launched run appears at top.
  // Ended runs sit underneath running ones until the user dismisses
  // them.
  const sorted = [...procs].sort(
    (a, b) => (b.started_at_ms ?? 0) - (a.started_at_ms ?? 0),
  );
  return (
    <div
      className="card"
      style={{
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 10,
        minHeight: 0,
        // Without min-width:0 the nowrap log lines inside the cards
        // push this grid column wider than the viewport. The two-column
        // grid's first track is `minmax(0, 1fr)`, but we also clamp
        // here for safety and so nested overflow works as expected.
        minWidth: 0,
        overflow: "hidden",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          Active runs{" "}
          <span style={{ color: "var(--app-text)", fontWeight: 600 }}>
            · {activeCount}
          </span>
        </div>
        <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
          live tail · not saved
        </span>
      </div>
      {sorted.length === 0 ? (
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
          ○ No runs · configure one on the right to start.
        </div>
      ) : (
        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: 8,
            overflow: "auto",
            minHeight: 0,
            // Same reasoning as the panel wrapper: nowrap log lines in
            // RunCard would otherwise widen this container past the
            // grid column. Belt-and-braces.
            minWidth: 0,
          }}
        >
          {sorted.map((p) => (
            <RunCard
              key={p.id}
              proc={p}
              tail={tails[p.id] ?? []}
              onKill={() => onKill(p.id)}
              onDismiss={() => onDismiss(p.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function RunCard({
  proc,
  tail,
  onKill,
  onDismiss,
}: {
  proc: ProcInfo;
  tail: PerfLogLine[];
  onKill: () => void;
  onDismiss: () => void;
}) {
  const display = perfDisplayState(proc, tail);
  const failed = display === "failed";
  const finished = proc.state === "done" || proc.state === "failed";
  const stopping = proc.state === "stopping";

  return (
    <div
      style={{
        background: "var(--app-surface-2)",
        border: failed
          ? "1px solid var(--ui-error)"
          : "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "10px 12px",
        display: "flex",
        flexDirection: "column",
        gap: 8,
        // Prevents the card from stretching past its grid column when
        // the inner log box has overly long lines.
        minWidth: 0,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <span className={`dot ${dotForDisplay(display)}`} />
        <span
          className="mono"
          style={{
            fontSize: "var(--fs-x-small)",
            fontWeight: 600,
            color: "var(--app-text)",
          }}
        >
          {proc.label}
        </span>
        <StatePill state={display} />
        <span
          className="dim"
          style={{
            fontSize: "var(--fs-xx-small)",
            marginLeft: "auto",
          }}
        >
          {humanStarted(proc, finished)}
        </span>
        {!finished ? (
          <button
            onClick={onKill}
            className="danger"
            disabled={stopping}
            style={{ padding: "2px 10px", fontSize: "var(--fs-xx-small)" }}
          >
            {stopping ? "stopping…" : "Kill"}
          </button>
        ) : (
          <>
            <span
              className="dim"
              style={{
                fontSize: "var(--fs-xxx-small)",
                textTransform: "uppercase",
              }}
            >
              {proc.state}
            </span>
            {failed && (
              <button
                onClick={onDismiss}
                title="Remove this card"
                style={{
                  padding: "2px 8px",
                  fontSize: "var(--fs-xxx-small)",
                }}
              >
                Dismiss ✕
              </button>
            )}
          </>
        )}
      </div>

      <div
        style={{
          display: "flex",
          gap: 6,
          flexWrap: "wrap",
          fontSize: "var(--fs-xx-small)",
        }}
      >
        <CommandSummary command={proc.command} />
      </div>

      {failed ? (
        <div
          style={{
            background: "rgba(224,120,136,0.10)",
            border: "1px solid var(--ui-error)",
            borderRadius: "var(--radius-sm)",
            padding: "6px 10px",
            fontSize: "var(--fs-xx-small)",
            color: "var(--ui-error)",
            fontFamily: "var(--font-mono)",
          }}
        >
          ✗ {failureLine(proc, tail)}
        </div>
      ) : (
        <MiniLogBox tail={tail} />
      )}
    </div>
  );
}

function StatePill({ state }: { state: PerfDisplay }) {
  const map: Record<PerfDisplay, { bg: string; text: string }> = {
    running: {
      bg: "var(--core-fleet-green)",
      text: "running",
    },
    starting: {
      bg: "var(--ui-warning)",
      text: "starting",
    },
    failed: { bg: "var(--ui-error)", text: "failed" },
    stopped: {
      bg: "var(--app-text-dim)",
      text: "stopped",
    },
  };
  const c = map[state];
  // Yellow background needs dark text for contrast; the rest read fine
  // against white.
  const fg =
    state === "starting" ? "var(--core-fleet-black)" : "var(--core-fleet-white)";
  return (
    <span
      style={{
        background: c.bg,
        color: fg,
        fontSize: "var(--fs-xxx-small)",
        padding: "1px 6px",
        borderRadius: 3,
        textTransform: "uppercase",
        letterSpacing: "0.05em",
        fontWeight: 600,
      }}
    >
      {c.text}
    </span>
  );
}

function CommandSummary({ command }: { command: string }) {
  // Pull out a friendly summary from the args the spawn line carries
  // (host_count + os_templates). Falls back to the raw command if we
  // can't find them — graceful for unexpected shapes.
  const hostMatch = command.match(/--host_count\s+(\d+)/);
  const tmplMatch = command.match(/--os_templates\s+(\S+)/);
  const mdmMatch = command.match(/--mdm_prob\s+(\S+)/);
  const hosts = hostMatch ? hostMatch[1] : null;
  const tmpls = tmplMatch ? tmplMatch[1].split(",") : [];
  return (
    <>
      {hosts && (
        <span
          style={{
            color: "var(--app-text)",
            fontWeight: 600,
            fontSize: "var(--fs-xx-small)",
          }}
        >
          {hosts} hosts
        </span>
      )}
      {tmpls.length > 0 && (
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          ·
        </span>
      )}
      {tmpls.map((t) => (
        <span
          key={t}
          className="mono"
          style={{
            background:
              "color-mix(in srgb, var(--core-fleet-purple) 10%, transparent)",
            border:
              "1px solid color-mix(in srgb, var(--core-fleet-purple) 50%, transparent)",
            color: "var(--app-text)",
            padding: "0 6px",
            borderRadius: 999,
            fontSize: "var(--fs-xxx-small)",
          }}
        >
          {t.split(":")[0]}
          {t.includes(":") && (
            <span className="dim" style={{ marginLeft: 4 }}>
              · {t.split(":")[1]}
            </span>
          )}
        </span>
      ))}
      {mdmMatch && (
        <>
          <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
            ·
          </span>
          <span
            className="mono"
            style={{
              background: "rgba(123,121,255,0.10)",
              border: "1px solid var(--core-vibrant-blue)",
              padding: "0 6px",
              borderRadius: 999,
              fontSize: "var(--fs-xxx-small)",
              color: "var(--app-text)",
            }}
          >
            MDM {mdmMatch[1]}
          </span>
        </>
      )}
    </>
  );
}

function MiniLogBox({ tail }: { tail: PerfLogLine[] }) {
  const ref = useRef<HTMLDivElement | null>(null);
  // Keep the box scrolled to the bottom as new lines come in. We don't
  // honor a user scroll-up here because the box is tiny — if they
  // want to read history they'll go to the Logs tab.
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [tail]);
  return (
    <div
      ref={ref}
      style={{
        background: "var(--log-bg)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-sm)",
        padding: "6px 8px",
        fontFamily: "var(--font-mono)",
        fontSize: "var(--fs-xxx-small)",
        lineHeight: 1.45,
        height: 110,
        overflow: "auto",
      }}
    >
      {tail.length === 0 ? (
        <div
          className="dim"
          style={{
            fontSize: "var(--fs-xxx-small)",
            opacity: 0.6,
          }}
        >
          (waiting for output…)
        </div>
      ) : (
        tail.map((l, i) => (
          <div
            key={i}
            style={{
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
              // osquery-perf writes ALL output to stderr (Go's log
              // package default), so stream tells us nothing — color by
              // content instead. White by default; red only when the
              // line actually reads as an error.
              color: looksLikeError(l.message)
                ? "var(--ui-error)"
                : "var(--app-text)",
            }}
            title={l.message}
          >
            <span className="dim" style={{ marginRight: 8 }}>
              {formatTime(l.ts_ms)}
            </span>
            {l.message}
          </div>
        ))
      )}
    </div>
  );
}

/* --------------- New run form --------------- */

function NewRunPanel({
  form,
  templates,
  onChange,
  onStart,
  canStart,
  activeCount,
}: {
  form: PerfFormConfig;
  templates: PerfTemplate[];
  onChange: (next: PerfFormConfig) => void;
  onStart: (form: PerfFormConfig, name: string) => void;
  canStart: boolean;
  activeCount: number;
}) {
  const [name, setName] = useState<string>(() => suggestName(form));
  // Track whether the user has manually edited the name — once they
  // have, stop auto-regenerating from the form. Avoids stomping a
  // custom name as they tweak host_count.
  const [nameDirty, setNameDirty] = useState(false);
  useEffect(() => {
    if (!nameDirty) setName(suggestName(form));
  }, [form, nameDirty]);

  // Secret + SCEP mirror locally so masked typing doesn't fire form
  // updates per keystroke.
  const [secretDraft, setSecretDraft] = useState(form.enroll_secret);
  const [scepDraft, setScepDraft] = useState(form.mdm_scep_challenge);
  useEffect(() => setSecretDraft(form.enroll_secret), [form.enroll_secret]);
  useEffect(() => setScepDraft(form.mdm_scep_challenge), [form.mdm_scep_challenge]);
  function commitSecret() {
    if (secretDraft !== form.enroll_secret) onChange({ ...form, enroll_secret: secretDraft });
  }
  function commitScep() {
    if (scepDraft !== form.mdm_scep_challenge) onChange({ ...form, mdm_scep_challenge: scepDraft });
  }

  const [advanced, setAdvanced] = useState(false);

  // Selected ids in template-list order — keeps the command string and
  // the even-split deterministic regardless of click order.
  const selectedIds = templates
    .map((t) => t.id)
    .filter((id) => id in form.os_counts);
  const total = totalHosts(form.os_counts);

  // Toggling an OS preserves the current total and re-splits it evenly
  // across the new selection (per-OS edits below let you override to
  // e.g. 60/20/20). Empty → seed total on first add.
  function toggleOs(id: string) {
    const has = id in form.os_counts;
    const nextIds = has
      ? selectedIds.filter((x) => x !== id)
      : templates.map((t) => t.id).filter((x) => x === id || x in form.os_counts);
    const t = total > 0 ? total : DEFAULT_TOTAL_HOSTS;
    onChange({ ...form, os_counts: evenSplit(t, nextIds) });
  }

  function setOsCount(id: string, n: number) {
    onChange({ ...form, os_counts: { ...form.os_counts, [id]: n } });
  }

  // Quick-pick sets the total and re-splits evenly across the current
  // selection — an explicit "reset to even at N" action.
  function setTotal(n: number) {
    if (selectedIds.length === 0) return;
    onChange({ ...form, os_counts: evenSplit(n, selectedIds) });
  }

  const onlyMobileSelected =
    selectedIds.length > 0 &&
    selectedIds.every((id) => templates.find((t) => t.id === id)?.mobile === true);
  const secretRequired = !onlyMobileSelected;

  // SCEP is only meaningful for Apple MDM enrollment — Windows MDM
  // doesn't use it. So it's required to start only when MDM is on AND
  // at least one Apple template (macOS/iOS/iPadOS) is in the run.
  const anyAppleSelected = selectedIds.some(
    (id) => templates.find((t) => t.id === id)?.apple === true,
  );
  const scepRequired = form.mdm_enabled && anyAppleSelected;

  const previewArgs = perfArgsFor({ ...form, enroll_secret: secretDraft, mdm_scep_challenge: scepDraft });

  function start() {
    // Commit any drafts first so the spawn matches the preview.
    const submitForm: PerfFormConfig = {
      ...form,
      enroll_secret: secretDraft,
      mdm_scep_challenge: scepDraft,
    };
    onChange(submitForm);
    onStart(submitForm, name);
  }

  const startDisabled =
    !canStart ||
    selectedIds.length === 0 ||
    total < 1 ||
    (secretRequired && !secretDraft.trim()) ||
    (scepRequired && !scepDraft.trim());

  return (
    <div
      className="card"
      style={{
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 12,
        minHeight: 0,
        overflow: "auto",
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          New run {!canStart && <span style={{ color: "var(--ui-error)" }}>· disabled</span>}
        </div>
        <span
          className="dim mono"
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          go run ./agent.go
        </span>
      </div>

      <Field label="Name">
        <input
          type="text"
          value={name}
          onChange={(e) => {
            setName(e.target.value);
            setNameDirty(true);
          }}
          {...noAutocorrect}
          className="mono"
          style={{ width: "100%" }}
        />
      </Field>

      <Field
        label="Fleet URL"
        hint="Defaults to https://localhost:8080. Edit if you're pointing at a different server."
      >
        <div style={{ display: "flex", gap: 6 }}>
          <input
            type="text"
            value={form.server_url}
            onChange={(e) => onChange({ ...form, server_url: e.target.value })}
            {...noAutocorrect}
            className="mono"
            style={{ flex: 1 }}
          />
          <button
            onClick={() => onChange({ ...form, server_url: "https://localhost:8080" })}
            style={{ padding: "4px 10px", fontSize: "var(--fs-xxx-small)" }}
            title="Reset to https://localhost:8080"
          >
            ↺
          </button>
        </div>
      </Field>

      <Field
        label={
          <>
            Enroll secret{" "}
            {secretRequired ? (
              <span style={{ color: "var(--ui-error)" }}>*</span>
            ) : (
              <span
                style={{
                  fontSize: "var(--fs-xxx-small)",
                  color: "var(--core-fleet-purple)",
                  textTransform: "uppercase",
                  letterSpacing: "0.05em",
                  marginLeft: 4,
                }}
              >
                not required · mobile only
              </span>
            )}
          </>
        }
        hint="Use fleetctl get enroll_secret or copy from the Fleet UI (Hosts → Manage enroll secret)."
      >
        <input
          type="text"
          value={secretDraft}
          onChange={(e) => setSecretDraft(e.target.value)}
          onBlur={commitSecret}
          {...noAutocorrect}
          className="mono"
          disabled={!secretRequired}
          style={{
            width: "100%",
            opacity: secretRequired ? 1 : 0.5,
          }}
        />
      </Field>

      <Field
        label={
          <span style={{ display: "flex", alignItems: "baseline", gap: 8 }}>
            <span>Host count</span>
            <span
              className="mono"
              style={{ color: "var(--app-text)", fontWeight: 600 }}
            >
              {total.toLocaleString()}
            </span>
            <span style={{ color: "var(--app-text-dim)" }}>
              = sum of per-OS counts
            </span>
          </span>
        }
      >
        <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
            even-split total:
          </span>
          {[10, 50, 100, 500, 1000].map((n) => (
            <button
              key={n}
              onClick={() => setTotal(n)}
              disabled={selectedIds.length === 0}
              title={
                selectedIds.length === 0
                  ? "Select an OS first"
                  : `Split ${n} hosts evenly across selected OSes`
              }
              style={{
                padding: "2px 8px",
                fontSize: "var(--fs-xxx-small)",
                background: n === total ? "rgba(0,194,139,0.10)" : undefined,
                borderColor:
                  n === total
                    ? "var(--core-fleet-green)"
                    : "var(--app-border)",
                color:
                  n === total
                    ? "var(--core-fleet-green)"
                    : "var(--app-text-dim)",
              }}
            >
              {n}
            </button>
          ))}
        </div>
      </Field>

      <Field
        label="OS templates"
        hint="Check an OS to include it; set its host count on the right. Toggling re-splits the current total evenly — then override any count (e.g. 60/20/20). Total = the sum."
      >
        <div
          style={{
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
            background: "var(--app-surface-2)",
            overflow: "hidden",
          }}
        >
          {templates.length === 0 ? (
            <div
              className="dim"
              style={{
                padding: 10,
                fontSize: "var(--fs-xx-small)",
                textAlign: "center",
              }}
            >
              loading templates…
            </div>
          ) : (
            templates.map((t, i) => {
              const selected = t.id in form.os_counts;
              return (
                // Grid (not flex) so the MDM-ONLY badge and the trailing
                // count/id always sit in the same columns regardless of
                // version-label width — the badges line up across rows.
                // The <label> uses display:contents so its children
                // (checkbox/label/version) become grid items while the
                // label still toggles the checkbox on click.
                <div
                  key={t.id}
                  style={{
                    display: "grid",
                    gridTemplateColumns: "auto 84px 1fr auto 80px",
                    alignItems: "center",
                    gap: 10,
                    padding: "5px 10px",
                    background: selected
                      ? "rgba(0,194,139,0.08)"
                      : "transparent",
                    borderTop:
                      i > 0 ? "1px solid var(--app-border)" : "none",
                  }}
                >
                  <label
                    style={{ display: "contents", cursor: "pointer" }}
                  >
                    <input
                      type="checkbox"
                      checked={selected}
                      onChange={() => toggleOs(t.id)}
                      style={{ accentColor: "var(--core-fleet-green)" }}
                    />
                    <span
                      style={{
                        fontSize: "var(--fs-x-small)",
                        color: "var(--app-text)",
                      }}
                    >
                      {t.label}
                    </span>
                    <span
                      className="mono dim"
                      style={{ fontSize: "var(--fs-xx-small)", minWidth: 0 }}
                    >
                      {t.version}
                    </span>
                  </label>
                  {/* col 4: MDM-ONLY badge slot (empty for non-mobile so
                      the column still reserves alignment) */}
                  {t.mobile ? (
                    <span
                      style={{
                        fontSize: "var(--fs-xxx-small)",
                        color: "var(--core-fleet-purple)",
                        padding: "0 5px",
                        border: "1px solid var(--core-fleet-purple)",
                        borderRadius: 3,
                        textTransform: "uppercase",
                        letterSpacing: "0.05em",
                        whiteSpace: "nowrap",
                      }}
                    >
                      mdm only
                    </span>
                  ) : (
                    <span />
                  )}
                  {/* col 5: count input when selected, else the id */}
                  {selected ? (
                    <input
                      type="number"
                      min={0}
                      max={100_000}
                      value={form.os_counts[t.id]}
                      onChange={(e) => {
                        const n = Math.max(0, Math.floor(Number(e.target.value)));
                        if (Number.isFinite(n)) setOsCount(t.id, n);
                      }}
                      className="mono"
                      title={`${t.label} host count`}
                      style={{ width: 80, textAlign: "right", fontWeight: 600 }}
                    />
                  ) : (
                    <span
                      className="mono dim"
                      style={{
                        fontSize: "var(--fs-xxx-small)",
                        textAlign: "right",
                      }}
                    >
                      {t.id}
                    </span>
                  )}
                </div>
              );
            })
          )}
        </div>
      </Field>

      <MdmSection
        form={form}
        onChange={onChange}
        scepRequired={scepRequired}
        scepDraft={scepDraft}
        setScepDraft={setScepDraft}
        commitScep={commitScep}
      />

      <AdvancedSection
        open={advanced}
        setOpen={setAdvanced}
        form={form}
        onChange={onChange}
      />

      <CommandPreview args={previewArgs} />

      <div
        style={{
          marginTop: "auto",
          paddingTop: 10,
          borderTop: "1px solid var(--app-border)",
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 8,
        }}
      >
        <span
          className="dim"
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          {activeCount} / {MAX_PERF_RUNS} slots used ·{" "}
          {MAX_PERF_RUNS - activeCount === 0 ? (
            <span style={{ color: "var(--ui-error)" }}>FULL</span>
          ) : (
            `${MAX_PERF_RUNS - activeCount} free`
          )}
        </span>
        <button
          className="primary"
          onClick={start}
          disabled={startDisabled}
          style={{ padding: "6px 16px" }}
        >
          ▶ Start run
        </button>
      </div>
    </div>
  );
}

function MdmSection({
  form,
  onChange,
  scepRequired,
  scepDraft,
  setScepDraft,
  commitScep,
}: {
  form: PerfFormConfig;
  onChange: (next: PerfFormConfig) => void;
  /// True only when MDM is on AND an Apple template is selected — Apple
  /// MDM enrollment needs SCEP; Windows MDM doesn't. Drives both the
  /// required marker and whether Start is blocked.
  scepRequired: boolean;
  scepDraft: string;
  setScepDraft: (s: string) => void;
  commitScep: () => void;
}) {
  return (
    <div
      style={{
        padding: 10,
        background: form.mdm_enabled
          ? "rgba(123,121,255,0.06)"
          : "var(--app-surface-2)",
        border: form.mdm_enabled
          ? "1px solid var(--core-vibrant-blue)"
          : "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        display: "flex",
        flexDirection: "column",
        gap: 10,
      }}
    >
      <label
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          cursor: "pointer",
        }}
      >
        <input
          type="checkbox"
          checked={form.mdm_enabled}
          onChange={(e) => onChange({ ...form, mdm_enabled: e.target.checked })}
          style={{ accentColor: "var(--core-vibrant-blue)" }}
        />
        <span style={{ fontWeight: 600, fontSize: "var(--fs-x-small)" }}>
          MDM enabled
        </span>
        {form.mdm_enabled && (
          <span style={{ marginLeft: "auto", display: "flex", gap: 6, alignItems: "center" }}>
            <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
              mdm_prob
            </span>
            <input
              type="number"
              min={0}
              max={1}
              step={0.1}
              value={form.mdm_prob}
              onChange={(e) => {
                const n = Number(e.target.value);
                if (Number.isFinite(n) && n >= 0 && n <= 1) {
                  onChange({ ...form, mdm_prob: n });
                }
              }}
              className="mono"
              style={{ width: 70, fontSize: "var(--fs-xxx-small)" }}
            />
          </span>
        )}
      </label>
      {form.mdm_enabled && (
        <div>
          <Field
            label={
              <>
                SCEP challenge{" "}
                {scepRequired ? (
                  <span style={{ color: "var(--ui-error)" }}>*</span>
                ) : (
                  <span
                    style={{
                      fontSize: "var(--fs-xxx-small)",
                      color: "var(--app-text-dim)",
                      marginLeft: 4,
                    }}
                  >
                    · Apple only · not needed for this selection
                  </span>
                )}
              </>
            }
            hint={
              <>
                Required for Apple MDM enrollment (macOS/iOS/iPadOS);
                Windows MDM doesn't use it. Extract with{" "}
                <span className="mono">
                  go run tools/mdm/assets/main.go export -key=&lt;server_private_key&gt; -dir=&lt;tmp&gt; -name=scep_challenge
                </span>{" "}
                in the Fleet repo, then paste the file contents here. The server also needs{" "}
                <span className="mono">FLEET_DEV_MDM_APPLE_DISABLE_PUSH=1</span> in its env.
              </>
            }
          >
            <input
              type="text"
              value={scepDraft}
              onChange={(e) => setScepDraft(e.target.value)}
              onBlur={commitScep}
              {...noAutocorrect}
              className="mono"
              style={{ width: "100%" }}
            />
          </Field>
        </div>
      )}
    </div>
  );
}

function AdvancedSection({
  open,
  setOpen,
  form,
  onChange,
}: {
  open: boolean;
  setOpen: (v: boolean) => void;
  form: PerfFormConfig;
  onChange: (next: PerfFormConfig) => void;
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
        gap: 8,
      }}
    >
      <button
        onClick={() => setOpen(!open)}
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          border: "none",
          padding: 0,
          background: "transparent",
          textAlign: "left",
          cursor: "pointer",
          fontSize: "var(--fs-xx-small)",
        }}
      >
        <span style={{ color: "var(--app-text)" }}>Advanced intervals</span>
        <span
          className="mono dim"
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          start_period {form.start_period} · query {form.query_interval} · config{" "}
          {form.config_interval} <span style={{ color: "var(--core-fleet-green)" }}>{open ? "▴" : "▾"}</span>
        </span>
      </button>
      {open && (
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 8 }}>
          <SmallField label="start_period">
            <input
              type="text"
              value={form.start_period}
              onChange={(e) => onChange({ ...form, start_period: e.target.value })}
              {...noAutocorrect}
              className="mono"
              style={{ width: "100%" }}
            />
          </SmallField>
          <SmallField label="query_interval">
            <input
              type="text"
              value={form.query_interval}
              onChange={(e) => onChange({ ...form, query_interval: e.target.value })}
              {...noAutocorrect}
              className="mono"
              style={{ width: "100%" }}
            />
          </SmallField>
          <SmallField label="config_interval">
            <input
              type="text"
              value={form.config_interval}
              onChange={(e) => onChange({ ...form, config_interval: e.target.value })}
              {...noAutocorrect}
              className="mono"
              style={{ width: "100%" }}
            />
          </SmallField>
        </div>
      )}
    </div>
  );
}

function CommandPreview({ args }: { args: string[] }) {
  // Highlight flag tokens (those starting with `--`) so the block
  // reads like a real shell preview without a syntax highlighter.
  return (
    <div
      style={{
        background: "var(--log-bg)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-sm)",
        padding: "8px 10px",
        fontFamily: "var(--font-mono)",
        fontSize: "var(--fs-xxx-small)",
        color: "var(--app-text)",
        lineHeight: 1.5,
        wordBreak: "break-all",
      }}
    >
      <span className="dim">$ </span>
      go run ./agent.go{" "}
      {args.map((a, i) => (
        <span key={i}>
          {a.startsWith("--") ? (
            <span style={{ color: "var(--core-fleet-green)" }}>{a}</span>
          ) : (
            a
          )}
          {i < args.length - 1 ? " " : ""}
        </span>
      ))}
    </div>
  );
}

/* --------------- Small helpers --------------- */

function Field({
  label,
  hint,
  children,
}: {
  label: React.ReactNode;
  hint?: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)" }}>
        {label}
      </div>
      {children}
      {hint && (
        <div className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
          {hint}
        </div>
      )}
    </div>
  );
}

function SmallField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 3 }}>
      <span
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)" }}
      >
        {label}
      </span>
      {children}
    </div>
  );
}

/// Builds the argv tail (after `agent.go`). Kept here so the preview
/// and spawn always come out of the same function — preview can never
/// drift from what's actually run.
export function perfArgsFor(form: PerfFormConfig): string[] {
  const args: string[] = [];
  if (form.server_url.trim()) {
    args.push("--server_url", form.server_url.trim());
  }
  // Mobile-only runs skip the secret per the osquery-perf README.
  if (form.enroll_secret.trim()) {
    args.push("--enroll_secret", form.enroll_secret.trim());
  }
  // Total = sum of per-template counts; the agent fatals if the
  // template counts don't add up to --host_count exactly, so we always
  // pass both and let the sum define the total.
  const ids = orderedSelectedIds(form.os_counts);
  const total = totalHosts(form.os_counts);
  args.push("--host_count", String(total));
  if (ids.length > 0) {
    const spec = ids.map((id) => `${id}:${form.os_counts[id]}`).join(",");
    args.push("--os_templates", spec);
  }
  if (form.mdm_enabled) {
    args.push("--mdm_prob", String(form.mdm_prob));
    if (form.mdm_scep_challenge.trim()) {
      args.push("--mdm_scep_challenge", form.mdm_scep_challenge.trim());
    }
  }
  if (form.start_period.trim()) {
    args.push("--start_period", form.start_period.trim());
  }
  if (form.query_interval.trim()) {
    args.push("--query_interval", form.query_interval.trim());
  }
  if (form.config_interval.trim()) {
    args.push("--config_interval", form.config_interval.trim());
  }
  return args;
}

/// Selected template ids in a stable order. We don't have the template
/// list here, so we fall back to insertion order of the count map keys,
/// which is good enough for the command string.
function orderedSelectedIds(counts: Record<string, number>): string[] {
  return Object.keys(counts);
}

function suggestName(form: PerfFormConfig): string {
  const ids = Object.keys(form.os_counts);
  const total = totalHosts(form.os_counts);
  if (ids.length === 0) {
    return `run-${total}`;
  }
  if (ids.length === 1) {
    const os = ids[0].split(".")[0].split("_")[0];
    return `run-${total}-${os}`;
  }
  const suffix = form.mdm_enabled ? "-mdm" : "";
  return `run-${total}-mixed${suffix}`;
}

function extractHostCount(p: ProcInfo): number {
  const m = p.command.match(/--host_count\s+(\d+)/);
  return m ? Number(m[1]) : 0;
}

type PerfDisplay = "starting" | "running" | "failed" | "stopped";

function perfDisplayState(p: ProcInfo, tail: PerfLogLine[]): PerfDisplay {
  if (p.state === "failed") return "failed";
  if (p.state === "done") return "stopped";
  if (p.state === "stopping") return "stopped";
  // Starting = spawn alive but no log output yet AND younger than ~6s.
  // Once a log line lands or the runtime crosses the threshold, we
  // flip to "running" so the user sees real progress.
  if (p.state === "running") {
    const age = p.started_at_ms != null ? Date.now() - p.started_at_ms : 0;
    if (tail.length === 0 && age < 6000) return "starting";
    return "running";
  }
  return "stopped";
}

function dotForDisplay(d: PerfDisplay): string {
  switch (d) {
    case "running":
      return "run";
    case "starting":
      return "warn";
    case "failed":
      return "fail";
    case "stopped":
      return "idle";
  }
}

function humanStarted(p: ProcInfo, finished: boolean): string {
  const ref = p.ended_at_ms ?? p.started_at_ms ?? null;
  if (ref == null) return "—";
  const ago = Date.now() - ref;
  const sec = Math.floor(ago / 1000);
  const verb = finished ? "ended" : "started";
  if (sec < 5) return `${verb} just now`;
  if (sec < 60) return `${verb} ${sec}s ago`;
  if (sec < 3600) return `${verb} ${Math.floor(sec / 60)}m ago`;
  return `${verb} ${Math.floor(sec / 3600)}h ago`;
}

function failureLine(p: ProcInfo, tail: PerfLogLine[]): string {
  // Prefer the synthetic [exit: …] line the backend appends on exit,
  // then any stderr-flavored tail, then a fallback message.
  const synth = [...p.recent_log].reverse().find((l) => l.startsWith("[exit:"));
  if (synth) return synth;
  const errish = [...tail]
    .reverse()
    .find((l) => l.stream === "stderr" || /error|fatal|panic/i.test(l.message));
  if (errish) return errish.message;
  if (p.exit_code != null) return `exit code ${p.exit_code}`;
  return "exited without diagnostic output — see Logs tab for the full channel";
}

/// Conservative error detector for the mini-log. osquery-perf has no
/// level tokens (plain Go `log` lines), so we look for strong signals
/// only and deliberately avoid bare "error" — the routine stats line
/// reads "…, error rate: 0.00, …" and must NOT turn red. We match
/// `error:`/`error=` (error immediately followed by a separator),
/// fatal/panic, "failed", and slog-style level=error.
function looksLikeError(message: string): boolean {
  return /\b(fatal|panic)\b|error[:=]|\bfailed\b|level=err/i.test(message);
}

function formatTime(ms: number): string {
  const d = new Date(ms);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
