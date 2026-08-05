import { useCallback, useEffect, useRef, useState } from "react";
import { listen } from "../../lib/events";
import { StatusPill } from "../StatusPill";
import { Toast } from "../Toast";
import { LogBox } from "../LogLines";
import {
  api,
  type LogEntry,
  type LogLine,
  type NgrokRunningTunnel,
  type ProcInfo,
  type Settings,
  type TufConfig,
  type TufServerStatus,
} from "../../lib/ipc";
import {
  FLEET_PORT,
  TUF_CHANNEL,
  TUF_PLATFORMS,
  TUF_PORT,
  TUF_PROC_ID,
  domainOf,
  tunnelForPort,
} from "../../lib/tuf";
import { copyText } from "../../lib/clipboard";

const LOG_LEVELS = ["debug", "info", "warn", "error"];

export function TufTab({
  settings,
  onSettingsChange,
  procs,
}: {
  settings: Settings;
  onSettingsChange: (s: Settings) => void;
  procs: ProcInfo[];
}) {
  const cfg = settings.tuf;
  const [tunnels, setTunnels] = useState<NgrokRunningTunnel[]>([]);
  const [server, setServer] = useState<TufServerStatus | null>(null);
  const [assetsExist, setAssetsExist] = useState(false);
  const [toast, setToast] = useState<{ kind: "ok" | "err"; msg: string } | null>(null);

  const flash = useCallback((kind: "ok" | "err", msg: string) => {
    setToast({ kind, msg });
    window.setTimeout(() => setToast(null), 2600);
  }, []);

  const persist = useCallback(
    (nextCfg: TufConfig) => {
      const next = { ...settings, tuf: nextCfg };
      onSettingsChange(next);
      api.saveSettings(next).catch((e) => {
        console.error("save settings failed", e);
        flash("err", "Failed to save settings");
      });
    },
    [settings, onSettingsChange, flash],
  );
  const set = useCallback(
    <K extends keyof TufConfig>(k: K, v: TufConfig[K]) => persist({ ...cfg, [k]: v }),
    [cfg, persist],
  );

  const togglePlatform = useCallback(
    (key: string) => {
      const has = cfg.platforms.includes(key);
      const platforms = has ? cfg.platforms.filter((p) => p !== key) : [...cfg.platforms, key];
      persist({ ...cfg, platforms });
    },
    [cfg, persist],
  );

  // Poll ngrok tunnels + TUF server status + whether assets exist, so the
  // prerequisites and cleanup controls reflect reality when the tab is open.
  const refreshStatus = useCallback(async () => {
    const [t, s, a] = await Promise.all([
      api.ngrokTunnels().catch(() => [] as NgrokRunningTunnel[]),
      api.tufServerStatus().catch(() => null),
      api.tufAssetsExist().catch(() => false),
    ]);
    setTunnels(t);
    setServer(s);
    setAssetsExist(a);
  }, []);
  useEffect(() => {
    refreshStatus();
    const poll = window.setInterval(refreshStatus, 4000);
    return () => window.clearInterval(poll);
  }, [refreshStatus]);

  const buildProc = procs.find((p) => p.id === TUF_PROC_ID);
  const building = buildProc?.state === "running" || buildProc?.state === "stopping";

  const fleetTunnel = tunnelForPort(tunnels, FLEET_PORT);
  const tufTunnel = tunnelForPort(tunnels, TUF_PORT);

  const useDetected = useCallback(() => {
    const next = { ...cfg };
    if (fleetTunnel?.public_url) next.fleet_url = fleetTunnel.public_url;
    if (tufTunnel?.public_url) next.tuf_url = tufTunnel.public_url;
    persist(next);
    flash("ok", "Filled URLs from running tunnels");
  }, [cfg, fleetTunnel, tufTunnel, persist, flash]);

  const runBuild = useCallback(async () => {
    try {
      await api.tufStartBuild(cfg);
    } catch (e) {
      flash("err", errText(e));
    }
  }, [cfg, flash]);

  const stopBuild = useCallback(async () => {
    try {
      await api.tufStopBuild();
    } catch (e) {
      flash("err", errText(e));
    }
  }, [flash]);

  const startServer = useCallback(async () => {
    try {
      await api.tufStartServer();
      await refreshStatus();
    } catch (e) {
      flash("err", errText(e));
    }
  }, [refreshStatus, flash]);

  const killServer = useCallback(async () => {
    try {
      const outcomes = await api.tufKillServer();
      await refreshStatus();
      flash("ok", outcomes.length ? `Killed ${outcomes.length} process(es)` : "Nothing on :8081");
    } catch (e) {
      flash("err", errText(e));
    }
  }, [refreshStatus, flash]);

  // When a build finishes cleanly, bring the TUF server up so the generated
  // packages have somewhere to fetch from — mirroring the old all-in-one run.
  // Guarded per-build (started_at_ms) and on server-down so it fires once.
  const autoStarted = useRef<number | null>(null);
  useEffect(() => {
    if (!buildProc || buildProc.state !== "done" || buildProc.exit_code !== 0) return;
    if (server?.up) return;
    const key = buildProc.started_at_ms ?? 0;
    if (autoStarted.current === key) return;
    autoStarted.current = key;
    startServer();
  }, [buildProc?.state, buildProc?.exit_code, buildProc?.started_at_ms, server?.up, startServer]);

  const deleteAssets = useCallback(async () => {
    try {
      await api.tufDeleteAssets();
      await refreshStatus();
      flash("ok", "Deleted test_tuf");
    } catch (e) {
      flash("err", errText(e));
    }
  }, [refreshStatus, flash]);

  const copy = useCallback(
    async (text: string, label: string) => {
      try {
        await copyText(text);
        flash("ok", `Copied ${label}`);
      } catch {
        flash("err", "Clipboard unavailable");
      }
    },
    [flash],
  );

  const missing: string[] = [];
  if (cfg.platforms.length === 0) missing.push("pick a platform");
  if (!cfg.enroll_secret.trim()) missing.push("enroll secret");
  if (!tufTunnel) missing.push("TUF tunnel");
  const canRun = missing.length === 0 && !building;

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column", fontSize: "var(--fs-xx-small)" }}>
      <div
        style={{
          flex: 1,
          minHeight: 0,
          overflowY: "auto",
          padding: "var(--pad-medium)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-medium)",
        }}
      >
        {/* Prerequisites: tunnels + server */}
        <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <div className="section-title" style={{ margin: 0 }}>Prerequisites</div>
            <div style={{ flex: 1 }} />
            <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
              runs <span className="mono">tools/tuf/test/main.sh</span>
            </span>
            <button onClick={useDetected} disabled={!fleetTunnel && !tufTunnel}>
              Use detected URLs
            </button>
          </div>

          <TunnelRow label="Fleet tunnel" port={FLEET_PORT} tunnel={fleetTunnel} domain={domainOf(cfg.fleet_url)} onCopy={copy} />
          <TunnelRow label="TUF tunnel" port={TUF_PORT} tunnel={tufTunnel} domain={domainOf(cfg.tuf_url)} onCopy={copy} />

          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)", flexWrap: "wrap" }}>
            <StatusPill label="TUF server" up={!!server?.up} />
            {server?.up ? (
              <span className="mono" style={{ fontSize: "var(--fs-xx-small)" }}>{server.url}</span>
            ) : (
              <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>not running on :{TUF_PORT}</span>
            )}
            <div style={{ flex: 1 }} />
            {server?.up ? (
              <button className="danger" onClick={killServer}>Kill server</button>
            ) : assetsExist ? (
              <button className="primary" onClick={startServer}>Start server</button>
            ) : (
              <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>nothing to clean up</span>
            )}
            {assetsExist && (
              <button className="danger" onClick={deleteAssets}>Delete assets</button>
            )}
          </div>
        </div>

        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(380px, 1fr))", gap: "var(--pad-medium)", alignItems: "start" }}>
        {/* Build config */}
        <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)" }}>
          <div className="section-title" style={{ margin: 0 }}>Platforms</div>
          <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--pad-small) var(--pad-large)" }}>
            {TUF_PLATFORMS.map((p) => (
              <label key={p.key} style={{ display: "flex", alignItems: "center", gap: 6, fontSize: "var(--fs-xx-small)" }}>
                <input type="checkbox" checked={cfg.platforms.includes(p.key)} onChange={() => togglePlatform(p.key)} />
                {p.label}
              </label>
            ))}
          </div>

          <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap" }}>
            <Field label="Fleet URL">
              <input value={cfg.fleet_url} placeholder="https://you.ngrok.app" onChange={(e) => set("fleet_url", e.target.value)} style={{ width: "100%" }} />
            </Field>
            <Field label="TUF URL">
              <input value={cfg.tuf_url} placeholder="https://tuf.you.ngrok.app" onChange={(e) => set("tuf_url", e.target.value)} style={{ width: "100%" }} />
            </Field>
          </div>

          <Field label="Enroll secret" hint={cfg.enroll_secret.trim() ? undefined : "Required to generate packages."} warn={!cfg.enroll_secret.trim()} full>
            <input value={cfg.enroll_secret} onChange={(e) => set("enroll_secret", e.target.value)} style={{ width: "100%" }} />
          </Field>

          <div style={{ display: "flex", gap: "var(--pad-large)" }}>
            <label style={{ display: "flex", alignItems: "center", gap: 6, fontSize: "var(--fs-xx-small)" }}>
              <input type="checkbox" checked={cfg.fleet_desktop} onChange={(e) => set("fleet_desktop", e.target.checked)} />
              Fleet Desktop
            </label>
            <label style={{ display: "flex", alignItems: "center", gap: 6, fontSize: "var(--fs-xx-small)" }}>
              <input type="checkbox" checked={cfg.debug} onChange={(e) => set("debug", e.target.checked)} />
              Debug
            </label>
          </div>

          <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap" }}>
            {building ? (
              <button className="danger" onClick={stopBuild}>Stop build</button>
            ) : (
              <button className="primary" onClick={runBuild} disabled={!canRun}>Generate &amp; run build</button>
            )}
            <BuildStatus proc={buildProc} building={building} />
            {!building && !canRun && (
              <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>needs: {missing.join(", ")}</span>
            )}
          </div>
        </div>

        <BuildLogView />
        </div>
      </div>

      {toast && <Toast kind={toast.kind} msg={toast.msg} />}
    </div>
  );
}

