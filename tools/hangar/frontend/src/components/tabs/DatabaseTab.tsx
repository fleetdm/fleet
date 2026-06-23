import { useCallback, useEffect, useMemo, useState } from "react";
import {
  api,
  type BackupEntry,
  type ProcInfo,
  type Settings,
} from "../../lib/tauri";
import { noAutocorrect } from "../../lib/noAutocorrect";
import { startServe, waitForExit } from "../../lib/orchestration";
import type { DockerHealth, ServeStatus } from "../../lib/useSystemHealth";

const BACKUP_EXT = ".sql.gz";
// Subdirectory under the repo where DB dumps are written and read back.
// Must stay in sync with BACKUPS_DIRNAME in src-tauri/src/db.rs.
const BACKUPS_DIR = "db-backups";
const RESET_DROP_ID = "db-reset-drop";
const RESET_PREPARE_ID = "db-reset-prepare";
const BACKUP_PROC_ID = "db-backup";
const RESTORE_PROC_ID = "db-restore";

type BusyKind =
  | { kind: "backup" }
  | { kind: "restore"; name: string }
  | { kind: "delete"; name: string }
  | { kind: "reset" }
  | null;

export function DatabaseTab({
  repoPath,
  settings,
  currentBranch,
  procs,
  serve,
  docker,
  goToLogs,
}: {
  repoPath: string | null;
  settings: Settings;
  currentBranch: string | null;
  procs: ProcInfo[];
  serve: ServeStatus;
  docker: DockerHealth;
  goToLogs: () => void;
}) {
  const [backups, setBackups] = useState<BackupEntry[]>([]);
  const [backupsDir, setBackupsDir] = useState<string | null>(null);
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const [busy, setBusy] = useState<BusyKind>(null);
  const [error, setError] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());

  // Tick once a minute so the "2m ago" style timestamps stay fresh
  // without us having to refresh the whole list on every render.
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(id);
  }, []);

  const refresh = useCallback(async () => {
    if (!repoPath) return;
    try {
      // ensureBackupsDir is cheap and idempotent; doing it here keeps the
      // "first time" case from showing an empty list because the folder
      // doesn't exist yet.
      const dir = await api.dbEnsureBackupsDir(repoPath);
      setBackupsDir(dir);
      const list = await api.dbListBackups(repoPath);
      setBackups(list);
    } catch (e) {
      setError(String(e));
    }
  }, [repoPath]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // Keep the selected backup in sync with the list — if it disappears
  // (deleted, renamed externally) drop the selection rather than show
  // ghost actions for a row that isn't there.
  useEffect(() => {
    if (selectedName && !backups.some((b) => b.name === selectedName)) {
      setSelectedName(null);
    }
  }, [backups, selectedName]);

  const selected = useMemo(
    () => backups.find((b) => b.name === selectedName) ?? null,
    [backups, selectedName],
  );

  if (!repoPath) {
    return (
      <div
        style={{
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          color: "var(--app-text-dim)",
        }}
      >
        No Fleet repo configured · open Settings to pick one
      </div>
    );
  }

  // Whether the MySQL container is actually up — not just "the compose
  // project has at least one running container". Otherwise a partial
  // start (e.g. only redis up) would tell the user MySQL is ready and
  // restore/backup would fail at connect time.
  const mysqlUp = docker.containers.some(
    (c) =>
      (c.name === "mysql" ||
        c.name === "mysql_test" ||
        c.name.includes("mysql")) &&
      c.state === "running",
  );
  const lastBackup = backups[0] ?? null;

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        padding: "var(--pad-large)",
        gap: "var(--pad-medium)",
        overflow: "auto",
      }}
    >
      <StatusHeader
        mysqlUp={mysqlUp}
        serveUp={serve.up}
        serveOwned={
          procs.find((p) => p.id === "fleet-serve")?.state === "running" ||
          procs.find((p) => p.id === "fleet-serve")?.state === "stopping"
        }
        repoPath={repoPath}
        settings={settings}
        lastBackup={lastBackup}
        backupCount={backups.length}
        backupsDir={backupsDir}
        now={now}
        onRefresh={refresh}
        onReveal={
          backupsDir ? () => api.openPath(backupsDir, false) : undefined
        }
        setError={setError}
      />

      {error && (
        <div
          style={{
            color: "var(--ui-error)",
            fontSize: "var(--fs-xx-small)",
            padding: "6px 10px",
            background: "var(--tint-error-soft)",
            border: "1px solid var(--ui-error)",
            borderRadius: "var(--radius-md)",
          }}
        >
          {error}
        </div>
      )}

      <div
        style={{
          display: "grid",
          // minmax(0, 1fr) instead of 1fr — the latter resolves to
          // minmax(auto, 1fr), which lets long content (e.g. a long
          // typed backup name) push columns wider than 50%.
          gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr)",
          gap: "var(--pad-medium)",
          flex: 1,
          minHeight: 0,
        }}
      >
        <BackupsPanel
          backups={backups}
          selectedName={selectedName}
          onSelect={setSelectedName}
          busy={busy}
          selected={selected}
          mysqlUp={mysqlUp}
          serveUp={serve.up}
          onRestore={async (entry) => {
            await runRestore(entry, {
              repoPath,
              setBusy,
              setError,
              refresh,
              goToLogs,
            });
          }}
          onDelete={async (entry) => {
            await runDelete(entry, { repoPath, setBusy, setError, refresh });
          }}
        />

        <div
          style={{
            display: "flex",
            flexDirection: "column",
            gap: "var(--pad-medium)",
            minHeight: 0,
            minWidth: 0,
          }}
        >
          <NewBackupPanel
            repoPath={repoPath}
            currentBranch={currentBranch}
            mysqlUp={mysqlUp}
            busy={busy}
            onSaved={refresh}
            setBusy={setBusy}
            setError={setError}
            goToLogs={goToLogs}
          />
          <ResetPanel
            repoPath={repoPath}
            mysqlUp={mysqlUp}
            serveUp={serve.up}
            busy={busy}
            procs={procs}
            onReset={async () => {
              await runReset({
                repoPath,
                setBusy,
                setError,
                refresh,
                goToLogs,
              });
            }}
          />
        </div>
      </div>
    </div>
  );
}

