// Start/Stop-all logic for the Fleet dev environment, scoped to one server.
// Extracted from MasterControl so the tray menu can drive the same flows
// without duplicating chain ordering / skip-if-running rules.
//
// Per-server processes (build chain, docker, prepare-db, serve) are namespaced
// `<serverID>:<base>`; ngrok / python remain global (one each, driven by the
// top-level settings).

import { listen } from "./events";
import { api, type ProcEvent, type ServerProfile, type Settings } from "./ipc";
import {
  dockerEnvFor,
  dockerUpArgs,
  prepareDbArgsFor,
  procId,
  serveArgsFor,
  serveChannel,
  serveEnvFor,
  serveLabelFor,
} from "./servers";

// Re-export the serve helpers so existing call sites (ServerTab) can keep
// importing them from here.
export { serveArgsFor, serveEnvFor, serveLabelFor };

export type BuildStep = {
  id: string;
  label: string;
  program: string;
  args: string[];
};

// Base build steps (server-agnostic); buildChainFor() namespaces the ids.
const BUILD_STEP_BASES: { base: string; label: string; program: string; args: string[] }[] = [
  { base: "make-deps", label: "make deps", program: "make", args: ["deps"] },
  { base: "make-generate", label: "make generate", program: "make", args: ["generate"] },
  { base: "make-build", label: "make build", program: "make", args: ["build"] },
];

/// The build chain for a server, with process ids namespaced to that server so
/// two servers' builds never collide.
export function buildChainFor(server: ServerProfile): BuildStep[] {
  return BUILD_STEP_BASES.map((s) => ({
    id: procId(server.id, s.base),
    label: s.label,
    program: s.program,
    args: s.args,
  }));
}

export function ngrokArgsFor(settings: Settings): string[] {
  const cfg = settings.ngrok;
  const baseArgs = cfg.yml_path ? ["--config", cfg.yml_path] : [];
  if (cfg.start_all) return ["start", ...baseArgs, "--all"];
  return ["start", ...baseArgs, ...cfg.default_tunnels];
}

export function pythonArgsFor(settings: Settings): string[] {
  const cfg = settings.python_server;
  return [
    "-m",
    "http.server",
    String(cfg.port),
    "--directory",
    cfg.directory || ".",
  ];
}

export function ngrokIsLaunchable(settings: Settings): boolean {
  const cfg = settings.ngrok;
  return cfg.enabled && (cfg.start_all || cfg.default_tunnels.length > 0);
}

/// Single-spawn helper: kicks off `fleet serve --dev` for one server without
/// the docker / prepare-db prerequisites that startAll handles. Suitable when
/// the rest of the stack is already up (e.g. restarting serve after a restore).
export async function startServe(server: ServerProfile): Promise<void> {
  if (!server.worktree_path) throw new Error("server has no worktree configured");
  await api.startProcess({
    id: procId(server.id, "fleet-serve"),
    label: serveLabelFor(server),
    cwd: server.worktree_path,
    program: "./build/fleet",
    args: serveArgsFor(server),
    log_channel: serveChannel(server.id),
    env: serveEnvFor(server).length > 0 ? serveEnvFor(server) : null,
  });
}

export type SystemHealth = {
  serveUp: boolean;
  dockerUp: boolean;
  ngrokRunning: boolean;
  pythonRunning: boolean;
};

export type StartAllArgs = {
  server: ServerProfile;
  settings: Settings;
  health: SystemHealth;
  onBuildStepRun?: (stepId: string) => void;
};

