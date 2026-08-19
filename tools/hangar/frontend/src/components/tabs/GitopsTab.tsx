import { useCallback, useEffect, useRef, useState } from "react";
import { listen } from "../../lib/events";
import {
  api,
  type ContextInfo,
  type GitopsDirScan,
  type GitopsFile,
  type GitopsRepo,
  type GitopsTargetCheck,
  type LogLine,
  type ProcEvent,
  type ResolvedBinary,
  type Settings,
} from "../../lib/ipc";
import { noAutocorrect } from "../../lib/noAutocorrect";
import { activeServer } from "../../lib/servers";

/// Stable proc ids — only one apply and one generate at a time, so the
/// existing process manager's "no duplicate id" check is the lock.
const APPLY_ID = "gitops-apply";
const GENERATE_ID = "gitops-generate";
/// Cap on accumulated output kept per run. Plenty for a real apply
/// (icon uploads + profile sync rarely exceed ~5k lines on a big repo),
/// not so much that the box becomes unscrollable.
const OUTPUT_LINE_CAP = 5000;

type OutputKind = "apply" | "generate";
type OutputState = "running" | "applied" | "dry-run" | "failed";

interface RunOutput {
  kind: OutputKind;
  repoName: string | null;
  /// Full argv joined for display — matches what the user would have
  /// typed into a shell. Stored separately from `lines` so the header
  /// line in the terminal box doesn't compete with the actual output.
  command: string;
  startedAt: number;
  endedAt: number | null;
  state: OutputState;
  exitCode: number | null;
  lines: { ts_ms: number; line: string; stream: "stdout" | "stderr" }[];
  /// True for `--dry-run` invocations so the final state pill can flip
  /// to "dry-run" (blue) on exit 0, vs "applied" (green) for real runs.
  dryRun: boolean;
}

