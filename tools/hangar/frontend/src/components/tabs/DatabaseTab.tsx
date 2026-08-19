import { useCallback, useEffect, useMemo, useState } from "react";
import {
  api,
  type BackupEntry,
  type ProcInfo,
  type ServerProfile,
} from "../../lib/ipc";
import { noAutocorrect } from "../../lib/noAutocorrect";
import { startServe, waitForExit } from "../../lib/orchestration";
import {
  dbBackupCommand,
  dbRestoreCommand,
  prepareDbArgsFor,
  procId,
} from "../../lib/servers";
import type { DockerHealth, ServeStatus } from "../../lib/useSystemHealth";

const BACKUP_EXT = ".sql.gz";
// Base process ids; namespaced per server via procId(server.id, ...).
const RESET_DROP = "db-reset-drop";
const RESET_PREPARE = "db-reset-prepare";
const BACKUP_PROC = "db-backup";
const RESTORE_PROC = "db-restore";

type BusyKind =
  | { kind: "backup" }
  | { kind: "restore"; name: string }
  | { kind: "delete"; name: string }
  | { kind: "reset" }
  | null;

// A backup plus where it came from, so a merged list (central app-data +
// worktree, possibly from another server) can key/select/delete unambiguously.
type ListedBackup = BackupEntry & {
  sourceDir: string; // directory it lives in — used for dir-scoped delete
  origin: "central" | "worktree";
};

