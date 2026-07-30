// Drop-in replacement for @tauri-apps/api/event's listen(), backed by Wails'
// event bus. Wails delivers events as { data }, whereas the Tauri handlers
// we ported read { payload } — so we adapt the shape here. Tauri's listen()
// resolves to an unlisten function; Wails' On() returns one synchronously, so
// we wrap it in a resolved promise to keep the `await listen(...)` callers
// working unchanged.
import { Events } from "@wailsio/runtime";

export function listen<T = unknown>(
  event: string,
  handler: (e: { payload: T }) => void,
): Promise<() => void> {
  const off = Events.On(event as never, (e: { data: T }) =>
    handler({ payload: e.data }),
  );
  return Promise.resolve(off);
}
