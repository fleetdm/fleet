import { useCallback, useEffect, useState } from "react";
import {
  api,
  type NgrokYamlInfo,
  type ProcInfo,
  type ServerProfile,
  type Settings,
} from "../../lib/ipc";
import {
  buildChainFor,
  dockerUpWithStaleCleanup,
  ngrokArgsFor,
  staleNgrokTunnels,
  ngrokIsLaunchable,
  pythonArgsFor,
  serveArgsFor,
  serveEnvFor,
  serveLabelFor,
  startAll as orchestrationStartAll,
  stopAll as orchestrationStopAll,
  waitForExit,
} from "../../lib/orchestration";
import {
  dockerUpArgs,
  prepareDbArgsFor,
  procId,
  serveChannel,
  updateServer,
} from "../../lib/servers";
import type {
  DockerHealth,
  ServeStatus,
} from "../../lib/useSystemHealth";
import type { SettingsSection } from "./SettingsTab";

type ChainStep = {
  id: string;
  label: string;
  program: string;
  args: string[];
  /// "skip awaiting spawn exit in chain runAll" — for things like fleet
  /// serve where the spawn never returns. Pure chain-flow concern.
  longRunning?: boolean;
  logChannel?: string;
  /// When true, the step row hides the ■ button while running because
  /// the process is also represented in the Active processes panel
  /// where stop control lives.
  hideStop?: boolean;
  /// When true, the step row's display state comes from the health probe
  /// (externalRunning), not the spawn exit code, and it gets the
  /// LONG-RUNNING tag in the meta column. Set for steps that start a
  /// persistent service (docker compose up -d, fleet serve --dev).
  service?: boolean;
  /// Identifies the special steps without string-matching ids (ids are now
  /// per-server-namespaced): "docker-up" routes ▶ through the stale-cleanup
  /// helper; "serve" gets the command preview rendered beneath it.
  kind?: "docker-up" | "serve";
  /// Env vars to apply on the spawn, as [key, value] tuples (matches
  /// the IPC shape). Only meaningful for the fleet-serve step right
  /// now — the build chain inherits the parent env unmodified.
  env?: Array<[string, string]>;
};

/// Build the run chain for a server — ids are namespaced to the server, the
/// fleet-serve argv/env derive from the server's serve config + ports, and
/// docker/prepare-db carry the per-server project/address flags.
function runChainFor(server: ServerProfile): ChainStep[] {
  return [
    {
      id: procId(server.id, "docker-compose-up"),
      label: "docker compose up -d",
      program: "docker",
      args: dockerUpArgs(server),
      hideStop: true,
      service: true,
      kind: "docker-up",
    },
    {
      id: procId(server.id, "fleet-prepare-db"),
      label: "fleet prepare db --dev",
      program: "./build/fleet",
      args: prepareDbArgsFor(server),
    },
    {
      id: procId(server.id, "fleet-serve"),
      label: serveLabelFor(server),
      program: "./build/fleet",
      args: serveArgsFor(server),
      env: serveEnvFor(server),
      longRunning: true,
      logChannel: serveChannel(server.id),
      hideStop: true,
      service: true,
      kind: "serve",
    },
  ];
}

export function ServerTab({
  server,
  settings,
  onSettingsChange,
  procs,
  currentBranch,
  serve,
  docker,
  goToLogs,
  goToSettings,
}: {
  server: ServerProfile;
  settings: Settings;
  onSettingsChange: (next: Settings) => void;
  procs: ProcInfo[];
  currentBranch: string | null;
  serve: ServeStatus;
  docker: DockerHealth;
  goToLogs: () => void;
  goToSettings: (section: SettingsSection) => void;
}) {
  const repoPath = server.worktree_path;
  const buildChain: ChainStep[] = buildChainFor(server);
  const [now, setNow] = useState(Date.now());
  // Tracks which branch each Build chain step ran against, so we can
  // visually reset to ○ idle when the user switches branches.
  const [buildBranchByStepId, setBuildBranchByStepId] = useState<
    Record<string, string>
  >({});

  const captureBuildBranch = useCallback(
    (stepId: string) => {
      if (!currentBranch) return;
      setBuildBranchByStepId((prev) => ({ ...prev, [stepId]: currentBranch }));
    },
    [currentBranch],
  );

  // Tick the clock so uptime labels refresh.
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

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

  const ngrokProc = procs.find((p) => p.id === "ngrok");
  const pythonProc = procs.find((p) => p.id === "python-server");
  const ngrokRunning =
    ngrokProc?.state === "running" || ngrokProc?.state === "stopping";
  const pythonRunning =
    pythonProc?.state === "running" || pythonProc?.state === "stopping";

  const runChain = runChainFor(server);

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        padding: "var(--pad-large)",
        gap: "var(--pad-medium)",
        overflow: "auto",
      }}
    >
      <MasterControl
        server={server}
        settings={settings}
        serve={serve}
        docker={docker}
        ngrokRunning={ngrokRunning}
        pythonRunning={pythonRunning}
        onBuildStepRun={captureBuildBranch}
      />
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: "var(--pad-medium)",
        }}
      >
        <ChainCard
          title="Build chain"
          subtitle="Install deps, bundle frontend, build fleet + fleetctl"
          steps={buildChain}
          server={server}
          repoPath={repoPath}
          procs={procs}
          now={now}
          branchByStepId={buildBranchByStepId}
          currentBranch={currentBranch}
          onStepRun={captureBuildBranch}
        />
        <ChainCard
          title="Run chain"
          subtitle="Bring up dev environment"
          steps={runChain}
          server={server}
          repoPath={repoPath}
          procs={procs}
          now={now}
          externalRunningByStepId={{
            [procId(server.id, "docker-compose-up")]: docker.up,
            [procId(server.id, "fleet-serve")]: serve.up,
          }}
          header={
            <div style={{ display: "flex", gap: 6 }}>
              <HealthChip label="serve" up={serve.up} />
              <HealthChip label="docker" up={docker.up} />
            </div>
          }
          fleetServePreview={
            <ServePreview
              server={server}
              onTogglePremium={(premium) => {
                const next = updateServer(settings, server.id, (s) => ({
                  ...s,
                  fleet_serve: { ...s.fleet_serve, premium },
                }));
                onSettingsChange(next);
                api.saveSettings(next).catch((e) =>
                  console.error("save serve settings failed", e),
                );
              }}
              onConfigure={() => goToSettings("fleet-server")}
            />
          }
        />
      </div>

      <ActiveProcessesPanel
        server={server}
        settings={settings}
        onSettingsChange={onSettingsChange}
        procs={procs}
        serve={serve}
        docker={docker}
        now={now}
        goToLogs={goToLogs}
        goToSettings={goToSettings}
      />

      <RecentPanel procs={procs} now={now} />
    </div>
  );
}

