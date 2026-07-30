// Start/Stop-all logic for the Fleet dev environment. Extracted from
// MasterControl so the tray menu can drive the same flows without
// duplicating chain ordering / skip-if-running rules.

import { listen } from "./events";
import { api, type ProcEvent, type Settings } from "./tauri";

export type BuildStep = {
  id: string;
  label: string;
  program: string;
  args: string[];
};

export const BUILD_CHAIN: BuildStep[] = [
  { id: "make-deps", label: "make deps", program: "make", args: ["deps"] },
  {
    id: "make-generate",
    label: "make generate",
    program: "make",
    args: ["generate"],
  },
  { id: "make-build", label: "make build", program: "make", args: ["build"] },
];

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

/// Resolves the current `fleet serve` argv based on settings. Order is
/// `serve --dev` first (fixed), then config, then license, then debug
/// flags — what we shipped before, just with each piece individually
/// optional. Used for both the spawn and the preview line so they can
/// never disagree.
export function serveArgsFor(settings: Settings): string[] {
  const cfg = settings.fleet_serve;
  const args = ["serve", "--dev"];
  const configPath = cfg.config_path?.trim();
  if (configPath) {
    args.push("--config", configPath);
  }
  if (cfg.premium) args.push("--dev_license");
  if (cfg.debug) args.push("--debug");
  if (cfg.logging_debug) args.push("--logging_debug");
  return args;
}

/// Tuples in [key, value] shape so it can go straight into
/// `api.startProcess({env})`. Empty-key rows are skipped (they're the
/// "draft row" state in the Settings editor); disabled rows are
/// skipped too so the toggle behaves the same as removing the row.
export function serveEnvFor(settings: Settings): Array<[string, string]> {
  return settings.fleet_serve.env
    .map((e) => ({ ...e, key: e.key.trim() }))
    .filter((e) => e.enabled && e.key.length > 0)
    .map((e) => [e.key, e.value] as [string, string]);
}

export function ngrokIsLaunchable(settings: Settings): boolean {
  const cfg = settings.ngrok;
  return cfg.enabled && (cfg.start_all || cfg.default_tunnels.length > 0);
}

/// Single-spawn helper: kicks off `fleet serve --dev` without the
/// docker / prepare-db prerequisites that startAll handles. Suitable
/// when the user just wants to restart serve in a state where the rest
/// of the stack is already up (e.g. after stopping serve from the
/// Database tab to run a restore). If docker is down the spawn will
/// fail at db connect time — the error lands in the Logs tab.
export async function startServe(
  repoPath: string,
  settings: Settings,
): Promise<void> {
  const args = serveArgsFor(settings);
  const env = serveEnvFor(settings);
  await api.startProcess({
    id: "fleet-serve",
    label: serveLabelFor(settings),
    cwd: repoPath,
    program: "./build/fleet",
    args,
    log_channel: "fleet-serve",
    env: env.length > 0 ? env : null,
  });
}

/// Label shown in the chain row and Active processes panel. Reflects
/// the premium/free state so the user can see at a glance which build
/// they're running — `--dev_license` is the only flag that
/// meaningfully changes the server's behavior.
export function serveLabelFor(settings: Settings): string {
  return settings.fleet_serve.premium
    ? "fleet serve --dev (premium)"
    : "fleet serve --dev (free)";
}

export type SystemHealth = {
  serveUp: boolean;
  dockerUp: boolean;
  ngrokRunning: boolean;
  pythonRunning: boolean;
};

export type StartAllArgs = {
  repoPath: string;
  settings: Settings;
  health: SystemHealth;
  onBuildStepRun?: (stepId: string) => void;
};

export async function startAll(args: StartAllArgs): Promise<void> {
  const { repoPath, settings, health, onBuildStepRun } = args;

  // 1. Build chain — always runs, sequentially.
  for (const step of BUILD_CHAIN) {
    onBuildStepRun?.(step.id);
    await api.startProcess({
      id: step.id,
      label: step.label,
      cwd: repoPath,
      program: step.program,
      args: step.args,
    });
    const ok = await waitForExit(step.id);
    if (!ok) return;
  }

  // 2. Run chain with skip-if-running.
  const dockerWasDown = !health.dockerUp;
  if (dockerWasDown) {
    const ok = await dockerUpWithStaleCleanup(repoPath);
    if (!ok) return;
  }
  // Re-prepare db only on a cold boot.
  if (dockerWasDown) {
    await api.startProcess({
      id: "fleet-prepare-db",
      label: "fleet prepare db --dev",
      cwd: repoPath,
      program: "./build/fleet",
      args: ["prepare", "db", "--dev"],
    });
    const ok = await waitForExit("fleet-prepare-db");
    if (!ok) return;
  }
  if (!health.serveUp) {
    await startServe(repoPath, settings);
    // long-running — don't wait
  }
  // 3. ngrok — skip if disabled, already running, or no tunnels picked.
  if (
    settings.ngrok.enabled &&
    !health.ngrokRunning &&
    ngrokIsLaunchable(settings)
  ) {
    await api.startProcess({
      id: "ngrok",
      label: "ngrok",
      cwd: repoPath,
      program: "ngrok",
      args: ngrokArgsFor(settings),
    });
  }
  // 4. python — skip if disabled or already running.
  if (settings.python_server.enabled && !health.pythonRunning) {
    await api.startProcess({
      id: "python-server",
      label: "python http.server",
      cwd: repoPath,
      program: "python3",
      args: pythonArgsFor(settings),
    });
  }
}

export type StopAllArgs = {
  repoPath: string;
  health: SystemHealth;
};

export async function stopAll(args: StopAllArgs): Promise<void> {
  const { repoPath, health } = args;
  // Stop in reverse dependency order.
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
      await api.stopProcess("fleet-serve");
    } catch {}
  }
  if (health.dockerUp) {
    try {
      await api.dockerComposeDown(repoPath);
    } catch {}
  }
}

/// Cold-start docker. Runs `docker compose down` first to clear any
/// stale state from a previous session — `docker compose ps` only
/// reports running containers, so our health check thinks docker is
/// down while old exited container shells still claim the names and
/// would conflict on `up -d`. On a truly cold start, the down is a
/// fast no-op. The trade-off: ~0.5s extra latency for a deterministic,
/// flicker-free start path.
export async function dockerUpWithStaleCleanup(
  repoPath: string,
): Promise<boolean> {
  try {
    await api.dockerComposeDown(repoPath);
  } catch {
    // If down itself fails (daemon offline, permissions), let `up -d`
    // surface the real error in the chain row.
  }
  await api.startProcess({
    id: "docker-compose-up",
    label: "docker compose up -d",
    cwd: repoPath,
    program: "docker",
    args: ["compose", "up", "-d"],
  });
  return waitForExit("docker-compose-up");
}

/// Resolves true on the matching proc:state "done", false on "failed",
/// and false on timeout. Subscribes to the backend event stream rather
/// than polling — saves an IPC roundtrip per 400ms per concurrent
/// chain step. Default timeout is 30 minutes to cover slow `make deps`
/// / vulnerability scans; callers that know they're shorter can pass
/// their own.
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
      // The process may have exited between startProcess returning and
      // the listener being attached. Check current state and short-
      // circuit if we already missed the event.
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
