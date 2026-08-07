import { useCallback, useEffect, useMemo, useState } from "react";
import { listen } from "../../lib/events";
import { Toast } from "../Toast";
import { LogBox } from "../LogLines";
import {
  api,
  type LogEntry,
  type LogLine,
  type ProcInfo,
  type ScepBinaryInfo,
  type ScepDepotInfo,
  type ScepInitCAParams,
  type ScepProfile,
  type Settings,
} from "../../lib/ipc";
import {
  removeScepProfile,
  scepChannel,
  scepPortConflict,
  scepProcId,
  scepUrl,
  upsertScepProfile,
} from "../../lib/scep";
import { copyText } from "../../lib/clipboard";

const LOG_LEVELS = ["debug", "info", "warn", "error"];

export function ScepTab({
  settings,
  onSettingsChange,
  procs,
}: {
  settings: Settings;
  onSettingsChange: (s: Settings) => void;
  procs: ProcInfo[];
}) {
  const profiles = settings.scep_profiles;
  const [binary, setBinary] = useState<ScepBinaryInfo | null>(null);
  const [building, setBuilding] = useState(false);
  const [lanIp, setLanIp] = useState("");
  const [depots, setDepots] = useState<Record<string, ScepDepotInfo>>({});
  const [editing, setEditing] = useState<{ profile: ScepProfile; isNew: boolean } | null>(null);
  const [initFor, setInitFor] = useState<ScepProfile | null>(null);
  const [openLogs, setOpenLogs] = useState<string | null>(null);
  const [toast, setToast] = useState<{ kind: "ok" | "err"; msg: string } | null>(null);

  const flash = useCallback((kind: "ok" | "err", msg: string) => {
    setToast({ kind, msg });
    window.setTimeout(() => setToast(null), 2600);
  }, []);

  const persist = useCallback(
    (next: Settings) => {
      onSettingsChange(next);
      api.saveSettings(next).catch((e) => {
        console.error("save settings failed", e);
        flash("err", "Failed to save settings");
      });
    },
    [onSettingsChange, flash],
  );

  useEffect(() => {
    api.scepBinaryStatus().then(setBinary).catch(() => setBinary(null));
    api.scepLanIp().then(setLanIp).catch(() => setLanIp(""));
  }, []);

  const depotSig = useMemo(() => profiles.map((p) => `${p.id}:${p.depot_path}`).join("|"), [profiles]);
  const reloadDepot = useCallback((p: ScepProfile) => {
    api
      .scepProfileDepotInfo(p)
      .then((di) => setDepots((d) => ({ ...d, [p.id]: di })))
      .catch((e) => console.error("depot info failed", e));
  }, []);
  useEffect(() => {
    profiles.forEach(reloadDepot);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [depotSig]);

  const isRunning = useCallback(
    (id: string) => procs.some((p) => p.id === scepProcId(id) && (p.state === "running" || p.state === "stopping")),
    [procs],
  );

  const buildBinary = useCallback(async () => {
    setBuilding(true);
    try {
      const info = await api.scepRebuildBinary();
      setBinary(info);
      flash("ok", "Built scepserver");
    } catch (e) {
      flash("err", `Build failed: ${errText(e)}`);
    } finally {
      setBuilding(false);
    }
  }, [flash]);

  const start = useCallback(
    async (p: ScepProfile) => {
      try {
        await api.scepStartProfile(p);
        setOpenLogs(p.id);
        api.scepBinaryStatus().then(setBinary).catch(() => {});
      } catch (e) {
        flash("err", errText(e));
      }
    },
    [flash],
  );

  const stop = useCallback(
    async (p: ScepProfile) => {
      try {
        await api.scepStopProfile(p.id);
      } catch (e) {
        flash("err", errText(e));
      }
    },
    [flash],
  );

  const addProfile = useCallback(async () => {
    try {
      const p = await api.newScepProfile();
      setEditing({ profile: p, isNew: true });
    } catch (e) {
      flash("err", errText(e));
    }
  }, [flash]);

  const saveProfile = useCallback(
    (p: ScepProfile) => {
      persist(upsertScepProfile(settings, p));
      setEditing(null);
      reloadDepot(p);
    },
    [persist, settings, reloadDepot],
  );

  const deleteProfile = useCallback(
    async (p: ScepProfile) => {
      if (isRunning(p.id)) await stop(p);
      persist(removeScepProfile(settings, p.id));
    },
    [isRunning, stop, persist, settings],
  );

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

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column", fontSize: "var(--fs-xx-small)" }}>
      <ScepHeader binary={binary} building={building} lanIp={lanIp} onBuild={buildBinary} />

      <div style={{ flex: 1, minHeight: 0, overflowY: "auto", padding: "var(--pad-medium)" }}>
        {profiles.length === 0 ? (
          <EmptyState onAdd={addProfile} />
        ) : (
          <div style={{ display: "flex", flexDirection: "column", gap: "var(--pad-medium)" }}>
            {profiles.map((p) => (
              <ScepCard
                key={p.id}
                profile={p}
                depot={depots[p.id]}
                running={isRunning(p.id)}
                lanIp={lanIp}
                logsOpen={openLogs === p.id}
                onToggleLogs={() => setOpenLogs((cur) => (cur === p.id ? null : p.id))}
                onStart={() => start(p)}
                onStop={() => stop(p)}
                onEdit={() => setEditing({ profile: p, isNew: false })}
                onDelete={() => deleteProfile(p)}
                onInit={() => setInitFor(p)}
                onCopy={copy}
              />
            ))}
            <div>
              <button onClick={addProfile}>+ Add SCEP server</button>
            </div>
          </div>
        )}
      </div>

      {editing && (
        <ProfileModal
          initial={editing.profile}
          isNew={editing.isNew}
          settings={settings}
          onCancel={() => setEditing(null)}
          onSave={saveProfile}
        />
      )}
      {initFor && (
        <InitCaModal
          profile={initFor}
          onCancel={() => setInitFor(null)}
          onDone={(di) => {
            setDepots((d) => ({ ...d, [initFor.id]: di }));
            setInitFor(null);
            flash("ok", "CA created");
          }}
          onError={(msg) => flash("err", msg)}
        />
      )}

      {toast && <Toast kind={toast.kind} msg={toast.msg} />}
    </div>
  );
}