export async function startAll(args: StartAllArgs): Promise<void> {
  const { server, settings, health, onBuildStepRun } = args;
  if (!server.worktree_path) return;
  const cwd = server.worktree_path;

  // 1. Build chain — always runs, sequentially.
  for (const step of buildChainFor(server)) {
    onBuildStepRun?.(step.id);
    await api.startProcess({
      id: step.id,
      label: step.label,
      cwd,
      program: step.program,
      args: step.args,
    });
    const ok = await waitForExit(step.id);
    if (!ok) return;
  }

  // 2. Run chain with skip-if-running.
  const dockerWasDown = !health.dockerUp;
  if (dockerWasDown) {
    const ok = await dockerUpWithStaleCleanup(server);
    if (!ok) return;
  }
  // Re-prepare db only on a cold boot.
  if (dockerWasDown) {
    await api.startProcess({
      id: procId(server.id, "fleet-prepare-db"),
      label: "fleet prepare db --dev",
      cwd,
      program: "./build/fleet",
      args: prepareDbArgsFor(server),
    });
    const ok = await waitForExit(procId(server.id, "fleet-prepare-db"));
    if (!ok) return;
  }
  if (!health.serveUp) {
    await startServe(server);
    // long-running — don't wait
  }
  // 3. ngrok — global; skip if disabled, already running, or no tunnels picked.
  if (
    settings.ngrok.enabled &&
    !health.ngrokRunning &&
    ngrokIsLaunchable(settings)
  ) {
    await api.startProcess({
      id: "ngrok",
      label: "ngrok",
      cwd,
      program: "ngrok",
      args: ngrokArgsFor(settings),
    });
  }
  // 4. python — global; skip if disabled or already running.
  if (settings.python_server.enabled && !health.pythonRunning) {
    await api.startProcess({
      id: "python-server",
      label: "python http.server",
      cwd,
      program: "python3",
      args: pythonArgsFor(settings),
    });
  }
}

export type StopAllArgs = {
  server: ServerProfile;
  health: SystemHealth;
};

export async function stopAll(args: StopAllArgs): Promise<void> {
  const { server, health } = args;
  // Stop in reverse dependency order. ngrok / python are global.
  if (health.pythonRunning) {
    try {
      await api.stopProcess("python-server");
    } catch {}
  }
  if (health.ngrokRunning) {
    try {
      await api.stopProcess("ngrok");
    } catch {}
  }
  if (health.serveUp) {
    try {
      await api.stopProcess(procId(server.id, "fleet-serve"));
    } catch {}
  }
  if (health.dockerUp && server.worktree_path) {
    try {
      await api.dockerComposeDown(
        procId(server.id, "docker-compose-up"),
        server.worktree_path,
        server.compose_project,
      );
    } catch {}
  }
}

/// Cold-start docker for a server. Runs `docker compose -p <project> down`
/// first to clear stale state from a previous session (compose ps only reports
/// running containers, so a leftover exited shell would claim names and
/// conflict on `up -d`). On a truly cold start the down is a fast no-op.
export async function dockerUpWithStaleCleanup(
  server: ServerProfile,
): Promise<boolean> {
  if (!server.worktree_path) return false;
  const cwd = server.worktree_path;
  const upId = procId(server.id, "docker-compose-up");
  try {
    await api.dockerComposeDown(upId, cwd, server.compose_project);
  } catch {
    // If down itself fails, let `up -d` surface the real error in the row.
  }
  await api.startProcess({
    id: upId,
    label: "docker compose up -d",
    cwd,
    program: "docker",
    args: dockerUpArgs(server),
    env: dockerEnvFor(server),
  });
  return waitForExit(upId);
}

/// Resolves true on the matching proc:state "done", false on "failed" or
/// timeout. Subscribes to the backend event stream rather than polling.
export function waitForExit(
  id: string,
  timeoutMs = 30 * 60_000,
): Promise<boolean> {
  return new Promise((resolve) => {
    let unlisten: (() => void) | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let settled = false;
    const finish = (ok: boolean) => {
      if (settled) return;
      settled = true;
      if (timer) clearTimeout(timer);
      unlisten?.();
      resolve(ok);
    };
    timer = setTimeout(() => finish(false), timeoutMs);

    listen<ProcEvent>("proc:state", (e) => {
      if (e.payload.proc_id !== id) return;
      if (e.payload.state === "done") finish(true);
      else if (e.payload.state === "failed") finish(false);
    }).then((u) => {
      if (settled) {
        u();
        return;
      }
      unlisten = u;
      // The process may have exited between startProcess returning and the
      // listener attaching. Check current state and short-circuit if we
      // already missed the event.
      api
        .listProcesses()
        .then((list) => {
          const p = list.find((x) => x.id === id);
          if (!p) return;
          if (p.state === "done") finish(true);
          else if (p.state === "failed") finish(false);
        })
        .catch(() => {
          // If listProcesses fails, fall through to the listener path.
        });
    });
  });
}