export function DatabaseTab({
  server,
  servers,
  currentBranch,
  procs,
  serve,
  docker,
  goToLogs,
}: {
  server: ServerProfile;
  servers: ServerProfile[];
  currentBranch: string | null;
  procs: ProcInfo[];
  serve: ServeStatus;
  docker: DockerHealth;
  goToLogs: () => void;
}) {
  const repoPath = server.worktree_path;
  const serveProcId = procId(server.id, "fleet-serve");
  const [backups, setBackups] = useState<ListedBackup[]>([]);
  // The active server's central (app-data) backups dir — where NEW backups are
  // written, and what "Reveal" opens. Resolved once per active server.
  const [activeCentralDir, setActiveCentralDir] = useState<string | null>(null);
  // Which server's backups we're browsing. Defaults to the active server; pick
  // another to restore its dumps into the active one.
  const [sourceServerId, setSourceServerId] = useState(server.id);
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [busy, setBusy] = useState<BusyKind>(null);
  const [error, setError] = useState<string | null>(null);
  const [now, setNow] = useState(Date.now());

  // When the active server changes, snap the browse-source back to it and
  // re-resolve the central dir new backups go into.
  useEffect(() => {
    setSourceServerId(server.id);
    let cancelled = false;
    api
      .dbServerBackupsDir(server.id)
      .then(async (dir) => {
        await api.dbEnsureDir(dir);
        if (!cancelled) setActiveCentralDir(dir);
      })
      .catch(() => {
        if (!cancelled) setActiveCentralDir(null);
      });
    return () => {
      cancelled = true;
    };
  }, [server.id]);

  // Tick once a minute so the "2m ago" style timestamps stay fresh
  // without us having to refresh the whole list on every render.
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(id);
  }, []);

  const sourceServer = useMemo(
    () => servers.find((s) => s.id === sourceServerId) ?? server,
    [servers, sourceServerId, server],
  );
  const sourceIsActive = sourceServer.id === server.id;

  const refresh = useCallback(async () => {
    try {
      const merged: ListedBackup[] = [];
      // Central app-data backups for the source server (survive worktree teardown).
      try {
        const centralDir = await api.dbServerBackupsDir(sourceServer.id);
        await api.dbEnsureDir(centralDir);
        const central = await api.dbListBackupsInDir(centralDir);
        for (const b of central) {
          merged.push({ ...b, sourceDir: centralDir, origin: "central" });
        }
      } catch (e) {
        console.warn("list central backups failed", e);
      }
      // Also surface `make db-backup` outputs from the source server's worktree.
      if (sourceServer.worktree_path) {
        try {
          const wtDir = await api.dbBackupsDir(sourceServer.worktree_path);
          const wt = await api.dbListBackups(sourceServer.worktree_path);
          for (const b of wt) {
            merged.push({ ...b, sourceDir: wtDir, origin: "worktree" });
          }
        } catch (e) {
          console.warn("list worktree backups failed", e);
        }
      }
      merged.sort((a, b) => b.mtime_ms - a.mtime_ms);
      setBackups(merged);
    } catch (e) {
      setError(String(e));
    }
  }, [sourceServer]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  // Keep the selection in sync with the list — if it disappears (deleted,
  // source switched) drop it rather than show ghost actions. Keyed by path
  // since names aren't unique across dirs/servers.
  useEffect(() => {
    if (selectedPath && !backups.some((b) => b.path === selectedPath)) {
      setSelectedPath(null);
    }
  }, [backups, selectedPath]);

  const selected = useMemo(
    () => backups.find((b) => b.path === selectedPath) ?? null,
    [backups, selectedPath],
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
          procs.find((p) => p.id === serveProcId)?.state === "running" ||
          procs.find((p) => p.id === serveProcId)?.state === "stopping"
        }
        server={server}
        serveProcId={serveProcId}
        lastBackup={lastBackup}
        backupCount={backups.length}
        backupsDir={activeCentralDir}
        now={now}
        onRefresh={refresh}
        onReveal={
          activeCentralDir
            ? () => api.openPath(activeCentralDir, false)
            : undefined
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
          selectedPath={selectedPath}
          onSelect={setSelectedPath}
          busy={busy}
          selected={selected}
          mysqlUp={mysqlUp}
          serveUp={serve.up}
          servers={servers}
          sourceServerId={sourceServerId}
          onSourceChange={setSourceServerId}
          activeServerName={server.name}
          sourceIsActive={sourceIsActive}
          onRestore={async (entry) => {
            await runRestore(entry, {
              server,
              setBusy,
              setError,
              refresh,
              goToLogs,
            });
          }}
          onDelete={async (entry) => {
            await runDelete(entry, { setBusy, setError, refresh });
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
            server={server}
            repoPath={repoPath}
            centralDir={activeCentralDir}
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
                server,
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
  server,
  serveProcId,
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
  server: ServerProfile;
  serveProcId: string;
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
      await api.stopProcess(serveProcId);
    } catch (e) {
      setError(String(e));
    }
    setServeBusy(null);
  }

  async function startServeAction() {
    if (serveBusy) return;
    setServeBusy("starting");
    setError(null);
    try {
      await startServe(server);
    } catch (e) {
      setError(String(e));
      setServeBusy(null);
      return;
    }
    // startServe only spawns the process — the server isn't listening yet.
    // Stay in "starting…" until health reports it up (see effect below); the
    // timeout is a safety net if it never binds (crash / port already in use)
    // so the button doesn't stay stuck.
    window.setTimeout(() => {
      setServeBusy((b) => (b === "starting" ? null : b));
    }, 90_000);
  }

  // Clear "starting…" the moment the server is actually up. Without this the
  // button would re-enable (and look spammable) the instant the process is
  // spawned, long before fleet is listening.
  useEffect(() => {
    if (serveUp) setServeBusy((b) => (b === "starting" ? null : b));
  }, [serveUp]);

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
          :{server.ports.mysql} · via docker
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
  selectedPath,
  onSelect,
  busy,
  selected,
  mysqlUp,
  serveUp,
  servers,
  sourceServerId,
  onSourceChange,
  activeServerName,
  sourceIsActive,
  onRestore,
  onDelete,
}: {
  backups: ListedBackup[];
  selectedPath: string | null;
  onSelect: (path: string) => void;
  busy: BusyKind;
  selected: ListedBackup | null;
  mysqlUp: boolean;
  serveUp: boolean;
  servers: ServerProfile[];
  sourceServerId: string;
  onSourceChange: (id: string) => void;
  activeServerName: string;
  sourceIsActive: boolean;
  onRestore: (b: ListedBackup) => void;
  onDelete: (b: ListedBackup) => void;
}) {
  const [confirmDelete, setConfirmDelete] = useState(false);

  // Whenever the selection changes, drop the inline-confirm state. Avoids
  // the "I selected a different row but delete still says CONFIRM?" trap.
  useEffect(() => {
    setConfirmDelete(false);
  }, [selectedPath]);

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
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 8,
          flexWrap: "wrap",
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          Backups
        </div>
        {servers.length > 1 && (
          <label
            className="dim"
            style={{
              display: "flex",
              alignItems: "center",
              gap: 6,
              fontSize: "var(--fs-xx-small)",
            }}
          >
            source
            <select
              value={sourceServerId}
              onChange={(e) => onSourceChange(e.target.value)}
              style={{ fontSize: "var(--fs-xx-small)", padding: "2px 6px" }}
            >
              {servers.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name}
                </option>
              ))}
            </select>
          </label>
        )}
      </div>

      {!sourceIsActive && (
        <div
          style={{
            fontSize: "var(--fs-xxx-small)",
            color: "var(--ui-warning)",
            border: "1px solid var(--ui-warning)",
            borderRadius: "var(--radius-md)",
            padding: "6px 8px",
            lineHeight: 1.4,
          }}
        >
          Browsing another server's backups. Restore imports into the active
          server (<span className="mono">{activeServerName}</span>) — make sure
          the branch/version lines up, or run prepare-db to migrate afterward.
        </div>
      )}

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
              key={b.path}
              entry={b}
              selected={b.path === selectedPath}
              onClick={() => onSelect(b.path)}
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
            : `Restore into ${activeServerName}`}
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
  entry: ListedBackup;
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
        {entry.origin === "worktree" && (
          <span
            className="dim"
            title="From the worktree's db-backups (e.g. make db-backup)"
            style={{
              fontSize: "var(--fs-xxx-small)",
              border: "1px solid var(--app-border)",
              borderRadius: 4,
              padding: "0 4px",
              flexShrink: 0,
            }}
          >
            repo
          </span>
        )}
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
  server,
  repoPath,
  centralDir,
  currentBranch,
  mysqlUp,
  busy,
  onSaved,
  setBusy,
  setError,
  goToLogs,
}: {
  server: ServerProfile;
  repoPath: string;
  // Central app-data dir new backups are written to (null until resolved).
  centralDir: string | null;
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
  const canSave =
    !stemEmpty && !stemInvalid && mysqlUp && busy == null && !!centralDir;

  // Surface "name already exists" as the user types so they see the
  // overwrite intent before clicking save — no popup. Debounced so we
  // don't hit IPC on every keystroke; cancellation flag ignores stale
  // responses if the user keeps typing.
  const [nameExists, setNameExists] = useState(false);
  useEffect(() => {
    if (stemEmpty || stemInvalid || !centralDir) {
      setNameExists(false);
      return;
    }
    let cancelled = false;
    const t = window.setTimeout(() => {
      api
        .dbCheckBackupNameInDir(centralDir, stem)
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
  }, [stem, stemEmpty, stemInvalid, centralDir]);

  async function onSave() {
    if (!canSave || !centralDir) return;
    setError(null);
    try {
      // No overwrite confirm — the inline "will overwrite" hint and the
      // Overwrite-labeled save button already make the intent explicit.
      // The command uses a shell `>` redirect so the file is replaced
      // atomically; we also rewrite the sidecar with the current
      // branch/note. Backups are written to the active server's central
      // app-data dir so they survive worktree teardown.
      const check = await api.dbCheckBackupNameInDir(centralDir, stem);
      const fullPath = `${centralDir}/${check.final_name}`;
      setBusy({ kind: "backup" });
      // Hangar builds the dump command itself (see dbBackupCommand) instead of
      // running the worktree's backup.sh — old/released refs ship a script that
      // hardcodes the primary compose network and would dump the wrong stack.
      await api.startProcess({
        id: procId(server.id, BACKUP_PROC),
        label: `db-backup ${check.final_name}`,
        cwd: repoPath,
        program: "bash",
        args: ["-c", dbBackupCommand(server, fullPath)],
      });
      const ok = await waitForExit(procId(server.id, BACKUP_PROC));
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
          title={centralDir ? `${centralDir}/${finalName}` : finalName}
        >
          {finalName}
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
  entry: ListedBackup,
  ctx: {
    server: ServerProfile;
    setBusy: (b: BusyKind) => void;
    setError: (e: string | null) => void;
    refresh: () => Promise<void>;
    goToLogs: () => void;
  },
) {
  const { server, setBusy, setError, refresh, goToLogs: _go } = ctx;
  // No confirmation: parity with the original, which restored immediately on
  // click. (The old window.confirm() never actually prompted — Tauri's webview
  // silently returned true — so the effective behavior was a direct restore.)
  setBusy({ kind: "restore", name: entry.name });
  setError(null);
  const restoreId = procId(server.id, RESTORE_PROC);
  try {
    // Hangar builds the restore command itself (see dbRestoreCommand) instead
    // of running the worktree's restore.sh: on old/released refs that script
    // hardcodes the primary compose network, so a "successful" restore would
    // silently import into the wrong stack. entry.path is absolute (it may live
    // in another server's dir); the import always targets the ACTIVE server.
    await api.startProcess({
      id: restoreId,
      label: `db-restore ${entry.name}`,
      cwd: server.worktree_path ?? "",
      program: "bash",
      args: ["-c", dbRestoreCommand(server, entry.path)],
    });
    const success = await waitForExit(restoreId);
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
  entry: ListedBackup,
  ctx: {
    setBusy: (b: BusyKind) => void;
    setError: (e: string | null) => void;
    refresh: () => Promise<void>;
  },
) {
  const { setBusy, setError, refresh } = ctx;
  setBusy({ kind: "delete", name: entry.name });
  setError(null);
  try {
    // Delete is scoped to the entry's own dir (central or worktree).
    await api.dbDeleteBackupInDir(entry.sourceDir, entry.path);
    await refresh();
  } catch (e) {
    setError(String(e));
  }
  setBusy(null);
}

async function runReset(ctx: {
  server: ServerProfile;
  repoPath: string;
  setBusy: (b: BusyKind) => void;
  setError: (e: string | null) => void;
  refresh: () => Promise<void>;
  goToLogs: () => void;
}) {
  const { server, repoPath, setBusy, setError, refresh } = ctx;
  setBusy({ kind: "reset" });
  setError(null);
  const dropId = procId(server.id, RESET_DROP);
  const prepareId = procId(server.id, RESET_PREPARE);
  try {
    // We split the Makefile's db-reset target into two managed steps
    // so the user sees the failure point in the Logs tab if either
    // half blows up. Same commands, just streamed — scoped to this
    // server's compose project so the right MySQL is reset.
    await api.startProcess({
      id: dropId,
      label: "db-reset · drop+create",
      cwd: repoPath,
      program: "docker",
      args: [
        "compose",
        "-p",
        server.compose_project,
        "exec",
        "-T",
        "mysql",
        "bash",
        "-c",
        'echo "drop database if exists fleet; create database fleet;" | MYSQL_PWD=toor mysql -uroot',
      ],
    });
    const dropOk = await waitForExit(dropId);
    if (!dropOk) {
      setError(
        "Drop/create failed. Check the Logs tab — fleet serve still holding connections is the usual cause.",
      );
      setBusy(null);
      return;
    }
    await api.startProcess({
      id: prepareId,
      label: "db-reset · fleet prepare db --dev",
      cwd: repoPath,
      program: "./build/fleet",
      args: prepareDbArgsFor(server),
    });
    const prepOk = await waitForExit(prepareId);
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
