import { useCallback, useEffect, useState } from "react";
import { api, type DockerStatus, type ProcInfo } from "./tauri";

export type ServeStatus = { up: boolean; upSinceMs: number | null };
export type DockerHealth = {
  up: boolean;
  upSinceMs: number | null;
  containers: DockerStatus["containers"];
};

/// Polls `fleet serve --dev` (TCP 8080) and `docker compose ps`, plus
/// fires an ad-hoc probe whenever the underlying spawn proc transitions
/// (start, stop, exit) so the UI doesn't lag behind reality. Lives at
/// app-level so the tray menu can read the same state without
/// duplicating probes.
///
/// The serve probe is a raw TCP connect — fleet listens TLS on 8080 in
/// dev, so each probe shows up in fleet's logs as `TLS handshake error:
/// EOF`. The proc-state-driven reprobe below catches transitions
/// instantly, so the periodic poll is just a fallback for cases the
/// proc events don't cover (docker restarted outside the app, fleet
/// crashed without our spawn dying). 10s keeps the spam at ~6/min while
/// still recovering from out-of-band changes within reasonable time.
const SERVE_POLL_MS = 10_000;
const DOCKER_POLL_MS = 3_000;
/// `enabled` gates the periodic probes off entirely — passed false while
/// the first-run gate is up so we don't probe serve / docker before the
/// user has even picked a repo or anything can be running.
export function useSystemHealth(
  repoPath: string | null,
  procs: ProcInfo[],
  enabled = true,
) {
  const [serve, setServe] = useState<ServeStatus>({ up: false, upSinceMs: null });
  const [docker, setDocker] = useState<DockerHealth>({
    up: false,
    upSinceMs: null,
    containers: [],
  });

  const probeServe = useCallback(async () => {
    try {
      const up = await api.serveTcpCheck(8080);
      setServe((prev) =>
        up
          ? { up: true, upSinceMs: prev.up ? prev.upSinceMs : Date.now() }
          : { up: false, upSinceMs: null },
      );
    } catch {
      // ignore
    }
  }, []);

  const probeDocker = useCallback(async () => {
    if (!repoPath) return;
    try {
      const s = await api.dockerComposeStatus(repoPath);
      setDocker((prev) => ({
        up: s.running,
        upSinceMs: s.running ? (prev.up ? prev.upSinceMs : Date.now()) : null,
        containers: s.containers,
      }));
    } catch {
      setDocker({ up: false, upSinceMs: null, containers: [] });
    }
  }, [repoPath]);

  useEffect(() => {
    if (!enabled) return;
    probeServe();
    const id = window.setInterval(probeServe, SERVE_POLL_MS);
    return () => window.clearInterval(id);
  }, [enabled, probeServe]);

  useEffect(() => {
    if (!enabled || !repoPath) {
      setDocker({ up: false, upSinceMs: null, containers: [] });
      return;
    }
    probeDocker();
    const id = window.setInterval(probeDocker, DOCKER_POLL_MS);
    return () => window.clearInterval(id);
  }, [enabled, repoPath, probeDocker]);

  // Ad-hoc reprobe on proc state transitions — saves up to one full
  // polling interval of "I clicked start, why is it still grey?".
  const dockerProcState = procs.find((p) => p.id === "docker-compose-up")?.state;
  useEffect(() => {
    if (dockerProcState === "done" || dockerProcState === "failed") {
      const t = window.setTimeout(probeDocker, 300);
      return () => window.clearTimeout(t);
    }
  }, [dockerProcState, probeDocker]);

  const serveProcState = procs.find((p) => p.id === "fleet-serve")?.state;
  useEffect(() => {
    if (!serveProcState) return;
    const t = window.setTimeout(probeServe, 300);
    return () => window.clearTimeout(t);
  }, [serveProcState, probeServe]);

  return { serve, docker, probeServe, probeDocker };
}