export function GitopsTab({
  settings,
  goToSettings,
}: {
  settings: Settings;
  goToSettings: () => void;
}) {
  const gitopsDir = settings.gitops_dir;
  const repoPath = activeServer(settings).worktree_path;

  const [scan, setScan] = useState<GitopsDirScan | null>(null);
  const [scanError, setScanError] = useState<string | null>(null);
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  // Persisted only within session: which team files are checked per
  // repo. Defaults to "all selected" the first time you see a repo.
  const [selectionByRepo, setSelectionByRepo] = useState<
    Record<string, Set<string>>
  >({});
  const [dryRunByRepo, setDryRunByRepo] = useState<Record<string, boolean>>({});
  const [output, setOutput] = useState<RunOutput | null>(null);

  // Active context for `--context <name>`. Same UI pattern as
  // FleetctlTab — read parsed contexts from ~/.fleet/config and let
  // the user pick. Default to "default" if no current is set.
  const [ctxInfo, setCtxInfo] = useState<ContextInfo | null>(null);
  const [selectedContext, setSelectedContext] = useState<string>("default");
  const [binary, setBinary] = useState<ResolvedBinary | null>(null);

  const refreshCtx = useCallback(async () => {
    try {
      const c = await api.fleetctlReadContext();
      setCtxInfo(c);
    } catch (e) {
      console.error("read context failed", e);
    }
  }, []);

  const refreshBinary = useCallback(async () => {
    try {
      const b = await api.fleetctlResolveBinary(
        repoPath ?? null,
        settings.fleetctl_path ?? null,
      );
      setBinary(b);
    } catch (e) {
      console.error("resolve fleetctl binary failed", e);
    }
  }, [repoPath, settings.fleetctl_path]);

  useEffect(() => {
    refreshCtx();
    refreshBinary();
  }, [refreshCtx, refreshBinary]);

  const rescan = useCallback(async () => {
    if (!gitopsDir) {
      setScan(null);
      return;
    }
    setScanError(null);
    try {
      const s = await api.gitopsListRepos(gitopsDir);
      setScan(s);
      // Auto-select the first repo when scan changes, or keep the
      // current selection if it still exists.
      if (s.repos.length > 0) {
        setSelectedRepo((cur) =>
          cur != null && s.repos.some((r) => r.name === cur) ? cur : s.repos[0].name,
        );
      } else {
        setSelectedRepo(null);
      }
    } catch (e) {
      setScanError(String(e));
      setScan(null);
    }
  }, [gitopsDir]);

  useEffect(() => {
    rescan();
  }, [rescan]);

  // Subscribe to proc:log + proc:state events for our two stable ids
  // and route them into the current `output` buffer. Same pattern as
  // perf runs: no log_channel on the spawn, so nothing hits disk.
  //
  // The `cancelled` flag closes the React-18-StrictMode race where the
  // effect runs → registers a listener (promise pending) → cleanup
  // runs before the promise resolves → effect runs again. Without it,
  // we end up with two listeners and every output line appears twice.
  useEffect(() => {
    let cancelled = false;
    const unlistens: Array<() => void> = [];
    const register = async <T,>(
      event: string,
      handler: (e: { payload: T }) => void,
    ) => {
      const u = await listen<T>(event, handler);
      if (cancelled) u();
      else unlistens.push(u);
    };

    register<LogLine>("proc:log", (e) => {
      const id = e.payload.proc_id;
      if (id !== APPLY_ID && id !== GENERATE_ID) return;
      setOutput((prev) => {
        if (!prev) return prev;
        // If a new run started, the output gets replaced (see startRun /
        // startGenerate). Only append if the event is for the current
        // displayed kind.
        if ((id === APPLY_ID) !== (prev.kind === "apply")) return prev;
        const next = [
          ...prev.lines,
          {
            ts_ms: e.payload.ts_ms,
            line: e.payload.line,
            stream: e.payload.stream,
          },
        ];
        if (next.length > OUTPUT_LINE_CAP) {
          next.splice(0, next.length - OUTPUT_LINE_CAP);
        }
        return { ...prev, lines: next };
      });
    });
    register<ProcEvent>("proc:state", (e) => {
      const id = e.payload.proc_id;
      if (id !== APPLY_ID && id !== GENERATE_ID) return;
      setOutput((prev) => {
        if (!prev) return prev;
        if ((id === APPLY_ID) !== (prev.kind === "apply")) return prev;
        let nextState: OutputState = prev.state;
        const code = e.payload.exit_code;
        if (e.payload.state === "done") {
          nextState = prev.dryRun ? "dry-run" : "applied";
        } else if (e.payload.state === "failed") {
          nextState = "failed";
        } else if (e.payload.state === "running") {
          nextState = "running";
        }
        return {
          ...prev,
          state: nextState,
          exitCode: code,
          endedAt: e.payload.state === "running" ? null : Date.now(),
        };
      });
    });

    return () => {
      cancelled = true;
      unlistens.forEach((u) => u());
    };
  }, []);

  // Empty-state when gitops dir isn't configured yet.
  if (!gitopsDir) {
    return (
      <EmptyState
        title="GitOps directory not configured"
        body="Pick the folder where your gitops repos live (or a single repo containing default.yml). Settings → GitOps directory."
        cta={{ label: "Open Settings", onClick: goToSettings }}
      />
    );
  }
  if (scanError) {
    return (
      <EmptyState
        title="Scan failed"
        body={scanError}
        cta={{ label: "Retry", onClick: rescan }}
      />
    );
  }
  if (!scan) {
    return (
      <EmptyState
        title="Scanning…"
        body={gitopsDir}
      />
    );
  }

  const repo = scan.repos.find((r) => r.name === selectedRepo) ?? null;
  // Unified file list: default.yml leads, team/fleet files follow. The
  // default entry uses an empty `subdir` as a sentinel — applyArgsFor
  // emits the bare filename for it (since fleetctl runs from the repo
  // cwd) and "teams/x.yml" / "fleets/x.yml" for the rest.
  const allFiles: GitopsFile[] = repo
    ? [
        {
          name: "default.yml",
          path: repo.default_path,
          size: repo.default_size,
          mtime_ms: repo.default_mtime_ms,
          subdir: "",
        },
        ...repo.team_files,
      ]
    : [];
  // Selection model: undefined / missing entry = nothing selected.
  // Most apply runs target a single file, so starting empty matches
  // the common case and avoids "I clicked Apply by accident and it
  // pushed every team."
  const repoSelection = repo ? selectionByRepo[repo.name] : undefined;
  // Selections are keyed by full file path, not basename: files in different
  // subdirs (e.g. teams/foo.yml vs fleets/foo.yml) share a name and would
  // otherwise collide, selecting/applying both at once.
  const selectedFiles = allFiles.filter((f) =>
    repoSelection != null && repoSelection.has(f.path),
  );
  const dryRun =
    repo != null ? (dryRunByRepo[repo.name] ?? true) : true;

  function toggleFile(path: string) {
    if (!repo) return;
    setSelectionByRepo((prev) => {
      const cur = new Set(prev[repo.name] ?? []);
      if (cur.has(path)) cur.delete(path);
      else cur.add(path);
      return { ...prev, [repo.name]: cur };
    });
  }
  function selectAll() {
    if (!repo) return;
    setSelectionByRepo((prev) => ({
      ...prev,
      [repo.name]: new Set(allFiles.map((f) => f.path)),
    }));
  }
  function selectNone() {
    if (!repo) return;
    setSelectionByRepo((prev) => ({
      ...prev,
      [repo.name]: new Set<string>(),
    }));
  }

  async function startApply() {
    if (!repo) return;
    if (!binary?.exists) {
      alert("fleetctl binary not found. Settings → Paths.");
      return;
    }
    const args = applyArgsFor(selectedFiles, selectedContext, dryRun);
    const command = `${binary.path} ${args.join(" ")}`;
    setOutput({
      kind: "apply",
      repoName: repo.name,
      command,
      startedAt: Date.now(),
      endedAt: null,
      state: "running",
      exitCode: null,
      lines: [],
      dryRun,
    });
    try {
      // Replace any prior run — proc id is stable. If a prior run is
      // still alive the user must stop it explicitly; we don't auto-
      // kill, since that would mask in-progress work.
      await api.startProcess({
        id: APPLY_ID,
        label: `gitops apply · ${repo.name}${dryRun ? " (dry-run)" : ""}`,
        cwd: repo.path,
        program: binary.path,
        args,
      });
    } catch (e) {
      setOutput((prev) =>
        prev
          ? {
              ...prev,
              state: "failed",
              endedAt: Date.now(),
              lines: [
                ...prev.lines,
                {
                  ts_ms: Date.now(),
                  stream: "stderr",
                  line: `failed to spawn fleetctl: ${e}`,
                },
              ],
            }
          : prev,
      );
    }
  }

  async function startGenerate(name: string, force: boolean) {
    if (!binary?.exists) {
      alert("fleetctl binary not found. Settings → Paths.");
      return;
    }
    if (!gitopsDir) return;
    const target = `${gitopsDir.replace(/\/$/, "")}/${name}`;
    const args = [
      "generate-gitops",
      "--context",
      selectedContext,
      "--dir",
      target,
    ];
    if (force) args.push("--force");
    const command = `${binary.path} ${args.join(" ")}`;
    setOutput({
      kind: "generate",
      repoName: null,
      command,
      startedAt: Date.now(),
      endedAt: null,
      state: "running",
      exitCode: null,
      lines: [],
      dryRun: false,
    });
    try {
      await api.startProcess({
        id: GENERATE_ID,
        label: `gitops generate · ${name}`,
        cwd: gitopsDir,
        program: binary.path,
        args,
      });
    } catch (e) {
      setOutput((prev) =>
        prev
          ? {
              ...prev,
              state: "failed",
              endedAt: Date.now(),
              lines: [
                ...prev.lines,
                {
                  ts_ms: Date.now(),
                  stream: "stderr",
                  line: `failed to spawn fleetctl: ${e}`,
                },
              ],
            }
          : prev,
      );
      // After failed generate, re-scan because partial files may exist.
      rescan();
      return;
    }
  }

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        overflow: "hidden",
      }}
    >
      <div
        style={{
          padding: "var(--pad-large) var(--pad-large) var(--pad-medium)",
          flexShrink: 0,
        }}
      >
        <StatusStrip
          gitopsDir={scan.root}
          repoCount={scan.repos.length}
          singleRepo={scan.single_repo_mode}
          ctx={ctxInfo}
          selectedContext={selectedContext}
          onContextChange={setSelectedContext}
          binary={binary}
          onRescan={rescan}
        />
      </div>

      <div
        style={{
          flex: 1,
          minHeight: 0,
          display: "grid",
          gridTemplateColumns: scan.single_repo_mode
            ? "1fr minmax(0, 480px)"
            : "minmax(220px, 240px) minmax(0, 1fr) minmax(0, 480px)",
          gap: "var(--pad-medium)",
          padding: "var(--pad-medium)",
          overflow: "hidden",
        }}
      >
        {!scan.single_repo_mode && (
          <ReposPanel
            scan={scan}
            selected={selectedRepo}
            onSelect={setSelectedRepo}
            onRescan={rescan}
          />
        )}

        <RepoPanel
          repo={repo}
          allFiles={allFiles}
          selection={repoSelection}
          onToggle={toggleFile}
          onSelectAll={selectAll}
          onSelectNone={selectNone}
          dryRun={dryRun}
          onToggleDryRun={(v) => {
            if (repo) setDryRunByRepo((p) => ({ ...p, [repo.name]: v }));
          }}
          selectedCount={selectedFiles.length}
          onApply={startApply}
          contextName={selectedContext}
        />

        <RightColumn
          gitopsDir={gitopsDir}
          singleRepo={scan.single_repo_mode}
          contextName={selectedContext}
          binary={binary}
          output={output}
          onStartGenerate={startGenerate}
          onStop={(kind) =>
            api
              .stopProcess(kind === "apply" ? APPLY_ID : GENERATE_ID)
              .catch(console.error)
          }
        />
      </div>
    </div>
  );
}

