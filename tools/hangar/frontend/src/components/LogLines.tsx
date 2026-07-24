import { useEffect, useRef } from "react";
import type { LogEntry } from "../lib/ipc";

// LogBox is the scrolling container for a compact log panel. It auto-scrolls to
// the newest line ONLY while the viewer is already at the bottom — scrolling up
// to read pins the view in place (matches the Logs tab's follow behavior).
export function LogBox({ entries, maxHeight = 240 }: { entries: LogEntry[]; maxHeight?: number }) {
  const ref = useRef<HTMLDivElement>(null);
  const follow = useRef(true);

  const onScroll = () => {
    const el = ref.current;
    if (!el) return;
    // Within 40px of the bottom counts as "following".
    follow.current = el.scrollHeight - el.clientHeight - el.scrollTop < 40;
  };

  useEffect(() => {
    if (!follow.current) return;
    const el = ref.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [entries]);

  return (
    <div
      ref={ref}
      onScroll={onScroll}
      className="mono"
      style={{
        maxHeight,
        overflowY: "auto",
        background: "var(--log-bg)",
        border: "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "var(--pad-small)",
        lineHeight: 1.5,
      }}
    >
      <LogLines entries={entries} />
    </div>
  );
}

// LogLines renders log entries the same way the Logs tab does — a HH:MM:SS
// timestamp, an uppercase level column, then the message — with error/warn
// tinting. Used by the compact SCEP / TUF log panels (non-virtualized: these
// are low-volume, wrapped rather than horizontally scrolled).
export function LogLines({ entries }: { entries: LogEntry[] }) {
  if (entries.length === 0) {
    return <span className="dim">No output yet.</span>;
  }
  return (
    <>
      {entries.map((e, i) => {
        const isErr = e.level === "error";
        const isWarn = e.level === "warn";
        return (
          <div
            key={i}
            style={{
              display: "flex",
              gap: 8,
              alignItems: "baseline",
              color: isErr ? "var(--ui-error)" : isWarn ? "var(--ui-warning)" : "var(--app-text)",
            }}
          >
            <span className="dim" style={{ flexShrink: 0 }}>{formatTime(e.ts_ms)}</span>
            <span style={{ color: levelColor(e.level), width: 44, flexShrink: 0, textTransform: "uppercase" }}>
              {e.level ?? "info"}
            </span>
            <span style={{ minWidth: 0, flex: 1, whiteSpace: "pre-wrap", wordBreak: "break-word" }}>{e.message}</span>
          </div>
        );
      })}
    </>
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
