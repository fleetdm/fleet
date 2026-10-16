import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { listen } from "../../lib/events";
import { Toast } from "../Toast";
import { StatusPill } from "../StatusPill";
import { copyText } from "../../lib/clipboard";
import { LogBox } from "../LogLines";
import {
  api,
  type ContextSummary,
  type InstallerServerConfig,
  type InstallersChannels,
  type InstallersLocalTUF,
  type InstallersManifestEntry,
  type InstallersOutputFile,
  type InstallersScanResult,
  type InstallersTargetInfo,
  type InstallersShareStatus,
  type InstallersSharedFile,
  type LogEntry,
  type LogLine,
  type ProcInfo,
  type ServerProfile,
  type Settings,
} from "../../lib/ipc";
import {
  CUSTOM_TARGET,
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
// The share controls row swaps between the port box and the URL + buttons;
// a fixed minimum height keeps the card from shifting when it does.
const SHARE_ROW: React.CSSProperties = { display: "flex", alignItems: "center", gap: "var(--pad-small)", flexWrap: "wrap", minHeight: 40 };
const SHARE_BUTTON: React.CSSProperties = { minWidth: 130 };
// Fleet dropdown value shown while the enroll secret was typed by hand.
const MANUAL_FLEET = "__manual__";

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

  const setSharePort = useCallback(
    (port: number) => persist((s) => ({ ...s, installers: { ...s.installers, share_port: port } })),
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
  // Only servers that can build: enabled, with a worktree to take fleetctl from.
  const enabledServers = settings.servers.filter((srv) => srv.enabled && !!srv.worktree_path);
  // Build targets: each Hangar server, plus a custom target for any other Fleet.
  const targets: Target[] = [
    ...enabledServers.map((srv) => ({ id: srv.id, label: srv.name, server: srv })),
    { id: CUSTOM_TARGET, label: "Custom Fleet server" },
  ];
  // Until a target is picked here, follow the server selected in Hangar's switcher.
  const targetId = targets.some((t) => t.id === cfg.target)
    ? cfg.target
    : (targets.find((t) => t.id === settings.active_server_id) ?? targets[0]).id;
  // Switching to a Hangar server resets its context to the one matching its
  // port (an explicit pick from earlier would otherwise win); the custom
  // target keeps the context chosen for it.
  const setTarget = useCallback(
    (id: string) =>
      persist((s) => {
        const servers = { ...s.installers.servers };
        if (id !== CUSTOM_TARGET && servers[id]?.context) servers[id] = { ...servers[id], context: "" };
        return { ...s, installers: { ...s.installers, target: id, servers } };
      }),
    [persist],
  );
  const buildingIds = new Set(
    procs.filter((p) => p.id.startsWith("installers:") && (p.state === "running" || p.state === "stopping")).map((p) => p.id.slice("installers:".length)),
  );

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
        <OutputShareCard
          outputDir={cfg.output_dir}
          defaultDir={defaultDir}
          outputRoot={outputRoot}
          onOutputDirChange={setOutputDir}
          onChooseDir={chooseDir}
          port={cfg.share_port}
          onPortChange={setSharePort}
          flash={flash}
        />

        <TargetCard
          key={targetId}
          targetId={targetId}
          targets={targets}
          buildingIds={buildingIds}
          onTargetChange={setTarget}
          servers={enabledServers}
          cfg={cfg.servers[targetId] ?? defaultInstallerServer()}
          onChange={(next) => setServerCfg(targetId, next)}
          outputDir={cfg.output_dir}
          contexts={contexts}
          official={official}
          localTuf={localTuf}
          proc={procs.find((p) => p.id === installersProcId(targetId))}
          flash={flash}
        />
      </div>
      {toast && <Toast kind={toast.kind} msg={toast.msg} />}
    </div>
  );
}

type Target = { id: string; label: string; server?: ServerProfile };