function MasterControl({
  server,
  settings,
  serve,
  docker,
  ngrokRunning,
  pythonRunning,
  onBuildStepRun,
}: {
  server: ServerProfile;
  settings: Settings;
  serve: ServeStatus;
  docker: DockerHealth;
  ngrokRunning: boolean;
  pythonRunning: boolean;
  onBuildStepRun?: (stepId: string) => void;
}) {
  const [busy, setBusy] = useState<"starting" | "stopping" | null>(null);

  // Toggle is keyed off "anything running" so the button always offers
  // the action the user is most likely to want: Stop all when anything is
  // alive, Start all when everything is down.
  const anyRunning =
    serve.up || docker.up || ngrokRunning || pythonRunning;
  const allRunning =
    serve.up &&
    docker.up &&
    (!settings.ngrok.enabled ||
      !ngrokIsLaunchable(settings) ||
      ngrokRunning) &&
    (!settings.python_server.enabled || pythonRunning);

  async function startAll() {
    setBusy("starting");
    try {
      await orchestrationStartAll({
        server,
        settings,
        health: {
          serveUp: serve.up,
          dockerUp: docker.up,
          ngrokRunning,
          pythonRunning,
        },
        onBuildStepRun,
      });
    } catch (e) {
      console.error("start all failed", e);
    }
    setBusy(null);
  }

  async function stopAll() {
    setBusy("stopping");
    try {
      await orchestrationStopAll({
        server,
        health: {
          serveUp: serve.up,
          dockerUp: docker.up,
          ngrokRunning,
          pythonRunning,
        },
      });
    } catch (e) {
      console.error("stop all failed", e);
    }
    setBusy(null);
  }

  const mode: "start" | "stop" = anyRunning ? "stop" : "start";

  return (
    <div
      style={{
        display: "flex",
        justifyContent: "flex-end",
        alignItems: "center",
        gap: 10,
      }}
    >
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        {busy === "starting"
          ? "starting build → run → optional services…"
          : busy === "stopping"
            ? "stopping everything…"
            : mode === "stop"
              ? allRunning
                ? "everything is up"
                : "some services running"
              : "ready to start"}
      </div>
      {mode === "stop" ? (
        <button
          onClick={stopAll}
          disabled={!!busy}
          className="danger"
          style={{ padding: "6px 14px" }}
        >
          {busy === "stopping" ? "stopping…" : "■ Stop all"}
        </button>
      ) : (
        <button
          onClick={startAll}
          disabled={!!busy}
          className="primary"
          style={{ padding: "6px 14px" }}
        >
          {busy === "starting" ? "starting…" : "▶ Start all"}
        </button>
      )}
    </div>
  );
}

