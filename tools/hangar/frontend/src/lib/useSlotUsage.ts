import { useEffect, useState } from "react";
import { api, type SlotUsage } from "./ipc";
import { listen } from "./events";

const POLL_MS = 10_000;

/// Slot reservations and agent jobs, keyed by slot id. The helper writes
/// reservations outside Hangar, so this also polls.
export function useSlotUsage(): Record<string, SlotUsage> {
  const [usage, setUsage] = useState<Record<string, SlotUsage>>({});
  useEffect(() => {
    let alive = true;
    const refresh = () =>
      api
        .slotUsage()
        .then((u) => alive && setUsage(u ?? {}))
        .catch(() => {});
    refresh();
    const timer = setInterval(refresh, POLL_MS);
    const offs = ["settings:changed", "agent:slots"].map((ev) =>
      listen(ev, refresh),
    );
    return () => {
      alive = false;
      clearInterval(timer);
      offs.forEach((o) => o.then((off) => off()));
    };
  }, []);
  return usage;
}
