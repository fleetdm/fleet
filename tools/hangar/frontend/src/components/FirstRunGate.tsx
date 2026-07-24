import { useCallback, useEffect, useState } from "react";
import { api, type RepoProbe, type Settings } from "../lib/ipc";
import { updateServer } from "../lib/servers";
import logoUrl from "../assets/logo.png";
import { DepCheckSection } from "./DepCheck";

const FLEET_CLONE_CMD = "git clone https://github.com/fleetdm/fleet.git";

export function FirstRunGate({
  onComplete,
}: {
  onComplete: (settings: Settings) => void;
}) {
  const [probes, setProbes] = useState<RepoProbe[]>([]);
  const [scanning, setScanning] = useState(true);
  const [selected, setSelected] = useState<string | null>(null);
  const [depsOk, setDepsOk] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Re-runnable so the user can rescan after cloning a repo without
  // having to restart the app. Keeps the current selection if it's still
  // valid; otherwise falls back to the first valid clone found.
  const scanRepos = useCallback(async () => {
    setScanning(true);
    try {
      const p = await api.probeFleetRepo();
      setProbes(p);
      setSelected((cur) =>
        cur && p.some((x) => x.path === cur && x.valid)
          ? cur
          : (p.find((x) => x.valid)?.path ?? null),
      );
    } catch (e) {
      setError(String(e));
    } finally {
      setScanning(false);
    }
  }, []);

  useEffect(() => {
    scanRepos();
  }, [scanRepos]);

  async function pickFolder() {
    const result = await api.pickFolder();
    if (!result) return;
    const path = typeof result === "string" ? result : null;
    if (!path) return;
    const probed = await api.probeFleetRepo(path);
    if (probed[0]?.valid) {
      setProbes([probed[0], ...probes.filter((p) => p.path !== probed[0].path)]);
      setSelected(probed[0].path);
      setError(null);
    } else {
      setError(probed[0]?.reason ?? "not a valid fleet repo");
    }
  }

  async function finish(skip: boolean) {
    setBusy(true);
    try {
      const baseline = await api.getSettings();
      // Auto-detect a fleet.yml in the repo root so serve points at it
      // when present; absent → leave config unset (env / dev defaults).
      const detectedConfig =
        !skip && selected ? await api.detectFleetConfig(selected) : null;
      // Seed the first server's worktree + config (baseline is already
      // migrated, so servers[0] exists).
      const firstId = baseline.servers[0]?.id ?? baseline.active_server_id;
      const seeded = updateServer(baseline, firstId, (srv) => ({
        ...srv,
        worktree_path: skip ? null : selected,
        fleet_serve: { ...srv.fleet_serve, config_path: detectedConfig },
      }));
      const s: Settings = { ...seeded, first_run_complete: true };
      await api.saveSettings(s);
      onComplete(s);
    } catch (e) {
      setError(String(e));
      setBusy(false);
    }
  }

  const anyValid = probes.some((p) => p.valid);

  return (
    <div
      style={{
        height: "100%",
        display: "flex",
        alignItems: "flex-start",
        justifyContent: "center",
        padding: "var(--pad-large)",
        overflowY: "auto",
        // Contain scroll to this panel and prevent the rubber-band
        // overscroll that lets the whole card be dragged off-screen.
        overscrollBehavior: "none",
      }}
    >
      <div
        style={{
          maxWidth: 620,
          width: "100%",
          background: "var(--app-surface)",
          border: "1px solid var(--app-border)",
          borderRadius: "var(--radius-xl)",
          padding: "var(--pad-xlarge)",
          display: "flex",
          flexDirection: "column",
          gap: "var(--pad-large)",
        }}
      >
        <div style={{ display: "flex", justifyContent: "center" }}>
          <img
            src={logoUrl}
            alt=""
            style={{
              width: 140,
              height: 140,
              objectFit: "contain",
              userSelect: "none",
            }}
            draggable={false}
          />
        </div>
        <div style={{ textAlign: "center" }}>
          <div
            style={{
              fontSize: "var(--fs-large)",
              fontWeight: 600,
              marginBottom: 4,
            }}
          >
            Welcome to Fleet Hangar
          </div>
          <div className="dim" style={{ fontSize: "var(--fs-xx-small)" }}>
            Quick setup · all paths are editable later in Settings
          </div>
        </div>

        <DepCheckSection repoPath={selected} onChange={setDepsOk} />

        <div>
          <div
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              marginBottom: 8,
            }}
          >
            <div className="section-title" style={{ margin: 0 }}>
              Fleet repository
            </div>
            <button
              onClick={scanRepos}
              disabled={scanning}
              title="Re-scan common dev folders and ~/fleet for clones"
              style={{ fontSize: "var(--fs-xxx-small)" }}
            >
              <span
                className={scanning ? "spin" : undefined}
                style={{ display: "inline-block" }}
              >
                ↻
              </span>{" "}
              Rescan
            </button>
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 8 }}>
            {scanning && (
              <div
                className="dim"
                style={{
                  fontSize: "var(--fs-xx-small)",
                  padding: "var(--pad-medium) 0",
                }}
              >
                Scanning common dev folders for Fleet clones…
              </div>
            )}
            {!scanning &&
              probes.map((p) => (
                <ProbeRow
                  key={p.path}
                  probe={p}
                  selected={selected === p.path}
                  onSelect={() => p.valid && setSelected(p.path)}
                />
              ))}
            {!scanning && !anyValid && <NoRepoFound />}
            <button onClick={pickFolder} style={{ alignSelf: "flex-start" }}>
              📁 Pick folder…
            </button>
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
          </div>
        </div>

        <div
          style={{
            display: "flex",
            justifyContent: "space-between",
            alignItems: "center",
            borderTop: "1px solid var(--app-border)",
            paddingTop: "var(--pad-medium)",
            marginTop: "auto",
            gap: 12,
          }}
        >
          <button onClick={() => finish(true)} disabled={busy}>
            Skip · start blank
          </button>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 10,
            }}
          >
            {selected && !depsOk && (
              <span
                className="dim"
                style={{ fontSize: "var(--fs-xxx-small)" }}
              >
                some dependencies still missing
              </span>
            )}
            <button
              className="primary"
              onClick={() => finish(false)}
              disabled={!selected || busy}
            >
              → Get started
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

