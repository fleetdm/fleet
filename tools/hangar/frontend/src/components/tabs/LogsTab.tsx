import { useEffect, useMemo, useState } from "react";
import {
  List,
  useListRef,
  type RowComponentProps,
} from "react-window";
import {
  api,
  type LogEntry,
  type LogWindow,
  type ServerProfile,
} from "../../lib/ipc";
import { serveChannel } from "../../lib/servers";
import { noAutocorrect } from "../../lib/noAutocorrect";

type Level = "debug" | "info" | "warn" | "error";

const WINDOWS: { label: string; ms: number | null }[] = [
  { label: "1m", ms: 60_000 },
  { label: "5m", ms: 5 * 60_000 },
  { label: "10m", ms: 10 * 60_000 },
  { label: "30m", ms: 30 * 60_000 },
  { label: "1h", ms: 60 * 60_000 },
  { label: "all", ms: null },
];

const ALL_LEVELS: Level[] = ["debug", "info", "warn", "error"];
const DEFAULT_LEVELS: Set<Level> = new Set(ALL_LEVELS);

const POLL_MS = 1500;
const MAX_RENDER_LINES = 2000;
const LOG_ROW_HEIGHT = 20;
// Approximate width of a monospace character at --fs-xx-small (12px).
// Empirical — slightly over the actual ~7.2px so end-of-line content
// isn't clipped by the row's right edge.
const LOG_MONO_CHAR_PX = 7.5;
// Fixed left columns: time span (~8 chars + gap) + level column (48px)
// + outer paddings. Adjust here if the row header changes.
const LOG_ROW_FIXED_LEFT_PX = 150;
const LOG_ROW_RIGHT_PAD_PX = 16;
const LOG_MIN_ROW_WIDTH_PX = 600;
// How far above the bottom edge a user has to scroll before we
// interpret it as "deliberately scrolled away" and disable follow.
// A few rows of slack lets a click-with-momentum coast a bit without
// flipping follow off.
const FOLLOW_BOTTOM_TOLERANCE_PX = 40;