function StatusHeader({
  mysqlUp,
  serveUp,
  serveOwned,
  repoPath,
  settings,
  lastBackup,
  backupCount,
  backupsDir,
  now,
  onRefresh,
  onReveal,
  setError,
}: {
  mysqlUp: boolean;
  serveUp: boolean;
  serveOwned: boolean;
  repoPath: string;
  settings: Settings;
  lastBackup: BackupEntry | null;
  backupCount: number;
  backupsDir: string | null;
  now: number;
  onRefresh: () => void;
  onReveal?: () => void;
  setError: (e: string | null) => void;
}) {
  const [serveBusy, setServeBusy] = useState<"starting" | "stopping" | null>(
    null,
  );

  async function stopServe() {
    setServeBusy("stopping");
    setError(null);
    try {
      await api.stopProcess("fleet-serve");
    } catch (e) {
      setError(String(e));
    }
    setServeBusy(null);
  }

  async function startServeAction() {
    setServeBusy("starting");
    setError(null);
    try {
      await startServe(repoPath, settings);
    } catch (e) {
      setError(String(e));
    }
    setServeBusy(null);
  }

  return (
    <div
      className="card"
      style={{
        display: "flex",
        alignItems: "center",
        gap: 18,
        padding: "12px 16px",
        flexWrap: "wrap",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span className={`dot ${mysqlUp ? "run" : "fail"}`} />
        <span style={{ fontWeight: 600 }}>
          MySQL {mysqlUp ? "up" : "down"}
        </span>
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          :3306 · via docker
        </span>
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span className={`dot ${serveUp ? "run" : "idle"}`} />
        <span style={{ fontWeight: 600 }}>
          fleet serve {serveUp ? "up" : "down"}
        </span>
        {serveUp && !serveOwned && (
          <span
            className="dim"
            style={{ fontSize: "var(--fs-xxx-small)" }}
          >
            external
          </span>
        )}
        {serveUp && serveOwned && (
          <button
            onClick={stopServe}
            disabled={serveBusy != null}
            className="danger"
            title="Stop so you can restore or reset"
            style={{ padding: "2px 10px", fontSize: "var(--fs-xxx-small)" }}
          >
            {serveBusy === "stopping" ? "stopping…" : "Stop"}
          </button>
        )}
        {!serveUp && (
          <button
            onClick={startServeAction}
            disabled={serveBusy != null}
            title="fleet serve --dev · assumes docker is up"
            style={{ padding: "2px 10px", fontSize: "var(--fs-xxx-small)" }}
          >
            {serveBusy === "starting" ? "starting…" : "Start"}
          </button>
        )}
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        last backup ·{" "}
        <span style={{ color: "var(--app-text)" }}>
          {lastBackup
            ? humanAgo(now - lastBackup.mtime_ms)
            : "none yet"}
        </span>
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        {backupCount} backup{backupCount === 1 ? "" : "s"}
      </div>
      <div style={{ marginLeft: "auto", display: "flex", gap: 6 }}>
        {onReveal && (
          <button
            onClick={onReveal}
            title={backupsDir ?? ""}
            style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
          >
            Reveal in Finder
          </button>
        )}
        <button
          onClick={onRefresh}
          style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
        >
          ↻ Refresh
        </button>
      </div>
    </div>
  );
}

function BackupsPanel({
  backups,
  selectedName,
  onSelect,
  busy,
  selected,
  mysqlUp,
  serveUp,
  onRestore,
  onDelete,
}: {
  backups: BackupEntry[];
  selectedName: string | null;
  onSelect: (name: string) => void;
  busy: BusyKind;
  selected: BackupEntry | null;
  mysqlUp: boolean;
  serveUp: boolean;
  onRestore: (b: BackupEntry) => void;
  onDelete: (b: BackupEntry) => void;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);

  // Whenever the selection changes, drop the inline-confirm state. Avoids
  // the "I selected a different row but delete still says CONFIRM?" trap.
  useEffect(() => {
    setConfirmDelete(false);
  }, [selectedName]);

  const restoreBlocked = !mysqlUp || serveUp || busy != null;
  const restoreReason = !mysqlUp
    ? "MySQL is down — start docker compose first"
    : serveUp
      ? "Stop fleet serve before restoring (it holds connections)"
      : undefined;

  return (
    <div
      className="card"
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 10,
        minHeight: 0,
      }}
    >
      <div className="section-title" style={{ margin: 0 }}>
        Backups
      </div>

      {backups.length === 0 ? (
        <div
          className="dim"
          style={{
            fontSize: "var(--fs-xx-small)",
            padding: "var(--pad-medium)",
            border: "1px dashed var(--app-border)",
            borderRadius: "var(--radius-md)",
            textAlign: "center",
          }}
        >
          No backups yet. Use the form on the right to create one.
        </div>
      ) : (
        <div
          style={{
            flex: 1,
            overflow: "auto",
            display: "flex",
            flexDirection: "column",
            gap: 4,
            minHeight: 0,
          }}
        >
          {backups.map((b) => (
            <BackupRow
              key={b.name}
              entry={b}
              selected={b.name === selectedName}
              onClick={() => onSelect(b.name)}
            />
          ))}
        </div>
      )}

      <div
        style={{
          borderTop: "1px solid var(--app-border)",
          paddingTop: 10,
          display: "flex",
          gap: 6,
          flexWrap: "wrap",
          alignItems: "center",
        }}
      >
        <button
          className="primary"
          disabled={!selected || restoreBlocked}
          onClick={() => selected && onRestore(selected)}
          title={selected ? restoreReason : "Select a backup first"}
          style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
        >
          {busy?.kind === "restore" && selected?.name === busy.name
            ? "restoring…"
            : "Restore selected"}
        </button>
        <button
          className="danger"
          disabled={!selected || busy != null}
          onClick={() => {
            if (!selected) return;
            if (!confirmDelete) {
              setConfirmDelete(true);
              return;
            }
            setConfirmDelete(false);
            onDelete(selected);
          }}
          style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
        >
          {confirmDelete ? "Click again to confirm" : "Delete"}
        </button>
      </div>
    </div>
  );
}

