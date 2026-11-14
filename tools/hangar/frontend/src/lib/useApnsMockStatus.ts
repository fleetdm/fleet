import { useEffect, useState } from "react";
import { api, type ApnsMockStatus } from "./ipc";

/// Polls the mock APNs server on `port` (null = don't poll). Probes the port
/// rather than the process list, so it also sees a mock Hangar didn't start.
export function useApnsMockStatus(port: number | null): ApnsMockStatus | null {
  const [status, setStatus] = useState<ApnsMockStatus | null>(null);
  useEffect(() => {
    setStatus(null);
    if (port === null) return;
    let cancelled = false;
    const poll = () =>
      api
        .apnsMockStatus(port)
        .then((s) => {
          if (!cancelled) setStatus(s);
        })
        .catch(() => {});
    poll();
    const id = window.setInterval(poll, 3000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [port]);
  return status;
}
