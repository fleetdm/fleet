import { useEffect, useState } from "react";
import { api, type DepCheck as DepCheckT } from "../lib/tauri";

async function openDocs(url: string) {
  try {
    await api.openUrl(url);
  } catch (e) {
    console.error("openUrl failed", e);
  }
}

export function DepCheckSection({
  repoPath,
  onChange,
}: {
  repoPath: string | null;
  onChange?: (allOk: boolean) => void;
}) {
  const [checks, setChecks] = useState<DepCheckT[]>([]);
  const [loading, setLoading] = useState(true);

  async function refresh(forcePath = false) {
    setLoading(true);
    try {
      const r = await api.checkDependencies(repoPath, forcePath);
      setChecks(r.checks);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh(false);
  }, [repoPath]);

  useEffect(() => {
    if (!onChange) return;
    onChange(checks.length > 0 && checks.every(isOk));
  }, [checks, onChange]);

  const missing = checks.filter((c) => !isOk(c)).length;

  return (
    <div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 8,
        }}
      >
        <div className="section-title">Dependencies</div>
        <button
          onClick={() => refresh(true)}
          disabled={loading}
          style={{ fontSize: "var(--fs-xxx-small)" }}
        >
          <span
            className={loading ? "spin" : undefined}
            style={{ display: "inline-block" }}
          >
            ↻
          </span>{" "}
          Recheck
        </button>
      </div>

      {loading && checks.length === 0 ? (
        <div
          className="dim"
          style={{ fontSize: "var(--fs-xx-small)", padding: "8px 0" }}
        >
          Checking your toolchain…
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          {checks.map((c) => (
            <DepRow key={c.id} check={c} />
          ))}
        </div>
      )}

      {!loading && checks.length > 0 && (
        <div
          className="dim"
          style={{
            fontSize: "var(--fs-xxx-small)",
            marginTop: 8,
            textAlign: "right",
          }}
        >
          {missing === 0
            ? "All set ✓"
            : `${missing} item${missing === 1 ? "" : "s"} need attention`}
        </div>
      )}
    </div>
  );
}

function isOk(c: DepCheckT): boolean {
  if (!c.installed) return false;
  if (c.version_ok === false) return false;
  if (c.runtime_ok === false) return false;
  return true;
}

function DepRow({ check }: { check: DepCheckT }) {
  const ok = isOk(check);

  return (
    <div
      style={{
        display: "flex",
        flexDirection: "column",
        gap: 4,
        padding: "8px 12px",
        background: "var(--app-surface-2)",
        border: `1px solid ${ok ? "transparent" : "var(--app-border)"}`,
        borderRadius: "var(--radius-md)",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
        <span className={`dot ${ok ? "ok" : "fail"}`} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: "var(--fs-x-small)" }}>{check.name}</div>
          {check.note && (
            <div
              className="dim"
              style={{ fontSize: "var(--fs-xxx-small)", marginTop: 2 }}
            >
              {check.note}
            </div>
          )}
        </div>
        <div
          className="mono dim"
          style={{ fontSize: "var(--fs-xxx-small)", textAlign: "right" }}
        >
          {statusLine(check)}
        </div>
      </div>

      {!ok && (
        <InstallCommand
          command={check.install_command}
          docUrl={check.doc_url}
        />
      )}
    </div>
  );
}

function statusLine(c: DepCheckT): string {
  if (!c.installed) return "not found";
  if (c.runtime_ok === false) return "stopped";
  if (c.version_ok === false && c.version && c.required) {
    return `${c.version} (need ${c.required})`;
  }
  return c.version ?? "ok";
}

function InstallCommand({
  command,
  docUrl,
}: {
  command: string;
  docUrl: string | null;
}) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    await navigator.clipboard.writeText(command);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        gap: 6,
        marginTop: 4,
        marginLeft: 18,
      }}
    >
      <code
        style={{
          flex: 1,
          minWidth: 0,
          padding: "6px 8px",
          background: "var(--app-surface)",
          border: "1px solid var(--app-border)",
          borderRadius: "var(--radius-sm)",
          fontSize: "var(--fs-xxx-small)",
          fontFamily: "var(--font-mono)",
          overflowX: "auto",
          whiteSpace: "nowrap",
          // Reserve space for the horizontal scrollbar so the box height
          // doesn't jump when content overflows — keeps the row aligned
          // with the Copy button next to it.
          scrollbarGutter: "stable",
        }}
      >
        {command}
      </code>
      <button
        onClick={copy}
        style={{ fontSize: "var(--fs-xxx-small)", padding: "4px 10px" }}
      >
        {copied ? "✓" : "Copy"}
      </button>
      {docUrl && (
        <button
          onClick={() => openDocs(docUrl)}
          style={{
            fontSize: "var(--fs-xxx-small)",
            padding: "4px 10px",
            color: "var(--app-text-dim)",
          }}
        >
          docs ↗
        </button>
      )}
    </div>
  );
}
