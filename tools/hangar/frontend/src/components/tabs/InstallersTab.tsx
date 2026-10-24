import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { listen } from "../../lib/events";
import { Toast } from "../Toast";
import { LogBox } from "../LogLines";
import {
  api,
  type ContextSummary,
  type InstallerServerConfig,
  type InstallersChannels,
  type InstallersLocalTUF,
  type InstallersOutputFile,
  type InstallersScanResult,
  type InstallersServerInfo,
  type LogEntry,
  type LogLine,
  type ProcInfo,
  type ServerProfile,
  type Settings,
} from "../../lib/ipc";
import {
  INSTALLER_PLATFORMS,
  UPDATE_LOCAL_TUF,
  UPDATE_OFFICIAL,
  defaultInstallerServer,
  formatBytes,
  installersChannel,
  installersProcId,
  jobStatuses,
  type JobStatus,
} from "../../lib/installers";
import { serverColorVar } from "../../lib/servers";

const LOG_LEVELS = ["debug", "info", "warn", "error"];

export function InstallersTab({
  settings,
  onSettingsChange,
  procs,
}: {
  settings: Settings;
  onSettingsChange: (s: Settings) => void;
  procs: ProcInfo[];
}) {
  const cfg = settings.installers;
  const [defaultDir, setDefaultDir] = useState("");
  const [contexts, setContexts] = useState<ContextSummary[]>([]);
  const [official, setOfficial] = useState<InstallersChannels | null>(null);
  const [localTuf, setLocalTuf] = useState<InstallersLocalTUF | null>(null);
  const [toast, setToast] = useState<{ kind: "ok" | "err"; msg: string } | null>(null);

  const toastTimer = useRef<number | undefined>(undefined);
  const flash = useCallback((kind: "ok" | "err", msg: string) => {
    setToast({ kind, msg });
    window.clearTimeout(toastTimer.current);
    toastTimer.current = window.setTimeout(() => setToast(null), 2600);
  }, []);
  useEffect(() => () => window.clearTimeout(toastTimer.current), []);

  useEffect(() => {
    api.installersDefaultOutputDir().then(setDefaultDir).catch(() => {});
    api.installersOfficialChannels().then(setOfficial).catch(() => {});
    api.fleetctlReadContext().then((c) => setContexts(c.contexts)).catch(() => {});
    const loadTuf = () => api.installersLocalTuf().then(setLocalTuf).catch(() => {});
    loadTuf();
    const poll = window.setInterval(loadTuf, 10000);
    return () => window.clearInterval(poll);
  }, []);

  // Updates build on the latest settings (not the render-time prop), so two
  // cards changing at once, or a change made while the folder picker is open,
  // don't overwrite each other. Saves are debounced so typing in a text field
  // doesn't race one disk write per keystroke.
  const latest = useRef(settings);
  latest.current = settings;
  const saveTimer = useRef<number | undefined>(undefined);
  const pendingSave = useRef<Settings | null>(null);
  const flushSave = useCallback(() => {
    window.clearTimeout(saveTimer.current);
    const next = pendingSave.current;
    pendingSave.current = null;
    if (!next) return;
    api.saveSettings(next).catch((e) => {
      console.error("save settings failed", e);
      flash("err", "Failed to save settings");
    });
  }, [flash]);
  useEffect(() => flushSave, [flushSave]);

  const persist = useCallback(
    (update: (s: Settings) => Settings) => {
      const next = update(latest.current);
      latest.current = next;
      onSettingsChange(next);
      pendingSave.current = next;
      window.clearTimeout(saveTimer.current);
      saveTimer.current = window.setTimeout(flushSave, 400);
    },
    [onSettingsChange, flushSave],
  );

  const setOutputDir = useCallback(
    (dir: string) => persist((s) => ({ ...s, installers: { ...s.installers, output_dir: dir } })),
    [persist],
  );

  const setServerCfg = useCallback(
    (serverId: string, next: InstallerServerConfig) =>
      persist((s) => ({ ...s, installers: { ...s.installers, servers: { ...s.installers.servers, [serverId]: next } } })),
    [persist],
  );

  const chooseDir = useCallback(async () => {
    try {
      const dir = await api.pickFolder();
      if (dir) setOutputDir(dir);
    } catch (e) {
      flash("err", errText(e));
    }
  }, [setOutputDir, flash]);

  const outputRoot = cfg.output_dir.trim() || defaultDir;

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
        <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <div className="section-title" style={{ margin: 0 }}>Output folder</div>
            <div style={{ flex: 1 }} />
            <span className="dim">
              runs <span className="mono">fleetctl package</span> with each server's own fleetctl
            </span>
          </div>
          <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap" }}>
            <input
              value={cfg.output_dir}
              placeholder={defaultDir}
              onChange={(e) => setOutputDir(e.target.value)}
              style={{ flex: 1, minWidth: 260 }}
            />
            <button onClick={chooseDir}>Choose…</button>
            <button onClick={() => outputRoot && api.openPath(outputRoot, false).catch((e) => flash("err", errText(e)))} disabled={!outputRoot}>
              Open
            </button>
            {cfg.output_dir.trim() && <button className="link-btn" onClick={() => setOutputDir("")}>use default</button>}
          </div>
          <span className="dim">Each server gets its own subfolder. Rebuilding with the same settings overwrites the same files.</span>
        </div>

        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(460px, 1fr))", gap: "var(--pad-medium)", alignItems: "start" }}>
          {settings.servers.map((srv) => (
            <ServerCard
              key={srv.id}
              server={srv}
              cfg={cfg.servers[srv.id] ?? defaultInstallerServer()}
              onChange={(next) => setServerCfg(srv.id, next)}
              outputDir={cfg.output_dir}
              contexts={contexts}
              official={official}
              localTuf={localTuf}
              proc={procs.find((p) => p.id === installersProcId(srv.id))}
              flash={flash}
            />
          ))}
        </div>
      </div>
      {toast && <Toast kind={toast.kind} msg={toast.msg} />}
    </div>
  );
}