/* --------------- Status strip --------------- */

function StatusStrip({
  gitopsDir,
  repoCount,
  singleRepo,
  ctx,
  selectedContext,
  onContextChange,
  binary,
  onRescan,
}: {
  gitopsDir: string;
  repoCount: number;
  singleRepo: boolean;
  ctx: ContextInfo | null;
  selectedContext: string;
  onContextChange: (name: string) => void;
  binary: ResolvedBinary | null;
  onRescan: () => void;
}) {
  // Match FleetctlTab's ContextHeader visual: dot+label, separators,
  // binary chip, context picker. We add a gitops-dir cell on the
  // left and a Reveal button on the right.
  const known = ctx?.contexts ?? [];
  const options = known.some((c) => c.name === selectedContext)
    ? known
    : [
        ...known,
        { name: selectedContext, address: null, email: null, has_token: false },
      ];

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
      <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0 }}>
        <span style={{ color: "var(--ui-warning)" }}>📁</span>
        <span
          className="mono"
          style={{
            fontSize: "var(--fs-xx-small)",
            color: "var(--app-text)",
            maxWidth: 320,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
          title={gitopsDir}
        >
          {gitopsDir}
        </span>
        <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          · {repoCount} repo{repoCount === 1 ? "" : "s"}
        </span>
        {singleRepo && (
          <span
            style={{
              fontSize: "var(--fs-xxx-small)",
              color: "var(--core-fleet-purple)",
              padding: "0 6px",
              border: "1px solid var(--core-fleet-purple)",
              borderRadius: 3,
              textTransform: "uppercase",
              letterSpacing: "0.05em",
            }}
          >
            single-repo
          </span>
        )}
        <button
          onClick={onRescan}
          title="rescan"
          style={{ padding: "2px 8px", fontSize: "var(--fs-xxx-small)" }}
        >
          ↺
        </button>
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
        binary ·{" "}
        <span
          className="mono"
          style={{
            color: binary?.exists ? "var(--app-text)" : "var(--ui-error)",
          }}
          title={binary?.path ?? ""}
        >
          {binary?.exists
            ? binary.source === "settings"
              ? "settings"
              : "build/fleetctl"
            : "missing"}
        </span>
      </div>
      <span style={{ color: "var(--app-border)" }}>│</span>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 6,
          fontSize: "var(--fs-xx-small)",
        }}
      >
        <span className="dim">context</span>
        <select
          value={selectedContext}
          onChange={(e) => onContextChange(e.target.value)}
          style={{
            background: "var(--app-surface-2)",
            color: "var(--app-text)",
            border: "1px solid var(--app-border)",
            borderRadius: 5,
            padding: "3px 6px",
            fontFamily: "var(--font-mono)",
            fontSize: "var(--fs-xx-small)",
          }}
        >
          {options.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name}
            </option>
          ))}
        </select>
      </div>
      <span style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
        <button
          onClick={() => api.openPath(gitopsDir).catch(console.error)}
          style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
        >
          Reveal in Finder
        </button>
      </span>
    </div>
  );
}

