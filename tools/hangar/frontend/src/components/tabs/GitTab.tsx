import { useEffect, useMemo, useRef, useState } from "react";
import { flushSync } from "react-dom";
import {
  api,
  type Branch,
  type BranchStatus,
} from "../../lib/ipc";
import { noAutocorrect } from "../../lib/noAutocorrect";

type Filter = "rc" | "main" | "all";

const DEFAULT_RC_MINORS = 10;
const ALL_LIMIT = 200;
// Max matches returned for a name search. Like ALL_LIMIT this bounds how many
// rows we render; unlike ALL_LIMIT the search runs server-side across the full
// ref set, so a match is found regardless of how stale the branch is.
const SEARCH_LIMIT = 200;
// Debounce before a search keystroke fires an IPC call. The client-side
// filter narrows the already-loaded list instantly; this only gates the
// server round-trip that widens the set to branches outside that window.
const SEARCH_DEBOUNCE_MS = 200;

// limitFor picks the row cap for a load: a flat search cap when searching,
// otherwise the per-filter recency cap (RC minors / all / unbounded main).
function limitFor(
  filter: Filter,
  rcMinors: number,
  query: string,
): number | undefined {
  if (query.trim()) return SEARCH_LIMIT;
  return filter === "rc" ? rcMinors : filter === "all" ? ALL_LIMIT : undefined;
}

// Cooldown for the auto-fetch on Git-tab open. GitTab remounts every time
// the tab is selected, so without this, rapid tab in/out would re-fetch
// (and could even run two `git fetch` at once → .git/index.lock errors).
// Module-level so it survives the unmount/remount cycle; keyed per repo.
// The manual Fetch button ignores this and always fetches.
const AUTO_FETCH_COOLDOWN_MS = 20_000;
const lastFetchAt = new Map<string, number>();