function ScepHeader({
  binary,
  building,
  lanIp,
  onBuild,
}: {
  binary: ScepBinaryInfo | null;
  building: boolean;
  lanIp: string;
  onBuild: () => void;
}) {
  return (
    <div style={{ padding: "var(--pad-medium) var(--pad-medium) 0" }}>
      <div className="card" style={{ display: "flex", alignItems: "center", gap: 18, padding: "12px 16px", flexWrap: "wrap" }}>
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          LAN IP · <span className="mono" style={{ color: "var(--app-text)" }}>{lanIp || "unknown"}</span>
        </span>
        <span style={{ color: "var(--app-border)" }}>│</span>
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          binary ·{" "}
          {binary?.exists ? (
            <span className="mono" style={{ color: "var(--app-text)" }}>built {shortTime(binary.built_at)}</span>
          ) : (
            <span className="mono" style={{ color: "var(--ui-error)" }}>not built</span>
          )}
        </span>
        <div style={{ flex: 1 }} />
        <button onClick={onBuild} disabled={building}>
          {building ? "Building…" : binary?.exists ? "↻ Rebuild" : "Build"}
        </button>
      </div>
    </div>
  );
}

function EmptyState({ onAdd }: { onAdd: () => void }) {
  return (
    <div style={{ maxWidth: 620, margin: "48px auto", display: "flex", flexDirection: "column", gap: "var(--pad-medium)", textAlign: "center" }}>
      <div style={{ fontSize: "var(--fs-medium)", fontWeight: 600 }}>No SCEP servers yet</div>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)", lineHeight: 1.6 }}>
        Run local SCEP CAs for QA using Fleet's in-repo scepserver (a fork of micromdm/scep). Add a
        server, then create its CA — each server uses its own depot, port, and challenge, and several
        can run at once so Fleet can register multiple Custom SCEP CAs simultaneously.
      </div>
      <div>
        <button className="primary" onClick={onAdd}>+ Add SCEP server</button>
      </div>
    </div>
  );
}

