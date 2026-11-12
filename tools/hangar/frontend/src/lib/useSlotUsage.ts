import { useSyncExternalStore } from "react";
import { api, type SlotUsage } from "./ipc";
import { listen } from "./events";

const POLL_MS = 10_000;

type Usage = Record<string, SlotUsage>;

// One shared store: the first subscriber starts polling and the event
// listeners, the last one stops them.
let usage: Usage = {};
const subscribers = new Set<() => void>();
let stop: (() => void) | null = null;

function start(): () => void {
  let alive = true;
  const refresh = () =>
    api
      .slotUsage()
      .then((u) => {
        if (!alive) return;
        usage = u ?? {};
        subscribers.forEach((fn) => fn());
      })
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
}

function subscribe(fn: () => void): () => void {
  subscribers.add(fn);
  stop ??= start();
  return () => {
    subscribers.delete(fn);
    if (subscribers.size === 0) {
      stop?.();
      stop = null;
    }
  };
}

/// Slot reservations and agent jobs, keyed by slot id. The helper writes
/// reservations outside Hangar, so this also polls.
export function useSlotUsage(): Usage {
  return useSyncExternalStore(subscribe, () => usage);
}