function TunnelRow({
  label,
  port,
  tunnel,
  domain,
  onCopy,
}: {
  label: string;
  port: number;
  tunnel: NgrokRunningTunnel | undefined;
  domain: string;
  onCopy: (text: string, label: string) => void;
}) {
  const cmd = `ngrok http --domain=${domain || "<your-domain>"} http://localhost:${port}`;
  return (
    <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)", flexWrap: "wrap" }}>
      <StatusPill label={label} up={!!tunnel} upText="up" downText="down" />
      {tunnel ? (
        <span className="mono" style={{ fontSize: "var(--fs-xx-small)" }}>{tunnel.public_url}</span>
      ) : (
        <>
          <span className="mono" style={{ fontSize: "var(--fs-xx-small)", userSelect: "all" }}>{cmd}</span>
          <button className="link-btn" onClick={() => onCopy(cmd, "ngrok command")}>copy</button>
        </>
      )}
    </div>
  );
}

function BuildStatus({ proc, building }: { proc: ProcInfo | undefined; building: boolean }) {
  if (building) {
    return (
      <span className="dim" style={{ fontSize: "var(--fs-xx-small)", display: "inline-flex", alignItems: "center", gap: 6 }}>
        <span className="dot run" /> building… (compile + docker cross-builds — several minutes)
      </span>
    );
  }
  if (!proc) return null;
  if (proc.state === "done") {
    return <span style={{ fontSize: "var(--fs-xx-small)", color: "var(--core-fleet-green)", fontWeight: 600 }}>build finished</span>;
  }
  if (proc.state === "failed") {
    return (
      <span style={{ fontSize: "var(--fs-xx-small)", color: "var(--ui-error)", fontWeight: 600 }}>
        build failed{proc.exit_code != null ? ` (exit ${proc.exit_code})` : ""}
      </span>
    );
  }
  return null;
}