function ScepCard({
  profile,
  depot,
  running,
  lanIp,
  logsOpen,
  onToggleLogs,
  onStart,
  onStop,
  onEdit,
  onDelete,
  onInit,
  onCopy,
}: {
  profile: ScepProfile;
  depot: ScepDepotInfo | undefined;
  running: boolean;
  lanIp: string;
  logsOpen: boolean;
  onToggleLogs: () => void;
  onStart: () => void;
  onStop: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onInit: () => void;
  onCopy: (text: string, label: string) => void;
}) {
  const [confirmDel, setConfirmDel] = useState(false);
  const hasCa = depot?.exists ?? false;
  const url = scepUrl(lanIp, profile.port);

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-small)" }}>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
        <span className={`dot ${running ? "run" : "idle"}`} />
        <span className="card-title">{profile.name || profile.id}</span>
        <span className="mono dim">:{profile.port}</span>
        <div style={{ flex: 1 }} />
        {running ? (
          <button className="danger" onClick={onStop}>Stop</button>
        ) : (
          <button className="primary" onClick={onStart} disabled={!hasCa} title={hasCa ? "" : "Create a CA first"}>
            Start
          </button>
        )}
        <button onClick={onToggleLogs}>{logsOpen ? "Hide logs" : "Logs"}</button>
        <button onClick={onEdit} disabled={running}>Edit</button>
        {confirmDel ? (
          <>
            <button className="danger" onClick={onDelete}>Confirm</button>
            <button onClick={() => setConfirmDel(false)}>Cancel</button>
          </>
        ) : (
          <button onClick={() => setConfirmDel(true)}>Delete</button>
        )}
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 5, fontSize: "var(--fs-xx-small)" }}>
        {depot && depot.depot_path && (
          <Row label="Depot">
            <span className="mono">{depot.depot_path}</span>
            <button className="link-btn" onClick={() => onCopy(depot.depot_path, "depot path")}>copy</button>
            {hasCa && (
              <button className="link-btn" onClick={() => api.openPath(depot.depot_path, true).catch(() => {})}>reveal</button>
            )}
          </Row>
        )}

        {hasCa ? (
          <>
            <Row label="Issuer"><span className="mono">{depot?.issuer_dn}</span></Row>
            <Row label="Thumbprint">
              <span className="mono">{depot?.thumbprint}</span>
              <button className="link-btn" onClick={() => onCopy(depot?.thumbprint ?? "", "thumbprint")}>copy</button>
            </Row>
            <Row label="URL">
              <span className="mono">{url}</span>
              <button className="link-btn" onClick={() => onCopy(url, "SCEP URL")}>copy</button>
            </Row>
            <Row label="Challenge">
              <span className="mono">{profile.challenge || "(none)"}</span>
              {profile.challenge ? <button className="link-btn" onClick={() => onCopy(profile.challenge, "challenge")}>copy</button> : null}
            </Row>
            {depot?.not_after ? <Row label="Expires"><span className="mono">{shortTime(depot.not_after)}</span></Row> : null}
          </>
        ) : (
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <span className="dim">{depot?.error ? `Depot problem: ${depot.error}` : "No CA in this depot yet."}</span>
            <button onClick={onInit}>Init CA…</button>
          </div>
        )}
      </div>

      {logsOpen && <ScepLogView profileId={profile.id} />}
    </div>
  );
}

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
      <span className="dim" style={{ width: 88, flexShrink: 0 }}>{label}</span>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)", minWidth: 0, flexWrap: "wrap" }}>{children}</div>
    </div>
  );
}