/* --------------- Repos panel (left, multi-repo only) --------------- */

function ReposPanel({
  scan,
  selected,
  onSelect,
  onRescan,
}: {
  scan: GitopsDirScan;
  selected: string | null;
  onSelect: (name: string) => void;
  onRescan: () => void;
}) {
  const [filter, setFilter] = useState("");
  const filtered = scan.repos.filter((r) =>
    r.name.toLowerCase().includes(filter.toLowerCase()),
  );
  return (
    <div
      className="card"
      style={{
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 8,
        minHeight: 0,
        minWidth: 0,
      }}
    >
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          Repos{" "}
          <span style={{ color: "var(--app-text)", fontWeight: 600 }}>
            · {scan.repos.length}
          </span>
        </div>
        <button
          onClick={onRescan}
          style={{ padding: "2px 8px", fontSize: "var(--fs-xxx-small)" }}
        >
          ↺
        </button>
      </div>
      <input
        type="text"
        placeholder="search…"
        value={filter}
        onChange={(e) => setFilter(e.target.value)}
        {...noAutocorrect}
        style={{ width: "100%" }}
      />
      <div
        style={{
          flex: 1,
          overflow: "auto",
          display: "flex",
          flexDirection: "column",
          gap: 6,
          minHeight: 0,
        }}
      >
        {filtered.length === 0 ? (
          <div
            className="dim"
            style={{
              fontSize: "var(--fs-xx-small)",
              textAlign: "center",
              padding: 12,
              border: "1px dashed var(--app-border)",
              borderRadius: "var(--radius-md)",
            }}
          >
            {scan.repos.length === 0
              ? "No repos detected. Drop a folder with default.yml here."
              : "no matches"}
          </div>
        ) : (
          filtered.map((r) => (
            <RepoCard
              key={r.name}
              repo={r}
              selected={r.name === selected}
              onClick={() => onSelect(r.name)}
            />
          ))
        )}
      </div>
      {scan.ignored.length > 0 && (
        <details
          style={{
            fontSize: "var(--fs-xxx-small)",
            color: "var(--app-text-dim)",
          }}
        >
          <summary style={{ cursor: "pointer" }}>
            {scan.ignored.length} folder(s) without default.yml
          </summary>
          <div style={{ paddingTop: 4 }}>
            {scan.ignored.map((n) => (
              <div key={n} className="mono">
                · {n}
              </div>
            ))}
          </div>
        </details>
      )}
    </div>
  );
}

function RepoCard({
  repo,
  selected,
  onClick,
}: {
  repo: GitopsRepo;
  selected: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        textAlign: "left",
        padding: "8px 10px",
        borderRadius: "var(--radius-md)",
        background: selected
          ? "var(--tint-success-soft)"
          : "var(--app-surface-2)",
        border: `1px solid ${selected ? "var(--core-fleet-green)" : "transparent"}`,
        display: "flex",
        flexDirection: "column",
        gap: 4,
        cursor: "pointer",
        minWidth: 0,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 6, minWidth: 0 }}>
        <span style={{ color: "var(--ui-warning)" }}>📁</span>
        <span
          className="mono"
          style={{
            fontWeight: 600,
            fontSize: "var(--fs-x-small)",
            color: "var(--app-text)",
            flex: 1,
            minWidth: 0,
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {repo.name}
        </span>
      </div>
      <div
        className="dim"
        style={{
          fontSize: "var(--fs-xxx-small)",
          display: "flex",
          gap: 8,
        }}
      >
        <span>{repo.team_files.length} files</span>
        <span>·</span>
        <span>{formatBytes(repo.default_size)} default.yml</span>
      </div>
    </button>
  );
}

/* --------------- Selected repo (middle column) --------------- */