function BuildLogView() {
  const [entries, setEntries] = useState<LogEntry[]>([]);

  const load = useCallback(async () => {
    try {
      const w = await api.readLogWindow({ source: TUF_CHANNEL, since_ms: 0, levels: LOG_LEVELS, max_lines: 1000 });
      setEntries(w.entries);
    } catch (e) {
      console.error("tuf readLogWindow", e);
    }
  }, []);

  useEffect(() => {
    load();
    const poll = window.setInterval(load, 2000);
    let cancelled = false;
    let un: (() => void) | undefined;
    listen<LogLine>("proc:log", (e) => {
      if (e.payload.proc_id === TUF_PROC_ID) load();
    }).then((u) => {
      if (cancelled) u();
      else un = u;
    });
    return () => {
      cancelled = true;
      window.clearInterval(poll);
      un?.();
    };
  }, [load]);

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-small)" }}>
      <div className="section-title" style={{ margin: 0 }}>Build output</div>
      <LogBox entries={entries} maxHeight={320} />
    </div>
  );
}

function Field({
  label,
  hint,
  warn,
  full,
  children,
}: {
  label: string;
  hint?: string;
  warn?: boolean;
  full?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4, flex: full ? undefined : 1, minWidth: full ? undefined : 220, width: full ? "100%" : undefined }}>
      <span style={{ fontSize: "var(--fs-xx-small)", fontWeight: 600 }}>{label}</span>
      {children}
      {hint ? (
        <span className={warn ? undefined : "dim"} style={{ fontSize: "var(--fs-xx-small)", color: warn ? "var(--ui-error)" : undefined }}>
          {hint}
        </span>
      ) : null}
    </div>
  );
}

function errText(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  return String(e);
}