function BackupRow({
  entry,
  selected,
  onClick,
}: {
  entry: BackupEntry;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onClick();
        }
      }}
      style={{
        padding: "8px 10px",
        borderRadius: 6,
        cursor: "pointer",
        background: selected
          ? "var(--tint-success-soft)"
          : "var(--app-surface-2)",
        border: `1px solid ${selected ? "var(--core-fleet-green)" : "transparent"}`,
        display: "flex",
        flexDirection: "column",
        gap: 2,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          gap: 8,
        }}
      >
        <span
          className="mono"
          style={{
            fontSize: "var(--fs-xx-small)",
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
            flex: 1,
          }}
          title={entry.name}
        >
          {entry.name}
        </span>
        <span
          className="dim"
          style={{ fontSize: "var(--fs-xxx-small)", flexShrink: 0 }}
        >
          {humanSize(entry.size)}
        </span>
      </div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 6,
          fontSize: "var(--fs-xxx-small)",
          color: "var(--app-text-dim)",
        }}
      >
        <span>{humanDate(entry.mtime_ms)}</span>
        {entry.branch && (
          <>
            <span>·</span>
            <span
              className="mono"
              style={{ color: "var(--core-fleet-purple)" }}
            >
              {entry.branch}
            </span>
          </>
        )}
        {entry.note && (
          <>
            <span>·</span>
            <span style={{ fontStyle: "italic" }} title={entry.note}>
              {truncate(entry.note, 60)}
            </span>
          </>
        )}
      </div>
    </div>
  );
}

