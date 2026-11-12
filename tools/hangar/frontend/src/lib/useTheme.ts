import { useEffect } from "react";
import type { ThemePreference } from "./ipc";

/// Applies the user's theme preference to <body>. Toggles the
/// `.dark-mode` class — same hook frontend/styles/var/colors.scss uses,
/// so the per-mode CSS-variable values in tokens.css just work.
///
/// "system" subscribes to the prefers-color-scheme media query so the
/// app switches live when the OS appearance flips. The other two modes
/// pin the class and ignore OS changes.
export function useApplyTheme(pref: ThemePreference | undefined): void {
  useEffect(() => {
    if (!pref) return;

    const mql = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
      const resolved =
        pref === "system" ? (mql.matches ? "dark" : "light") : pref;
      document.body.classList.toggle("dark-mode", resolved === "dark");
    };

    apply();
    if (pref !== "system") return;

    mql.addEventListener("change", apply);
    return () => mql.removeEventListener("change", apply);
  }, [pref]);
}