function RepoPanel({
  repo,
  allFiles,
  selection,
  onToggle,
  onSelectAll,
  onSelectNone,
  dryRun,
  onToggleDryRun,
  selectedCount,
  onApply,
  contextName,
}: {
  repo: GitopsRepo | null;
  allFiles: GitopsFile[];
  selection: Set<string> | undefined;
  onToggle: (name: string) => void;
  onSelectAll: () => void;
  onSelectNone: () => void;
  dryRun: boolean;
  onToggleDryRun: (v: boolean) => void;
  selectedCount: number;
  onApply: () => void;
  contextName: string;
}) {
  if (!repo) {
    return (
      <div
        className="card"
        style={{
          padding: "var(--pad-medium)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          color: "var(--app-text-dim)",
          fontSize: "var(--fs-xx-small)",
          minHeight: 0,
          minWidth: 0,
        }}
      >
        Pick a repo on the left.
      </div>
    );
  }
  const isChecked = (path: string) => selection?.has(path) ?? false;

  return (
    <div
      className="card"
      style={{
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 10,
        minHeight: 0,
        minWidth: 0,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline", gap: 8 }}>
        <div style={{ minWidth: 0 }}>
          <div
            className="mono"
            style={{
              fontSize: "var(--fs-medium)",
              fontWeight: 600,
              color: "var(--app-text)",
            }}
          >
            {repo.name}
          </div>
          <div
            className="dim mono"
            style={{
              fontSize: "var(--fs-xxx-small)",
              whiteSpace: "nowrap",
              overflow: "hidden",
              textOverflow: "ellipsis",
            }}
            title={repo.path}
          >
            {repo.path}
          </div>
        </div>
        <button
          onClick={() => api.openPath(repo.path).catch(console.error)}
          style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
        >
          Reveal
        </button>
      </div>

      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "baseline",
          marginTop: 4,
        }}
      >
        <div className="section-title" style={{ margin: 0 }}>
          files · {allFiles.length}
        </div>
        <div style={{ display: "flex", gap: 8, fontSize: "var(--fs-xxx-small)" }}>
          <button onClick={onSelectAll} className="link-btn">
            select all
          </button>
          <span className="dim">·</span>
          <button onClick={onSelectNone} className="link-btn">
            none
          </button>
        </div>
      </div>

      <div
        style={{
          flex: 1,
          minHeight: 0,
          overflow: "auto",
          border: "1px solid var(--app-border)",
          borderRadius: "var(--radius-md)",
          background: "var(--app-surface-2)",
        }}
      >
        {allFiles.length === 0 ? (
          <div
            className="dim"
            style={{
              fontSize: "var(--fs-xx-small)",
              textAlign: "center",
              padding: "var(--pad-large)",
            }}
          >
            No files found.
          </div>
        ) : (
          allFiles.map((f, i) => {
            const isDefault = f.subdir === "";
            return (
              <label
                key={f.path}
                style={{
                  display: "grid",
                  gridTemplateColumns: "auto auto auto 1fr auto auto",
                  alignItems: "center",
                  gap: 10,
                  padding: "6px 10px",
                  background: isChecked(f.path)
                    ? "var(--tint-success-soft)"
                    : undefined,
                  borderTop: i > 0 ? "1px solid var(--app-border)" : "none",
                  cursor: "pointer",
                  minWidth: 0,
                }}
              >
                <input
                  type="checkbox"
                  checked={isChecked(f.path)}
                  onChange={() => onToggle(f.path)}
                  style={{ accentColor: "var(--core-fleet-green)" }}
                />
                <span
                  style={{
                    color: isDefault
                      ? "var(--core-fleet-green)"
                      : "var(--app-text-dim)",
                    width: 14,
                    textAlign: "center",
                  }}
                  title={isDefault ? "default.yml — global settings" : undefined}
                >
                  {isDefault ? "★" : "·"}
                </span>
                <span
                  style={{
                    color: "var(--app-text-dim)",
                    fontSize: "var(--fs-xxx-small)",
                    width: 38,
                  }}
                  title={isDefault ? "(root)" : `${f.subdir}/`}
                >
                  {isDefault ? "" : f.subdir}
                </span>
                <span
                  className="mono"
                  style={{
                    fontSize: "var(--fs-x-small)",
                    color: "var(--app-text)",
                    fontWeight: isDefault ? 600 : 400,
                    minWidth: 0,
                    overflow: "hidden",
                    textOverflow: "ellipsis",
                    whiteSpace: "nowrap",
                  }}
                  title={f.name}
                >
                  {f.name}
                </span>
                <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
                  {formatBytes(f.size)}
                </span>
                <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
                  {formatTimeAgo(f.mtime_ms)}
                </span>
              </label>
            );
          })
        )}
      </div>

      <ApplyBar
        repoName={repo.name}
        selectedCount={selectedCount}
        contextName={contextName}
        dryRun={dryRun}
        onToggleDryRun={onToggleDryRun}
        onApply={onApply}
      />
    </div>
  );
}