function NewBackupPanel({
  repoPath,
  currentBranch,
  mysqlUp,
  busy,
  onSaved,
  setBusy,
  setError,
  goToLogs,
}: {
  repoPath: string;
  currentBranch: string | null;
  mysqlUp: boolean;
  busy: BusyKind;
  onSaved: () => Promise<void>;
  setBusy: (b: BusyKind) => void;
  setError: (e: string | null) => void;
  goToLogs: () => void;
}) {
  const defaultStem = useMemo(() => {
    if (!currentBranch) return "backup";
    return `${currentBranch}__clean`;
  }, [currentBranch]);

  const [stem, setStem] = useState(defaultStem);
  const [note, setNote] = useState("");
  const [userEdited, setUserEdited] = useState(false);

  // Until the user touches the field, keep it in sync with the branch.
  useEffect(() => {
    if (!userEdited) setStem(defaultStem);
  }, [defaultStem, userEdited]);

  const finalName = stem.trim().endsWith(BACKUP_EXT)
    ? stem.trim()
    : `${stem.trim()}${BACKUP_EXT}`;
  const trimmedStem = stem.trim();
  // Strip the extension before validating, otherwise the dot in
  // "foo.sql.gz" would fail the safe-chars regex below.
  const stemForValidate = trimmedStem.endsWith(BACKUP_EXT)
    ? trimmedStem.slice(0, -BACKUP_EXT.length)
    : trimmedStem;
  const stemEmpty = stemForValidate.length === 0;
  // Mirror the backend regex (db.rs): letters, digits, dot, underscore,
  // dash; can't start with a dot. Surfaces the actual rule to the user
  // instead of waiting for the IPC roundtrip to fail.
  const stemInvalid =
    !stemEmpty &&
    (stemForValidate.startsWith(".") ||
      !/^[A-Za-z0-9._-]+$/.test(stemForValidate));
  const canSave = !stemEmpty && !stemInvalid && mysqlUp && busy == null;

  // Surface "name already exists" as the user types so they see the
  // overwrite intent before clicking save — no popup. Debounced so we
  // don't hit IPC on every keystroke; cancellation flag ignores stale
  // responses if the user keeps typing.
  const [nameExists, setNameExists] = useState(false);
  useEffect(() => {
    if (stemEmpty || stemInvalid) {
      setNameExists(false);
      return;
    }
    let cancelled = false;
    const t = window.setTimeout(() => {
      api
        .dbCheckBackupName(repoPath, stem)
        .then((c) => {
          if (!cancelled) setNameExists(c.exists);
        })
        .catch(() => {
          if (!cancelled) setNameExists(false);
        });
    }, 200);
    return () => {
      cancelled = true;
      window.clearTimeout(t);
    };
  }, [stem, stemEmpty, stemInvalid, repoPath]);

  async function onSave() {
    if (!canSave) return;
    setError(null);
    try {
      // No overwrite confirm — the inline "will overwrite" hint and the
      // Overwrite-labeled save button already make the intent explicit.
      // backup.sh uses shell `>` redirect so the file is replaced
      // atomically; we also rewrite the sidecar with the current
      // branch/note.
      const check = await api.dbCheckBackupName(repoPath, stem);
      setBusy({ kind: "backup" });
      await api.startProcess({
        id: BACKUP_PROC_ID,
        label: `db-backup ${check.final_name}`,
        cwd: repoPath,
        program: "./tools/backup_db/backup.sh",
        args: [check.relative_path],
      });
      const ok = await waitForExit(BACKUP_PROC_ID);
      if (!ok) {
        setError(
          "Backup failed. Check the Logs tab for details.",
        );
        setBusy(null);
        return;
      }
      // Sidecar is best-effort metadata — we don't fail the save if it
      // can't be written, the dump itself is what matters.
      try {
        const fullPath = `${repoPath}/${BACKUPS_DIR}/${check.final_name}`;
        await api.dbSaveBackupMeta(
          fullPath,
          currentBranch,
          note.trim() || null,
        );
      } catch (e) {
        console.warn("save backup meta failed", e);
      }
      setNote("");
      setUserEdited(false);
      await onSaved();
    } catch (e) {
      setError(String(e));
    }
    setBusy(null);
  }

  return (
    <div className="card" style={{ padding: 14 }}>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 10,
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          New backup
        </div>
        <span
          className="mono"
          style={{
            color: "var(--app-text-dim)",
            background: "var(--app-surface-2)",
            padding: "1px 6px",
            borderRadius: 3,
            fontSize: "var(--fs-xxx-small)",
          }}
          title="What this runs: ./tools/backup_db/backup.sh db-backups/<name>.sql.gz"
        >
          make db-backup
        </span>
      </div>

      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)", marginBottom: 4 }}
      >
        Name
      </div>
      <div
        style={{
          display: "flex",
          border: stemInvalid
            ? "1.5px solid var(--ui-error)"
            : "1.5px solid var(--core-fleet-green)",
          borderRadius: 5,
          fontFamily: "var(--font-mono)",
          fontSize: "var(--fs-xx-small)",
          alignItems: "stretch",
          marginBottom: 6,
          // minWidth: 0 on both this flex container and the input
          // below — without it the input's content-based intrinsic
          // width can push the whole flex parent (and its grid column)
          // beyond the panel.
          minWidth: 0,
        }}
      >
        <input
          value={stem}
          onChange={(e) => {
            setStem(e.target.value);
            setUserEdited(true);
          }}
          placeholder={defaultStem}
          {...noAutocorrect}
          style={{
            flex: 1,
            minWidth: 0,
            background: "transparent",
            border: "none",
            outline: "none",
            padding: "6px 10px",
            color: "var(--app-text)",
            fontFamily: "var(--font-mono)",
            fontSize: "var(--fs-xx-small)",
          }}
        />
        <span
          style={{
            padding: "6px 10px",
            borderLeft: "1px solid var(--app-border)",
            color: "var(--app-text-dim)",
            background: "var(--app-surface-2)",
          }}
        >
          {BACKUP_EXT}
        </span>
      </div>
      <div
        style={{
          fontSize: "var(--fs-xxx-small)",
          marginBottom: 10,
          color: nameExists ? "var(--ui-error)" : "var(--app-text-dim)",
          // Single-line layout so a long backup name doesn't wrap the
          // whole panel taller and taller. The prefix label stays
          // visible at the start; the mono path ellipsizes from the
          // right, with full value available via the title tooltip.
          display: "flex",
          gap: "0.3em",
          minWidth: 0,
        }}
      >
        <span style={{ flexShrink: 0 }}>
          {nameExists ? "will overwrite" : "will save to"}
        </span>
        <span
          className="mono"
          style={{
            color: nameExists ? "var(--ui-error)" : "var(--app-text)",
            fontWeight: nameExists ? 600 : 400,
            whiteSpace: "nowrap",
            overflow: "hidden",
            textOverflow: "ellipsis",
            minWidth: 0,
          }}
          title={`${BACKUPS_DIR}/${finalName}`}
        >
          {BACKUPS_DIR}/{finalName}
        </span>
      </div>

      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)", marginBottom: 4 }}
      >
        optional note
      </div>
      <input
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder={'e.g. "after fixing MDM enroll bug"'}
        {...noAutocorrect}
        style={{
          width: "100%",
          boxSizing: "border-box",
          background: "var(--app-surface-2)",
          border: "1px solid var(--app-border)",
          borderRadius: 5,
          padding: "6px 10px",
          fontSize: "var(--fs-xx-small)",
          color: "var(--app-text)",
          marginBottom: 12,
        }}
      />

      <div style={{ display: "flex", justifyContent: "flex-end", gap: 6 }}>
        {!mysqlUp && (
          <span
            className="dim"
            style={{
              fontSize: "var(--fs-xxx-small)",
              alignSelf: "center",
              marginRight: "auto",
            }}
          >
            MySQL is down · start docker compose first
          </span>
        )}
        {busy?.kind === "backup" && (
          <button
            onClick={goToLogs}
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            logs ↗
          </button>
        )}
        <button
          className={nameExists ? "danger" : "primary"}
          disabled={!canSave}
          onClick={onSave}
          style={{ padding: "6px 14px" }}
        >
          {busy?.kind === "backup"
            ? "saving…"
            : nameExists
              ? "Overwrite backup"
              : "Save backup"}
        </button>
      </div>
    </div>
  );
}

