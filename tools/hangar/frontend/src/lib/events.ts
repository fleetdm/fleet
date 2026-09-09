// Event subscription helper over the Wails event bus. Wails delivers events as
// { data }, whereas our handlers read { payload } — so we adapt the shape here.
// Wails' On() returns an unlisten function synchronously; we wrap it in a
// resolved promise so the `await listen(...)` callers work unchanged. (The
// `{ payload }` shape and the promise are carry-overs from the original
// @tauri-apps/api/event listen() this replaced.)
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