/// Preview + premium toggle rendered just under the fleet-serve row.
/// Shows the exact `./build/fleet serve …` line that will spawn plus
/// the env-var keys (values redacted). Premium toggle persists into the
/// active server's `fleet_serve.premium`; everything else (config path,
/// debug flags, env values) lives in Settings → Fleet server.
function ServePreview({
  server,
  onTogglePremium,
  onConfigure,
}: {
  server: ServerProfile;
  onTogglePremium: (next: boolean) => void;
  onConfigure: () => void;
}) {
  const args = serveArgsFor(server);
  const env = serveEnvFor(server);
  const cmdPreview = `./build/fleet ${args.join(" ")}`;
  const premium = server.fleet_serve.premium;

  return (
    <div
      style={{
        marginLeft: 28,
        marginTop: 4,
        marginBottom: 2,
        padding: "8px 10px",
        background: "var(--app-surface)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
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
          gap: 8,
        }}
      >
        <PremiumToggle premium={premium} onChange={onTogglePremium} />
        <button
          onClick={onConfigure}
          className="link-btn"
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          Settings → Fleet server ↗
        </button>
      </div>
      <div
        className="mono"
        style={{
          fontSize: "var(--fs-xxx-small)",
          color: "var(--app-text-dim)",
          wordBreak: "break-all",
        }}
        title={cmdPreview}
      >
        {cmdPreview}
      </div>
      {env.length > 0 && (
        <div
          style={{
            fontSize: "var(--fs-xxx-small)",
            color: "var(--app-text-dim)",
            display: "flex",
            gap: 4,
            flexWrap: "wrap",
          }}
        >
          <span style={{ textTransform: "uppercase", letterSpacing: "0.06em" }}>
            env
          </span>
          {env.map(([k]) => (
            <span
              key={k}
              className="mono"
              style={{
                background: "var(--app-surface-2)",
                border: "1px solid var(--app-border)",
                borderRadius: "var(--radius-sm)",
                padding: "0 6px",
                color: "var(--app-text)",
              }}
              title={`${k}=··· (value hidden)`}
            >
              {k}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}

function PremiumToggle({
  premium,
  onChange,
}: {
  premium: boolean;
  onChange: (next: boolean) => void;
}) {
  return (
    <div
      role="group"
      aria-label="License mode"
      style={{
        display: "inline-flex",
        background: "var(--app-surface-2)",
        border: "1px solid var(--app-border)",
        borderRadius: 999,
        padding: 2,
        gap: 0,
      }}
    >
      <SegmentButton
        active={premium}
        onClick={() => onChange(true)}
        title="Spawns with --dev_license"
      >
        Premium
      </SegmentButton>
      <SegmentButton
        active={!premium}
        onClick={() => onChange(false)}
        title="Spawns without --dev_license"
      >
        Free
      </SegmentButton>
    </div>
  );
}

function SegmentButton({
  active,
  onClick,
  title,
  children,
}: {
  active: boolean;
  onClick: () => void;
  title?: string;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      title={title}
      style={{
        padding: "3px 12px",
        fontSize: "var(--fs-xxx-small)",
        textTransform: "uppercase",
        letterSpacing: "0.06em",
        border: "none",
        borderRadius: 999,
        background: active ? "var(--core-fleet-green)" : "transparent",
        color: active ? "var(--core-fleet-white)" : "var(--app-text-dim)",
        cursor: "pointer",
      }}
    >
      {children}
    </button>
  );
}

function HealthChip({ label, up }: { label: string; up: boolean }) {
  const state = up ? "up" : "down";
  const color = up ? "var(--core-fleet-green)" : "var(--ui-error)";
  const bg = up ? "var(--tint-success-soft)" : "var(--tint-danger-soft)";
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 6,
        padding: "3px 10px 3px 8px",
        background: bg,
        border: `1px solid ${color}`,
        borderRadius: 999,
        fontSize: "var(--fs-xx-small)",
        color,
      }}
    >
      <span className={`dot ${up ? "run" : "fail"}`} />
      <span style={{ fontWeight: 600 }}>{label}</span>
      <span
        style={{
          fontSize: "var(--fs-xxx-small)",
          textTransform: "uppercase",
          letterSpacing: "0.05em",
          color: "var(--core-fleet-white)",
          background: color,
          padding: "1px 5px",
          borderRadius: 3,
        }}
      >
        {state}
      </span>
    </div>
  );
}

function ChainCard({
  title,
  subtitle,
  steps,
  server,
  repoPath,
  procs,
  now,
  header,
  externalRunningByStepId,
  branchByStepId,
  currentBranch,
  onStepRun,
  fleetServePreview,
}: {
  title: string;
  subtitle: string;
  steps: ChainStep[];
  server: ServerProfile;
  repoPath: string;
  procs: ProcInfo[];
  now: number;
  header?: React.ReactNode;
  externalRunningByStepId?: Record<string, boolean>;
  branchByStepId?: Record<string, string>;
  currentBranch?: string | null;
  onStepRun?: (stepId: string) => void;
  /// Optional preview rendered under the fleet-serve row so the user
  /// can see the resolved argv before clicking Play. Only the Run chain
  /// passes this — the Build chain has no equivalent toggle to preview.
  fleetServePreview?: React.ReactNode;
}) {
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function runStep(step: ChainStep): Promise<boolean> {
    setError(null);
    onStepRun?.(step.id);
    try {
      // docker compose up -d is special — a leftover container from a prior
      // session causes "container name already in use" and the spawn exits 1.
      // The helper does an idempotent `compose down` first to guarantee a
      // clean slate (and applies this server's project + ports). Start all has
      // always used this path; routing the individual ▶ and Run all through it
      // too closes the inconsistency.
      if (step.kind === "docker-up") {
        return await dockerUpWithStaleCleanup(server);
      }
      await api.startProcess({
        id: step.id,
        label: step.label,
        cwd: repoPath,
        program: step.program,
        args: step.args,
        log_channel: step.logChannel,
        env: step.env && step.env.length > 0 ? step.env : null,
      });
      if (step.longRunning) return true;
      return await waitForExit(step.id);
    } catch (e) {
      setError(String(e));
      return false;
    }
  }

  async function runAll() {
    setRunning(true);
    for (const step of steps) {
      const ok = await runStep(step);
      if (!ok) break;
    }
    setRunning(false);
  }

  function isStepStale(stepId: string): boolean {
    if (!branchByStepId || !currentBranch) return false;
    const captured = branchByStepId[stepId];
    return captured != null && captured !== currentBranch;
  }

  // Disable Run all when any of this chain's service steps are already up
  // externally — running docker compose up -d / fleet serve again would
  // either no-op or conflict. Build chain has no services so this is moot.
  const serviceUp = steps.some(
    (s) => s.service && (externalRunningByStepId?.[s.id] ?? false),
  );
  const runAllDisabled = running || serviceUp;
  const runAllLabel = running
    ? "running…"
    : serviceUp
      ? "already up"
      : "▶ Run all";

  return (
    <div
      className="card"
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 10,
        minWidth: 0,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 8,
        }}
      >
        <div style={{ minWidth: 0 }}>
          <div className="card-title">{title}</div>
          <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
            {subtitle}
          </div>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          {header}
          <button
            className="primary"
            onClick={runAll}
            disabled={runAllDisabled}
            title={
              serviceUp
                ? "Stop running services before re-running"
                : undefined
            }
            style={{ padding: "6px 14px" }}
          >
            {runAllLabel}
          </button>
        </div>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        {steps.map((s) => (
          <div key={s.id}>
            <StepRow
              step={s}
              proc={procs.find((p) => p.id === s.id)}
              onRun={() => runStep(s)}
              onStop={() => api.stopProcess(s.id)}
              now={now}
              externalRunning={externalRunningByStepId?.[s.id] ?? false}
              treatAsIdle={isStepStale(s.id)}
              actionsDisabled={running}
            />
            {s.kind === "serve" && fleetServePreview}
          </div>
        ))}
      </div>
      {error && (
        <div style={{ color: "var(--ui-error)", fontSize: "var(--fs-xx-small)" }}>
          {error}
        </div>
      )}
    </div>
  );
}

type StepDisplayState = "idle" | "running" | "done" | "failed";
const STALE_MS = 15 * 60 * 1000;

/// Computes how a chain step should appear *right now*, given its proc
/// state and (for long-running steps) the live health-probe result. This
/// is the single source of truth for the row's glyph, sub-message, and
/// staleness — keeps the render code below boring.
function deriveStepDisplay(
  step: ChainStep,
  proc: ProcInfo | undefined,
  externalRunning: boolean,
  treatAsIdle: boolean, // e.g. branch changed since last build
  now: number,
): {
  state: StepDisplayState;
  subMessage: string | null;
  isStale: boolean;
  ourProcRunning: boolean;
} {
  const procState = proc?.state ?? "idle";
  const ourProcRunning = procState === "running" || procState === "stopping";
  const procDone = procState === "done";
  const procFailed = procState === "failed";

  if (treatAsIdle) {
    return { state: "idle", subMessage: null, isStale: false, ourProcRunning };
  }

  // If the user explicitly stopped this process (via stop, stop all, or
  // docker compose down) treat the row as idle — not failed. Avoids the
  // "I clicked stop and now it's red" confusion.
  // ...unless it's a service we don't own that's currently up externally —
  // then the health probe (externalRunning) should win over the stale
  // user-stopped flag, so we don't show idle/▶ for something that's running.
  if (proc?.was_user_stopped && !ourProcRunning && !(step.service && externalRunning)) {
    return { state: "idle", subMessage: null, isStale: false, ourProcRunning };
  }

  if (step.service) {
    // Service step state comes from the health probe, NOT the spawn exit
    // code. The spawn might be alive while the service is still starting;
    // or the spawn might be done but the service running (docker -d). We
    // therefore check externalRunning below BEFORE falling back to idle, so
    // a service started outside Hangar isn't shown as idle (with a ▶ action).
    // Stopping wins over externalRunning: when the user just clicked stop,
    // docker may still report containers up for a beat while compose tears
    // them down — we want to show "stopping…", not ✓ running.
    if (procState === "stopping") {
      return {
        state: "running",
        subMessage: "stopping…",
        isStale: false,
        ourProcRunning,
      };
    }
    if (externalRunning) {
      return { state: "done", subMessage: null, isStale: false, ourProcRunning };
    }
    if (procFailed) {
      return {
        state: "failed",
        subMessage:
          proc?.exit_code != null
            ? `failed to start · exit ${proc.exit_code}`
            : "failed to start",
        isStale: false,
        ourProcRunning,
      };
    }
    // Spawn alive OR spawn done-with-success but probe hasn't confirmed up
    // yet. For docker compose up -d the spawn exits in a few hundred ms
    // while containers are still coming up, so procDone here is normal —
    // treat it as transitional, not a failure.
    if (ourProcRunning || procDone) {
      return {
        state: "running",
        subMessage: "starting…",
        isStale: false,
        ourProcRunning,
      };
    }
    return { state: "idle", subMessage: null, isStale: false, ourProcRunning };
  }

  // Short-running step — state is the proc state, basically.
  if (ourProcRunning) {
    return { state: "running", subMessage: null, isStale: false, ourProcRunning };
  }
  if (procFailed) {
    return { state: "failed", subMessage: null, isStale: false, ourProcRunning };
  }
  if (procDone) {
    const isStale =
      proc?.ended_at_ms != null && now - proc.ended_at_ms > STALE_MS;
    return { state: "done", subMessage: null, isStale, ourProcRunning };
  }
  return { state: "idle", subMessage: null, isStale: false, ourProcRunning };
}

function StepGlyph({
  state,
  service,
}: {
  state: StepDisplayState;
  service?: boolean;
}) {
  // Long-running service that's up gets the same pulsing dot as Active
  // Processes — checkmark is reserved for short-lived tasks that completed
  // (make deps, fleet prepare db, etc.).
  if (service && state === "done") {
    return (
      <span
        className="step-glyph done"
        style={{ display: "inline-flex", alignItems: "center", justifyContent: "center" }}
      >
        <span className="dot run" />
      </span>
    );
  }
  const ch =
    state === "running" ? "⏵" : state === "done" ? "✓" : state === "failed" ? "✗" : "○";
  return <span className={`step-glyph ${state}`}>{ch}</span>;
}

function StepRow({
  step,
  proc,
  onRun,
  onStop,
  now,
  externalRunning,
  treatAsIdle,
  actionsDisabled,
}: {
  step: ChainStep;
  proc?: ProcInfo;
  onRun: () => void;
  onStop: () => void;
  now: number;
  externalRunning?: boolean;
  treatAsIdle?: boolean;
  actionsDisabled?: boolean;
}) {
  const display = deriveStepDisplay(
    step,
    proc,
    externalRunning ?? false,
    treatAsIdle ?? false,
    now,
  );
  const isStopping = proc?.state === "stopping";
  const isRunningOrUp = display.state === "running" || display.state === "done";
  // Don't dim while running. Treat-as-idle is a separate visual story
  // (the glyph already reverts to ○).
  const failed = display.state === "failed";

  // Latest log line stays visible while running and after a failure (so
  // the user can see the error) — but for clean completions, we replace
  // the "Done in 0.13s." noise with a rough "finished N ago" stamp,
  // which actually tells the user something useful (build freshness).
  const showLastLine =
    !step.service &&
    !treatAsIdle &&
    (display.state === "running" || display.state === "failed");
  const lastLine = showLastLine
    ? (proc?.recent_log[proc.recent_log.length - 1] ?? "")
    : "";
  const finishedAgo =
    !step.service &&
    !treatAsIdle &&
    display.state === "done" &&
    proc?.ended_at_ms != null
      ? `finished ${humanRoughAgo(now - proc.ended_at_ms)}`
      : null;

  // Meta column priority: EXIT N → LONG-RUNNING → live elapsed.
  let metaText = "";
  let metaColor = "var(--app-text-dim)";
  if (failed && proc?.exit_code != null && !step.service) {
    metaText = `EXIT ${proc.exit_code}`;
    metaColor = "var(--ui-error)";
  } else if (step.service && isRunningOrUp) {
    metaText = "LONG-RUNNING";
    metaColor = "var(--app-text-dim)";
  } else if (
    display.ourProcRunning &&
    !step.service &&
    proc?.started_at_ms != null
  ) {
    metaText = formatChainElapsed(now - proc.started_at_ms);
    metaColor = "var(--core-fleet-green)";
  }

  // Col 2 content: long-running sub-message takes priority, then the
  // "finished N ago" stamp for completed steps, then the live log line.
  const col2Text = display.subMessage ?? finishedAgo ?? lastLine;
  const col2Color = display.subMessage
    ? display.state === "failed"
      ? "var(--ui-error)"
      : "var(--app-text-dim)"
    : "var(--app-text-dim)";

  const cls = `chain-row${display.isStale ? " is-stale" : ""}`;

  return (
    <div
      className={cls}
      style={{
        display: "grid",
        gridTemplateColumns: "minmax(140px, auto) minmax(0, 1fr) 100px 56px",
        alignItems: "center",
        columnGap: 10,
        padding: "6px 10px",
        minWidth: 0,
        background: failed
          ? "var(--tint-error-soft)"
          : "var(--app-surface-2)",
        border: failed ? "1px solid var(--ui-error)" : "1px solid transparent",
        borderRadius: "var(--radius-md)",
      }}
    >
      {/* col 1: glyph + label */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
          minWidth: 0,
        }}
      >
        <StepGlyph state={display.state} service={step.service} />
        <span
          className="mono"
          style={{ color: "var(--app-text)", whiteSpace: "nowrap" }}
        >
          {step.label}
        </span>
      </div>
      {/* col 2: sub-message OR latest log line, ellipsizes on overflow */}
      <span
        className="mono"
        style={{
          minWidth: 0,
          overflow: "hidden",
          textOverflow: "ellipsis",
          whiteSpace: "nowrap",
          fontSize: "var(--fs-xxx-small)",
          color: col2Color,
        }}
        title={col2Text || undefined}
      >
        {col2Text}
      </span>
      {/* col 3: meta */}
      <span
        className="mono"
        style={{
          fontSize: "var(--fs-xxx-small)",
          color: metaColor,
          fontVariantNumeric: "tabular-nums",
          textAlign: "right",
          textTransform:
            metaText === "LONG-RUNNING" || metaText.startsWith("EXIT")
              ? "uppercase"
              : "none",
          letterSpacing:
            metaText === "LONG-RUNNING" || metaText.startsWith("EXIT")
              ? "0.06em"
              : undefined,
        }}
      >
        {metaText}
      </span>
      {/* col 4: action */}
      <div
        style={{
          display: "flex",
          justifyContent: "flex-end",
          alignItems: "center",
        }}
      >
        {isRunningOrUp && step.hideStop ? (
          <span
            style={{
              fontSize: "var(--fs-xxx-small)",
              color: isStopping
                ? "var(--app-text-dim)"
                : "var(--core-fleet-green)",
              textTransform: "uppercase",
              letterSpacing: "0.06em",
            }}
          >
            {isStopping ? "stopping" : "running"}
          </span>
        ) : display.state === "running" && !step.hideStop ? (
          <button
            onClick={onStop}
            className="danger"
            disabled={isStopping}
            style={{ padding: "2px 8px", fontSize: "var(--fs-xx-small)" }}
          >
            {isStopping ? "…" : "■"}
          </button>
        ) : (
          <button
            onClick={onRun}
            disabled={actionsDisabled}
            style={{ padding: "2px 8px", fontSize: "var(--fs-xx-small)" }}
          >
            ▶
          </button>
        )}
      </div>
    </div>
  );
}