function ScepLogView({ profileId }: { profileId: string }) {
  const source = scepChannel(profileId);
  const [entries, setEntries] = useState<LogEntry[]>([]);

  const load = useCallback(async () => {
    try {
      const w = await api.readLogWindow({ source, since_ms: 0, levels: LOG_LEVELS, max_lines: 500 });
      setEntries(w.entries);
    } catch (e) {
      console.error("scep readLogWindow", e);
    }
  }, [source]);

  useEffect(() => {
    load();
    const poll = window.setInterval(load, 2000);
    let cancelled = false;
    let un: (() => void) | undefined;
    listen<LogLine>("proc:log", (e) => {
      if (e.payload.proc_id === scepProcId(profileId)) load();
    }).then((u) => {
      if (cancelled) u();
      else un = u;
    });
    return () => {
      cancelled = true;
      window.clearInterval(poll);
      un?.();
    };
  }, [load, profileId]);

  return <LogBox entries={entries} maxHeight={220} />;
}

function ProfileModal({
  initial,
  isNew,
  settings,
  onCancel,
  onSave,
}: {
  initial: ScepProfile;
  isNew: boolean;
  settings: Settings;
  onCancel: () => void;
  onSave: (p: ScepProfile) => void;
}) {
  const [p, setP] = useState<ScepProfile>(initial);
  const portConflict = scepPortConflict(settings, p.id, p.port);
  const set = <K extends keyof ScepProfile>(k: K, v: ScepProfile[K]) => setP((cur) => ({ ...cur, [k]: v }));

  const pickDepot = async () => {
    const dir = await api.pickFolder();
    if (dir) set("depot_path", dir);
  };

  return (
    <ModalShell title={isNew ? "Add SCEP server" : "Edit SCEP server"} onCancel={onCancel}>
      <Field label="Name">
        <input value={p.name} onChange={(e) => set("name", e.target.value)} style={{ width: "100%" }} />
      </Field>
      <Field label="Depot" hint="Leave blank to use the managed default under app-data.">
        <div style={{ display: "flex", gap: "var(--pad-small)" }}>
          <input value={p.depot_path} placeholder="(managed default)" onChange={(e) => set("depot_path", e.target.value)} style={{ flex: 1 }} />
          <button onClick={pickDepot}>Browse…</button>
        </div>
      </Field>
      <Field label="Port" hint={portConflict ? "⚠ Another profile uses this port" : undefined} warn={portConflict}>
        <input type="number" value={p.port} onChange={(e) => set("port", Number(e.target.value) || 0)} style={{ width: 140 }} />
      </Field>
      <Field label="Challenge">
        <input value={p.challenge} onChange={(e) => set("challenge", e.target.value)} style={{ width: "100%" }} />
      </Field>
      <Field label="Allow renew (days)" hint="0 = always allow renewal.">
        <input type="number" value={p.allow_renew} onChange={(e) => set("allow_renew", Number(e.target.value) || 0)} style={{ width: 140 }} />
      </Field>
      <Field label="Extra flags">
        <input value={p.extra_flags} placeholder="-sign-server-attrs" onChange={(e) => set("extra_flags", e.target.value)} style={{ width: "100%" }} />
      </Field>
      <label style={{ display: "flex", alignItems: "center", gap: 8, fontSize: "var(--fs-xx-small)" }}>
        <input type="checkbox" checked={p.debug} onChange={(e) => set("debug", e.target.checked)} />
        Debug logging (-debug)
      </label>

      <ModalButtons
        onCancel={onCancel}
        confirmLabel="Save"
        confirmDisabled={!p.name.trim() || p.port <= 0}
        onConfirm={() => onSave({ ...p, name: p.name.trim(), depot_path: p.depot_path.trim() })}
      />
    </ModalShell>
  );
}