function ResetPanel({
  repoPath: _repoPath,
  mysqlUp,
  serveUp,
  busy,
  procs: _procs,
  onReset,
}: {
  repoPath: string;
  mysqlUp: boolean;
  serveUp: boolean;
  busy: BusyKind;
  procs: ProcInfo[];
  onReset: () => void;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  // Reset is gated the same way as Restore: needs MySQL up AND fleet
  // serve down. With serve up, the drop will block forever on the
  // open connections instead of cleanly recreating the schema.
  const blocked = !mysqlUp || serveUp || busy != null;
  const blockedReason = !mysqlUp
    ? "MySQL is down — start docker compose first"
    : serveUp
      ? "Stop fleet serve before resetting (it holds connections)"
      : undefined;

  return (
    <>
      <div
        className="card"
        style={{
          background: "var(--tint-error-soft)",
          border: "1px solid var(--tint-error-border)",
          padding: 14,
          marginTop: "auto",
        }}
      >
        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            marginBottom: 6,
          }}
        >
          <span
            className="card-title"
            style={{ color: "var(--ui-error)" }}
          >
            Reset database
          </span>
          <span
            className="mono"
            style={{
              color: "var(--app-text-dim)",
              background: "var(--app-surface-2)",
              padding: "1px 6px",
              borderRadius: 3,
              fontSize: "var(--fs-xxx-small)",
            }}
          >
            make db-reset
          </span>
        </div>
        <div
          className="dim"
          style={{ fontSize: "var(--fs-xxx-small)", lineHeight: 1.5 }}
        >
          Drops &amp; recreates the <span className="mono">fleet</span>{" "}
          schema, then runs{" "}
          <span className="mono">fleet prepare db --dev</span>. All local
          data gone.
          {serveUp && (
            <>
              {" "}
              <span style={{ color: "var(--ui-error)" }}>
                fleet serve is running — stop it first or the drop will
                hang on open connections.
              </span>
            </>
          )}
        </div>
        <div
          style={{
            marginTop: 8,
            display: "flex",
            justifyContent: "flex-end",
            gap: 6,
          }}
        >
          <button
            className="danger"
            disabled={blocked}
            onClick={() => setConfirmOpen(true)}
            title={blockedReason}
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
          >
            {busy?.kind === "reset" ? "resetting…" : "Reset…"}
          </button>
        </div>
      </div>

      {confirmOpen && (
        <ResetConfirmModal
          onCancel={() => setConfirmOpen(false)}
          onConfirm={() => {
            setConfirmOpen(false);
            onReset();
          }}
        />
      )}
    </>
  );
}