function ServerCard({
  server,
  cfg,
  onChange,
  outputDir,
  contexts,
  official,
  localTuf,
  proc,
  flash,
}: {
  server: ServerProfile;
  cfg: InstallerServerConfig;
  onChange: (next: InstallerServerConfig) => void;
  outputDir: string;
  contexts: ContextSummary[];
  official: InstallersChannels | null;
  localTuf: InstallersLocalTUF | null;
  proc: ProcInfo | undefined;
  flash: (kind: "ok" | "err", msg: string) => void;
}) {
  const [info, setInfo] = useState<InstallersServerInfo | null>(null);
  const [scan, setScan] = useState<InstallersScanResult | null>(null);
  const [scanErr, setScanErr] = useState("");
  const [scanning, setScanning] = useState(false);
  const [planned, setPlanned] = useState<Record<string, string>>({});
  const [outputs, setOutputs] = useState<InstallersOutputFile[]>([]);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [showLog, setShowLog] = useState(false);

  const set = <K extends keyof InstallerServerConfig>(k: K, v: InstallerServerConfig[K]) => onChange({ ...cfg, [k]: v });

  const building = proc?.state === "running" || proc?.state === "stopping";
  const contextName = cfg.context || info?.matched_context || "";
  const accent = serverColorVar(server.color);

  useEffect(() => {
    api.installersServerInfo(server.id, outputDir).then(setInfo).catch(() => setInfo(null));
  }, [server.id, server.name, server.worktree_path, outputDir]);

  const plannedKey = JSON.stringify(cfg);
  useEffect(() => {
    api.installersPlannedFiles(server.id, cfg).then(setPlanned).catch(() => setPlanned({}));
  }, [server.id, server.name, plannedKey]);

  // Only the latest scan may update state, so a slow scan of a previous
  // context can't overwrite the current one.
  const scanSeq = useRef(0);
  const runScan = useCallback(async () => {
    if (!contextName) return;
    const seq = ++scanSeq.current;
    setScanning(true);
    try {
      const res = await api.installersScan(contextName);
      if (seq !== scanSeq.current) return;
      setScan(res);
      setScanErr("");
    } catch (e) {
      if (seq !== scanSeq.current) return;
      setScan(null);
      setScanErr(errText(e));
    } finally {
      if (seq === scanSeq.current) setScanning(false);
    }
  }, [contextName]);

  // Scan once a context is known, and again whenever it changes (dropping the
  // old context's fleets right away).
  useEffect(() => {
    setScan(null);
    setScanErr("");
    if (contextName) runScan();
  }, [contextName, runScan]);

  const loadOutputs = useCallback(() => {
    api.installersListOutputs(server.id, outputDir).then(setOutputs).catch(() => setOutputs([]));
  }, [server.id, outputDir]);
  useEffect(() => {
    loadOutputs();
  }, [loadOutputs, proc?.state, info?.output_dir]);

  // Tail this server's build log (statuses come from build.sh's marker lines).
  const procId = installersProcId(server.id);
  const logSeq = useRef(0);
  const loadLog = useCallback(async () => {
    const seq = ++logSeq.current;
    try {
      const w = await api.readLogWindow({ source: installersChannel(server.id), since_ms: 0, levels: LOG_LEVELS, max_lines: 5000 });
      if (seq === logSeq.current) setEntries(w.entries);
    } catch (e) {
      console.error("installers readLogWindow", e);
    }
  }, [server.id]);
  useEffect(() => {
    loadLog();
    // Builds stream many lines (docker pulls); reload at most every 300ms.
    let timer: number | undefined;
    let cancelled = false;
    let un: (() => void) | undefined;
    listen<LogLine>("proc:log", (e) => {
      if (e.payload.proc_id !== procId || timer !== undefined) return;
      timer = window.setTimeout(() => {
        timer = undefined;
        loadLog();
      }, 300);
    })
      .then((u) => {
        if (cancelled) u();
        else un = u;
      })
      .catch((e) => console.error("installers listen", e));
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
      un?.();
    };
  }, [loadLog, procId]);
  // A final reload when the build ends catches lines that raced the listener.
  useEffect(() => {
    loadLog();
  }, [proc?.state, loadLog]);

  const statuses = useMemo(() => jobStatuses(entries), [entries]);
  // Open the output when a build fails while the tab is open — not for an old
  // failure on remount.
  const prevState = useRef(proc?.state);
  useEffect(() => {
    if (proc?.state === "failed" && prevState.current !== "failed") setShowLog(true);
    prevState.current = proc?.state;
  }, [proc?.state]);

  const channels: InstallersChannels | null = cfg.update_source === UPDATE_LOCAL_TUF ? (localTuf?.channels ?? null) : official;

  const setUpdateSource = (source: string) => {
    const list = source === UPDATE_LOCAL_TUF ? localTuf?.channels : official;
    const keep = (ch: string, options: string[] | undefined) => (options?.includes(ch) ? ch : "stable");
    onChange({
      ...cfg,
      update_source: source,
      orbit_channel: keep(cfg.orbit_channel, list?.orbit),
      desktop_channel: keep(cfg.desktop_channel, list?.desktop),
      osqueryd_channel: keep(cfg.osqueryd_channel, list?.osqueryd),
    });
  };

  const togglePlatform = (key: string) => {
    const has = cfg.platforms.includes(key);
    set("platforms", has ? cfg.platforms.filter((p) => p !== key) : [...cfg.platforms, key]);
  };

  const pickTeam = (value: string) => {
    const team = scan?.teams.find((t) => (t.id == null ? "" : String(t.id)) === value);
    onChange({ ...cfg, team_id: team?.id ?? null, team_name: team?.name ?? "" });
  };

  const build = async () => {
    try {
      setShowLog(false);
      await api.installersStartBuild(server.id, cfg, outputDir);
    } catch (e) {
      flash("err", errText(e));
    }
  };
  const stop = async () => {
    try {
      await api.installersStopBuild(server.id);
    } catch (e) {
      flash("err", errText(e));
    }
  };

  const selectedTeam = scan?.teams.find((t) => (t.id ?? null) === (cfg.team_id ?? null));
  const missing: string[] = [];
  if (!info?.fleetctl_exists) missing.push("fleetctl build");
  if (!contextName) missing.push("fleetctl context");
  if (!scan) missing.push("a successful scan");
  else if (!selectedTeam) missing.push("a fleet");
  else if (!selectedTeam.has_secret) missing.push("an enroll secret for that fleet");
  if (cfg.platforms.length === 0) missing.push("a platform");
  if (cfg.update_source === UPDATE_LOCAL_TUF && !localTuf?.available) missing.push("local TUF");
  const canBuild = missing.length === 0 && !building;

  const fleetUrl = cfg.fleet_url.trim() || scan?.server_url || "";
  const loopback = /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])(:|\/|$)/.test(fleetUrl);

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)", borderTop: `3px solid ${accent}` }}>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
        <span className="dot" style={{ background: accent }} />
        <div className="section-title" style={{ margin: 0 }}>{server.name}</div>
        <span className="dim mono">:{server.ports.server}</span>
        <div style={{ flex: 1 }} />
        {info && !info.fleetctl_exists && (
          <span style={{ color: "var(--ui-error)" }} title={info.fleetctl}>fleetctl not built</span>
        )}
      </div>

      {/* Connection: context, scan, fleet */}
      <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap", alignItems: "flex-end" }}>
        <Field label="fleetctl context">
          <select value={cfg.context} onChange={(e) => set("context", e.target.value)} style={{ width: "100%" }}>
            <option value="">{info?.matched_context ? `auto (${info.matched_context})` : "auto (no match)"}</option>
            {contexts.map((c) => (
              <option key={c.name} value={c.name}>
                {c.name}
                {c.address ? ` — ${c.address}` : ""}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Fleet">
          <div style={{ display: "flex", gap: "var(--pad-small)" }}>
            <select
              value={cfg.team_id == null ? "" : String(cfg.team_id)}
              onChange={(e) => pickTeam(e.target.value)}
              disabled={!scan}
              style={{ flex: 1, minWidth: 0 }}
            >
              {!scan && <option value={cfg.team_id == null ? "" : String(cfg.team_id)}>{cfg.team_name || "Unassigned"}</option>}
              {scan && !selectedTeam && <option value={cfg.team_id == null ? "" : String(cfg.team_id)}>{cfg.team_name || "?"} (not found)</option>}
              {scan?.teams.map((t) => (
                <option key={t.id ?? "none"} value={t.id == null ? "" : String(t.id)} disabled={!t.has_secret}>
                  {t.name}
                  {t.has_secret ? "" : t.secret_hidden ? " (secret hidden from this user)" : " (no enroll secret)"}
                </option>
              ))}
            </select>
            {/* The label stays in the layout (hidden) under the spinner so the button never changes size. */}
            <button onClick={runScan} disabled={!contextName || scanning} aria-label={scanning ? "Scanning" : "Scan"} style={{ position: "relative" }}>
              <span style={{ visibility: scanning ? "hidden" : "visible" }}>Scan</span>
              {scanning && (
                <span style={{ position: "absolute", inset: 0, display: "flex", alignItems: "center", justifyContent: "center" }}>
                  <span className="spin">↻</span>
                </span>
              )}
            </button>
          </div>
        </Field>
      </div>
      {scanErr && <span style={{ color: "var(--ui-error)" }}>{scanErr}</span>}
      {scan && (
        <span className="dim">
          {scan.premium ? "Premium" : "Free (no fleets — Unassigned only)"} · {scan.teams.length - 1} fleet{scan.teams.length === 2 ? "" : "s"} · via{" "}
          <span className="mono">{scan.context}</span>
        </span>
      )}

      <Field label="Fleet URL baked into the packages" full hint={loopback ? "This is a loopback address — other machines can't reach it. Use the public (e.g. ngrok) URL." : undefined} warn={loopback}>
        <input value={cfg.fleet_url} placeholder={scan?.server_url || "server_url from the server's settings"} onChange={(e) => set("fleet_url", e.target.value)} style={{ width: "100%" }} />
      </Field>

      {/* Platforms + planned file names + per-build status */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
          <span style={{ fontWeight: 600 }}>Platforms</span>
          <div style={{ flex: 1 }} />
          <button className="link-btn" onClick={() => set("platforms", INSTALLER_PLATFORMS.map((p) => p.key))}>all</button>
          <button className="link-btn" onClick={() => set("platforms", [])}>none</button>
        </div>
        {INSTALLER_PLATFORMS.map((p) => (
          <label key={p.key} style={{ display: "grid", gridTemplateColumns: "auto 150px 1fr auto", alignItems: "center", gap: 8 }}>
            <input type="checkbox" checked={cfg.platforms.includes(p.key)} onChange={() => togglePlatform(p.key)} />
            <span>{p.label}</span>
            <span className="mono dim" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={planned[p.key]}>
              {planned[p.key] ?? ""}
            </span>
            <JobBadge status={statuses[p.key]} active={building} userStopped={!!proc?.was_user_stopped} />
          </label>
        ))}
      </div>

      <div style={{ display: "flex", gap: "var(--pad-large)", flexWrap: "wrap" }}>
        <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <input type="checkbox" checked={cfg.fleet_desktop} onChange={(e) => set("fleet_desktop", e.target.checked)} />
          Fleet Desktop
        </label>
        <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
          <input type="checkbox" checked={cfg.debug} onChange={(e) => set("debug", e.target.checked)} />
          Debug logging
        </label>
      </div>

      {/* Updates: source + channels */}
      <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
        <div style={{ display: "flex", gap: "var(--pad-large)", alignItems: "center", flexWrap: "wrap" }}>
          <span style={{ fontWeight: 600 }}>Updates</span>
          <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <input type="radio" checked={cfg.update_source !== UPDATE_LOCAL_TUF} onChange={() => setUpdateSource(UPDATE_OFFICIAL)} />
            Official (updates.fleetdm.com)
          </label>
          <label style={{ display: "flex", alignItems: "center", gap: 6 }} title={localTuf?.available ? localTuf.url : localTuf?.reason}>
            <input
              type="radio"
              checked={cfg.update_source === UPDATE_LOCAL_TUF}
              disabled={!localTuf?.available && cfg.update_source !== UPDATE_LOCAL_TUF}
              onChange={() => setUpdateSource(UPDATE_LOCAL_TUF)}
            />
            Local TUF
          </label>
        </div>
        {cfg.update_source === UPDATE_LOCAL_TUF && (
          <span className={localTuf?.available ? "dim" : undefined} style={{ color: localTuf?.available ? undefined : "var(--ui-error)" }}>
            {localTuf?.available ? (
              <>
                <span className="mono">{localTuf.url}</span> — a TUF tab rebuild rotates its keys, so rebuild these installers after one.
              </>
            ) : (
              localTuf?.reason ?? "checking local TUF…"
            )}
          </span>
        )}
        <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap" }}>
          <ChannelSelect label="orbit" value={cfg.orbit_channel} options={channels?.orbit} onChange={(v) => set("orbit_channel", v)} />
          <ChannelSelect label="Fleet Desktop" value={cfg.desktop_channel} options={channels?.desktop} onChange={(v) => set("desktop_channel", v)} />
          <ChannelSelect label="osquery" value={cfg.osqueryd_channel} options={channels?.osqueryd} onChange={(v) => set("osqueryd_channel", v)} />
        </div>
      </div>

      <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap" }}>
        {building ? (
          <button className="danger" onClick={stop}>Stop</button>
        ) : (
          <button className="primary" onClick={build} disabled={!canBuild}>
            Generate {cfg.platforms.length} installer{cfg.platforms.length === 1 ? "" : "s"}
          </button>
        )}
        <BuildStatus proc={proc} building={building} />
        {!building && !canBuild && <span className="dim">needs: {missing.join(", ")}</span>}
        <div style={{ flex: 1 }} />
        {entries.length > 0 && (
          <button className="link-btn" onClick={() => setShowLog((v) => !v)}>{showLog ? "hide output" : "show output"}</button>
        )}
      </div>
      {showLog && <LogBox entries={entries} maxHeight={260} />}

      {/* What's in this server's folder */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
          <span style={{ fontWeight: 600 }}>Files</span>
          <span className="mono dim" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{info?.output_dir}</span>
          <div style={{ flex: 1 }} />
          {info && outputs.length > 0 && (
            <button className="link-btn" onClick={() => api.openPath(info.output_dir, false).catch((e) => flash("err", errText(e)))}>open folder</button>
          )}
        </div>
        {outputs.length === 0 ? (
          <span className="dim">none yet</span>
        ) : (
          outputs.map((f) => (
            <div key={f.name} style={{ display: "grid", gridTemplateColumns: "1fr auto auto", gap: 8, alignItems: "center" }}>
              <span className="mono" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={f.path}>{f.name}</span>
              <span className="dim">{formatBytes(f.size)}</span>
              <span className="dim">{new Date(f.mod_time_ms).toLocaleString()}</span>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function ChannelSelect({
  label,
  value,
  options,
  onChange,
}: {
  label: string;
  value: string;
  options: string[] | undefined;
  onChange: (v: string) => void;
}) {
  const list = options && options.length > 0 ? options : ["stable"];
  const current = value || "stable";
  return (
    <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
      {label}
      <select value={current} onChange={(e) => onChange(e.target.value)}>
        {!list.includes(current) && <option value={current}>{current} (unavailable)</option>}
        {list.map((ch) => (
          <option key={ch} value={ch}>{ch}</option>
        ))}
      </select>
    </label>
  );
}

function JobBadge({ status, active, userStopped }: { status: JobStatus | undefined; active: boolean; userStopped: boolean }) {
  if (!status) return <span />;
  if (status === "building") {
    if (active) {
      return (
        <span className="dim" style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
          <span className="dot run" /> building
        </span>
      );
    }
    // No result line after the run ended: stopped by the user, or the build died.
    return userStopped ? <span className="dim">stopped</span> : <span style={{ color: "var(--ui-error)", fontWeight: 600 }}>✗ interrupted</span>;
  }
  if (status === "ok") return <span style={{ color: "var(--core-fleet-green)", fontWeight: 600 }}>✓ built</span>;
  return <span style={{ color: "var(--ui-error)", fontWeight: 600 }}>✗ failed</span>;
}

function BuildStatus({ proc, building }: { proc: ProcInfo | undefined; building: boolean }) {
  if (building) {
    return (
      <span className="dim" style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
        <span className="dot run" /> building… (.msi builds use Docker and take a few minutes)
      </span>
    );
  }
  if (!proc) return null;
  if (proc.state === "done") {
    if (proc.was_user_stopped) return <span className="dim">stopped</span>;
    return <span style={{ color: "var(--core-fleet-green)", fontWeight: 600 }}>all built</span>;
  }
  if (proc.state === "failed") {
    return <span style={{ color: "var(--ui-error)", fontWeight: 600 }}>some builds failed — see output</span>;
  }
  return null;
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
    <div style={{ display: "flex", flexDirection: "column", gap: 4, flex: full ? undefined : 1, minWidth: full ? undefined : 200, width: full ? "100%" : undefined }}>
      <span style={{ fontWeight: 600 }}>{label}</span>
      {children}
      {hint ? (
        <span className={warn ? undefined : "dim"} style={{ color: warn ? "var(--ui-error)" : undefined }}>
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
