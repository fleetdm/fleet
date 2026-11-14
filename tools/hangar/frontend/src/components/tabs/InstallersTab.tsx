import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { listen } from "../../lib/events";
import { Toast } from "../Toast";
import { StatusPill } from "../StatusPill";
import { copyText } from "../../lib/clipboard";
import { LogBox } from "../LogLines";
import {
  api,
  type ContextSummary,
  type InstallerOptions,
  type InstallerTarget,
  type InstallersChannels,
  type InstallersContextInfo,
  type InstallersLocalTUF,
  type InstallersManifestEntry,
  type InstallersOutputFile,
  type InstallersScanResult,
  type InstallersShareStatus,
  type InstallersSharedFile,
  type LogEntry,
  type LogLine,
  type ProcInfo,
  type Settings,
} from "../../lib/ipc";
import {
  INSTALLERS_CHANNEL,
  INSTALLERS_PROC_ID,
  INSTALLER_PLATFORMS,
  UPDATE_LOCAL_TUF,
  UPDATE_OFFICIAL,
  emptyInstallerTarget,
  formatBytes,
  jobStatuses,
  type JobStatus,
} from "../../lib/installers";

const LOG_LEVELS = ["debug", "info", "warn", "error"];
// The share controls row swaps between the port box and the URL + buttons;
// a fixed minimum height keeps the card from shifting when it does.
const SHARE_ROW: React.CSSProperties = { display: "flex", alignItems: "center", gap: "var(--pad-small)", flexWrap: "wrap", minHeight: 40 };
const SHARE_BUTTON: React.CSSProperties = { minWidth: 130 };
// Fleet dropdown values: the enroll secret was typed by hand, or no fleet has
// been picked yet. Real fleets use fleetOptionValue.
const MANUAL_FLEET = "__manual__";
const NO_FLEET = "__none__";
// Context dropdown value for building without a fleetctl context.
const NO_CONTEXT = "__none__";

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
  // null until ~/.fleet/config has been read, so the card doesn't mount in
  // manual mode first and then jump to a context.
  const [contexts, setContexts] = useState<ContextSummary[] | null>(null);
  const [contextsErr, setContextsErr] = useState("");
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

  const loadContexts = useCallback(() => {
    api
      .fleetctlReadContext()
      .then((c) => {
        setContexts(c.contexts);
        setContextsErr("");
      })
      .catch((e) => {
        setContexts([]);
        setContextsErr(errText(e));
      });
  }, []);
  useEffect(() => {
    loadContexts();
    api.installersOfficialChannels().then(setOfficial).catch(() => {});
    const loadTuf = () => api.installersLocalTuf().then(setLocalTuf).catch(() => {});
    loadTuf();
    const poll = window.setInterval(loadTuf, 10000);
    return () => window.clearInterval(poll);
  }, [loadContexts]);

  // Updates build on the latest settings (not the render-time prop), so a
  // scan finishing while the user types doesn't overwrite either change.
  // Saves are debounced so typing doesn't race one disk write per keystroke.
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

  // Only contexts with an address can be scanned or built for.
  const usable = (contexts ?? []).filter((c) => !!c.address);
  const manual = cfg.manual || usable.length === 0;
  // Until a context is picked here, use "default" (fleetctl's own default).
  const contextName = manual
    ? ""
    : usable.some((c) => c.name === cfg.context)
      ? cfg.context
      : (usable.find((c) => c.name === "default") ?? usable[0]).name;
  const target = manual ? cfg.manual_target : (cfg.targets[contextName] ?? emptyInstallerTarget());

  const selectContext = useCallback(
    (value: string) =>
      persist((s) => ({
        ...s,
        installers: value === NO_CONTEXT ? { ...s.installers, manual: true } : { ...s.installers, manual: false, context: value },
      })),
    [persist],
  );
  // Keyed by the context it was started for, so a slow scan of the previous
  // context can't write into the newly picked one.
  const setTarget = useCallback(
    (forContext: string, forManual: boolean, update: (t: InstallerTarget) => InstallerTarget) =>
      persist((s) => {
        const inst = s.installers;
        if (forManual) return { ...s, installers: { ...inst, manual_target: update(inst.manual_target) } };
        const cur = inst.targets[forContext] ?? emptyInstallerTarget();
        return { ...s, installers: { ...inst, targets: { ...inst.targets, [forContext]: update(cur) } } };
      }),
    [persist],
  );
  const setOptions = useCallback(
    (update: (o: InstallerOptions) => InstallerOptions) =>
      persist((s) => ({ ...s, installers: { ...s.installers, options: update(s.installers.options) } })),
    [persist],
  );
  const setSharePort = useCallback(
    (port: number) => persist((s) => ({ ...s, installers: { ...s.installers, share_port: port } })),
    [persist],
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
        {contexts === null ? (
          <div className="dim">Reading fleetctl contexts…</div>
        ) : (
          <BuildCard
            key={manual ? NO_CONTEXT : contextName}
            contextName={contextName}
            manual={manual}
            contexts={usable}
            contextsErr={contextsErr}
            onContextChange={selectContext}
            onReloadContexts={loadContexts}
            target={target}
            onTargetChange={(update) => setTarget(contextName, manual, update)}
            options={cfg.options}
            onOptionsChange={setOptions}
            official={official}
            localTuf={localTuf}
            proc={procs.find((p) => p.id === INSTALLERS_PROC_ID)}
            flash={flash}
          />
        )}

        <OutputShareCard
          settings={settings}
          port={cfg.share_port}
          onPortChange={setSharePort}
          flash={flash}
        />
      </div>
      {toast && <Toast kind={toast.kind} msg={toast.msg} />}
    </div>
  );
}