function ApplyBar({
  repoName,
  selectedCount,
  contextName,
  dryRun,
  onToggleDryRun,
  onApply,
}: {
  repoName: string;
  selectedCount: number;
  contextName: string;
  dryRun: boolean;
  onToggleDryRun: (v: boolean) => void;
  onApply: () => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 10,
        padding: "8px 10px",
        background: "var(--app-surface-2)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
      }}
    >
      <span style={{ fontSize: "var(--fs-xx-small)" }}>
        {selectedCount === 0 ? (
          <span className="dim">pick at least one file →</span>
        ) : (
          <>
            applying{" "}
            <span style={{ color: "var(--core-fleet-green)", fontWeight: 600 }}>
              {selectedCount} {selectedCount === 1 ? "file" : "files"}
            </span>{" "}
            → <span className="mono">{contextName}</span>
          </>
        )}
      </span>
      <span style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 10 }}>
        <label
          style={{
            display: "flex",
            alignItems: "center",
            gap: 6,
            cursor: "pointer",
            fontSize: "var(--fs-xx-small)",
          }}
          title="When on, runs with --dry-run (no mutation)"
        >
          <input
            type="checkbox"
            checked={dryRun}
            onChange={(e) => onToggleDryRun(e.target.checked)}
            style={{ accentColor: "var(--core-vibrant-blue)" }}
          />
          <span style={{ color: dryRun ? "var(--core-vibrant-blue)" : "var(--app-text-dim)", fontWeight: 600 }}>
            dry-run
          </span>
        </label>
        <button
          className="primary"
          onClick={onApply}
          disabled={selectedCount === 0}
          title={`fleetctl gitops -f … on ${repoName}`}
          style={{ padding: "6px 14px" }}
        >
          ▶ Apply
        </button>
      </span>
    </div>
  );
}

/* --------------- Right column: generate + output --------------- */

function RightColumn({
  gitopsDir,
  singleRepo,
  contextName,
  binary,
  output,
  onStartGenerate,
  onStop,
}: {
  gitopsDir: string;
  singleRepo: boolean;
  contextName: string;
  binary: ResolvedBinary | null;
  output: RunOutput | null;
  onStartGenerate: (name: string, force: boolean) => void;
  onStop: (kind: OutputKind) => void;
}) {
  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: "var(--pad-medium)",
        minHeight: 0,
        minWidth: 0,
      }}
    >
      <GenerateCard
        gitopsDir={gitopsDir}
        disabled={singleRepo || !binary?.exists}
        disabledReason={
          singleRepo
            ? "Switch to a multi-repo dir in Settings to enable"
            : !binary?.exists
              ? "fleetctl binary missing"
              : null
        }
        contextName={contextName}
        onStart={onStartGenerate}
      />
      <OutputCard output={output} onStop={onStop} />
    </div>
  );
}