function NoRepoFound() {
  const [copied, setCopied] = useState(false);
  async function copy() {
    await navigator.clipboard.writeText(FLEET_CLONE_CMD);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }
  return (
    <div
      style={{
        background: "var(--app-surface-2)",
        border: "1px dashed var(--app-border)",
        borderRadius: "var(--radius-md)",
        padding: "var(--pad-medium)",
        display: "flex",
        flexDirection: "column",
        gap: 8,
      }}
    >
      <div
        style={{
          fontSize: "var(--fs-xx-small)",
          color: "var(--app-text-dim)",
        }}
      >
        No Fleet clones found in common dev folders. Clone it with:
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
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
            scrollbarGutter: "stable",
          }}
        >
          {FLEET_CLONE_CMD}
        </code>
        <button
          onClick={copy}
          style={{ fontSize: "var(--fs-xxx-small)", padding: "4px 10px" }}
        >
          {copied ? "✓" : "Copy"}
        </button>
      </div>
      <div
        className="dim"
        style={{ fontSize: "var(--fs-xxx-small)" }}
      >
        Then click Rescan above, or pick the folder manually below.
      </div>
    </div>
  );
}

function ProbeRow({
  probe,
  selected,
  onSelect,
}: {
  probe: RepoProbe;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      onClick={onSelect}
      disabled={!probe.valid}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "10px 12px",
        background: selected ? "var(--tint-success-soft)" : "var(--app-surface-2)",
        border: selected
          ? "1px solid var(--core-fleet-green)"
          : "1px solid var(--app-border)",
        borderRadius: "var(--radius-md)",
        textAlign: "left",
        opacity: probe.valid ? 1 : 0.55,
        cursor: probe.valid ? "pointer" : "default",
      }}
    >
      <span className={`dot ${probe.valid ? "ok" : "fail"}`} />
      <div style={{ flex: 1, minWidth: 0 }}>
        <div
          className="mono"
          style={{
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {probe.path}
        </div>
        {probe.reason && (
          <div
            className="dim"
            style={{ fontSize: "var(--fs-xxx-small)", marginTop: 2 }}
          >
            {probe.reason}
          </div>
        )}
      </div>
      <span
        style={{
          fontSize: "var(--fs-xxx-small)",
          textTransform: "uppercase",
          letterSpacing: "0.06em",
          color: probe.valid
            ? "var(--core-fleet-green)"
            : "var(--ui-error)",
        }}
      >
        {probe.valid ? "found" : "not found"}
      </span>
    </button>
  );
}