function ResetConfirmModal({
  onCancel,
  onConfirm,
}: {
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const [typed, setTyped] = useState("");
  const armed = typed.trim().toLowerCase() === "reset";

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
          maxWidth: 480,
          width: "90%",
          padding: "var(--pad-large)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-medium)",
          border: "1.5px solid var(--ui-error)",
        }}
      >
        <div
          style={{
            fontSize: "var(--fs-medium)",
            fontWeight: 600,
            color: "var(--ui-error)",
          }}
        >
          ! Reset local dev database
        </div>
        <div
          className="dim"
          style={{ fontSize: "var(--fs-x-small)", lineHeight: 1.5 }}
        >
          This will run:
        </div>
        <div
          className="mono"
          style={{
            background: "var(--app-surface-2)",
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
            padding: "8px 10px",
            fontSize: "var(--fs-xx-small)",
            color: "var(--app-text)",
          }}
        >
          docker compose exec mysql ... drop database fleet
          <br />
          docker compose exec mysql ... create database fleet
          <br />
          ./build/fleet prepare db --dev
        </div>
        <div
          className="dim"
          style={{ fontSize: "var(--fs-x-small)", lineHeight: 1.5 }}
        >
          Type <span className="mono">reset</span> below to enable the
          button.
        </div>
        <input
          value={typed}
          onChange={(e) => setTyped(e.target.value)}
          autoFocus
          {...noAutocorrect}
          style={{
            background: "var(--app-surface-2)",
            border: "1.5px solid var(--app-border)",
            borderRadius: 5,
            padding: "6px 10px",
            fontFamily: "var(--font-mono)",
            fontSize: "var(--fs-x-small)",
            color: "var(--app-text)",
          }}
        />
        <div
          style={{ display: "flex", gap: 8, justifyContent: "flex-end" }}
        >
          <button onClick={onCancel} style={{ padding: "6px 14px" }}>
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={!armed}
            className="danger"
            style={{ padding: "6px 14px" }}
          >
            Reset database
          </button>
        </div>
      </div>
    </div>
  );
}