export function LogsTab({ server }: { server: ServerProfile }) {
  // Logs are per-server: each server's fleet-serve output lives in its own
  // structured channel (fleet-serve-<id>).
  const SOURCE = serveChannel(server.id);
  const [windowMs, setWindowMs] = useState<number | null>(10 * 60_000);
  const [search, setSearch] = useState("");
  const [levels, setLevels] = useState<Set<Level>>(new Set(DEFAULT_LEVELS));
  const [follow, setFollow] = useState(true);
  const [data, setData] = useState<LogWindow>({
    entries: [],
    total_in_window: 0,
    warn_count: 0,
    error_count: 0,
  });
  // Snapshot result toast — the path is shown so the user knows where
  // the file landed and the Reveal button drives the system file
  // manager. Auto-dismisses on the next snapshot or after 6s.
  const [snapToast, setSnapToast] = useState<{
    path: string;
    clipboardOk: boolean;
  } | null>(null);
  const listApiRef = useListRef(null);

  useEffect(() => {
    if (!snapToast) return;
    const t = window.setTimeout(() => setSnapToast(null), 6000);
    return () => window.clearTimeout(t);
  }, [snapToast]);

  async function takeSnapshot() {
    if (data.entries.length === 0) return;
    const text = formatEntriesForSnapshot(data.entries);
    const filename = `${SOURCE}-${snapshotStamp(new Date())}.log`;
    let clipboardOk = false;
    try {
      await navigator.clipboard.writeText(text);
      clipboardOk = true;
    } catch (e) {
      // Webview may refuse clipboard in some configurations — fail the
      // clipboard write loudly to console but still try to save the
      // file, since the on-disk artifact is the more durable half.
      console.warn("clipboard write failed", e);
    }
    try {
      const path = await api.saveLogSnapshot(filename, text);
      setSnapToast({ path, clipboardOk });
    } catch (e) {
      console.error("saveLogSnapshot", e);
    }
  }

  async function refresh() {
    try {
      const since = windowMs ? Date.now() - windowMs : 0;
      const w = await api.readLogWindow({
        source: SOURCE,
        since_ms: since,
        levels: Array.from(levels),
        search: search.trim() ? search.trim() : null,
        max_lines: MAX_RENDER_LINES,
      });
      setData(w);
    } catch (e) {
      console.error("readLogWindow", e);
    }
  }

  // Debounce search input so each keystroke doesn't fire a Rust call
  // (the underlying ring buffer can be up to LOG_CHANNEL_CAP entries —
  // scanning it on every key is wasteful and visibly stutters).
  const [debouncedSearch, setDebouncedSearch] = useState(search);
  useEffect(() => {
    const t = window.setTimeout(() => setDebouncedSearch(search), 200);
    return () => window.clearTimeout(t);
  }, [search]);

  useEffect(() => {
    refresh();
    if (!follow) return;
    const id = window.setInterval(refresh, POLL_MS);
    return () => window.clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [windowMs, debouncedSearch, levels, follow, SOURCE]);

  // Auto-scroll to bottom when following and new lines come in. With
  // the virtualized list we drive this via the imperative API rather
  // than directly mutating scrollTop on the container.
  useEffect(() => {
    if (!follow || data.entries.length === 0) return;
    listApiRef.current?.scrollToRow({
      index: data.entries.length - 1,
      align: "end",
    });
  }, [data.entries, follow]);

  function toggleLevel(l: Level) {
    setLevels((prev) => {
      const next = new Set(prev);
      if (next.has(l)) next.delete(l);
      else next.add(l);
      return next;
    });
  }

  const windowLabel =
    WINDOWS.find((w) => w.ms === windowMs)?.label ?? "custom";

  // Width of every row in the virtualized list. Sized to the *longest
  // visible* message so a single horizontal scrollbar lives at the
  // bottom of the list (instead of one per row). Recomputed on every
  // poll — cheap (linear scan), keeps the scroll travel honest as new
  // long lines arrive.
  const rowWidth = useMemo(() => {
    let maxLen = 0;
    for (const e of data.entries) {
      if (e.message.length > maxLen) maxLen = e.message.length;
    }
    const messagePx = maxLen * LOG_MONO_CHAR_PX;
    return Math.max(
      LOG_MIN_ROW_WIDTH_PX,
      LOG_ROW_FIXED_LEFT_PX + messagePx + LOG_ROW_RIGHT_PAD_PX,
    );
  }, [data.entries]);

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        background: "var(--app-bg)",
        position: "relative",
      }}
    >
      {/* Toolbar */}
      <div
        style={{
          padding: "var(--pad-small) var(--pad-medium)",
          borderBottom: "1px solid var(--app-border)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-small)",
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "var(--pad-small)",
            flexWrap: "wrap",
          }}
        >
          <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
            show last
          </span>
          <WindowPicker value={windowMs} onChange={setWindowMs} />
          <span
            className="dim mono"
            style={{ fontSize: "var(--fs-xx-small)", marginLeft: 4 }}
          >
            ~ {data.total_in_window} lines
          </span>
          <input
            placeholder="filter — type a word, or /regex/"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="mono"
            autoComplete="off"
            {...noAutocorrect}
            // Fixed width + auto left margin: input stays glued to the
            // follow/snapshot buttons on the right and doesn't jitter
            // when the "~ N lines" counter changes width.
            style={{ width: 360, minWidth: 200, marginLeft: "auto" }}
          />
          <button
            onClick={() => setFollow((v) => !v)}
            style={{
              padding: "4px 10px",
              fontSize: "var(--fs-xx-small)",
              background: follow ? "var(--tint-success-soft)" : undefined,
              borderColor: follow
                ? "var(--core-fleet-green)"
                : "var(--app-border)",
              color: follow ? "var(--core-fleet-green)" : "var(--app-text-dim)",
            }}
          >
            <span className={`dot ${follow ? "run" : "idle"}`} /> Follow
          </button>
          <ClearButton
            onClear={async () => {
              await api.clearLogChannel(SOURCE);
              await refresh();
            }}
          />
          <button
            style={{ padding: "4px 10px", fontSize: "var(--fs-xx-small)" }}
            onClick={async () => {
              try {
                const dir = await api.logsDir();
                await api.openPath(dir);
              } catch (e) {
                console.error("logsDir/openPath", e);
              }
            }}
            title="Open the on-disk log folder in Finder"
          >
            Reveal in Finder
          </button>
          <button
            className="primary"
            style={{ padding: "4px 12px", fontSize: "var(--fs-xx-small)" }}
            onClick={takeSnapshot}
            disabled={data.entries.length === 0}
            title={
              data.entries.length === 0
                ? "Nothing to snapshot in this view"
                : "Save the current view to disk and copy to clipboard"
            }
          >
            📋 Snapshot
          </button>
        </div>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: "var(--pad-small)",
          }}
        >
          {ALL_LEVELS.map((l) => (
            <LevelChip
              key={l}
              level={l}
              on={levels.has(l)}
              onClick={() => toggleLevel(l)}
            />
          ))}
          <div style={{ marginLeft: "auto", display: "flex", gap: 12 }}>
            {data.warn_count > 0 && (
              <span
                style={{
                  color: "var(--ui-warning)",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                {data.warn_count} warning{data.warn_count === 1 ? "" : "s"}
              </span>
            )}
            {data.error_count > 0 && (
              <span
                style={{
                  color: "var(--ui-error)",
                  fontSize: "var(--fs-xx-small)",
                }}
              >
                {data.error_count} error{data.error_count === 1 ? "" : "s"}
              </span>
            )}
            <span className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
              in last {windowLabel}
            </span>
          </div>
        </div>
      </div>

      {/* Body — virtualized so a 2000-row window stays responsive even
          while the 1.5s poll keeps re-rendering. */}
      <div
        style={{
          flex: 1,
          minHeight: 0,
          background: "var(--log-bg)",
        }}
      >
        {data.entries.length === 0 ? (
          <div
            className="dim"
            style={{
              padding: "var(--pad-large)",
              textAlign: "center",
              fontSize: "var(--fs-xx-small)",
            }}
          >
            No log lines in this window. Start{" "}
            <span className="mono">fleet serve</span> from the Server tab, then
            come back.
          </div>
        ) : (
          <List
            listRef={listApiRef}
            rowCount={data.entries.length}
            rowHeight={LOG_ROW_HEIGHT}
            rowComponent={LogRow}
            rowProps={{ entries: data.entries, rowWidth }}
            // Single horizontal scrollbar at the bottom of the list:
            // we widen every row to the longest visible message's
            // width, so the List's outer container becomes the
            // horizontal scroller. macOS overlay scrollbar = it only
            // appears when you scroll, so short content reads clean.
            style={{ height: "100%", padding: "var(--pad-small) 0" }}
            onScroll={(e) => {
              const el = e.currentTarget as HTMLDivElement;
              // "Distance from bottom" instead of absolute scrollTop.
              // When entries age out of a short window the list shrinks
              // and the browser clamps scrollTop downward — under the
              // old `scrollTop < prev` heuristic that looked identical
              // to a user scroll-up and killed follow. Tracking the
              // distance from the bottom edge avoids the false positive:
              // if we were at the bottom, we stay at the bottom after a
              // shrink, distance ~= 0 either way.
              const distFromBottom =
                el.scrollHeight - el.clientHeight - el.scrollTop;
              if (follow && distFromBottom > FOLLOW_BOTTOM_TOLERANCE_PX) {
                setFollow(false);
              }
            }}
          />
        )}
      </div>

      {/* Bottom ribbon */}
      <div
        style={{
          padding: "4px var(--pad-medium)",
          borderTop: "1px solid var(--app-border)",
          fontSize: "var(--fs-xx-small)",
          color: "var(--app-text-dim)",
          display: "flex",
          justifyContent: "space-between",
        }}
      >
        <span>
          ━━ {follow ? "live" : "paused"} · last {windowLabel} window ·{" "}
          {data.entries.length} lines ━━
        </span>
      </div>

      {snapToast && (
        <SnapshotToast
          path={snapToast.path}
          clipboardOk={snapToast.clipboardOk}
          onDismiss={() => setSnapToast(null)}
        />
      )}
    </div>
  );
}

