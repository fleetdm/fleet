import { useCallback, useEffect, useState } from "react";
import { Toast } from "../Toast";
import {
  api,
  type MdmAssetsConfig,
  type MdmAssetsExportResult,
  type MdmAssetsFile,
} from "../../lib/ipc";
import { MDM_ASSET_NAMES, newMdmAssetsConfig } from "../../lib/mdmassets";
import { copyText } from "../../lib/clipboard";

export function MdmAssetsTab() {
  const [configs, setConfigs] = useState<MdmAssetsConfig[]>([]);
  const [current, setCurrent] = useState<MdmAssetsConfig | null>(null);
  const [defaultDir, setDefaultDir] = useState("");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<MdmAssetsExportResult | null>(null);
  const [showLog, setShowLog] = useState(false);
  const [toast, setToast] = useState<{ kind: "ok" | "err"; msg: string } | null>(null);

  const flash = useCallback((kind: "ok" | "err", msg: string) => {
    setToast({ kind, msg });
    window.setTimeout(() => setToast(null), 2600);
  }, []);

  useEffect(() => {
    (async () => {
      const [list, dir] = await Promise.all([
        api.mdmAssetsConfigsList().catch(() => [] as MdmAssetsConfig[]),
        api.mdmAssetsDefaultDir().catch(() => ""),
      ]);
      setDefaultDir(dir);
      setConfigs(list);
      setCurrent(list.length > 0 ? list[0] : newMdmAssetsConfig(dir));
    })();
  }, []);

  const set = useCallback(
    <K extends keyof MdmAssetsConfig>(k: K, v: MdmAssetsConfig[K]) =>
      setCurrent((cur) => (cur ? { ...cur, [k]: v } : cur)),
    [],
  );

  const selectConfig = useCallback(
    (id: string) => {
      if (id === "") {
        setCurrent(newMdmAssetsConfig(defaultDir));
        setResult(null);
        return;
      }
      const c = configs.find((c) => c.id === id);
      if (c) {
        setCurrent(c);
        setResult(null);
      }
    },
    [configs, defaultDir],
  );

  const saveConfig = useCallback(async () => {
    if (!current) return;
    try {
      const saved = await api.mdmAssetsConfigSave(current);
      const list = await api.mdmAssetsConfigsList();
      setConfigs(list);
      setCurrent(saved);
      flash("ok", "Config saved");
    } catch (e) {
      flash("err", errText(e));
    }
  }, [current, flash]);

  const deleteConfig = useCallback(async () => {
    if (!current) return;
    const isSaved = configs.some((c) => c.id === current.id);
    if (!isSaved) {
      setCurrent(newMdmAssetsConfig(defaultDir));
      return;
    }
    try {
      await api.mdmAssetsConfigDelete(current.id);
      const list = await api.mdmAssetsConfigsList();
      setConfigs(list);
      setCurrent(list.length > 0 ? list[0] : newMdmAssetsConfig(defaultDir));
      flash("ok", "Config deleted");
    } catch (e) {
      flash("err", errText(e));
    }
  }, [current, configs, defaultDir, flash]);

  const runExport = useCallback(async () => {
    if (!current) return;
    setBusy(true);
    setResult(null);
    try {
      const res = await api.mdmAssetsExport(current);
      setResult(res);
      if (res.exit_code === 0) {
        flash("ok", `Exported ${res.files.length} file(s)`);
      } else {
        flash("err", "Export failed — see log output");
        setShowLog(true);
      }
    } catch (e) {
      flash("err", errText(e));
    } finally {
      setBusy(false);
    }
  }, [current, flash]);

  const pickDir = useCallback(async () => {
    const dir = await api.pickFolder();
    if (dir) set("dir", dir);
  }, [set]);

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

  const copyFile = useCallback(
    async (f: MdmAssetsFile) => {
      try {
        const contents = await api.mdmAssetsReadFile(f.path);
        await copyText(contents);
        flash("ok", `Copied ${f.name}`);
      } catch (e) {
        flash("err", errText(e));
      }
    },
    [flash],
  );

  if (!current) return null;

  const currentIsSaved = configs.some((c) => c.id === current.id);
  const keyMissing = !current.key.trim();

  return (
    <div style={{ height: "100%", display: "flex", flexDirection: "column", fontSize: "var(--fs-xx-small)" }}>
      <div style={{ padding: "var(--pad-medium) var(--pad-medium) 0" }}>
        <div className="card" style={{ display: "flex", alignItems: "center", gap: 18, padding: "12px 16px", flexWrap: "wrap" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <span className="card-title">Saved config</span>
            <select value={currentIsSaved ? current.id : ""} onChange={(e) => selectConfig(e.target.value)} style={{ minWidth: 180 }}>
              {!currentIsSaved && <option value="">(unsaved)</option>}
              {configs.map((c) => (
                <option key={c.id} value={c.id}>{c.name || c.id}</option>
              ))}
            </select>
            <button onClick={() => selectConfig("")}>New</button>
            <button onClick={deleteConfig}>Delete</button>
          </div>
          <span style={{ color: "var(--app-border)" }}>│</span>
          <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
            runs <span className="mono">tools/mdm/assets export</span>
          </span>
        </div>
      </div>

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
        <div style={{ display: "grid", gridTemplateColumns: "repeat(auto-fit, minmax(340px, 1fr))", gap: "var(--pad-medium)", alignItems: "start" }}>
        {/* Form */}
        <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-smedium)" }}>
          <Field label="Name">
            <input value={current.name} onChange={(e) => set("name", e.target.value)} style={{ width: "100%" }} />
          </Field>

          <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap" }}>
            <Field label="DB user" grow><input value={current.db_user} onChange={(e) => set("db_user", e.target.value)} style={{ width: "100%" }} /></Field>
            <Field label="DB password" grow><input value={current.db_password} onChange={(e) => set("db_password", e.target.value)} style={{ width: "100%" }} /></Field>
          </div>
          <div style={{ display: "flex", gap: "var(--pad-medium)", flexWrap: "wrap" }}>
            <Field label="DB address" grow><input value={current.db_address} onChange={(e) => set("db_address", e.target.value)} style={{ width: "100%" }} /></Field>
            <Field label="DB name" grow><input value={current.db_name} onChange={(e) => set("db_name", e.target.value)} style={{ width: "100%" }} /></Field>
          </div>

          <Field label="Encryption key (server private key)" hint={keyMissing ? "Required for export." : undefined} warn={keyMissing}>
            <input value={current.key} placeholder="FLEET_SERVER_PRIVATE_KEY" onChange={(e) => set("key", e.target.value)} style={{ width: "100%" }} />
          </Field>

          <Field label="Output directory" hint="Where files are written; defaults to the Fleet repo root.">
            <div style={{ display: "flex", gap: "var(--pad-small)" }}>
              <input value={current.dir} placeholder={defaultDir || "(Fleet repo root)"} onChange={(e) => set("dir", e.target.value)} style={{ flex: 1 }} />
              <button onClick={pickDir}>Browse…</button>
            </div>
          </Field>

          <Field label="Asset" hint="Export a single asset, or all of them.">
            <select value={current.asset_name} onChange={(e) => set("asset_name", e.target.value)} style={{ width: 240 }}>
              <option value="">All assets</option>
              {MDM_ASSET_NAMES.map((n) => (<option key={n} value={n}>{n}</option>))}
            </select>
          </Field>

          <div style={{ display: "flex", gap: "var(--pad-small)", alignItems: "center", flexWrap: "wrap" }}>
            <button className="primary" onClick={runExport} disabled={busy || keyMissing}>
              {busy ? "Exporting…" : "Run export"}
            </button>
            <button onClick={saveConfig}>Save config</button>
            {busy && (
              <span className="dim" style={{ fontSize: "var(--fs-xx-small)", display: "inline-flex", alignItems: "center", gap: 6 }}>
                <span className="dot run" /> first run compiles the tool — may take a bit
              </span>
            )}
          </div>
        </div>

        {result ? (
          <ResultView result={result} onCopy={copy} onCopyFile={copyFile} showLog={showLog} onToggleLog={() => setShowLog((s) => !s)} />
        ) : (
          <div className="card" style={{ display: "flex", alignItems: "center", justifyContent: "center", textAlign: "center", color: "var(--app-text-dim)", minHeight: 140 }}>
            Run an export to see the generated files here.
          </div>
        )}
        </div>
      </div>

      {toast && <Toast kind={toast.kind} msg={toast.msg} />}
    </div>
  );
}

function ResultView({
  result,
  onCopy,
  onCopyFile,
  showLog,
  onToggleLog,
}: {
  result: MdmAssetsExportResult;
  onCopy: (text: string, label: string) => void;
  onCopyFile: (f: MdmAssetsFile) => void;
  showLog: boolean;
  onToggleLog: () => void;
}) {
  const ok = result.exit_code === 0;
  const env = result.stdout.trim();

  return (
    <div className="card" style={{ display: "flex", flexDirection: "column", gap: "var(--pad-small)" }}>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
        <span className={`dot ${ok ? "ok" : "fail"}`} />
        <span style={{ fontWeight: 600, color: ok ? "var(--core-fleet-green)" : "var(--ui-error)" }}>
          {ok ? "Export succeeded" : `Export failed${result.exit_code != null ? ` (exit ${result.exit_code})` : ""}`}
        </span>
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>{result.files.length} file(s)</span>
      </div>

      {result.files.length > 0 && (
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          {result.files.map((f) => (
            <div
              key={f.path}
              style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)", fontSize: "var(--fs-xx-small)", padding: "4px 0", borderTop: "1px solid var(--app-border)" }}
            >
              <span className="mono" style={{ minWidth: 150 }}>{f.name}</span>
              <span className="dim">{formatBytes(f.size)}</span>
              <span className="dim">{formatTime(f.mod_time_ms)}</span>
              <div style={{ flex: 1 }} />
              <button className="link-btn" onClick={() => onCopyFile(f)}>copy contents</button>
              <button className="link-btn" onClick={() => onCopy(f.path, "path")}>copy path</button>
            </div>
          ))}
        </div>
      )}

      {env && (
        <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
          <div style={{ display: "flex", alignItems: "center", gap: "var(--pad-small)" }}>
            <div className="section-title" style={{ margin: 0 }}>Fleet config</div>
            <button className="link-btn" onClick={() => onCopy(env, "config")}>copy</button>
          </div>
          <pre style={preStyle}>{env}</pre>
        </div>
      )}

      <button className="link-btn" style={{ alignSelf: "flex-start" }} onClick={onToggleLog}>
        {showLog ? "Hide log output" : "Show log output"}
      </button>
      {showLog && <pre style={preStyle}>{result.stderr || "(no log output)"}</pre>}
    </div>
  );
}

function Field({ label, hint, warn, grow, children }: { label: string; hint?: string; warn?: boolean; grow?: boolean; children: React.ReactNode }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 4, flex: grow ? 1 : undefined, minWidth: grow ? 200 : undefined }}>
      <span style={{ fontSize: "var(--fs-xx-small)", fontWeight: 600 }}>{label}</span>
      {children}
      {hint ? (
        <span className={warn ? undefined : "dim"} style={{ fontSize: "var(--fs-xxx-small)", color: warn ? "var(--ui-error)" : undefined }}>{hint}</span>
      ) : null}
    </div>
  );
}

const preStyle: React.CSSProperties = {
  margin: 0,
  background: "var(--log-bg)",
  border: "1px solid var(--app-border)",
  borderRadius: "var(--radius-md)",
  padding: "var(--pad-small)",
  fontFamily: "var(--font-mono)",
  fontSize: "var(--fs-xx-small)",
  lineHeight: 1.5,
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
  maxHeight: 260,
  overflowY: "auto",
};

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / (1024 * 1024)).toFixed(1)} MB`;
}

function formatTime(ms: number): string {
  if (!ms) return "";
  return new Date(ms).toLocaleString(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

function errText(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "string") return e;
  return String(e);
}