// ---------- runners ----------

async function runRestore(
  entry: BackupEntry,
  ctx: {
    repoPath: string;
    setBusy: (b: BusyKind) => void;
    setError: (e: string | null) => void;
    refresh: () => Promise<void>;
    goToLogs: () => void;
  },
) {
  const { repoPath, setBusy, setError, refresh, goToLogs: _go } = ctx;
  // No confirmation: parity with the original, which restored immediately on
  // click. (The old window.confirm() never actually prompted — Tauri's webview
  // silently returned true — so the effective behavior was a direct restore.)
  setBusy({ kind: "restore", name: entry.name });
  setError(null);
  try {
    // restore.sh wants the path relative to cwd (the repo). We always
    // store backups under <repo>/db-backups so this stays stable.
    await api.startProcess({
      id: RESTORE_PROC_ID,
      label: `db-restore ${entry.name}`,
      cwd: repoPath,
      program: "./tools/backup_db/restore.sh",
      args: [`${BACKUPS_DIR}/${entry.name}`],
    });
    const success = await waitForExit(RESTORE_PROC_ID);
    if (!success) {
      setError("Restore failed. Check the Logs tab for details.");
    }
    await refresh();
  } catch (e) {
    setError(String(e));
  }
  setBusy(null);
}

async function runDelete(
  entry: BackupEntry,
  ctx: {
    repoPath: string;
    setBusy: (b: BusyKind) => void;
    setError: (e: string | null) => void;
    refresh: () => Promise<void>;
  },
) {
  const { repoPath, setBusy, setError, refresh } = ctx;
  setBusy({ kind: "delete", name: entry.name });
  setError(null);
  try {
    await api.dbDeleteBackup(repoPath, entry.path);
    await refresh();
  } catch (e) {
    setError(String(e));
  }
  setBusy(null);
}