function SnapshotToast({
  path,
  clipboardOk,
  onDismiss,
}: {
  path: string;
  clipboardOk: boolean;
  onDismiss: () => void;
}) {
  // Show just the basename so the toast stays narrow — the full path
  // is on the title attribute for hover, and Reveal opens the file in
  // Finder which makes the full location moot.
  const basename = path.split("/").pop() ?? path;
  return (
    <div
      role="status"
      style={{
        position: "absolute",
        right: "var(--pad-medium)",
        bottom: 36,
        background: "var(--app-surface)",
        border: "1px solid var(--core-fleet-green)",
        borderRadius: "var(--radius-md)",
        padding: "8px 12px",
        display: "flex",
        alignItems: "center",
        gap: 10,
        boxShadow: "var(--shadow-popover)",
        fontSize: "var(--fs-xx-small)",
        zIndex: 20,
      }}
    >
      <span style={{ color: "var(--core-fleet-green)" }}>✓ Saved</span>
      <span className="mono" title={path}>
        {basename}
      </span>
      {clipboardOk && (
        <span className="dim" style={{ fontSize: "var(--fs-xxx-small)" }}>
          · copied to clipboard
        </span>
      )}
      <button
        onClick={() => api.openPath(path, true).catch(console.error)}
        style={{ padding: "2px 8px", fontSize: "var(--fs-xxx-small)" }}
      >
        Reveal
      </button>
      <button
        onClick={onDismiss}
        style={{ padding: "2px 8px", fontSize: "var(--fs-xxx-small)" }}
        title="Dismiss"
      >
        ✕
      </button>
    </div>
  );
}

