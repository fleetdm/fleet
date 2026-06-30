import { useCallback, useEffect, useRef, useState } from "react";
import { listen } from "./lib/events";
import { TabBar, type TabId } from "./components/TabBar";
import { StatusRail } from "./components/StatusRail";
import { FirstRunGate } from "./components/FirstRunGate";
import { DatabaseTab } from "./components/tabs/DatabaseTab";
import { FleetctlTab } from "./components/tabs/FleetctlTab";
import { GitTab } from "./components/tabs/GitTab";
import { ServerTab } from "./components/tabs/ServerTab";
import { LogsTab } from "./components/tabs/LogsTab";
import {
  SettingsTab,
  type SettingsSection,
} from "./components/tabs/SettingsTab";
import { OsqueryPerfTab } from "./components/tabs/OsqueryPerfTab";
import { GitopsTab } from "./components/tabs/GitopsTab";
import {
  api,
  type BranchStatus,
  type LogLine,
  type ProcEvent,
  type ProcInfo,
  type Settings,
} from "./lib/tauri";
import { useSystemHealth } from "./lib/useSystemHealth";
import { useApplyTheme } from "./lib/useTheme";
import { startAll, stopAll } from "./lib/orchestration";

export default function App() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [active, setActive] = useState<TabId>("git");
  const [settingsSection, setSettingsSection] =
    useState<SettingsSection>("paths");
  const [branchStatus, setBranchStatus] = useState<BranchStatus | null>(null);
  const [procs, setProcs] = useState<ProcInfo[]>([]);
  const procPoll = useRef<number | null>(null);
  // No need to probe serve / docker / processes until the user is past
  // the first-run gate — nothing can be running on the welcome screen.
  const monitoringEnabled = settings?.first_run_complete ?? false;
  const { serve, docker } = useSystemHealth(
    settings?.repo_path ?? null,
    procs,
    monitoringEnabled,
  );
  // Quit flow phases: idle = no modal, confirm = "stop everything and
  // quit?" prompt, stopping = backend is shutting down. Triggered by
  // tray > Quit emitting "app:quit-requested", or by any window close
  // path (X button / Cmd+W / Cmd+Q) when services are running.
  const [quitPhase, setQuitPhase] = useState<"idle" | "confirm" | "stopping">(
    "idle",
  );

  useEffect(() => {
    api.getSettings().then(setSettings);
  }, []);

  useApplyTheme(settings?.theme);

  const refreshBranchStatus = useCallback(async () => {
    if (!settings?.repo_path) {
      setBranchStatus(null);
      return;
    }
    try {
      const s = await api.gitBranchStatus(settings.repo_path);
      setBranchStatus(s);
    } catch {
      setBranchStatus(null);
    }
  }, [settings?.repo_path]);

  useEffect(() => {
    refreshBranchStatus();
  }, [refreshBranchStatus]);

  useEffect(() => {
    if (!monitoringEnabled) return;
    let cancelled = false;
    const unlistens: Array<() => void> = [];

    const refresh = async () => {
      try {
        const list = await api.listProcesses();
        if (!cancelled) setProcs(list);
      } catch (e) {
        console.error("listProcesses failed", e);
      }
    };
    refresh();

    // listen() is async — the unlisten function isn't available until
    // the promise resolves. If cleanup runs before that, we'd leak the
    // listener; gate registration on `cancelled` so a late resolve
    // immediately tears down.
    const register = async <T,>(
      event: string,
      handler: (e: { payload: T }) => void,
    ) => {
      const u = await listen<T>(event, handler);
      if (cancelled) u();
      else unlistens.push(u);
    };

    register<ProcEvent>("proc:state", () => {
      refresh();
    });
    // We don't keep a per-line frontend copy of recent_log — the backend
    // already maintains one and exposes it via listProcesses(), and the
    // 4s poll plus proc:state-driven refresh keeps it fresh. Spreading
    // a new procs array on every log line caused App-wide re-renders
    // under load (fleet serve emits many lines/sec at --logging_debug).
    register<LogLine>("proc:log", (e) => {
      const { proc_id, stream, line } = e.payload;
      const tag = `[${proc_id}/${stream}]`;
      if (stream === "stderr") console.warn(tag, line);
      else console.log(tag, line);
    });

    procPoll.current = window.setInterval(refresh, 4000);

    return () => {
      cancelled = true;
      unlistens.forEach((u) => u());
      if (procPoll.current) window.clearInterval(procPoll.current);
    };
  }, [monitoringEnabled]);

  // Compute the slices of state the tray menu cares about. Re-runs cheaply
  // on each render, but updateTray itself only fires when something
  // changed (see effect below).
  const ngrokRunning = procs.some(
    (p) => p.id === "ngrok" && (p.state === "running" || p.state === "stopping"),
  );
  const pythonRunning = procs.some(
    (p) =>
      p.id === "python-server" &&
      (p.state === "running" || p.state === "stopping"),
  );
  const trayState = {
    branch: branchStatus?.branch ?? null,
    serve_up: serve.up,
    docker_up: docker.up,
    ngrok_running: ngrokRunning,
    python_running: pythonRunning,
  };
  const traySig = JSON.stringify(trayState);

  useEffect(() => {
    api.updateTray(trayState).catch(() => {
      // tray may not be ready immediately at startup
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [traySig]);

  // Tray menu callbacks: start-all / stop-all. The tray is unaware of
  // orchestration — it just emits events, and we route them here to the
  // same code paths the Server tab uses.
  useEffect(() => {
    let cancelled = false;
    const unlistens: Array<() => void> = [];
    const register = async (event: string, handler: () => Promise<void>) => {
      const u = await listen(event, handler);
      if (cancelled) u();
      else unlistens.push(u);
    };
    register("tray:start-all", async () => {
      if (!settings?.repo_path) return;
      try {
        await startAll({
          repoPath: settings.repo_path,
          settings,
          health: {
            serveUp: serve.up,
            dockerUp: docker.up,
            ngrokRunning,
            pythonRunning,
          },
        });
      } catch (e) {
        console.error("tray:start-all failed", e);
      }
    });
    register("tray:stop-all", async () => {
      if (!settings?.repo_path) return;
      try {
        await stopAll({
          repoPath: settings.repo_path,
          health: {
            serveUp: serve.up,
            dockerUp: docker.up,
            ngrokRunning,
            pythonRunning,
          },
        });
      } catch (e) {
        console.error("tray:stop-all failed", e);
      }
    });

    return () => {
      cancelled = true;
      unlistens.forEach((u) => u());
    };
  }, [settings, serve.up, docker.up, ngrokRunning, pythonRunning]);

  // Listen for tray Quit / Cmd+Q / dock > Quit. Separate effect with
  // empty deps so the listener is registered once and never churns —
  // previously this re-registered on every state change, which created
  // brief windows with no listener and let click events get dropped.
  // We read latest running-state via refs so the closure doesn't see
  // stale values.
  const runningRef = useRef({
    serveUp: false,
    dockerUp: false,
    ngrokRunning: false,
    pythonRunning: false,
  });
  const repoPathRef = useRef<string | null>(null);
  runningRef.current = {
    serveUp: serve.up,
    dockerUp: docker.up,
    ngrokRunning,
    pythonRunning,
  };
  repoPathRef.current = settings?.repo_path ?? null;
  useEffect(() => {
    let cancelled = false;
    let unlisten: (() => void) | undefined;
    listen("app:quit-requested", () => {
      const r = runningRef.current;
      const anyRunning =
        r.serveUp || r.dockerUp || r.ngrokRunning || r.pythonRunning;
      if (anyRunning) {
        setQuitPhase("confirm");
      } else {
        // Nothing to clean up — skip the modal and exit straight away.
        api.shutdownNow(repoPathRef.current).catch((e) => {
          console.error("shutdown_now failed", e);
        });
      }
    }).then((u) => {
      if (cancelled) u();
      else unlisten = u;
    });
    return () => {
      cancelled = true;
      unlisten?.();
    };
  }, []);

  const confirmQuit = useCallback(async () => {
    setQuitPhase("stopping");
    // shutdown_now does its own SIGTERM → SIGKILL escalation, runs
    // docker compose down, then calls app.exit(0) — so this promise
    // never actually resolves on the happy path; the app dies first.
    try {
      await api.shutdownNow(settings?.repo_path ?? null);
    } catch (e) {
      console.error("shutdown_now failed", e);
      setQuitPhase("idle");
    }
  }, [settings]);

  const cancelQuit = useCallback(() => setQuitPhase("idle"), []);

  const goToLogs = useCallback(() => {
    setActive("logs");
  }, []);

  const goToSettings = useCallback((section: SettingsSection) => {
    setSettingsSection(section);
    setActive("settings");
  }, []);

  if (!settings) {
    return null;
  }

  if (!settings.first_run_complete) {
    return <FirstRunGate onComplete={setSettings} />;
  }

  return (
    <div
      style={{
        height: "100vh",
        display: "flex",
        flexDirection: "column",
        background: "var(--app-bg)",
      }}
    >
      <TabBar active={active} onChange={setActive} />
      <main style={{ flex: 1, minHeight: 0, overflow: "hidden" }}>
        {active === "server" && (
          <ServerTab
            repoPath={settings.repo_path}
            settings={settings}
            onSettingsChange={setSettings}
            procs={procs}
            currentBranch={branchStatus?.branch ?? null}
            serve={serve}
            docker={docker}
            goToLogs={goToLogs}
            goToSettings={goToSettings}
          />
        )}
        {active === "git" && (
          <GitTab
            repoPath={settings.repo_path}
            branchStatus={branchStatus}
            refreshBranchStatus={refreshBranchStatus}
          />
        )}
        {active === "settings" && (
          <SettingsTab
            settings={settings}
            onChange={(s) => {
              // Only re-probe git when the repo actually changed —
              // otherwise every per-keystroke save (e.g. python port,
              // ngrok flags) was firing a git command.
              const repoChanged = s.repo_path !== settings.repo_path;
              setSettings(s);
              if (repoChanged) refreshBranchStatus();
            }}
            section={settingsSection}
            onSectionChange={setSettingsSection}
          />
        )}
        {active === "logs" && <LogsTab />}
        {active === "database" && (
          <DatabaseTab
            repoPath={settings.repo_path}
            settings={settings}
            currentBranch={branchStatus?.branch ?? null}
            procs={procs}
            serve={serve}
            docker={docker}
            goToLogs={goToLogs}
          />
        )}
        {active === "fleetctl" && (
          <FleetctlTab
            settings={settings}
            onSettingsChange={setSettings}
            serve={serve}
            goToSettings={() => goToSettings("fleetctl")}
            goToServer={() => setActive("server")}
            goToLogs={goToLogs}
          />
        )}
        {active === "gitops" && (
          <GitopsTab
            settings={settings}
            goToSettings={() => goToSettings("gitops")}
          />
        )}
        {active === "osquery-perf" && (
          <OsqueryPerfTab settings={settings} procs={procs} />
        )}
      </main>
      <StatusRail
        branchStatus={branchStatus}
        procs={procs}
        dockerUp={docker.up}
      />
      {quitPhase !== "idle" && (
        <QuitModal
          phase={quitPhase}
          onCancel={cancelQuit}
          onConfirm={confirmQuit}
        />
      )}
    </div>
  );
}

function QuitModal({
  phase,
  onCancel,
  onConfirm,
}: {
  phase: "confirm" | "stopping";
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      style={{
        position: "fixed",
        inset: 0,
        background: "var(--overlay-modal)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1100,
      }}
    >
      <div
        className="card"
        style={{
          maxWidth: 440,
          padding: "var(--pad-large)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-medium)",
        }}
      >
        {phase === "confirm" ? (
          <>
            <div style={{ fontSize: "var(--fs-medium)", fontWeight: 600 }}>
              Stop everything and quit?
            </div>
            <div
              className="dim"
              style={{ fontSize: "var(--fs-x-small)", lineHeight: 1.5 }}
            >
              This will stop fleet serve, ngrok, the python server, and run{" "}
              <span className="mono">docker compose down</span> before closing
              the app.
            </div>
            <div
              style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}
            >
              <button onClick={onCancel} style={{ padding: "6px 14px" }}>
                Cancel
              </button>
              <button
                onClick={onConfirm}
                className="danger"
                style={{ padding: "6px 14px" }}
              >
                Stop everything and quit
              </button>
            </div>
          </>
        ) : (
          <div
            style={{
              display: "flex",
              flexDirection: "column",
              gap: "var(--pad-small)",
              alignItems: "center",
              textAlign: "center",
            }}
          >
            <div style={{ fontSize: "var(--fs-medium)", fontWeight: 600 }}>
              Stopping everything…
            </div>
            <div
              className="dim"
              style={{ fontSize: "var(--fs-x-small)", lineHeight: 1.5 }}
            >
              Shutting down services and tearing down docker compose. This
              usually takes a few seconds.
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