export function GitTab({
  repoPath,
  branchStatus,
  refreshBranchStatus,
}: {
  repoPath: string | null;
  branchStatus: BranchStatus | null;
  refreshBranchStatus: () => Promise<void>;
}) {
  const [branches, setBranches] = useState<Branch[]>([]);
  const [filter, setFilter] = useState<Filter>("rc");
  const [rcMinors, setRcMinors] = useState<number>(DEFAULT_RC_MINORS);
  const [search, setSearch] = useState("");
  // The debounced search actually sent to the backend. `search` updates per
  // keystroke (drives the instant client-side filter); this trails it.
  const [debouncedSearch, setDebouncedSearch] = useState("");
  // The query the currently-loaded `branches` were fetched for. When it lags
  // `search` a server fetch is pending, so we show "Searching…" rather than a
  // premature "No branches match." for a branch outside the loaded window.
  const [loadedQuery, setLoadedQuery] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pendingCheckout, setPendingCheckout] = useState<string | null>(null);

  async function loadBranches() {
    if (!repoPath) return;
    try {
      const query = search.trim();
      const list = await api.gitListBranches(
        repoPath,
        filter,
        query,
        limitFor(filter, rcMinors, query),
      );
      setBranches(list);
      setLoadedQuery(query);
    } catch (e) {
      setError(String(e));
    }
  }

  // Debounce the search box into debouncedSearch (the value sent to the
  // backend). Each keystroke still narrows the loaded list instantly via the
  // client-side `filtered` memo; this just gates the server round-trip.
  useEffect(() => {
    const id = setTimeout(() => setDebouncedSearch(search), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(id);
  }, [search]);

  // Rapid filter switches (rc → main → all) or fast typing can resolve out of
  // order — "last response wins" is whichever IPC happened to finish last,
  // not the most recent input. Bump a request id and ignore stale responses.
  const loadReqRef = useRef(0);
  useEffect(() => {
    if (!repoPath) return;
    const reqId = ++loadReqRef.current;
    const query = debouncedSearch.trim();
    api
      .gitListBranches(repoPath, filter, query, limitFor(filter, rcMinors, query))
      .then((list) => {
        if (reqId === loadReqRef.current) {
          setBranches(list);
          setLoadedQuery(query);
        }
      })
      .catch((e) => {
        if (reqId === loadReqRef.current) setError(String(e));
      });
  }, [repoPath, filter, rcMinors, debouncedSearch]);

  // Auto-fetch when the tab is opened. GitTab is conditionally rendered
  // in App, so it unmounts on tab-switch and remounts on return — this
  // mount effect is effectively "fetch on Git-tab selection." The bet:
  // if you opened Git, you want the latest remote state (ahead/behind
  // reflecting real upstream), so pay one network fetch now rather than
  // make you click. Bounded — one fetch per tab open, never background.
  //
  // We deliberately wait two animation frames before firing so the tab
  // paints its local branches/status FIRST, then the fetch kicks off
  // through the same `doFetch` path as the button — the user sees the
  // tab open instantly and watches the Fetch spinner spin, instead of
  // staring at what feels like a frozen open while we block on the
  // network. Reusing doFetch keeps the busy/error UX identical to a
  // manual click.
  //
  // The ref guard fires it once per real mount. It's set INSIDE the rAF
  // (not at effect entry) on purpose: in dev StrictMode the effect runs
  // → cleanup (cancels the rAF) → runs again on the same instance. If we
  // set the guard at entry, the second run would see it already true and
  // bail, and since the first rAF was cancelled the fetch would never
  // fire at all. Setting it just before doFetch means a cancelled frame
  // doesn't consume the guard, so the second run re-schedules a frame
  // that actually runs. A real remount (new instance) fetches again.
  const didAutoFetch = useRef(false);
  useEffect(() => {
    if (!repoPath || didAutoFetch.current) return;
    // Skip if we fetched this repo recently — keeps re-opening the tab
    // from re-fetching (and from racing an in-flight fetch on a fast
    // tab-out/tab-in). Local branches/status still render immediately.
    if (Date.now() - (lastFetchAt.get(repoPath) ?? 0) < AUTO_FETCH_COOLDOWN_MS) {
      didAutoFetch.current = true;
      return;
    }
    const id = requestAnimationFrame(() =>
      requestAnimationFrame(() => {
        didAutoFetch.current = true;
        void doFetch();
      }),
    );
    return () => cancelAnimationFrame(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repoPath]);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return branches;
    return branches.filter((b) => b.name.toLowerCase().includes(q));
  }, [branches, search]);

  // A server fetch for the current query hasn't landed yet (debounce window
  // or in-flight IPC). Used to avoid flashing "No branches match." before a
  // stale branch outside the loaded window has had a chance to load.
  const searchPending = search.trim() !== loadedQuery;

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

  async function withBusy(
    tag: string,
    work: () => Promise<void>,
  ): Promise<void> {
    flushSync(() => {
      setBusy(tag);
      setError(null);
    });
    // Wait for an actual paint. requestAnimationFrame fires BEFORE paint,
    // so we need two: the second rAF callback runs after the first frame
    // has rendered to the screen.
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() =>
        requestAnimationFrame(() => resolve()),
      ),
    );
    try {
      await work();
    } catch (e) {
      setError(String(e));
    }
    setBusy(null);
  }

  async function doFetch() {
    // Record at start so a remount during the in-flight fetch (or right
    // after) honors the cooldown instead of kicking off a second fetch.
    if (repoPath) lastFetchAt.set(repoPath, Date.now());
    await withBusy("fetch", async () => {
      await api.gitFetch(repoPath!);
      await loadBranches();
      await refreshBranchStatus();
    });
  }

  async function doPull() {
    await withBusy("pull", async () => {
      await api.gitPull(repoPath!);
      await loadBranches();
      await refreshBranchStatus();
    });
  }

  function isDirtyBlockError(msg: string): boolean {
    return (
      msg.includes("would be overwritten") ||
      msg.includes("Please commit your changes or stash them") ||
      msg.includes("Please move or remove them")
    );
  }

  async function checkout(branch: string) {
    setBusy(branch);
    setError(null);
    try {
      await api.gitCheckout(repoPath!, branch);
      await loadBranches();
      await refreshBranchStatus();
    } catch (e) {
      const msg = String(e);
      if (isDirtyBlockError(msg)) {
        setPendingCheckout(branch);
      } else {
        setError(msg);
      }
    }
    setBusy(null);
  }

  async function doCheckout(branch: string, mode: "stash" | "discard") {
    setBusy(branch);
    setError(null);
    setPendingCheckout(null);
    try {
      if (mode === "stash") {
        await api.gitStashAndCheckout(repoPath!, branch);
      } else {
        await api.gitDiscardAndCheckout(repoPath!, branch);
      }
      await loadBranches();
      await refreshBranchStatus();
    } catch (e) {
      setError(String(e));
    }
    setBusy(null);
  }

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        padding: "var(--pad-large)",
        gap: "var(--pad-medium)",
        overflow: "hidden",
      }}
    >
      {/* Hero */}
      <div className="card" style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 10,
              marginBottom: 4,
            }}
          >
            <span
              className={`dot ${
                !branchStatus
                  ? "idle"
                  : branchStatus.behind > 0
                    ? "warn"
                    : "ok"
              }`}
            />
            <span
              className="mono"
              style={{
                fontSize: "var(--fs-x-small)",
                fontWeight: 600,
                color: "var(--app-text)",
              }}
            >
              {branchStatus?.branch ?? "—"}
            </span>
            <span
              className="dim"
              style={{ fontSize: "var(--fs-xx-small)" }}
            >
              {branchStatus
                ? branchStatus.behind > 0
                  ? `${branchStatus.behind} behind`
                  : "up to date"
                : ""}
              {branchStatus && branchStatus.ahead > 0
                ? ` · ${branchStatus.ahead} ahead`
                : ""}
              {branchStatus && branchStatus.modified.length > 0
                ? ` · ${branchStatus.modified.length} modified`
                : ""}
            </span>
          </div>
          {branchStatus?.last_commit && (
            <div
              className="dim"
              style={{
                fontSize: "var(--fs-xx-small)",
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
              }}
            >
              <span className="mono">{branchStatus.last_commit.sha}</span> ·{" "}
              {branchStatus.last_commit.subject} · {branchStatus.last_commit.author}{" "}
              · {branchStatus.last_commit.time_ago}
            </div>
          )}
        </div>
        <div style={{ display: "flex", gap: 6 }}>
          <button onClick={doFetch} disabled={!!busy}>
            {busy === "fetch" ? (
              <>
                <span className="spin" style={{ display: "inline-block" }}>
                  ↻
                </span>{" "}
                fetching…
              </>
            ) : (
              "↻ Fetch"
            )}
          </button>
          <button
            className="primary"
            onClick={doPull}
            disabled={!!busy || !branchStatus || branchStatus.behind === 0}
            title={
              branchStatus && branchStatus.behind === 0
                ? "Already up to date"
                : undefined
            }
          >
            {busy === "pull" ? (
              <>
                <span className="spin" style={{ display: "inline-block" }}>
                  ↻
                </span>{" "}
                pulling…
              </>
            ) : (
              "↓ Pull"
            )}
          </button>
        </div>
      </div>

      {/* Picker */}
      <div
        className="card"
        style={{
          flex: 1,
          minHeight: 0,
          display: "flex",
          flexDirection: "column",
          padding: 0,
        }}
      >
        <div
          style={{
            padding: "var(--pad-medium)",
            display: "flex",
            gap: 10,
            alignItems: "center",
            borderBottom: "1px solid var(--app-border)",
          }}
        >
          <input
            placeholder="Search branches…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{ flex: 1 }}
            {...noAutocorrect}
          />
          <FilterPill
            label={
              filter === "rc"
                ? `RC branches · last ${rcMinors} minors`
                : "RC branches"
            }
            active={filter === "rc"}
            onClick={() => setFilter("rc")}
          />
          {filter === "rc" && (
            <select
              value={rcMinors}
              onChange={(e) => setRcMinors(Number(e.target.value))}
              style={{ padding: "2px 6px", fontSize: "var(--fs-xx-small)" }}
              title="How many recent minor release lines to load (plus all their patches)"
            >
              <option value={5}>5</option>
              <option value={10}>10</option>
              <option value={20}>20</option>
              <option value={50}>50</option>
            </select>
          )}
          <FilterPill
            label="main"
            active={filter === "main"}
            onClick={() => setFilter("main")}
          />
          <FilterPill
            label="all"
            active={filter === "all"}
            onClick={() => setFilter("all")}
          />
        </div>
        <div style={{ flex: 1, overflow: "auto" }}>
          {filtered.map((b) => (
            <BranchRow
              key={b.name}
              branch={b}
              busy={busy === b.name}
              onCheckout={() => checkout(b.name)}
            />
          ))}
          {filtered.length === 0 && (
            <div
              style={{
                padding: "var(--pad-large)",
                textAlign: "center",
                color: "var(--app-text-dim)",
                fontSize: "var(--fs-xx-small)",
              }}
            >
              {searchPending ? "Searching…" : "No branches match."}
            </div>
          )}
        </div>
      </div>

      {error && (
        <div
          style={{
            color: "var(--ui-error)",
            fontSize: "var(--fs-xx-small)",
          }}
        >
          {error}
        </div>
      )}

      {pendingCheckout && branchStatus && (
        <DirtyConfirm
          target={pendingCheckout}
          status={branchStatus}
          onCancel={() => setPendingCheckout(null)}
          onStash={() => doCheckout(pendingCheckout, "stash")}
          onDiscard={() => doCheckout(pendingCheckout, "discard")}
        />
      )}
    </div>
  );
}