/// Plain-text rendering that mirrors what the user sees in the row
/// list: `HH:MM:SS LEVEL message`. Level pads to 5 chars so columns
/// line up in a fixed-width viewer. Entries are already sorted
/// ascending by ts in the LogWindow payload.
function formatEntriesForSnapshot(entries: LogEntry[]): string {
  const lines = entries.map((e) => {
    const t = formatTime(e.ts_ms);
    const lvl = (e.level ?? "info").toUpperCase().padEnd(5);
    return `${t} ${lvl} ${e.message}`;
  });
  return lines.join("\n") + "\n";
}

/// Filename-safe timestamp like `2026-05-25_14-32-01`. Colons and
/// spaces avoided because some shells / older tooling get cranky.
function snapshotStamp(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}_${pad(
    d.getHours(),
  )}-${pad(d.getMinutes())}-${pad(d.getSeconds())}`;
}

/// Two-step confirm: first click arms the button (3s window), second
/// click does the wipe. A modal would be overkill for a Logs-tab
/// action; an unconfirmed click would be too easy to fire by accident
/// since the button sits next to Follow. The 3s timer is short enough
/// that a stale armed state doesn't surprise a returning user.
function ClearButton({ onClear }: { onClear: () => Promise<void> }) {
  const [phase, setPhase] = useState<"idle" | "armed" | "clearing">("idle");

  useEffect(() => {
    if (phase !== "armed") return;
    const t = window.setTimeout(() => setPhase("idle"), 3000);
    return () => window.clearTimeout(t);
  }, [phase]);

  async function onClick() {
    if (phase === "clearing") return;
    if (phase === "idle") {
      setPhase("armed");
      return;
    }
    setPhase("clearing");
    try {
      await onClear();
    } catch (e) {
      console.error("clearLogChannel", e);
    }
    setPhase("idle");
  }

  const label =
    phase === "armed"
      ? "Click again to clear"
      : phase === "clearing"
        ? "clearing…"
        : "Clear";
  const armed = phase === "armed";
  return (
    <button
      onClick={onClick}
      disabled={phase === "clearing"}
      title="Wipes in-memory buffer and on-disk log file"
      className={armed ? "danger" : undefined}
      style={{
        padding: "4px 10px",
        fontSize: "var(--fs-xx-small)",
        ...(armed
          ? {}
          : {
              borderColor: "var(--app-border)",
              color: "var(--app-text-dim)",
            }),
      }}
    >
      🗑 {label}
    </button>
  );
}

function WindowPicker({
  value,
  onChange,
}: {
  value: number | null;
  onChange: (v: number | null) => void;
}) {
  return (
    <div style={{ display: "flex", gap: 4 }}>
      {WINDOWS.map((w) => {
        const active = value === w.ms;
        return (
          <button
            key={w.label}
            onClick={() => onChange(w.ms)}
            className="mono"
            style={{
              padding: "3px 10px",
              fontSize: "var(--fs-xx-small)",
              borderRadius: "var(--radius)",
              background: active ? "var(--core-fleet-green)" : undefined,
              borderColor: active
                ? "var(--core-fleet-green)"
                : "var(--app-border)",
              color: active ? "var(--core-fleet-white)" : "var(--app-text-dim)",
              fontWeight: active ? 600 : 400,
            }}
          >
            {w.label}
          </button>
        );
      })}
    </div>
  );
}

function LevelChip({
  level,
  on,
  onClick,
}: {
  level: Level;
  on: boolean;
  onClick: () => void;
}) {
  const colors: Record<Level, string> = {
    debug: "var(--app-text-dim)",
    info: "var(--core-vibrant-blue)",
    warn: "var(--ui-warning)",
    error: "var(--core-vibrant-red)",
  };
  const c = colors[level];
  return (
    <button
      onClick={onClick}
      style={{
        padding: "3px 10px",
        fontSize: "var(--fs-xx-small)",
        borderRadius: 999,
        borderColor: c,
        color: on ? c : "var(--app-text-dim)",
        background: on ? `${c}22` : undefined,
        opacity: on ? 1 : 0.55,
        textTransform: "uppercase",
        letterSpacing: "0.05em",
        fontWeight: 600,
      }}
    >
      {level}
    </button>
  );
}

function LogRow({
  index,
  style,
  entries,
  rowWidth,
}: RowComponentProps<{ entries: LogEntry[]; rowWidth: number }>) {
  const entry = entries[index];
  const isErr = entry.level === "error";
  const isWarn = entry.level === "warn";
  // Every row is widened to the same `rowWidth` (longest visible
  // message) — that's what turns the List's outer container into a
  // single horizontal scroller for the whole pane.
  return (
    <div
      style={{
        ...style,
        width: rowWidth,
        padding: "1px 12px 1px 8px",
        background: isErr ? "var(--tint-error-soft)" : undefined,
        borderLeft: isErr
          ? "2px solid var(--ui-error)"
          : "2px solid transparent",
        display: "flex",
        gap: 8,
        alignItems: "baseline",
        fontFamily: "var(--font-mono)",
        fontSize: "var(--fs-xx-small)",
        color: isErr
          ? "var(--ui-error)"
          : isWarn
            ? "var(--ui-warning)"
            : "var(--app-text)",
        whiteSpace: "nowrap",
      }}
    >
      <span className="dim" style={{ flexShrink: 0 }}>
        {formatTime(entry.ts_ms)}
      </span>
      <span
        style={{
          color: levelColor(entry.level),
          width: 48,
          flexShrink: 0,
          textTransform: "uppercase",
        }}
      >
        {entry.level ?? "info"}
      </span>
      <span style={{ flexShrink: 0 }}>{entry.message}</span>
    </div>
  );
}

function levelColor(l: LogEntry["level"]): string {
  switch (l) {
    case "error":
      return "var(--ui-error)";
    case "warn":
      return "var(--ui-warning)";
    case "debug":
      return "var(--app-text-dim)";
    default:
      return "var(--core-vibrant-blue)";
  }
}

function formatTime(ms: number): string {
  const d = new Date(ms);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