function BuildCard({
  contextName,
  manual,
  contexts,
  contextsErr,
  onContextChange,
  onReloadContexts,
  target,
  onTargetChange,
  options,
  onOptionsChange,
  official,
  localTuf,
  proc,
  flash,
}: {
  contextName: string;
  manual: boolean;
  contexts: ContextSummary[];
  contextsErr: string;
  onContextChange: (value: string) => void;
  onReloadContexts: () => void;
  target: InstallerTarget;
  onTargetChange: (update: (t: InstallerTarget) => InstallerTarget) => void;
  options: InstallerOptions;
  onOptionsChange: (update: (o: InstallerOptions) => InstallerOptions) => void;
  official: InstallersChannels | null;
  localTuf: InstallersLocalTUF | null;
  proc: ProcInfo | undefined;
  flash: (kind: "ok" | "err", msg: string) => void;
}) {
  const [info, setInfo] = useState<InstallersContextInfo | null>(null);
  const [showSecret, setShowSecret] = useState(false);
  const [flagsErr, setFlagsErr] = useState("");
  const [scan, setScan] = useState<InstallersScanResult | null>(null);
  const [scanErr, setScanErr] = useState("");
  const [secretErr, setSecretErr] = useState("");
  const [infoErr, setInfoErr] = useState("");
  const [scanning, setScanning] = useState(false);
  const [planned, setPlanned] = useState<Record<string, string>>({});
  const [outputs, setOutputs] = useState<InstallersOutputFile[]>([]);
  const [entries, setEntries] = useState<LogEntry[]>([]);
  const [showLog, setShowLog] = useState(false);
  const [showFiles, setShowFiles] = useState(false);

  // Async results (scan, fleet secret) merge into the latest target, not the
  // one from the render that started them.
  const targetRef = useRef(target);
  targetRef.current = target;
  const update = (patch: Partial<InstallerTarget>) => onTargetChange((t) => ({ ...t, ...patch }));
  const setOpt = <K extends keyof InstallerOptions>(k: K, v: InstallerOptions[K]) => onOptionsChange((o) => ({ ...o, [k]: v }));

  // One build runs at a time, for whichever context started it; its label says
  // which (built the same way in InstallersStartBuild).
  const building = proc?.state === "running" || proc?.state === "stopping";
  const ownProc = proc?.label === (manual ? "fleetd packages (manual)" : `fleetd packages (${contextName})`) ? proc : undefined;
  const ownBuilding = building && !!ownProc;
  const address = contexts.find((c) => c.name === contextName)?.address ?? "";

  // Re-read now and then: building fleetctl from the Server tab, or changing
  // the file server's directory, changes the answer.
  useEffect(() => {
    let cancelled = false;
    const load = () =>
      api
        .installersContextInfo(contextName, manual)
        .then((i) => {
          if (cancelled) return;
          setInfo(i);
          setInfoErr("");
        })
        .catch((e) => {
          if (!cancelled) setInfoErr(errText(e));
        });
    load();
    const poll = window.setInterval(load, 10000);
    return () => {
      cancelled = true;
      window.clearInterval(poll);
    };
  }, [contextName, manual, proc?.state]);

  const plannedKey = JSON.stringify([target.team_name, target.team_id, options]);
  useEffect(() => {
    let cancelled = false;
    api
      .installersPlannedFiles(target, options)
      .then((p) => !cancelled && setPlanned(p))
      .catch(() => !cancelled && setPlanned({}));
    return () => {
      cancelled = true;
    };
  }, [plannedKey]);

  useEffect(() => {
    if (!options.extra_flags.trim()) {
      setFlagsErr("");
      return;
    }
    let cancelled = false;
    api
      .installersCheckExtraFlags(options.extra_flags)
      .then((err) => !cancelled && setFlagsErr(err))
      .catch(() => !cancelled && setFlagsErr(""));
    return () => {
      cancelled = true;
    };
  }, [options.extra_flags]);

  // Reads the context's server: its server_url, license and fleets. A fleet
  // picked earlier gets its secret re-read (secrets can be rotated); on a first
  // visit it's the global (Unassigned) secret, like fleetctl's own examples. A
  // secret typed (or cleared) by hand is left alone. When the server can't be
  // read, the URL falls back to the context's address and everything stays
  // editable by hand.
  const scanSeq = useRef(0);
  const runScan = useCallback(async () => {
    if (manual || !contextName) return;
    const seq = ++scanSeq.current;
    setScanning(true);
    setSecretErr("");
    try {
      const res = await api.installersScan(contextName);
      if (seq !== scanSeq.current) return;
      setScan(res);
      setScanErr("");
      const cur = targetRef.current;
      const firstVisit = !cur.fleet_url.trim() && !cur.enroll_secret.trim() && !cur.team_name;
      if (!cur.fleet_url.trim()) update({ fleet_url: res.server_url || address });
      const team = cur.team_name
        ? res.teams.find((t) => (t.id ?? null) === (cur.team_id ?? null))
        : firstVisit
          ? res.teams.find((t) => t.id == null)
          : undefined;
      if (!team) return;
      if (!team.has_secret) {
        if (cur.team_name) update({ enroll_secret: "" });
        return;
      }
      let secret: string;
      try {
        secret = await api.installersFleetSecret(contextName, team.id ?? null);
      } catch (e) {
        if (seq === scanSeq.current) setSecretErr(errText(e));
        return;
      }
      const now = targetRef.current;
      // Typed, cleared or picked something else while the secret was read.
      if (seq !== scanSeq.current || now.team_name !== cur.team_name || now.enroll_secret !== cur.enroll_secret) return;
      update({ enroll_secret: secret, team_id: team.id ?? null, team_name: team.name });
    } catch (e) {
      if (seq !== scanSeq.current) return;
      setScan(null);
      setScanErr(errText(e));
      if (!targetRef.current.fleet_url.trim() && address) update({ fleet_url: address });
    } finally {
      if (seq === scanSeq.current) setScanning(false);
    }
  }, [contextName, manual, address]);

  // The card is keyed by context, so this runs whenever the context changes.
  useEffect(() => {
    runScan();
  }, [runScan]);

  const loadOutputs = useCallback(() => {
    api.installersListOutputs(contextName, manual).then(setOutputs).catch(() => setOutputs([]));
  }, [contextName, manual]);
  useEffect(() => {
    loadOutputs();
  }, [loadOutputs, proc?.state, info?.output_dir]);

  // Tail the build log (statuses come from build.sh's marker lines).
  const logSeq = useRef(0);
  const loadLog = useCallback(async () => {
    const seq = ++logSeq.current;
    try {
      const w = await api.readLogWindow({ source: INSTALLERS_CHANNEL, since_ms: 0, levels: LOG_LEVELS, max_lines: 5000 });
      if (seq === logSeq.current) setEntries(w.entries);
    } catch (e) {
      console.error("installers readLogWindow", e);
    }
  }, []);
  useEffect(() => {
    loadLog();
    // Builds stream many lines (docker pulls); reload at most every 300ms.
    let timer: number | undefined;
    let cancelled = false;
    let un: (() => void) | undefined;
    listen<LogLine>("proc:log", (e) => {
      if (e.payload.proc_id !== INSTALLERS_PROC_ID || timer !== undefined) return;
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
  }, [loadLog]);
  // A final reload when the build ends catches lines that raced the listener.
  useEffect(() => {
    loadLog();
  }, [proc?.state, loadLog]);

  // The log and per-platform results belong to whichever context built last.
  const statuses = useMemo(() => (ownProc ? jobStatuses(entries) : {}), [entries, ownProc]);
  // Open the output when this context's build fails while the tab is open —
  // not for an old failure on remount.
  const prevState = useRef(ownProc?.state);
  useEffect(() => {
    if (ownProc?.state === "failed" && prevState.current !== "failed") setShowLog(true);
    prevState.current = ownProc?.state;
  }, [ownProc?.state]);

  const channels: InstallersChannels | null = options.update_source === UPDATE_LOCAL_TUF ? (localTuf?.channels ?? null) : official;

  const setUpdateSource = (source: string) => {
    const list = source === UPDATE_LOCAL_TUF ? localTuf?.channels : official;
    const keep = (ch: string, opts: string[] | undefined) => (opts?.includes(ch) ? ch : "stable");
    onOptionsChange((o) => ({
      ...o,
      update_source: source,
      orbit_channel: keep(o.orbit_channel, list?.orbit),
      desktop_channel: keep(o.desktop_channel, list?.desktop),
      osqueryd_channel: keep(o.osqueryd_channel, list?.osqueryd),
    }));
  };

  const togglePlatform = (key: string) =>
    onOptionsChange((o) => ({
      ...o,
      platforms: o.platforms.includes(key) ? o.platforms.filter((p) => p !== key) : [...o.platforms, key],
    }));

  // Picking a fleet fills its enroll secret (and the Fleet URL if empty). The
  // fleet is only applied once its secret is in hand, so a failed read leaves
  // the old fleet and secret as they were, and a slower earlier pick can't land
  // after a later one.
  const pickSeq = useRef(0);
  const pickTeam = async (value: string) => {
    const team = scan?.teams.find((t) => fleetOptionValue(t.id ?? null) === value);
    if (!team?.has_secret) return;
    const seq = ++pickSeq.current;
    // A rescan finishing now must not re-read the previous fleet over this one.
    scanSeq.current++;
    setScanning(false);
    try {
      const secret = await api.installersFleetSecret(contextName, team.id ?? null);
      if (seq !== pickSeq.current) return;
      setSecretErr("");
      update({
        team_id: team.id ?? null,
        team_name: team.name,
        enroll_secret: secret,
        fleet_url: targetRef.current.fleet_url.trim() || scan?.server_url || address,
      });
    } catch (e) {
      if (seq === pickSeq.current) flash("err", errText(e));
    }
  };

  const build = async () => {
    try {
      setShowLog(false);
      await api.installersStartBuild(contextName, manual, target, options);
    } catch (e) {
      flash("err", errText(e));
    }
  };
  const stop = async () => {
    try {
      await api.installersStopBuild();
    } catch (e) {
      flash("err", errText(e));
    }
  };

  const manualSecret = !!target.enroll_secret.trim() && !target.team_name;
  const fleetValue = manualSecret ? MANUAL_FLEET : target.team_name ? fleetOptionValue(target.team_id ?? null) : NO_FLEET;
  const selectedTeam = target.team_name ? scan?.teams.find((t) => (t.id ?? null) === (target.team_id ?? null)) : undefined;
  const missing: string[] = [];
  if (!info) missing.push(infoErr ? `context details (${infoErr})` : "context details");
  if (info && !info.fleetctl_exists) missing.push("fleetctl build");
  if (info?.output_err) missing.push("a packages folder");
  if (!target.fleet_url.trim()) missing.push("Fleet URL");
  if (!target.enroll_secret.trim()) missing.push("enroll secret");
  if (options.platforms.length === 0) missing.push("a platform");
  if (options.update_source === UPDATE_LOCAL_TUF && !localTuf?.available) missing.push("local TUF");
  if (flagsErr) missing.push("valid extra flags");
  const canBuild = !!info && missing.length === 0 && !building;

  // Selected platforms whose file name already exists in the folder: building
  // replaces them (only on success, but the old file is gone afterwards).
  const existing = new Map(outputs.map((f) => [f.name, f]));
  const overwrites = options.platforms
    .map((k) => planned[k])
    .filter((name): name is string => !!name && existing.has(name))
    .map((name) => existing.get(name)!);

  const fleetUrl = target.fleet_url.trim();
  const loopback = /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])(:|\/|$)/.test(fleetUrl);
  const urlMatchesScan = !!scan?.server_url && fleetUrl.replace(/\/+$/, "") === scan.server_url.replace(/\/+$/, "");

  // Flag non-default agent options so a customized build isn't a surprise.
  const nonStable = [options.orbit_channel, options.desktop_channel, options.osqueryd_channel].some((ch) => ch && ch !== "stable");
  const customized =
    !options.fleet_desktop || !options.enable_scripts || options.update_source === UPDATE_LOCAL_TUF || nonStable || options.debug || !!options.extra_flags.trim();

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)" }}>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)", flexWrap: "wrap" }}>
        <div className="section-title" style={{ margin: 0 }}>Build fleetd packages for</div>
        <select value={manual ? NO_CONTEXT : contextName} onChange={(e) => onContextChange(e.target.value)} disabled={building} style={{ minWidth: 260 }}>
          {contexts.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name} — {c.address}
            </option>
          ))}
          <option value={NO_CONTEXT}>No context — enter the URL and secret by hand</option>
        </select>
        <button className="link-btn" onClick={onReloadContexts} title="Re-read ~/.fleet/config">
          reload contexts
        </button>
        <div style={{ flex: 1 }} />
        {!info && infoErr && <span style={{ color: "var(--ui-error)" }}>{infoErr}</span>}
        {info && (
          <span className={info.fleetctl_exists ? "dim" : undefined} style={{ color: info.fleetctl_exists ? undefined : "var(--ui-error)" }} title={info.fleetctl}>
            {info.fleetctl_exists
              ? info.fleetctl_server
                ? `builds with ${info.fleetctl_server}'s fleetctl`
                : "builds with Hangar's fleetctl"
              : "fleetctl not built — build it from the Server tab"}
          </span>
        )}
      </div>
      {contextsErr ? (
        <span style={{ color: "var(--ui-warning)" }}>Couldn't read ~/.fleet/config ({contextsErr}). Enter the URL and secret by hand, or reload contexts.</span>
      ) : (
        contexts.length === 0 && (
          <span className="dim">No fleetctl contexts with an address in ~/.fleet/config. Log in from the fleetctl tab to pick fleets from a server.</span>
        )
      )}

      {/* Where the enroll secret comes from: a fleet on the context's server, or typed in */}
      {!manual && (
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <Field label="Fleet">
            <div style={{ display: "flex", gap: "var(--pad-small)" }}>
              <select value={fleetValue} onChange={(e) => pickTeam(e.target.value)} disabled={!scan} style={{ flex: 1, minWidth: 0 }}>
                {/* A hand-entered secret belongs to no fleet Hangar knows about, so
                    the dropdown says so instead of showing the last fleet picked. */}
                {manualSecret && <option value={MANUAL_FLEET}>— secret entered by hand</option>}
                {!scan && !manualSecret && (
                  <option value={fleetValue}>{scanning ? "loading fleets…" : target.team_name || "fleets unavailable — enter the secret below"}</option>
                )}
                {scan && !manualSecret && !selectedTeam && (
                  <option value={fleetValue}>{target.team_name ? `${target.team_name} (not found)` : "pick a fleet"}</option>
                )}
                {scan?.teams.map((t) => (
                  <option key={t.id ?? "none"} value={fleetOptionValue(t.id ?? null)} disabled={!t.has_secret}>
                    {t.name}
                    {t.has_secret ? "" : t.secret_hidden ? " (secret hidden from this user)" : " (no enroll secret)"}
                  </option>
                ))}
              </select>
              {/* The label stays in the layout (hidden) under the spinner so the button never changes size. */}
              <button onClick={runScan} disabled={scanning} aria-label={scanning ? "Loading" : "Reload"} title="Read the fleets again" style={{ position: "relative" }}>
                <span style={{ visibility: scanning ? "hidden" : "visible" }}>Reload</span>
                {scanning && (
                  <span style={{ position: "absolute", inset: 0, display: "flex", alignItems: "center", justifyContent: "center" }}>
                    <span className="spin">↻</span>
                  </span>
                )}
              </button>
            </div>
          </Field>
          {scanErr ? (
            <span style={{ color: "var(--ui-warning)" }}>
              Couldn't read fleets through <span className="mono">{contextName}</span>: {scanErr}. You can still build. Check the Fleet URL and paste an
              enroll secret below.
            </span>
          ) : scan ? (
            <span className="dim">
              {scan.premium
                ? `Premium · ${scan.teams.length - 1} fleet${scan.teams.length === 2 ? "" : "s"}`
                : "Free · Unassigned only (fleets need Premium)"}
            </span>
          ) : null}
          {secretErr && (
            <span style={{ color: "var(--ui-warning)" }}>Couldn't read the enroll secret: {secretErr}. Pick another fleet or paste a secret below.</span>
          )}
        </div>
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
          <div style={{ display: "flex", gap: "var(--pad-small)" }}>
            <input value={target.fleet_url} placeholder="https://fleet.example.com" onChange={(e) => update({ fleet_url: e.target.value })} style={{ flex: 1, minWidth: 0 }} />
            {scan?.server_url && !urlMatchesScan && (
              <button onClick={() => update({ fleet_url: scan.server_url })} title={scan.server_url}>
                Use server_url
              </button>
            )}
          </div>
        </Field>
        <Field
          label="Enroll secret"
          hint={target.enroll_secret.trim() ? (target.team_name ? `from ${target.team_name}` : "entered by hand") : undefined}
        >
          <div style={{ display: "flex", gap: "var(--pad-small)" }}>
            <input
              type={showSecret ? "text" : "password"}
              value={target.enroll_secret}
              placeholder={manual ? "paste an enroll secret" : "pick a fleet, or paste one"}
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
            <button className="link-btn" onClick={() => setOpt("platforms", INSTALLER_PLATFORMS.map((p) => p.key))}>all</button>
            <button className="link-btn" onClick={() => setOpt("platforms", [])}>none</button>
          </div>
          {INSTALLER_PLATFORMS.map((p) => (
            <label key={p.key} style={{ display: "grid", gridTemplateColumns: "auto 190px 1fr auto", alignItems: "center", gap: 8 }}>
              <input type="checkbox" checked={options.platforms.includes(p.key)} onChange={() => togglePlatform(p.key)} />
              <span>{p.label}</span>
              <span className="mono dim" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={planned[p.key]}>
                {planned[p.key] ?? ""}
              </span>
              <JobBadge status={statuses[p.key]} active={ownBuilding} userStopped={!!ownProc?.was_user_stopped} />
            </label>
          ))}
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: "var(--pad-small)" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <span style={{ fontWeight: 600 }}>Agent options</span>
            {customized && <span style={{ color: "var(--ui-warning)" }}>· not the defaults</span>}
          </div>
          <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <input type="checkbox" checked={options.fleet_desktop} onChange={(e) => setOpt("fleet_desktop", e.target.checked)} />
            Include Fleet Desktop
          </label>
          <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <input type="checkbox" checked={options.enable_scripts} onChange={(e) => setOpt("enable_scripts", e.target.checked)} />
            Enable scripts (<span className="mono">--enable-scripts</span>)
          </label>
          <div style={{ display: "flex", gap: "var(--pad-large)", alignItems: "center", flexWrap: "wrap" }}>
            <span style={{ fontWeight: 600 }}>Updates</span>
            <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
              <input type="radio" checked={options.update_source !== UPDATE_LOCAL_TUF} onChange={() => setUpdateSource(UPDATE_OFFICIAL)} />
              Official (updates.fleetdm.com)
            </label>
            <label style={{ display: "flex", alignItems: "center", gap: 6 }} title={localTuf?.available ? localTuf.url : localTuf?.reason}>
              <input
                type="radio"
                checked={options.update_source === UPDATE_LOCAL_TUF}
                disabled={!localTuf?.available && options.update_source !== UPDATE_LOCAL_TUF}
                onChange={() => setUpdateSource(UPDATE_LOCAL_TUF)}
              />
              Local TUF
            </label>
          </div>
          {options.update_source === UPDATE_LOCAL_TUF && (
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
            <ChannelSelect label="orbit" value={options.orbit_channel} options={channels?.orbit} onChange={(v) => setOpt("orbit_channel", v)} />
            <ChannelSelect label="Fleet Desktop" value={options.desktop_channel} options={channels?.desktop} onChange={(v) => setOpt("desktop_channel", v)} />
            <ChannelSelect label="osquery" value={options.osqueryd_channel} options={channels?.osqueryd} onChange={(v) => setOpt("osqueryd_channel", v)} />
          </div>
          <label style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <input type="checkbox" checked={options.debug} onChange={(e) => setOpt("debug", e.target.checked)} />
            Debug logging in orbit
          </label>
          <Field
            label="Extra fleetctl package flags"
            full
            hint={flagsErr || "Added to every build, e.g. --host-identifier=instance. Quote values with spaces."}
            warn={!!flagsErr}
          >
            <input
              className="mono"
              value={options.extra_flags}
              placeholder="--host-identifier=instance"
              onChange={(e) => setOpt("extra_flags", e.target.value)}
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
            Generate {options.platforms.length} package{options.platforms.length === 1 ? "" : "s"}
          </button>
        )}
        {building && !ownProc ? (
          <span className="dim">{proc?.label} is building — one build at a time</span>
        ) : (
          <BuildStatus proc={ownProc} building={ownBuilding} statuses={statuses} />
        )}
        {!building && !canBuild && (info || infoErr) && <span className="dim">needs: {missing.join(", ")}</span>}
        {!building && overwrites.length > 0 && (
          <span
            style={{ color: "var(--ui-warning)" }}
            title={overwrites.map((f) => `${f.name} (built ${new Date(f.mod_time_ms).toLocaleString()})`).join("\n")}
          >
            ⚠ replaces {overwrites.length} existing package{overwrites.length === 1 ? "" : "s"} (newest built{" "}
            {new Date(Math.max(...overwrites.map((f) => f.mod_time_ms))).toLocaleString()})
          </span>
        )}
        <div style={{ flex: 1 }} />
        {ownProc && entries.length > 0 && (
          <button className="link-btn" onClick={() => setShowLog((v) => !v)}>{showLog ? "hide output" : "show output"}</button>
        )}
      </div>
      {ownProc && showLog && <LogBox entries={entries} maxHeight={260} />}

      {/* What's in this context's folder, with how each file was built (from manifest.json) */}
      <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
        <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
          {outputs.length === 0 ? (
            <span className="dim">No packages in this folder yet</span>
          ) : (
            <button
              className="link-btn"
              onClick={() => setShowFiles((v) => !v)}
              aria-expanded={showFiles}
              style={{ display: "flex", alignItems: "center", gap: 6, padding: 0 }}
            >
              <span style={{ fontWeight: 600, color: "var(--app-text)" }}>{showFiles ? "▾" : "▸"} Files</span>
              <span className="dim">
                {outputs.length} package{outputs.length === 1 ? "" : "s"} · newest {new Date(outputs[0].mod_time_ms).toLocaleString()}
              </span>
            </button>
          )}
          <div style={{ flex: 1 }} />
          {info?.output_err ? (
            <span style={{ color: "var(--ui-error)" }}>{info.output_err}</span>
          ) : (
            info && (
              <>
                <span className="mono dim" style={{ overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={info.output_dir}>
                  {info.output_dir}
                </span>
                {outputs.length > 0 && (
                  <button className="link-btn" onClick={() => api.openPath(info.output_dir, false).catch((e) => flash("err", errText(e)))}>
                    open folder
                  </button>
                )}
              </>
            )
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
  settings,
  port,
  onPortChange,
  flash,
}: {
  settings: Settings;
  port: number;
  onPortChange: (port: number) => void;
  flash: (kind: "ok" | "err", msg: string) => void;
}) {
  // The packages folder follows the file server's directory, else the primary
  // repo (see internal/installers.OutputDir).
  const fileServerDir = settings.python_server.directory ?? "";
  const primaryRepo = settings.servers[0]?.worktree_path ?? "";
  const [outputRoot, setOutputRoot] = useState("");
  const [rootErr, setRootErr] = useState("");
  useEffect(() => {
    api
      .installersPackagesRoot()
      .then((r) => {
        setOutputRoot(r);
        setRootErr("");
      })
      .catch((e) => {
        setOutputRoot("");
        setRootErr(errText(e));
      });
  }, [fileServerDir, primaryRepo]);
  const [lanIp, setLanIp] = useState("");
  useEffect(() => {
    api.lanIp().then(setLanIp).catch(() => setLanIp(""));
  }, []);
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
      const list = await api.installersSharedFiles().catch(() => [] as InstallersSharedFile[]);
      if (id !== seq.current) return;
      setStatus(st);
      setFiles(list);
    } catch (e) {
      console.error("share status", e);
    }
  }, [outputRoot]);
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
        await api.installersShareStart(port);
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
        <div className="section-title" style={{ margin: 0 }}>Packages folder &amp; sharing</div>
        <StatusPill label="HTTP server" up={running} upText="sharing" downText="not sharing" />
      </div>

      {/* One subfolder per context; manual builds go in the folder itself. */}
      <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap" }}>
        {rootErr ? (
          <span style={{ color: "var(--ui-error)" }}>{rootErr}</span>
        ) : (
          <span className="mono" style={{ flex: 1, minWidth: 260, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={outputRoot}>
            {outputRoot}
          </span>
        )}
        <button onClick={() => outputRoot && api.openPath(outputRoot, false).catch((e) => flash("err", errText(e)))} disabled={!outputRoot}>
          Open
        </button>
      </div>
      <span className="dim">
        {fileServerDir ? (
          <>
            Inside the file server's directory, so they're also at{" "}
            <span className="mono">http://{lanIp || "<this Mac>"}:{settings.python_server.port}/packages/</span> while python http.server runs.
          </>
        ) : (
          <>Inside the Fleet repo. Set a directory for python http.server in Settings to put packages there instead.</>
        )}
      </span>

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

function fleetOptionValue(id: number | null): string {
  return id == null ? "unassigned" : String(id);
}

// describeBuild summarizes a manifest entry: fleet, update server, channels and
// options, e.g. "💻 Workstations · official updates · stable · Fleet Desktop".
function describeBuild(b: InstallersManifestEntry): string {
  const parts = [b.team || "Unassigned"];
  if (b.context) parts.push(`via ${b.context}`);
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
  parts.push(b.enable_scripts ? "scripts" : "no scripts");
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
        <span className="dot run" /> building… {ok > 0 ? `${ok} done` : "(.msi builds use Docker and take the longest)"}
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