function formatChainElapsed(ms: number): string {
  const sec = Math.max(0, Math.floor(ms / 1000));
  if (sec < 60) return `${sec}s`;
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  if (m < 60) return `${m}:${String(s).padStart(2, "0")}`;
  const h = Math.floor(m / 60);
  return `${h}h ${m % 60}m`;
}

function ActiveProcessesPanel({
  server,
  settings,
  onSettingsChange,
  procs,
  serve,
  docker,
  now,
  goToLogs,
  goToSettings,
}: {
  server: ServerProfile;
  settings: Settings;
  onSettingsChange: (next: Settings) => void;
  procs: ProcInfo[];
  serve: ServeStatus;
  docker: DockerHealth;
  now: number;
  goToLogs: () => void;
  goToSettings: (section: SettingsSection) => void;
}) {
  const repoPath = server.worktree_path as string;
  const serveId = procId(server.id, "fleet-serve");
  const serveProc = procs.find((p) => p.id === serveId);
  const serveOwned = serveProc?.state === "running" || serveProc?.state === "stopping";
  const ngrokProc = procs.find((p) => p.id === "ngrok");
  const pythonProc = procs.find((p) => p.id === "python-server");
  const ngrokRunning =
    ngrokProc?.state === "running" || ngrokProc?.state === "stopping";
  const pythonRunning =
    pythonProc?.state === "running" || pythonProc?.state === "stopping";

  const [busy, setBusy] = useState<string | null>(null);
  const [ngrokInfo, setNgrokInfo] = useState<NgrokYamlInfo | null>(null);

  // Re-parse ngrok.yml whenever its configured path changes.
  useEffect(() => {
    let cancelled = false;
    api
      .parseNgrokYml(settings.ngrok.yml_path)
      .then((r) => {
        if (cancelled) return;
        setNgrokInfo(r);
        // Self-heal: if a selected tunnel no longer exists in the yml (renamed
        // or removed), drop it so `ngrok start` doesn't try to launch a
        // phantom tunnel. staleNgrokTunnels only fires on a valid parse.
        const stale = staleNgrokTunnels(settings, r);
        if (stale.length > 0) {
          const next: Settings = {
            ...settings,
            ngrok: {
              ...settings.ngrok,
              default_tunnels: settings.ngrok.default_tunnels.filter(
                (n) => !stale.includes(n),
              ),
            },
          };
          api
            .saveSettings(next)
            .then(() => onSettingsChange(next))
            .catch((e) =>
              console.error("failed to prune stale ngrok tunnels", e),
            );
        }
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [settings.ngrok.yml_path]);

  async function serveStop() {
    setBusy("serve-stop");
    try {
      await api.stopProcess(serveId);
    } catch {}
    setBusy(null);
  }
  async function dockerStop() {
    setBusy("docker-stop");
    try {
      await api.dockerComposeDown(
        procId(server.id, "docker-compose-up"),
        repoPath,
        server.compose_project,
      );
    } catch {}
    setBusy(null);
  }

  function ngrokArgs(): { args: string[]; preview: string } {
    const args = ngrokArgsFor(settings);
    return { args, preview: `ngrok ${args.join(" ")}` };
  }

  async function ngrokStart() {
    setBusy("ngrok-start");
    try {
      const { args } = ngrokArgs();
      await api.startProcess({
        id: "ngrok",
        label: "ngrok",
        cwd: repoPath,
        program: "ngrok",
        args,
      });
    } catch (e) {
      console.error("ngrok start failed", e);
    }
    setBusy(null);
  }
  async function ngrokStop() {
    setBusy("ngrok-stop");
    try {
      await api.stopProcess("ngrok");
    } catch {}
    setBusy(null);
  }

  function pythonArgs(): { args: string[]; preview: string } {
    const args = pythonArgsFor(settings);
    return {
      args,
      preview: `python3 ${args.join(" ")}`,
    };
  }

  async function pythonStart() {
    setBusy("python-start");
    try {
      const { args } = pythonArgs();
      await api.startProcess({
        id: "python-server",
        label: "python http.server",
        cwd: repoPath,
        program: "python3",
        args,
      });
    } catch (e) {
      console.error("python start failed", e);
    }
    setBusy(null);
  }
  async function pythonStop() {
    setBusy("python-stop");
    try {
      await api.stopProcess("python-server");
    } catch {}
    setBusy(null);
  }

  async function toggleNgrokTunnel(name: string) {
    if (settings.ngrok.start_all) return;
    const cur = settings.ngrok.default_tunnels;
    const next = cur.includes(name)
      ? cur.filter((n) => n !== name)
      : [...cur, name];
    const newSettings: Settings = {
      ...settings,
      ngrok: { ...settings.ngrok, default_tunnels: next },
    };
    try {
      await api.saveSettings(newSettings);
      onSettingsChange(newSettings);
    } catch (e) {
      console.error("failed to save ngrok tunnel selection", e);
    }
  }

  // Cells render when enabled OR currently running (so you can still stop
  // a process you just toggled off in settings).
  const showNgrok = settings.ngrok.enabled || ngrokRunning;
  const showPython = settings.python_server.enabled || pythonRunning;

  const runningCount =
    (serve.up ? 1 : 0) +
    (docker.up ? 1 : 0) +
    (ngrokRunning ? 1 : 0) +
    (pythonRunning ? 1 : 0);
  const configuredCount =
    (showNgrok && !ngrokRunning ? 1 : 0) +
    (showPython && !pythonRunning ? 1 : 0);

  return (
    <div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: "var(--pad-small)",
          gap: "var(--pad-medium)",
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          Active processes ·{" "}
          <span style={{ color: "var(--core-fleet-green)" }}>
            {runningCount} running
          </span>{" "}
          · {configuredCount} configured
        </div>
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "1fr 1fr",
          gap: "var(--pad-medium)",
        }}
      >
        <Cell>
          <ProcRow
            dotState={serve.up ? "run" : "idle"}
            name="fleet serve --dev"
            subline={
              serve.up
                ? `up · :${server.ports.server} · uptime ${formatUptime(now, serve.upSinceMs)}${serveOwned ? "" : " · external"}`
                : `down · nothing on :${server.ports.server}`
            }
            onLogs={goToLogs}
            onStop={serveOwned ? serveStop : undefined}
            busy={busy?.startsWith("serve-") ?? false}
            down={!serve.up}
          />
        </Cell>
        <Cell>
          <ProcRow
            dotState={docker.up ? "run" : "idle"}
            name="docker compose"
            subline={
              docker.up
                ? `running · uptime ${formatUptime(now, docker.upSinceMs)}`
                : "not running"
            }
            onStop={dockerStop}
            busy={busy?.startsWith("docker-") ?? false}
            down={!docker.up}
          />
        </Cell>
        {/* Always rendered, even when disabled, so users can see these
            optional services exist and jump to Settings to enable them. */}
        <NgrokCell
          running={ngrokRunning}
          info={ngrokInfo}
          settings={settings}
          preview={ngrokArgs().preview}
          busy={busy?.startsWith("ngrok-") ?? false}
          onStart={ngrokStart}
          onStop={ngrokStop}
          onSettings={() => goToSettings("ngrok")}
          onToggleTunnel={toggleNgrokTunnel}
        />
        <PythonCell
          running={pythonRunning}
          settings={settings}
          preview={pythonArgs().preview}
          busy={busy?.startsWith("python-") ?? false}
          onStart={pythonStart}
          onStop={pythonStop}
          onSettings={() => goToSettings("python")}
        />
      </div>
    </div>
  );
}

function Cell({ children }: { children: React.ReactNode }) {
  return (
    <div
      className="card"
      style={{ padding: 0, display: "flex", flexDirection: "column" }}
    >
      {children}
    </div>
  );
}

function StarterCell({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div
      style={{
        background: "var(--app-surface-2)",
        border: "1px dashed var(--app-border)",
        borderRadius: "var(--radius-lg)",
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 8,
      }}
    >
      {children}
    </div>
  );
}

/// Greyed-out placeholder for an optional service (ngrok / python) that
/// is turned off in Settings. Keeps the box visible on the Server tab so
/// users know the service exists, with a link to enable it. The header
/// is dimmed; the Settings link stays at full opacity so it reads as the
/// one actionable element.
function DisabledServiceCell({
  name,
  settingsLabel,
  onSettings,
}: {
  name: string;
  settingsLabel: string;
  onSettings: () => void;
}) {
  return (
    <StarterCell>
      <div style={{ opacity: 0.55, display: "flex", flexDirection: "column", gap: 8 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span className="dot idle" />
          <div className="card-title">
            {name}{" "}
            <span className="dim" style={{ fontWeight: 400 }}>
              · disabled
            </span>
          </div>
        </div>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          Optional service — enable it in Settings to run it from here.
        </div>
      </div>
      <div style={{ textAlign: "right" }}>
        <button
          onClick={onSettings}
          className="link-btn"
          style={{ fontSize: "var(--fs-xx-small)" }}
        >
          {settingsLabel}
        </button>
      </div>
    </StarterCell>
  );
}

function NgrokCell({
  running,
  info,
  settings,
  preview,
  busy,
  onStart,
  onStop,
  onSettings,
  onToggleTunnel,
}: {
  running: boolean;
  info: NgrokYamlInfo | null;
  settings: Settings;
  preview: string;
  busy: boolean;
  onStart: () => void;
  onStop: () => void;
  onSettings: () => void;
  onToggleTunnel: (name: string) => void;
}) {
  const cfg = settings.ngrok;
  if (running) {
    const tunnelLabel = cfg.start_all
      ? "all tunnels"
      : cfg.default_tunnels.length === 0
        ? ""
        : cfg.default_tunnels.join(", ");
    return (
      <Cell>
        <ProcRow
          dotState="run"
          name="ngrok"
          subline={tunnelLabel ? `running · ${tunnelLabel}` : "running"}
          onStop={onStop}
          busy={busy}
          down={false}
        />
      </Cell>
    );
  }

  if (!cfg.enabled) {
    return (
      <DisabledServiceCell
        name="ngrok"
        settingsLabel="Settings → ngrok ↗"
        onSettings={onSettings}
      />
    );
  }

  // Starter cell
  const hasConfig =
    cfg.start_all || cfg.default_tunnels.length > 0;
  const tunnels = info?.tunnels ?? [];
  return (
    <StarterCell>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span className="dot idle" />
          <div className="card-title">
            ngrok{" "}
            <span className="dim" style={{ fontWeight: 400 }}>
              · off
            </span>
          </div>
        </div>
        <button
          className="primary"
          onClick={onStart}
          disabled={busy || !hasConfig}
          style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          title={!hasConfig ? "Configure tunnels in Settings" : undefined}
        >
          {busy ? "…" : "▶ Start"}
        </button>
      </div>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        {info && info.valid
          ? `${tunnels.length} tunnel${tunnels.length === 1 ? "" : "s"} in ngrok.yml`
          : "ngrok.yml not found"}
        {hasConfig && (
          <>
            {" · will run: "}
            <span className="mono" style={{ color: "var(--app-text)" }}>
              {preview}
            </span>
          </>
        )}
      </div>
      {tunnels.length > 0 && (
        <div style={{ display: "flex", flexWrap: "wrap", gap: 6 }}>
          {tunnels.map((t) => {
            const selected = cfg.start_all || cfg.default_tunnels.includes(t.name);
            const lockedByStartAll = cfg.start_all;
            const cls = [
              "tunnel-chip",
              selected ? "is-selected" : "",
              lockedByStartAll ? "is-locked" : "",
            ]
              .filter(Boolean)
              .join(" ");
            return (
              <div
                key={t.name}
                role="button"
                tabIndex={lockedByStartAll ? -1 : 0}
                onClick={() => !lockedByStartAll && onToggleTunnel(t.name)}
                onKeyDown={(e) => {
                  if (lockedByStartAll) return;
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    onToggleTunnel(t.name);
                  }
                }}
                className={cls}
                title={
                  lockedByStartAll
                    ? `${t.name} · ${t.proto} · ${t.addr} · locked by start all`
                    : `Click to ${selected ? "exclude" : "include"} · ${t.proto} · ${t.addr}`
                }
              >
                <span className={`dot ${selected ? "ok" : "idle"}`} />
                <span className="mono">{t.name}</span>
                <span style={{ fontSize: "var(--fs-xxx-small)" }}>
                  :{t.addr}
                </span>
              </div>
            );
          })}
        </div>
      )}
      <div style={{ textAlign: "right" }}>
        <button
          onClick={onSettings}
          className="link-btn"
          style={{ fontSize: "var(--fs-xx-small)" }}
        >
          Settings → ngrok ↗
        </button>
      </div>
    </StarterCell>
  );
}

function PythonCell({
  running,
  settings,
  preview,
  busy,
  onStart,
  onStop,
  onSettings,
}: {
  running: boolean;
  settings: Settings;
  preview: string;
  busy: boolean;
  onStart: () => void;
  onStop: () => void;
  onSettings: () => void;
}) {
  const cfg = settings.python_server;
  if (running) {
    return (
      <Cell>
        <ProcRow
          dotState="run"
          name="python http.server"
          subline={`serving ${cfg.directory ?? "repo root"} on :${cfg.port}`}
          onStop={onStop}
          busy={busy}
          down={false}
        />
      </Cell>
    );
  }

  if (!cfg.enabled) {
    return (
      <DisabledServiceCell
        name="python http.server"
        settingsLabel="Settings → http.server ↗"
        onSettings={onSettings}
      />
    );
  }

  return (
    <StarterCell>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
          <span className="dot idle" />
          <div className="card-title">
            python http.server{" "}
            <span className="dim" style={{ fontWeight: 400 }}>
              · off
            </span>
          </div>
        </div>
        <button
          className="primary"
          onClick={onStart}
          disabled={busy}
          style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
        >
          {busy ? "…" : "▶ Start"}
        </button>
      </div>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        will run:{" "}
        <span className="mono" style={{ color: "var(--app-text)" }}>
          {preview}
        </span>
      </div>
      <div style={{ display: "flex", gap: 8, fontSize: "var(--fs-xx-small)" }}>
        <span className="dim">port</span>
        <span className="mono">{cfg.port}</span>
        <span className="dim">·</span>
        <span className="dim">dir</span>
        <span className="mono">{cfg.directory ?? "repo root"}</span>
      </div>
      <div style={{ textAlign: "right" }}>
        <button
          onClick={onSettings}
          className="link-btn"
          style={{ fontSize: "var(--fs-xx-small)" }}
        >
          Settings → http.server ↗
        </button>
      </div>
    </StarterCell>
  );
}

function ProcRow({
  dotState,
  name,
  subline,
  onLogs,
  onStop,
  busy,
  down,
}: {
  dotState: string;
  name: string;
  subline: string;
  onLogs?: () => void;
  onStop?: () => void;
  busy: boolean;
  down: boolean;
}) {
  return (
    <div
      style={{
        padding: "var(--pad-medium)",
        display: "flex",
        alignItems: "center",
        gap: "var(--pad-medium)",
        opacity: down ? 0.65 : 1,
      }}
    >
      <div
        style={{
          flex: 1,
          minWidth: 0,
          display: "flex",
          alignItems: "center",
          gap: 10,
        }}
      >
        <span className={`dot ${dotState}`} />
        <div style={{ minWidth: 0 }}>
          <div className="card-title">{name}</div>
          <div
            className="dim"
            style={{
              fontSize: "var(--fs-xx-small)",
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
          >
            {subline}
          </div>
        </div>
      </div>
      <div style={{ display: "flex", gap: 6, flexShrink: 0 }}>
        {onLogs && (
          <button
            onClick={onLogs}
            style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
          >
            Logs ↗
          </button>
        )}
        {onStop && !down && (
          <button
            onClick={onStop}
            disabled={busy}
            className="danger"
            style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
          >
            ■
          </button>
        )}
      </div>
    </div>
  );
}

function RecentPanel({
  procs,
  now,
}: {
  procs: ProcInfo[];
  now: number;
}) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const recent = procs
    .filter((p) => p.state === "done" || p.state === "failed")
    .sort((a, b) => (b.started_at_ms ?? 0) - (a.started_at_ms ?? 0))
    .slice(0, 3);

  return (
    <div>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          marginBottom: "var(--pad-small)",
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          Recent · last 3
        </div>
      </div>
      {recent.length === 0 ? (
        <div
          className="dim"
          style={{
            fontSize: "var(--fs-xx-small)",
            padding: "var(--pad-medium)",
            border: "1px dashed var(--app-border)",
            borderRadius: "var(--radius-md)",
            textAlign: "center",
          }}
        >
          No completed runs yet.
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
          {recent.map((p) => (
            <RecentRow
              key={p.id}
              proc={p}
              expanded={expanded === p.id}
              onToggleExpanded={() =>
                setExpanded(expanded === p.id ? null : p.id)
              }
              now={now}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function RecentRow({
  proc,
  expanded,
  onToggleExpanded,
  now,
}: {
  proc: ProcInfo;
  expanded: boolean;
  onToggleExpanded: () => void;
  now: number;
}) {
  const failed = proc.state === "failed";
  // What to show as the failure reason. Priority:
  //   1. Synthetic [exit: ...] line that the backend appends on exit —
  //      tells us "killed by signal 9 (SIGKILL)" / "exit code N" etc.
  //   2. A line containing error/fatal/panic.
  //   3. Last line as a fallback — but only if we have one AND it
  //      isn't a normal request-log line, because surfacing a 200 OK
  //      as "the error" reads as a UI bug.
  const errorLine = failed
    ? (() => {
        const tail = proc.recent_log;
        const synth = [...tail].reverse().find((l) => l.startsWith("[exit:"));
        if (synth) return synth;
        const realErr = [...tail]
          .reverse()
          .find((l) => /error|fatal|panic/i.test(l));
        if (realErr) return realErr;
        // Nothing diagnostic in the buffer — be honest about it rather
        // than misrepresenting the last request as an error.
        if (proc.exit_signal != null) {
          return `[killed by signal ${proc.exit_signal}]`;
        }
        if (proc.exit_code != null) {
          return `[exit code ${proc.exit_code}, no error in log tail]`;
        }
        return "(no error logged before exit — see View error for context)";
      })()
    : null;
  const ago = proc.started_at_ms
    ? humanAgo(now - proc.started_at_ms)
    : "—";

  return (
    <div
      style={{
        background: failed ? "var(--tint-error-soft)" : "var(--app-surface)",
        border: failed
          ? "1px solid var(--tint-danger-border)"
          : "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "10px 12px",
      }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 10,
        }}
      >
        <span
          style={{
            fontWeight: 600,
            color: failed ? "var(--ui-error)" : "var(--ui-success)",
            fontSize: "var(--fs-x-small)",
          }}
        >
          {failed ? "✗" : "✓"}
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              display: "flex",
              // center (not baseline) so the label sits in the middle
              // of a multi-line wrapped error message instead of at the
              // top of it.
              alignItems: "center",
              gap: 8,
              fontSize: "var(--fs-x-small)",
            }}
          >
            {/* Label refuses to shrink — otherwise a long error message
                squeezes it down to a single-word column and "fleet serve
                --dev" wraps to 3 lines. */}
            <span
              style={{
                fontWeight: 600,
                whiteSpace: "nowrap",
                flexShrink: 0,
              }}
            >
              {proc.label}
            </span>
            <span
              className="mono dim"
              style={{
                fontSize: "var(--fs-xx-small)",
                flex: 1,
                minWidth: 0,
                wordBreak: "break-word",
              }}
            >
              {failed && errorLine ? errorLine : proc.command}
            </span>
          </div>
        </div>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          {ago}
        </div>
        {failed && (
          <button
            onClick={onToggleExpanded}
            style={{ padding: "2px 8px", fontSize: "var(--fs-xx-small)" }}
          >
            {expanded ? "Hide" : "View error"}
          </button>
        )}
      </div>
      {expanded && failed && proc.recent_log.length > 0 && (
        <pre
          style={{
            margin: "10px 0 0 0",
            padding: "10px 12px",
            background: "var(--log-bg)",
            color: "var(--app-text-dim)",
            fontFamily: "var(--font-mono)",
            fontSize: "var(--fs-xx-small)",
            borderRadius: "var(--radius-sm)",
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            maxHeight: 240,
            overflow: "auto",
          }}
        >
          {proc.recent_log.slice(-30).join("\n")}
        </pre>
      )}
    </div>
  );
}

function formatUptime(now: number, since: number | null): string {
  if (since == null) return "—";
  const sec = Math.max(0, Math.floor((now - since) / 1000));
  if (sec < 60) return `${sec}s`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m`;
  return `${Math.floor(sec / 3600)}h ${Math.floor((sec % 3600) / 60)}m`;
}

function humanAgo(ms: number): string {
  const sec = Math.floor(ms / 1000);
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s ago`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  return `${Math.floor(sec / 3600)}h ago`;
}

// Coarse "when did this happen" buckets for the chain step "finished N
// ago" stamp. Within the hour we use `~` (rough midpoint — friendlier,
// and minute-level precision isn't worth claiming). Past the hour we
// switch to `>` because "more than 1h" carries useful staleness signal
// that "~1h" would blur.
function humanRoughAgo(ms: number): string {
  const sec = Math.floor(ms / 1000);
  if (sec < 60) return "just now";
  const min = Math.floor(sec / 60);
  if (min < 5) return "~1m ago";
  if (min < 15) return "~5m ago";
  if (min < 30) return "~15m ago";
  if (min < 60) return "~30m ago";
  const hr = Math.floor(min / 60);
  return `>${hr}h ago`;
}
