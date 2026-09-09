import { useCallback, useEffect, useMemo, useState } from "react";
import {
  api,
  type DockerStatus,
  type ProcInfo,
  type ServerProfile,
} from "./ipc";

export type ServeStatus = { up: boolean; upSinceMs: number | null };
export type DockerHealth = {
  up: boolean;
  upSinceMs: number | null;
  containers: DockerStatus["containers"];
};
export type ServerHealth = { serve: ServeStatus; docker: DockerHealth };

const SERVE_DOWN: ServeStatus = { up: false, upSinceMs: null };
const DOCKER_DOWN: DockerHealth = { up: false, upSinceMs: null, containers: [] };

/// Polls each server's `fleet serve` (TCP on its server port) and its docker
/// compose project, returning a per-server health map. Lives at app level so
/// the switcher, status strip, and tray all read the same state.
///
/// Serve is probed on a slower cadence than docker because each serve probe is
/// a raw TLS connect that shows up in fleet's log as a handshake error — with
/// multiple servers that spam adds up, so we keep it to ~6/min/server and lean
/// on the proc-transition reprobe for snappy updates. `enabled` gates the
/// periodic probes off (e.g. during the first-run gate).
const SERVE_POLL_MS = 10_000;
const DOCKER_POLL_MS = 3_000;

export function useMultiServerHealth(
  servers: ServerProfile[],
  procs: ProcInfo[],
  enabled = true,
): Record<string, ServerHealth> {
  const [serveMap, setServeMap] = useState<Record<string, ServeStatus>>({});
  const [dockerMap, setDockerMap] = useState<Record<string, DockerHealth>>({});

  // Signature of just the fields the probes depend on, so unrelated settings
  // edits (theme, ngrok, …) don't churn the probe callbacks.
  const sig = useMemo(
    () =>
      servers
        .map(
          (s) =>
            `${s.id}:${s.ports.server}:${s.worktree_path ?? ""}:${s.compose_project}`,
        )
        .join("|"),
    [servers],
  );

  const probeServe = useCallback(async () => {
    const entries = await Promise.all(
      servers.map(async (s) => {
        let up = false;
        try {
          up = await api.serveTcpCheck(s.ports.server);
        } catch {
          // ignore
        }
        return [s.id, up] as const;
      }),
    );
    setServeMap((prev) => {
      const next: Record<string, ServeStatus> = {};
      for (const [id, up] of entries) {
        next[id] = up
          ? { up: true, upSinceMs: prev[id]?.up ? prev[id].upSinceMs : Date.now() }
          : SERVE_DOWN;
      }
      return next;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sig]);

  const probeDocker = useCallback(async () => {
    const entries = await Promise.all(
      servers.map(async (s) => {
        if (!s.worktree_path) {
          return [s.id, false, [] as DockerStatus["containers"]] as const;
        }
        try {
          const d = await api.dockerComposeStatus(
            s.worktree_path,
            s.compose_project,
          );
          return [s.id, d.running, d.containers] as const;
        } catch {
          return [s.id, false, [] as DockerStatus["containers"]] as const;
        }
      }),
    );
    setDockerMap((prev) => {
      const next: Record<string, DockerHealth> = {};
      for (const [id, running, containers] of entries) {
        next[id] = running
          ? {
              up: true,
              upSinceMs: prev[id]?.up ? prev[id].upSinceMs : Date.now(),
              containers,
            }
          : DOCKER_DOWN;
      }
      return next;
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sig]);

  useEffect(() => {
    if (!enabled) {
      setServeMap({});
      return;
    }
    probeServe();
    const id = window.setInterval(probeServe, SERVE_POLL_MS);
    return () => window.clearInterval(id);
  }, [enabled, probeServe]);

  useEffect(() => {
    if (!enabled) {
      setDockerMap({});
      return;
    }
    probeDocker();
    const id = window.setInterval(probeDocker, DOCKER_POLL_MS);
    return () => window.clearInterval(id);
  }, [enabled, probeDocker]);

  // Ad-hoc reprobe when any managed process transitions — saves up to a full
  // polling interval of "I clicked start, why is it still grey?".
  const procSig = useMemo(
    () => procs.map((p) => `${p.id}:${p.state}`).join("|"),
    [procs],
  );
  useEffect(() => {
    if (!enabled) return;
    const t = window.setTimeout(() => {
      probeServe();
      probeDocker();
    }, 300);
    return () => window.clearTimeout(t);
  }, [procSig, enabled, probeServe, probeDocker]);

  return useMemo(() => {
    const out: Record<string, ServerHealth> = {};
    for (const s of servers) {
      out[s.id] = {
        serve: serveMap[s.id] ?? SERVE_DOWN,
        docker: dockerMap[s.id] ?? DOCKER_DOWN,
      };
    }
    return out;
  }, [servers, serveMap, dockerMap]);
}