function GenerateCard({
  gitopsDir,
  disabled,
  disabledReason,
  contextName,
  onStart,
}: {
  gitopsDir: string;
  disabled: boolean;
  disabledReason: string | null;
  contextName: string;
  onStart: (name: string, force: boolean) => void;
}) {
  const [name, setName] = useState("");
  const [force, setForce] = useState(false);
  const [check, setCheck] = useState<GitopsTargetCheck | null>(null);
  // The name `check` was computed for. While this lags the current input
  // (debounce window or in-flight request) the check is stale and must not
  // gate Generate — otherwise typing a valid name, switching to an existing
  // one, and clicking Generate before the new check lands bypasses the gate.
  const [checkedName, setCheckedName] = useState<string | null>(null);
  const trimmedName = name.trim();

  // Debounce target check by 200ms — typing fires once after the user
  // stops, not on every keystroke. Empty name resets the check.
  useEffect(() => {
    if (!trimmedName) {
      setCheck(null);
      setCheckedName(null);
      return;
    }
    const requestedName = trimmedName;
    const t = window.setTimeout(async () => {
      try {
        const r = await api.gitopsCheckTarget(gitopsDir, requestedName);
        setCheck(r);
        setCheckedName(requestedName);
      } catch (e) {
        console.error("check target failed", e);
      }
    }, 200);
    return () => window.clearTimeout(t);
  }, [gitopsDir, trimmedName]);

  // The check applies to the current input only when it was computed for it.
  const checkCurrent = check != null && checkedName === trimmedName;

  // When the target doesn't exist, the force toggle is irrelevant —
  // reset it so a previous "force" state doesn't carry over silently.
  useEffect(() => {
    if (checkCurrent && !check.exists) setForce(false);
  }, [checkCurrent, check]);

  const exists = checkCurrent ? check.exists : false;
  const valid = checkCurrent && check.reason == null && check.writable;
  const startDisabled =
    disabled || !trimmedName || !valid || (exists && !force);

  const previewArgs: string[] = [
    "generate-gitops",
    "--context",
    contextName,
    "--dir",
    `${gitopsDir.replace(/\/$/, "")}/${name || "<name>"}`,
  ];
  if (force) previewArgs.push("--force");

  return (
    <div
      className="card"
      style={{
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 10,
        opacity: disabled ? 0.55 : 1,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "baseline" }}>
        <div className="section-title" style={{ margin: 0 }}>
          Generate new {disabled && <span style={{ color: "var(--ui-error)" }}>· disabled</span>}
        </div>
        <span className="dim mono" style={{ fontSize: "var(--fs-xxx-small)" }}>
          fleetctl generate-gitops
        </span>
      </div>
      <div className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
        Scaffolds a new gitops directory from active context{" "}
        <span className="mono" style={{ color: "var(--core-fleet-green)" }}>
          {contextName}
        </span>
        .
      </div>
      <div>
        <div style={{ fontSize: "var(--fs-xx-small)", color: "var(--app-text-dim)", marginBottom: 4 }}>
          Subdirectory name
        </div>
        <div
          style={{
            display: "flex",
            border: `1px solid ${exists ? "var(--ui-warning)" : "var(--app-border)"}`,
            borderRadius: "var(--radius-md)",
            background: "var(--app-surface-2)",
            overflow: "hidden",
          }}
        >
          <span
            className="mono dim"
            style={{
              padding: "6px 10px",
              borderRight: "1px solid var(--app-border)",
              fontSize: "var(--fs-xx-small)",
              whiteSpace: "nowrap",
            }}
          >
            {gitopsDir.replace(/\/$/, "")}/
          </span>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="new-main-v2"
            {...noAutocorrect}
            className="mono"
            disabled={disabled}
            style={{
              flex: 1,
              background: "transparent",
              border: "none",
              outline: "none",
              padding: "6px 10px",
              fontSize: "var(--fs-x-small)",
              color: "var(--app-text)",
              minWidth: 0,
            }}
          />
        </div>
        <div style={{ marginTop: 4, fontSize: "var(--fs-xxx-small)" }}>
          {check == null ? (
            <span className="dim">type a name…</span>
          ) : check.reason ? (
            <span style={{ color: "var(--ui-error)" }}>✗ {check.reason}</span>
          ) : check.exists ? (
            <span style={{ color: "var(--ui-warning)" }}>
              ⚠ exists · {check.file_count >= 200 ? "200+" : check.file_count} file
              {check.file_count === 1 ? "" : "s"} · force required to overwrite
            </span>
          ) : (
            <span style={{ color: "var(--core-fleet-green)" }}>
              ✓ available · directory does not exist yet
            </span>
          )}
        </div>
      </div>

      <label
        style={{
          display: "flex",
          alignItems: "center",
          gap: 8,
          padding: "6px 10px",
          background: exists ? "var(--tint-warning-soft)" : "var(--app-surface-2)",
          border: `1px solid ${exists ? "var(--ui-warning)" : "var(--app-border)"}`,
          borderRadius: "var(--radius-md)",
          opacity: exists ? 1 : 0.55,
          cursor: exists ? "pointer" : "default",
        }}
      >
        <input
          type="checkbox"
          checked={force}
          onChange={(e) => setForce(e.target.checked)}
          disabled={!exists}
          style={{ accentColor: "var(--ui-warning)" }}
        />
        <span style={{ fontSize: "var(--fs-xx-small)", fontWeight: 600 }}>
          force overwrite
        </span>
        <span
          className="dim"
          style={{ fontSize: "var(--fs-xxx-small)", marginLeft: "auto" }}
        >
          {exists
            ? `will replace ${check?.file_count ?? 0} existing file${check?.file_count === 1 ? "" : "s"}`
            : "not needed · target is empty"}
        </span>
      </label>

      <div
        style={{
          background: "var(--log-bg)",
          border: "1px solid var(--app-border)",
          borderRadius: "var(--radius-sm)",
          padding: "8px 10px",
          fontFamily: "var(--font-mono)",
          fontSize: "var(--fs-xxx-small)",
          color: "var(--app-text)",
          wordBreak: "break-all",
          lineHeight: 1.5,
        }}
      >
        <span className="dim">$ </span>
        fleetctl{" "}
        {previewArgs.map((a, i) => (
          <span key={i}>
            {a.startsWith("--") ? (
              <span style={{ color: "var(--core-fleet-green)" }}>{a}</span>
            ) : (
              a
            )}
            {i < previewArgs.length - 1 ? " " : ""}
          </span>
        ))}
      </div>

      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        <button
          className="primary"
          onClick={() => onStart(name.trim(), force)}
          disabled={startDisabled}
          title={disabledReason ?? undefined}
          style={{
            padding: "6px 16px",
            ...(exists && !startDisabled
              ? {
                  // Destructive-overwrite variant: bright yellow with dark
                  // text. Use --ui-on-warning (a fixed dark that doesn't flip
                  // between themes); --core-fleet-black flips to a light color
                  // in dark mode, which rendered unreadable white-on-yellow.
                  background: "var(--ui-warning)",
                  borderColor: "var(--ui-warning)",
                  color: "var(--ui-on-warning)",
                }
              : {}),
          }}
        >
          ▶ Generate{exists ? " · overwrite" : ""}
        </button>
      </div>
    </div>
  );
}