function TargetCard({
  targetId,
  targets,
  buildingIds,
  onTargetChange,
  servers,
  cfg,
  onChange,
  outputDir,
  contexts,
  official,
  localTuf,
  proc,
  flash,
}: {
  targetId: string;
  targets: Target[];
  buildingIds: Set<string>;
  onTargetChange: (id: string) => void;
  servers: ServerProfile[];
  cfg: InstallerServerConfig;
  onChange: (next: InstallerServerConfig) => void;
  outputDir: string;
  contexts: ContextSummary[];
  official: InstallersChannels | null;
  localTuf: InstallersLocalTUF | null;
  proc: ProcInfo | undefined;
  flash: (kind: "ok" | "err", msg: string) => void;
}) {
  const [info, setInfo] = useState<InstallersTargetInfo | null>(null);
  const [showSecret, setShowSecret] = useState(false);
  const [flagsErr, setFlagsErr] = useState("");
  const [scan, setScan] = useState<InstallersScanResult | null>(null);
  const [scanErr, setScanErr] = useState("");
  const [scanning, setScanning] = useState(false);
  const [planned, setPlanned] = useState<Record<string, string>>({});
  const [outputs, setOutputs] = useState<InstallersOutputFile[]>([]);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [showLog, setShowLog] = useState(false);
  const [showFiles, setShowFiles] = useState(false);

  // Async results (scan, fleet secret) merge into the latest config, not the
  // one from the render that started them.
  const cfgRef = useRef(cfg);
  cfgRef.current = cfg;
  const update = (patch: Partial<InstallerServerConfig>) => onChange({ ...cfgRef.current, ...patch });
  const set = <K extends keyof InstallerServerConfig>(k: K, v: InstallerServerConfig[K]) => update({ [k]: v } as Partial<InstallerServerConfig>);

  const server = targets.find((t) => t.id === targetId)?.server;
  const isCustom = targetId === CUSTOM_TARGET;
  const building = proc?.state === "running" || proc?.state === "stopping";
  const contextName = cfg.context || info?.matched_context || "";
  const usableContexts = contexts.filter((c) => !!c.address);
  const accent = server ? serverColorVar(server.color) : "var(--app-border)";

  useEffect(() => {
    api.installersTargetInfo(targetId, cfg, outputDir).then(setInfo).catch(() => setInfo(null));
  }, [targetId, cfg.name, cfg.fleetctl, server?.name, server?.worktree_path, outputDir]);

  const plannedKey = JSON.stringify(cfg);
  useEffect(() => {
    api.installersPlannedFiles(targetId, cfg).then(setPlanned).catch(() => setPlanned({}));
  }, [targetId, server?.name, plannedKey]);

  useEffect(() => {
    if (!cfg.extra_flags.trim()) {
      setFlagsErr("");
      return;
    }
    api.installersCheckExtraFlags(cfg.extra_flags).then(setFlagsErr).catch(() => setFlagsErr(""));
  }, [cfg.extra_flags]);

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
      // Fill in what's empty; never overwrite what's been typed.
      const cur = cfgRef.current;
      if (!cur.fleet_url.trim() && res.server_url) update({ fleet_url: res.server_url });
      const team = res.teams.find((t) => (t.id ?? null) === (cur.team_id ?? null));
      if (!cur.enroll_secret.trim() && team?.has_secret) {
        const secret = await api.installersFleetSecret(contextName, team.id ?? null);
        if (seq === scanSeq.current) update({ enroll_secret: secret, team_name: team.name });
      }
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
    api.installersListOutputs(targetId, cfgRef.current, outputDir).then(setOutputs).catch(() => setOutputs([]));
  }, [targetId, outputDir]);
  useEffect(() => {
    loadOutputs();
  }, [loadOutputs, proc?.state, info?.output_dir]);

  // Tail this target's build log (statuses come from build.sh's marker lines).
  const procId = installersProcId(targetId);
  const logSeq = useRef(0);
  const loadLog = useCallback(async () => {
    const seq = ++logSeq.current;
    try {
      const w = await api.readLogWindow({ source: installersChannel(targetId), since_ms: 0, levels: LOG_LEVELS, max_lines: 5000 });
      if (seq === logSeq.current) setEntries(w.entries);
    } catch (e) {
      console.error("installers readLogWindow", e);
    }
  }, [targetId]);
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
    update({
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

  // Switching context drops what came from the old one — the fleet, a secret
  // filled from it, and a Fleet URL filled from its server_url — but keeps
  // anything typed by hand.
  const changeContext = (next: string) => {
    const cur = cfgRef.current;
    const urlFromScan = !!scan?.server_url && cur.fleet_url.trim() === scan.server_url;
    update({
      context: next,
      team_id: null,
      team_name: "",
      ...(cur.team_name ? { enroll_secret: "" } : {}),
      ...(urlFromScan ? { fleet_url: "" } : {}),
    });
  };

  // Picking a fleet fills its enroll secret (and the Fleet URL if empty).
  const pickTeam = async (value: string) => {
    if (value === MANUAL_FLEET) return;
    const team = scan?.teams.find((t) => (t.id == null ? "" : String(t.id)) === value);
    update({ team_id: team?.id ?? null, team_name: team?.name ?? "" });
    if (!team?.has_secret) return;
    try {
      const secret = await api.installersFleetSecret(contextName, team.id ?? null);
      update({ enroll_secret: secret, team_name: team.name, fleet_url: cfgRef.current.fleet_url.trim() || scan?.server_url || "" });
    } catch (e) {
      flash("err", errText(e));
    }
  };

  const build = async () => {
    try {
      setShowLog(false);
      await api.installersStartBuild(targetId, cfg, outputDir);
    } catch (e) {
      flash("err", errText(e));
    }
  };
  const stop = async () => {
    try {
      await api.installersStopBuild(targetId);
    } catch (e) {
      flash("err", errText(e));
    }
  };

  const teamValue = cfg.team_id == null ? "" : String(cfg.team_id);
  const manualSecret = !!cfg.enroll_secret.trim() && !cfg.team_name;
  const selectedTeam = scan?.teams.find((t) => (t.id ?? null) === (cfg.team_id ?? null));
  const missing: string[] = [];
  if (!info?.fleetctl_exists) missing.push("fleetctl build");
  if (!cfg.fleet_url.trim()) missing.push("Fleet URL");
  if (!cfg.enroll_secret.trim()) missing.push("enroll secret");
  if (cfg.platforms.length === 0) missing.push("a platform");
  if (cfg.update_source === UPDATE_LOCAL_TUF && !localTuf?.available) missing.push("local TUF");
  if (flagsErr) missing.push("valid extra flags");
  const canBuild = missing.length === 0 && !building;

  // Selected platforms whose file name already exists in the folder: building
  // replaces them (only on success, but the old file is gone afterwards).
  const existing = new Map(outputs.map((f) => [f.name, f]));
  const overwrites = cfg.platforms
    .map((k) => planned[k])
    .filter((name): name is string => !!name && existing.has(name))
    .map((name) => existing.get(name)!);

  const fleetUrl = cfg.fleet_url.trim();
  const loopback = /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])(:|\/|$)/.test(fleetUrl);
  const urlMatchesScan = !!scan?.server_url && fleetUrl.replace(/\/+$/, "") === scan.server_url.replace(/\/+$/, "");

  // Flag non-default agent options so a customized build isn't a surprise.
  const nonStable = [cfg.orbit_channel, cfg.desktop_channel, cfg.osqueryd_channel].some((ch) => ch && ch !== "stable");
  const customized = cfg.update_source === UPDATE_LOCAL_TUF || nonStable || cfg.debug || !!cfg.extra_flags.trim();

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)", borderTop: `3px solid ${accent}` }}>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)", flexWrap: "wrap" }}>
        <div className="section-title" style={{ margin: 0 }}>Build for</div>
        <select value={targetId} onChange={(e) => onTargetChange(e.target.value)} style={{ minWidth: 220 }}>
          {targets.map((t) => (
            <option key={t.id} value={t.id}>
              {t.label}
              {t.server ? ` (:${t.server.ports.server})` : ""}
              {buildingIds.has(t.id) ? " · building" : ""}
            </option>
          ))}
        </select>
        {server && <span className="dot" style={{ background: accent }} />}
        <div style={{ flex: 1 }} />
        {info && !info.fleetctl_exists && (
          <span style={{ color: "var(--ui-error)" }} title={info.fleetctl}>fleetctl not built</span>
        )}
      </div>

      {isCustom && (
        <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap" }}>
          <Field label="Name (used for the folder and file names)">
            <input value={cfg.name} placeholder="custom" onChange={(e) => set("name", e.target.value)} style={{ width: "100%" }} />
          </Field>
          <Field label="Build with fleetctl from">
            <select value={cfg.fleetctl} onChange={(e) => set("fleetctl", e.target.value)} style={{ width: "100%" }}>
              <option value="">{servers[0] ? `${servers[0].name} (default)` : "no server with a worktree"}</option>
              {servers.slice(1).map((srv) => (
                <option key={srv.id} value={srv.id}>{srv.name}</option>
              ))}
            </select>
          </Field>
        </div>
      )}

      {/* Connection: an optional context scan fills the URL and secret below */}
      <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap", alignItems: "flex-end" }}>
        <Field label={isCustom ? "fleetctl context (optional)" : "fleetctl context"}>
          {/* Preselects the context matching this server's port; only contexts with
              an address can scan, so others aren't offered. */}
          <select value={contextName} onChange={(e) => changeContext(e.target.value)} style={{ width: "100%" }}>
            {(isCustom || !contextName) && <option value="">{isCustom ? "none — enter URL and secret below" : "no context matches this server — pick one"}</option>}
            {contextName && !usableContexts.some((c) => c.name === contextName) && <option value={contextName}>{contextName}</option>}
            {usableContexts.map((c) => (
              <option key={c.name} value={c.name}>
                {c.name} — {c.address}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Fleet">
          <div style={{ display: "flex", gap: "var(--pad-small)" }}>
            <select
              value={manualSecret ? MANUAL_FLEET : teamValue}
              onChange={(e) => pickTeam(e.target.value)}
              disabled={!scan}
              style={{ flex: 1, minWidth: 0 }}
            >
              {/* A hand-entered secret belongs to no fleet Hangar knows about, so
                  the dropdown says so instead of showing the last fleet picked. */}
              {manualSecret && <option value={MANUAL_FLEET}>— secret entered by hand</option>}
              {!scan && !manualSecret && (
                <option value={teamValue}>
                  {cfg.team_name || (contextName ? "click Scan to list fleets" : "no context — enter the secret below")}
                </option>
              )}
              {scan && !manualSecret && !selectedTeam && <option value={teamValue}>{cfg.team_name || "?"} (not found)</option>}
              {scan?.teams.map((t) => (
                <option key={t.id ?? "none"} value={t.id == null ? "" : String(t.id)} disabled={!t.has_secret}>
                  {t.name}
                  {t.has_secret ? "" : t.secret_hidden ? " (secret hidden from this user)" : " (no enroll secret)"}
                </option>
              ))}
            </select>
            {/* The label stays in the layout (hidden) under the spinner so the button never changes size. */}
            <button
              onClick={runScan}
              disabled={!contextName || scanning}
              aria-label={scanning ? "Scanning" : "Scan"}
              title={contextName ? "List this server's fleets via the fleetctl context" : "Scanning needs a fleetctl context — pick one, or enter the URL and secret below"}
              style={{ position: "relative" }}
            >
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

      {/* What gets baked into the packages — visible so it's easy to confirm */}
      <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap" }}>
        <Field
          label="Fleet URL"
          hint={
            loopback
              ? "Loopback address — other machines can't reach it. Use the public (e.g. ngrok) URL."
              : scan?.server_url
                ? urlMatchesScan
                  ? "✓ matches the server's server_url"
                  : `differs from the server's server_url (${scan.server_url})`
                : undefined
          }
          warn={loopback}
        >
          <input value={cfg.fleet_url} placeholder="https://fleet.example.com" onChange={(e) => set("fleet_url", e.target.value)} style={{ width: "100%" }} />
        </Field>
        <Field
          label="Enroll secret"
          hint={cfg.enroll_secret.trim() ? (cfg.team_name ? `from ${cfg.team_name}` : "entered by hand") : undefined}
        >
          <div style={{ display: "flex", gap: "var(--pad-small)" }}>
            <input
              type={showSecret ? "text" : "password"}
              value={cfg.enroll_secret}
              placeholder="scan and pick a fleet, or paste one"
              onChange={(e) => update({ enroll_secret: e.target.value, team_id: null, team_name: "" })}
              autoComplete="off"
              spellCheck={false}
              style={{ flex: 1, minWidth: 0 }}
            />
            <button onClick={() => setShowSecret((v) => !v)} aria-pressed={showSecret}>{showSecret ? "Hide" : "Show"}</button>
          </div>
        </Field>
      </div>

      {/* Platforms on the left, agent options on the right (stacked when narrow) */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(420px, 1fr))", gap: "var(--pad-large)", alignItems: "start" }}>
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
          <label style={{ display: "flex", alignItems: "center", gap: 6, marginTop: 4 }}>
            <input type="checkbox" checked={cfg.fleet_desktop} onChange={(e) => set("fleet_desktop", e.target.checked)} />
            Include Fleet Desktop
          </label>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: "var(--pad-small)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <span style={{ fontWeight: 600 }}>Agent options</span>
            {customized && <span style={{ color: "var(--ui-warning)" }}>· not the defaults</span>}
          </div>
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
          <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <input type="checkbox" checked={cfg.debug} onChange={(e) => set("debug", e.target.checked)} />
            Debug logging in orbit
          </label>
          <Field
            label="Extra fleetctl package flags"
            full
            hint={flagsErr || "Added to every build, e.g. --enable-scripts --host-identifier=instance. Quote values with spaces."}
            warn={!!flagsErr}
          >
            <input
              className="mono"
              value={cfg.extra_flags}
              placeholder="--enable-scripts"
              onChange={(e) => set("extra_flags", e.target.value)}
              spellCheck={false}
              style={{ width: "100%" }}
            />
          </Field>
        </div>
      </div>

      {/* Build */}
      <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap", borderTop: "1px solid var(--app-border)", paddingTop: "var(--pad-smedium)" }}>
        {building ? (
          <button className="danger" onClick={stop}>Stop</button>
        ) : (
          <button className="primary" onClick={build} disabled={!canBuild}>
            Generate {cfg.platforms.length} installer{cfg.platforms.length === 1 ? "" : "s"}
          </button>
        )}
        <BuildStatus proc={proc} building={building} statuses={statuses} />
        {!building && !canBuild && <span className="dim">needs: {missing.join(", ")}</span>}
        {!building && overwrites.length > 0 && (
          <span
            style={{ color: "var(--ui-warning)" }}
            title={overwrites.map((f) => `${f.name} (built ${new Date(f.mod_time_ms).toLocaleString()})`).join("\n")}
          >
            ⚠ replaces {overwrites.length} existing installer{overwrites.length === 1 ? "" : "s"} (newest built{" "}
            {new Date(Math.max(...overwrites.map((f) => f.mod_time_ms))).toLocaleString()})
          </span>
        )}
        <div style={{ flex: 1 }} />
        {entries.length > 0 && (
          <button className="link-btn" onClick={() => setShowLog((v) => !v)}>{showLog ? "hide output" : "show output"}</button>
        )}
      </div>
      {showLog && <LogBox entries={entries} maxHeight={260} />}

      {/* What's in this target's folder, with how each file was built (from manifest.json) */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
          {outputs.length === 0 ? (
            <span className="dim">No installers in this target's folder yet</span>
          ) : (
            <button
              className="link-btn"
              onClick={() => setShowFiles((v) => !v)}
              aria-expanded={showFiles}
              style={{ display: "flex", alignItems: "center", gap: 6, padding: 0 }}
            >
              <span style={{ fontWeight: 600, color: "var(--app-text)" }}>{showFiles ? "▾" : "▸"} Files</span>
              <span className="dim">
                {outputs.length} installer{outputs.length === 1 ? "" : "s"} · newest {new Date(outputs[0].mod_time_ms).toLocaleString()}
              </span>
            </button>
          )}
          <div style={{ flex: 1 }} />
          {info && outputs.length > 0 && (
            <button className="link-btn" onClick={() => api.openPath(info.output_dir, false).catch((e) => flash("err", errText(e)))} title={info.output_dir}>
              open folder
            </button>
          )}
        </div>
        {showFiles &&
          outputs.map((f) => (
            <div key={f.name} style={{ display: "flex", flexDirection: "column", paddingLeft: 14 }}>
              <div style={{ display: "grid", gridTemplateColumns: "1fr auto auto", gap: 8, alignItems: "center" }}>
                <span className="mono" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={f.path}>{f.name}</span>
                <span className="dim">{formatBytes(f.size)}</span>
                <span className="dim">{new Date(f.mod_time_ms).toLocaleString()}</span>
              </div>
              <span className="dim">{f.build ? describeBuild(f.build) : "no build details (not built by Hangar, or built before it kept a manifest)"}</span>
            </div>
          ))}
      </div>
    </div>
  );
}

function OutputShareCard({
  outputDir,
  defaultDir,
  outputRoot,
  onOutputDirChange,
  onChooseDir,
  port,
  onPortChange,
  flash,
}: {
  outputDir: string;
  defaultDir: string;
  outputRoot: string;
  onOutputDirChange: (dir: string) => void;
  onChooseDir: () => void;
  port: number;
  onPortChange: (port: number) => void;
  flash: (kind: "ok" | "err", msg: string) => void;
}) {
  const [status, setStatus] = useState<InstallersShareStatus | null>(null);
  const [files, setFiles] = useState<InstallersSharedFile[]>([]);
  const [suggested, setSuggested] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const busyRef = useRef(false);
  const [portDraft, setPortDraft] = useState(port ? String(port) : "");
  const [showFiles, setShowFiles] = useState(false);
  const running = !!status?.running;

  // Every refresh/start/stop takes a sequence number; a response that isn't the
  // latest is dropped, so a poll begun before Stop can't flip the card back to
  // "sharing".
  const seq = useRef(0);
  const refresh = useCallback(async () => {
    const id = ++seq.current;
    try {
      const st = await api.installersShareStatus();
      // Listed either way: with install commands while sharing, plain otherwise.
      const list = await api.installersSharedFiles(outputDir).catch(() => [] as InstallersSharedFile[]);
      if (id !== seq.current) return;
      setStatus(st);
      setFiles(list);
    } catch (e) {
      console.error("share status", e);
    }
  }, [outputDir]);
  useEffect(() => {
    refresh();
    const poll = window.setInterval(() => {
      if (!busyRef.current) refresh();
    }, 3000);
    return () => window.clearInterval(poll);
  }, [refresh]);

  // Suggest a free port whenever sharing is known to be off (ports come and go).
  const suggest = useCallback(() => {
    api.installersSuggestSharePort().then(setSuggested).catch(() => setSuggested(null));
  }, []);
  const knownStopped = status !== null && !status.running;
  useEffect(() => {
    if (knownStopped) suggest();
  }, [knownStopped, suggest]);

  const act = async (fn: () => Promise<void>) => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    seq.current++;
    try {
      await fn();
    } finally {
      busyRef.current = false;
      setBusy(false);
      await refresh();
    }
  };
  const start = () =>
    act(async () => {
      try {
        await api.installersShareStart(outputDir, port);
      } catch (e) {
        flash("err", errText(e));
        suggest();
      }
    });
  const stop = () =>
    act(async () => {
      try {
        await api.installersShareStop();
      } catch (e) {
        flash("err", errText(e));
      }
    });

  // Empty means auto; anything else must be a whole port number before it's saved.
  const portValid = portDraft === "" || (/^\d+$/.test(portDraft) && Number(portDraft) >= 1 && Number(portDraft) <= 65535);
  const onPortInput = (v: string) => {
    setPortDraft(v);
    if (v === "") onPortChange(0);
    else if (/^\d+$/.test(v) && Number(v) >= 1 && Number(v) <= 65535) onPortChange(Number(v));
  };
  const copy = async (text: string, label: string) => {
    try {
      await copyText(text);
      flash("ok", `Copied ${label}`);
    } catch {
      flash("err", "Clipboard unavailable");
    }
  };

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)" }}>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
        <div className="section-title" style={{ margin: 0 }}>Output &amp; sharing</div>
        <StatusPill label="HTTP server" up={running} upText="sharing" downText="not sharing" />
        <div style={{ flex: 1 }} />
        <span className="dim">
          builds with each server's own <span className="mono">fleetctl package</span>
        </span>
      </div>

      {/* Folder: locked while sharing, since the server keeps serving the folder it started with. */}
      <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap" }}>
        <input
          value={outputDir}
          placeholder={defaultDir}
          onChange={(e) => onOutputDirChange(e.target.value)}
          disabled={running}
          title={running ? "Stop sharing to change the folder" : undefined}
          style={{ flex: 1, minWidth: 260 }}
        />
        <button onClick={onChooseDir} disabled={running}>Choose…</button>
        <button onClick={() => outputRoot && api.openPath(outputRoot, false).catch((e) => flash("err", errText(e)))} disabled={!outputRoot}>
          Open
        </button>
        {outputDir.trim() && !running && <button className="link-btn" onClick={() => onOutputDirChange("")}>use default</button>}
      </div>

      {/* Sharing controls on one line. Like the SCEP and TUF servers, Start/Stop
          sit at the right at a fixed width, so the button doesn't move. */}
      {running && status ? (
        <div style={SHARE_ROW}>
          <span className="mono" style={{ fontSize: "var(--fs-x-small)", fontWeight: 600, userSelect: "all" }}>{status.url}</span>
          <button onClick={() => copy(status.url, "URL")}>Copy URL</button>
          <button onClick={() => api.openUrl(status.url).catch((e) => flash("err", errText(e)))}>Open in browser</button>
          <div style={{ flex: 1 }} />
          <button className="danger" onClick={stop} disabled={busy} style={SHARE_BUTTON}>Stop sharing</button>
        </div>
      ) : (
        <div style={SHARE_ROW}>
          <span>Port</span>
          <input
            inputMode="numeric"
            value={portDraft}
            placeholder={suggested ? `${suggested} (free)` : "auto"}
            onChange={(e) => onPortInput(e.target.value.trim())}
            aria-invalid={!portValid}
            title={portValid ? "Leave empty to use a free port" : "Enter a port from 1 to 65535"}
            style={{ width: 120, borderColor: portValid ? undefined : "var(--ui-error)" }}
          />
          {!portValid && <span style={{ color: "var(--ui-error)" }}>enter a port from 1 to 65535</span>}
          <div style={{ flex: 1 }} />
          <button className="primary" onClick={start} disabled={busy || !outputRoot || !portValid} style={SHARE_BUTTON}>Start sharing</button>
        </div>
      )}
      <span className="dim">
        Sharing serves this folder read-only to your network. Installers contain enroll secrets, so stop sharing when you're done. macOS may ask
        once to allow incoming connections for Hangar.
      </span>

      {/* The folder's installers: plain while not sharing, with copyable install
          commands once the server is up (they need its URL). */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <button
          className="link-btn"
          onClick={() => setShowFiles((v) => !v)}
          aria-expanded={showFiles}
          style={{ display: "flex", alignItems: "center", gap: 6, alignSelf: "flex-start", padding: 0 }}
        >
          <span style={{ fontWeight: 600, color: "var(--app-text)" }}>
            {showFiles ? "▾" : "▸"} {running ? "Install commands" : "Installers"}
          </span>
          <span className="dim">
            {files.length} installer{files.length === 1 ? "" : "s"}
            {!running && files.length > 0 ? " · start sharing for install commands" : ""}
          </span>
        </button>
        {!showFiles ? null : files.length === 0 ? (
          <span className="dim">no installers in the folder yet</span>
        ) : (
          files.map((f) => (
            <div key={f.path} style={{ display: "grid", gridTemplateColumns: "1fr auto auto", gap: 8, alignItems: "center" }}>
              <span className="mono" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={f.url || f.path}>{f.path}</span>
              <span className="dim">{formatBytes(f.size)}</span>
              {running && f.command ? (
                <button className="link-btn" onClick={() => copy(f.command, "install command")} title={f.command}>
                  copy {f.shell === "powershell" ? "PowerShell (admin)" : "shell"} command
                </button>
              ) : (
                <span />
              )}
            </div>
          ))
        )}
      </div>

      {running && status && (
        <>
          {status.recent.length > 0 && (
            <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
              <span style={{ fontWeight: 600 }}>Recent downloads</span>
              {status.recent.map((d, i) => (
                <span key={`${i}-${d.at_ms}-${d.path}`} className="dim">
                  {new Date(d.at_ms).toLocaleTimeString()} · <span className="mono">{d.client}</span> · <span className="mono">{d.path}</span>
                </span>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

// describeBuild summarizes a manifest entry: fleet, update server, channels and
// options, e.g. "💻 Workstations · official updates · stable · Fleet Desktop".
function describeBuild(b: InstallersManifestEntry): string {
  const parts = [b.team || "Unassigned"];
  let host = "";
  try {
    host = new URL(b.update_url).host;
  } catch {
    host = b.update_url;
  }
  parts.push(host === "updates.fleetdm.com" ? "official updates" : `TUF ${host}`);
  const custom = [
    ["orbit", b.orbit_channel],
    ["desktop", b.desktop_channel],
    ["osquery", b.osqueryd_channel],
  ].filter(([, ch]) => ch && ch !== "stable");
  parts.push(custom.length === 0 ? "stable" : custom.map(([c, ch]) => `${c} ${ch}`).join(", "));
  parts.push(b.fleet_desktop ? "Fleet Desktop" : "no Desktop");
  if (b.debug) parts.push("debug");
  if (b.extra_flags) parts.push(b.extra_flags);
  if (b.fleet_url) parts.push(b.fleet_url);
  return parts.join(" · ");
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

function BuildStatus({
  proc,
  building,
  statuses,
}: {
  proc: ProcInfo | undefined;
  building: boolean;
  statuses: Record<string, JobStatus>;
}) {
  const values = Object.values(statuses);
  const ok = values.filter((v) => v === "ok").length;
  const failed = values.filter((v) => v === "failed" || (v === "building" && !building && !proc?.was_user_stopped)).length;
  if (building) {
    return (
      <span className="dim" style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
        <span className="dot run" /> building… {ok > 0 ? `${ok} done` : "(.msi builds use Docker and take a few minutes)"}
      </span>
    );
  }
  if (!proc) return null;
  if (proc.was_user_stopped) return <span className="dim">stopped{ok > 0 ? ` · ${ok} built` : ""}</span>;
  return (
    <span style={{ display: "inline-flex", gap: 8, fontWeight: 600 }}>
      {ok > 0 && <span style={{ color: "var(--core-fleet-green)" }}>✓ {ok} built</span>}
      {failed > 0 && <span style={{ color: "var(--ui-error)" }}>✗ {failed} failed</span>}
    </span>
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
