import { useSyncExternalStore } from "react";

const subscribe = () => () => {};
const getSnapshot = () => true;
const getServerSnapshot = () => false;

// Returns false during SSR and the first client render, true after hydration.
// Use to gate content that depends on localStorage so SSR and first-client
// render match and React doesn't throw a hydration mismatch.
export function useIsHydrated(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