function OutputCard({
  output,
  onStop,
}: {
  output: RunOutput | null;
  onStop: (kind: OutputKind) => void;
}) {
  const bodyRef = useRef<HTMLDivElement | null>(null);
  // Stick to the bottom as new lines come in (live tail UX). We don't
  // track user scroll-up for v1 — output box is short, the user can
  // grab the scrollbar and we'll just respect it briefly.
  useEffect(() => {
    const el = bodyRef.current;
    if (!el) return;
    el.scrollTop = el.scrollHeight;
  }, [output?.lines.length]);

  if (!output) {
    return (
      <div
        className="card"
        style={{
          flex: 1,
          padding: "var(--pad-medium)",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          color: "var(--app-text-dim)",
          fontSize: "var(--fs-xx-small)",
          minHeight: 200,
        }}
      >
        Latest run will appear here.
      </div>
    );
  }

  const pillColor = stateColor(output.state);
  const failed = output.state === "failed";

  return (
    <div
      className="card"
      style={{
        flex: 1,
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 8,
        minHeight: 0,
        minWidth: 0,
        border: failed ? "1px solid var(--ui-error)" : undefined,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <div className="section-title" style={{ margin: 0 }}>
          Last run {output.repoName ? `· ${output.repoName}` : ""}
        </div>
        <span className="dim" style={{ fontSize: "var(--fs-xxx-small)", marginLeft: "auto" }}>
          {formatTimeAgo(output.endedAt ?? output.startedAt)}
        </span>
        <span
          style={{
            fontSize: "var(--fs-xxx-small)",
            background: pillColor,
            // Yellow ("running") pill needs fixed dark text — --ui-on-warning
            // doesn't flip between themes (--core-fleet-black does, which made
            // it unreadable white-on-yellow in dark mode). Other states
            // (green / blue / red) read fine with white.
            color:
              output.state === "running"
                ? "var(--ui-on-warning)"
                : "var(--core-fleet-white)",
            padding: "1px 6px",
            borderRadius: 3,
            textTransform: "uppercase",
            letterSpacing: "0.05em",
            fontWeight: 600,
          }}
        >
          {pillLabel(output.state)}
        </span>
        {output.state === "running" && (
          <button
            className="danger"
            onClick={() => onStop(output.kind)}
            style={{ padding: "2px 8px", fontSize: "var(--fs-xxx-small)" }}
          >
            Stop
          </button>
        )}
      </div>
      <div
        ref={bodyRef}
        style={{
          flex: 1,
          background: "var(--log-bg)",
          border: failed
            ? "1px solid var(--ui-error)"
            : "1px solid var(--app-border)",
          borderRadius: "var(--radius-sm)",
          padding: "8px 10px",
          fontFamily: "var(--font-mono)",
          fontSize: "var(--fs-xxx-small)",
          color: "var(--app-text)",
          lineHeight: 1.5,
          overflow: "auto",
          minHeight: 0,
        }}
      >
        <div
          className="dim"
          style={{
            whiteSpace: "pre-wrap",
            wordBreak: "break-all",
            marginBottom: 6,
          }}
        >
          $ {output.command}
        </div>
        {output.lines.length === 0 ? (
          <div className="dim" style={{ opacity: 0.7 }}>
            (no output yet…)
          </div>
        ) : (
          output.lines.map((l, i) => (
            <div
              key={i}
              style={{
                color:
                  l.stream === "stderr"
                    ? "var(--ui-error)"
                    : "var(--app-text)",
                whiteSpace: "pre-wrap",
                wordBreak: "break-all",
              }}
            >
              {l.line}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

/* --------------- Small helpers --------------- */

function EmptyState({
  title,
  body,
  cta,
}: {
  title: string;
  body: string;
  cta?: { label: string; onClick: () => void };
}) {
  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        justifyContent: "center",
        color: "var(--app-text-dim)",
        gap: 8,
        padding: "var(--pad-large)",
      }}
    >
      <div style={{ fontSize: "var(--fs-medium)", color: "var(--app-text)" }}>
        {title}
      </div>
      <div style={{ fontSize: "var(--fs-xx-small)", textAlign: "center", maxWidth: 480 }}>
        {body}
      </div>
      {cta && (
        <button
          className="primary"
          onClick={cta.onClick}
          style={{ marginTop: 8, padding: "6px 14px" }}
        >
          {cta.label}
        </button>
      )}
    </div>
  );
}

function applyArgsFor(
  selected: GitopsFile[],
  ctx: string,
  dryRun: boolean,
): string[] {
  // `fleetctl gitops -f <path> [-f <path> ...] --context <name>
  // [--dry-run]`. Paths are relative to repo cwd (we spawn from the
  // repo dir). default.yml uses the empty-subdir sentinel so it lands
  // as a bare filename; team/fleet entries get their subdir prefix.
  const args: string[] = ["gitops"];
  for (const f of selected) {
    const rel = f.subdir === "" ? f.name : `${f.subdir}/${f.name}`;
    args.push("-f", rel);
  }
  args.push("--context", ctx);
  if (dryRun) args.push("--dry-run");
  return args;
}

function stateColor(s: OutputState): string {
  switch (s) {
    case "applied":
      return "var(--core-fleet-green)";
    case "dry-run":
      return "var(--core-vibrant-blue)";
    case "failed":
      return "var(--ui-error)";
    case "running":
      return "var(--ui-warning)";
  }
}

function pillLabel(s: OutputState): string {
  switch (s) {
    case "applied":
      return "applied";
    case "dry-run":
      return "dry run";
    case "failed":
      return "failed";
    case "running":
      return "running";
  }
}

function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

function formatTimeAgo(ms: number): string {
  if (!ms) return "—";
  const diff = Date.now() - ms;
  const sec = Math.floor(diff / 1000);
  if (sec < 5) return "just now";
  if (sec < 60) return `${sec}s ago`;
  const min = Math.floor(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr}h ago`;
  return `${Math.floor(hr / 24)}d ago`;
}
