import { useEffect } from "react";

/** Browser navigation guard — shows the leave-confirm prompt while `block`
 * is true. Tied to the `beforeunload` event, which fires on tab close, hard
 * navigation, and reload. Used during multi-step uploads where losing the
 * in-flight request would discard the user's work.
 *
 * Note: this does NOT block in-app react-router navigation; only browser-level
 * navigation. Soft navigation between Fleet routes still proceeds. */
const useBlockNavigation = (block: boolean): void => {
  useEffect(() => {
    if (!block) return undefined;

    const handler = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      // Legacy support for Chrome/Edge < 119, which only respect
      // `returnValue` rather than the modern `preventDefault()`.
      e.returnValue = true;
    };

    addEventListener("beforeunload", handler);
    return () => removeEventListener("beforeunload", handler);
  }, [block]);
};

export default useBlockNavigation;
