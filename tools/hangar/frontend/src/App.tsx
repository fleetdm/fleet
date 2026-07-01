import { useCallback, useEffect, useRef, useState } from "react";
import { listen } from "./lib/events";
import { TabBar, type TabId } from "./components/TabBar";
import { StatusRail } from "./components/StatusRail";
import { ServerSwitcher } from "./components/ServerSwitcher";
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
  type ComposeTarget,
  type LogLine,
  type ProcEvent,
  type ProcInfo,
  type Settings,
} from "./lib/ipc";
import { activeServer } from "./lib/servers";
import {
  useMultiServerHealth,
  type DockerHealth,
  type ServeStatus,
} from "./lib/useSystemHealth";
import { useApplyTheme } from "./lib/useTheme";
import { startAll, stopAll } from "./lib/orchestration";

const SERVE_DOWN: ServeStatus = { up: false, upSinceMs: null };
const DOCKER_DOWN: DockerHealth = { up: false, upSinceMs: null, containers: [] };

export default function App() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [active, setActive] = useState<TabId>("git");
  const [settingsSection, setSettingsSection] =
    useState<SettingsSection>("servers");
  const [branchStatus, setBranchStatus] = useState<BranchStatus | null>(null);
  const [procs, setProcs] = useState<ProcInfo[]>([]);
  const procPoll = useRef<number | null>(null);

  // The active server profile drives which worktree / ports / serve config the
  // Server, Logs, Database, and Git tabs operate on.
  const activeSrv = settings ? activeServer(settings) : null;
  const repoPath = activeSrv?.worktree_path ?? null;

  // No need to probe serve / docker / processes until the user is past
  // the first-run gate — nothing can be running on the welcome screen.
  const monitoringEnabled = settings?.first_run_complete ?? false;
  const healthMap = useMultiServerHealth(
    settings?.servers ?? [],
    procs,
    monitoringEnabled,
  );
  const activeHealth = activeSrv ? healthMap[activeSrv.id] : undefined;
  const serve = activeHealth?.serve ?? SERVE_DOWN;
  const docker = activeHealth?.docker ?? DOCKER_DOWN;

  // Quit flow phases: idle = no modal, confirm = "stop everything and
  // quit?" prompt, stopping = backend is shutting down.
  const [quitPhase, setQuitPhase] = useState<"idle" | "confirm" | "stopping">(
    "idle",
  );

  useEffect(() => {
    api.getSettings().then(setSettings);
  }, []);

  useApplyTheme(settings?.theme);

  const refreshBranchStatus = useCallback(async () => {
    if (!repoPath) {
      setBranchStatus(null);
      return;
    }
    try {
      const s = await api.gitBranchStatus(repoPath);
      setBranchStatus(s);
    } catch {
      setBranchStatus(null);
    }
  }, [repoPath]);

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
    // already maintains one and exposes it via listProcesses(). Spreading a
    // new procs array on every log line caused App-wide re-renders under load.
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

  // Global optional services (one each, shared across servers).
  const ngrokRunning = procs.some(
    (p) => p.id === "ngrok" && (p.state === "running" || p.state === "stopping"),
  );
  const pythonRunning = procs.some(
    (p) =>
      p.id === "python-server" &&
      (p.state === "running" || p.state === "stopping"),
  );

  // Tray reflects the active server's serve/docker plus the global services.
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

  const switchServer = useCallback((id: string) => {
    setSettings((prev) => {
      if (!prev || prev.active_server_id === id) return prev;
      const next = { ...prev, active_server_id: id };
      api
        .saveSettings(next)
        .catch((e) => console.error("save active server failed", e));
      return next;
    });
  }, []);

  // Tray menu start-all / stop-all → drive the active server's stack.
  useEffect(() => {
    if (!settings) return;
    const srv = activeServer(settings);
    let cancelled = false;
    const unlistens: Array<() => void> = [];
    const register = async (event: string, handler: () => Promise<void>) => {
      const u = await listen(event, handler);
      if (cancelled) u();
      else unlistens.push(u);
    };
    register("tray:start-all", async () => {
      if (!srv.worktree_path) return;
      try {
        await startAll({
          server: srv,
          settings,
          health: { serveUp: serve.up, dockerUp: docker.up, ngrokRunning, pythonRunning },
        });
      } catch (e) {
        console.error("tray:start-all failed", e);
      }
    });
    register("tray:stop-all", async () => {
      try {
        await stopAll({
          server: srv,
          health: { serveUp: serve.up, dockerUp: docker.up, ngrokRunning, pythonRunning },
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

  // Quit flow. anyRunning + shutdown targets span ALL servers, so Cmd+Q tears
  // down every stack, not just the active one. Read via a ref so the
  // register-once listener never sees stale state.
  const anyServerUp = settings
    ? settings.servers.some(
        (s) => healthMap[s.id]?.serve.up || healthMap[s.id]?.docker.up,
      )
    : false;
  const anyRunning = anyServerUp || ngrokRunning || pythonRunning;
  const shutdownTargets: ComposeTarget[] = settings
    ? settings.servers
        .filter((s) => s.worktree_path)
        .map((s) => ({ cwd: s.worktree_path as string, project: s.compose_project }))
    : [];
  const quitInfoRef = useRef<{ anyRunning: boolean; targets: ComposeTarget[] }>({
    anyRunning: false,
    targets: [],
  });
  quitInfoRef.current = { anyRunning, targets: shutdownTargets };

  useEffect(() => {
    let cancelled = false;
    let unlisten: (() => void) | undefined;
    listen("app:quit-requested", () => {
      if (quitInfoRef.current.anyRunning) {
        setQuitPhase("confirm");
      } else {
        // Nothing to clean up — still pass targets so any stray compose stack
        // gets a down, then exit straight away.
        api.shutdownNow(quitInfoRef.current.targets).catch((e) => {
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
    try {
      await api.shutdownNow(quitInfoRef.current.targets);
    } catch (e) {
      console.error("shutdown_now failed", e);
      setQuitPhase("idle");
    }
  }, []);

  const cancelQuit = useCallback(() => setQuitPhase("idle"), []);

  const goToLogs = useCallback(() => {
    setActive("logs");
  }, []);

  const goToSettings = useCallback((section: SettingsSection) => {
    setSettingsSection(section);
    setActive("settings");
  }, []);

  if (!settings || !activeSrv) {
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
      <ServerSwitcher
        settings={settings}
        healthMap={healthMap}
        onSwitch={switchServer}
        onManage={() => goToSettings("servers")}
      />
      <TabBar active={active} onChange={setActive} />
      <main style={{ flex: 1, minHeight: 0, overflow: "hidden" }}>
        {active === "server" && (
          <ServerTab
            server={activeSrv}
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
            repoPath={repoPath}
            branchStatus={branchStatus}
            refreshBranchStatus={refreshBranchStatus}
          />
        )}
        {active === "settings" && (
          <SettingsTab
            settings={settings}
            onChange={(s) => {
              // Only re-probe git when the active server's worktree actually
              // changed — otherwise every per-keystroke save (python port,
              // ngrok flags) was firing a git command.
              const prevWt = activeServer(settings).worktree_path;
              const nextWt = activeServer(s).worktree_path;
              setSettings(s);
              if (nextWt !== prevWt) refreshBranchStatus();
            }}
            section={settingsSection}
            onSectionChange={setSettingsSection}
          />
        )}
        {active === "logs" && <LogsTab server={activeSrv} />}
        {active === "database" && (
          <DatabaseTab
            server={activeSrv}
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
              This will stop every server's fleet serve, ngrok, the python
              server, and run <span className="mono">docker compose down</span>{" "}
              for each server before closing the app.
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