function InitCaModal({
  profile,
  onCancel,
  onDone,
  onError,
}: {
  profile: ScepProfile;
  onCancel: () => void;
  onDone: (di: ScepDepotInfo) => void;
  onError: (msg: string) => void;
}) {
  const [depot, setDepot] = useState("");
  const [params, setParams] = useState<ScepInitCAParams>({
    common_name: `Fleet ${profile.name || profile.id} SCEP CA`,
    organization: "Fleet Device Management Inc.",
    organizational_unit: "QA",
    country: "US",
    key_size: 2048,
    years: 10,
    key_password: "",
  });
  const [busy, setBusy] = useState(false);
  const set = <K extends keyof ScepInitCAParams>(k: K, v: ScepInitCAParams[K]) => setParams((cur) => ({ ...cur, [k]: v }));

  useEffect(() => {
    api.scepResolveDepot(profile).then(setDepot).catch((e) => onError(errText(e)));
  }, [profile, onError]);

  const run = async () => {
    setBusy(true);
    try {
      const di = await api.scepInitCa(depot, params);
      onDone(di);
    } catch (e) {
      onError(errText(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <ModalShell title={`Init CA — ${profile.name || profile.id}`} onCancel={onCancel}>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)", lineHeight: 1.5 }}>
        Creates a new CA in <span className="mono">{depot || "…"}</span>. This generates a fresh signing
        cert — its thumbprint must be set on every profile referencing it.
      </div>
      <Field label="Common name"><input value={params.common_name} onChange={(e) => set("common_name", e.target.value)} style={{ width: "100%" }} /></Field>
      <Field label="Organization"><input value={params.organization} onChange={(e) => set("organization", e.target.value)} style={{ width: "100%" }} /></Field>
      <Field label="Org. unit"><input value={params.organizational_unit} onChange={(e) => set("organizational_unit", e.target.value)} style={{ width: "100%" }} /></Field>
      <Field label="Country"><input value={params.country} onChange={(e) => set("country", e.target.value)} style={{ width: 100 }} /></Field>
      <Field label="Key size"><input type="number" value={params.key_size} onChange={(e) => set("key_size", Number(e.target.value) || 0)} style={{ width: 140 }} /></Field>
      <Field label="Years"><input type="number" value={params.years} onChange={(e) => set("years", Number(e.target.value) || 0)} style={{ width: 140 }} /></Field>

      <ModalButtons
        onCancel={onCancel}
        confirmLabel={busy ? "Creating…" : "Create CA"}
        confirmDisabled={busy || !params.common_name.trim() || !depot}
        onConfirm={run}
      />
    </ModalShell>
  );
}

function ModalShell({ title, onCancel, children }: { title: string; onCancel: () => void; children: React.ReactNode }) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      onClick={onCancel}
      style={{ position: "fixed", inset: 0, background: "var(--overlay-modal)", display: "flex", alignItems: "center", justifyContent: "center", zIndex: 1100 }}
    >
      <div
        className="card"
        onClick={(e) => e.stopPropagation()}
        style={{ width: 520, maxWidth: "92vw", maxHeight: "88vh", overflowY: "auto", padding: "var(--pad-large)", display: "flex", flexDirection: "column", gap: "var(--pad-small)", boxShadow: "var(--shadow-modal)" }}
      >
        <div style={{ fontSize: "var(--fs-medium)", fontWeight: 600, marginBottom: 4 }}>{title}</div>
        {children}
      </div>
    </div>
  );
}

function Field({ label, hint, warn, children }: { label: string; hint?: string; warn?: boolean; children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
      <span style={{ fontSize: "var(--fs-xx-small)", fontWeight: 600 }}>{label}</span>
      {children}
      {hint ? (
        <span className={warn ? undefined : "dim"} style={{ fontSize: "var(--fs-xxx-small)", color: warn ? "var(--ui-error)" : undefined }}>{hint}</span>
      ) : null}
    </div>
  );
}

function ModalButtons({ onCancel, onConfirm, confirmLabel, confirmDisabled }: { onCancel: () => void; onConfirm: () => void; confirmLabel: string; confirmDisabled?: boolean }) {
  return (
    <div style={{ display: "flex", gap: "var(--pad-small)", justifyContent: "flex-end", marginTop: "var(--pad-small)" }}>
      <button onClick={onCancel}>Cancel</button>
      <button className="primary" onClick={onConfirm} disabled={confirmDisabled}>{confirmLabel}</button>
    </div>
  );
}

function shortTime(iso: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  const opts: Intl.DateTimeFormatOptions = { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" };
  // Include the year when it isn't the current one — otherwise a 10-year CA
  // expiry (e.g. 2036) reads as if it already lapsed today.
  if (d.getFullYear() !== new Date().getFullYear()) opts.year = "numeric";
  return d.toLocaleString(undefined, opts);
}

function errText(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  return String(e);
}