function FilterPill({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      style={{
        padding: "4px 10px",
        fontSize: "var(--fs-xx-small)",
        borderRadius: 999,
        background: active ? "var(--tint-success-strong)" : undefined,
        borderColor: active
          ? "var(--core-fleet-green)"
          : "var(--app-border)",
        color: active ? "var(--core-fleet-green)" : "var(--app-text-dim)",
      }}
    >
      {label}
    </button>
  );
}

function BranchRow({
  branch,
  busy,
  onCheckout,
}: {
  branch: Branch;
  busy: boolean;
  onCheckout: () => void;
}) {
  const tag = branch.is_current
    ? "checked out"
    : branch.is_local
      ? "local"
      : "remote";
  const dotClass = branch.is_current
    ? "ok"
    : branch.is_local
      ? "warn"
      : "idle";
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px var(--pad-medium)",
        borderBottom: "1px solid var(--app-border)",
      }}
    >
      <span className={`dot ${dotClass}`} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          style={{
            display: "flex",
            alignItems: "baseline",
            gap: 8,
            overflow: "hidden",
          }}
        >
          <span
            className="mono"
            style={{
              color: "var(--app-text)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {branch.name}
          </span>
          <span
            style={{
              fontSize: "var(--fs-xxx-small)",
              textTransform: "uppercase",
              letterSpacing: "0.06em",
              color: "var(--app-text-dim)",
            }}
          >
            {tag}
          </span>
        </div>
        {branch.last_commit && (
          <div
            className="dim"
            style={{
              fontSize: "var(--fs-xx-small)",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {branch.last_commit.subject} · {branch.last_commit.author} ·{" "}
            {branch.last_commit.time_ago}
          </div>
        )}
      </div>
      {!branch.is_current && (
        <button onClick={onCheckout} disabled={busy}>
          {busy ? "…" : "Checkout"}
        </button>
      )}
    </div>
  );
}

function DirtyConfirm({
  target,
  status,
  onCancel,
  onStash,
  onDiscard,
}: {
  target: string;
  status: BranchStatus;
  onCancel: () => void;
  onStash: () => void;
  onDiscard: () => void;
}) {
  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "var(--overlay-modal)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 10,
      }}
      onClick={onCancel}
    >
      <div
        onClick={(e) => e.stopPropagation()}
        style={{
          background: "var(--app-surface)",
          border: "1px solid var(--ui-warning)",
          borderRadius: "var(--radius-xl)",
          padding: "var(--pad-large)",
          width: 520,
          maxHeight: "70vh",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-medium)",
          boxShadow: "var(--shadow-modal)",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 8,
            color: "var(--ui-warning)",
            fontWeight: 600,
          }}
        >
          ⚠ Working tree has uncommitted changes
        </div>
        <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
          You're about to check out <span className="mono">{target}</span>.{" "}
          {status.modified.length} modified files would be carried over or lost.
        </div>
        <div
          style={{
            flex: 1,
            overflow: "auto",
            background: "var(--app-surface-2)",
            border: "1px solid var(--app-border)",
            borderRadius: "var(--radius-md)",
            padding: "var(--pad-small)",
            fontSize: "var(--fs-xx-small)",
          }}
        >
          {status.modified.map((m, i) => (
            <div key={i} className="mono" style={{ display: "flex", gap: 8 }}>
              <span style={{ color: "var(--ui-warning)", width: 16 }}>
                {m.status}
              </span>
              <span>{m.path}</span>
            </div>
          ))}
        </div>
        <div style={{ display: "flex", justifyContent: "flex-end", gap: 6 }}>
          <button onClick={onCancel}>Cancel</button>
          <button onClick={onStash}>Stash & checkout</button>
          <button className="danger" onClick={onDiscard}>
            Discard & checkout
          </button>
        </div>
      </div>
    </div>
  );
}