async function runReset(ctx: {
  repoPath: string;
  setBusy: (b: BusyKind) => void;
  setError: (e: string | null) => void;
  refresh: () => Promise<void>;
  goToLogs: () => void;
}) {
  const { repoPath, setBusy, setError, refresh } = ctx;
  setBusy({ kind: "reset" });
  setError(null);
  try {
    // We split the Makefile's db-reset target into two managed steps
    // so the user sees the failure point in the Logs tab if either
    // half blows up. Same commands, just streamed.
    await api.startProcess({
      id: RESET_DROP_ID,
      label: "db-reset · drop+create",
      cwd: repoPath,
      program: "docker",
      args: [
        "compose",
        "exec",
        "-T",
        "mysql",
        "bash",
        "-c",
        'echo "drop database if exists fleet; create database fleet;" | MYSQL_PWD=toor mysql -uroot',
      ],
    });
    const dropOk = await waitForExit(RESET_DROP_ID);
    if (!dropOk) {
      setError(
        "Drop/create failed. Check the Logs tab — fleet serve still holding connections is the usual cause.",
      );
      setBusy(null);
      return;
    }
    await api.startProcess({
      id: RESET_PREPARE_ID,
      label: "db-reset · fleet prepare db --dev",
      cwd: repoPath,
      program: "./build/fleet",
      args: ["prepare", "db", "--dev"],
    });
    const prepOk = await waitForExit(RESET_PREPARE_ID);
    if (!prepOk) {
      setError(
        "fleet prepare db --dev failed. Check the Logs tab for details.",
      );
    }
    await refresh();
  } catch (e) {
    setError(String(e));
  }
  setBusy(null);
}

// ---------- formatting helpers ----------

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(0)} KB`;
  const mb = kb / 1024;
  if (mb < 1024) return `${mb.toFixed(mb < 10 ? 1 : 0)} MB`;
  const gb = mb / 1024;
  return `${gb.toFixed(1)} GB`;
}

function humanAgo(ms: number): string {
  const sec = Math.floor(ms / 1000);
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s ago`;
  if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
  if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
  return `${Math.floor(sec / 86400)}d ago`;
}

function humanDate(ms: number): string {
  const d = new Date(ms);
  const today = new Date();
  const sameDay =
    d.getFullYear() === today.getFullYear() &&
    d.getMonth() === today.getMonth() &&
    d.getDate() === today.getDate();
  const hh = String(d.getHours()).padStart(2, "0");
  const mm = String(d.getMinutes()).padStart(2, "0");
  if (sameDay) return `today · ${hh}:${mm}`;
  const month = d.toLocaleString(undefined, { month: "short" });
  return `${month} ${d.getDate()} · ${hh}:${mm}`;
}

function truncate(s: string, n: number): string {
  if (s.length <= n) return s;
  return s.slice(0, n - 1) + "…";
}
