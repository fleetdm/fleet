const THEME_KEY = "fleet-theme";
const TRANSITION_MS = 300;

export type ThemeMode = "system" | "light" | "dark" | "solarized";

const THEME_CLASSES = ["dark-mode", "solarized-mode"] as const;

const isThemeMode = (value: unknown): value is ThemeMode =>
  value === "system" ||
  value === "light" ||
  value === "dark" ||
  value === "solarized";

const systemPrefersDark = (): boolean => {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  );
};

export const getThemeMode = (): ThemeMode => {
  let stored: string | null = null;
  try {
    stored = localStorage.getItem(THEME_KEY);
  } catch {
    // localStorage can throw in restricted environments; fall back to system.
  }
  return isThemeMode(stored) ? stored : "system";
};

/** Returns the body class to apply for a given mode, or null for light. */
const resolveThemeClass = (mode: ThemeMode): string | null => {
  if (mode === "dark") return "dark-mode";
  if (mode === "solarized") return "solarized-mode";
  if (mode === "system") return systemPrefersDark() ? "dark-mode" : null;
  return null; // light
};

const resolveDark = (mode: ThemeMode): boolean =>
  mode === "system" ? systemPrefersDark() : mode === "dark";

export const isDarkMode = (): boolean => resolveDark(getThemeMode());

// Apply a theme change to the DOM and notify listeners. `animate` adds a
// blanket transition class so the whole UI cross-fades instead of snapping.
const applyTheme = (mode: ThemeMode, animate: boolean): void => {
  if (animate) {
    document.body.classList.add("theme-transition");
    setTimeout(() => {
      document.body.classList.remove("theme-transition");
    }, TRANSITION_MS);
  }
  // Clear all theme classes, then set the resolved one.
  THEME_CLASSES.forEach((cls) => document.body.classList.remove(cls));
  const cls = resolveThemeClass(mode);
  if (cls) {
    document.body.classList.add(cls);
  }
  window.dispatchEvent(
    new CustomEvent("fleet-theme-change", {
      detail: { dark: resolveDark(mode) },
    })
  );
};

export const setThemeMode = (mode: ThemeMode): void => {
  try {
    if (mode === "system") {
      localStorage.removeItem(THEME_KEY);
    } else {
      localStorage.setItem(THEME_KEY, mode);
    }
  } catch {
    // localStorage can throw in restricted environments; the DOM still
    // gets the updated theme below, the choice just won't persist.
  }
  applyTheme(mode, true);
};

export const initTheme = (): void => {
  const mode = getThemeMode();
  const cls = resolveThemeClass(mode);
  if (cls) {
    document.body.classList.add(cls);
  }

  // Follow OS theme changes live -- but only while the user is on "system".
  // Once they've picked light, dark, or solarized explicitly, their choice
  // sticks regardless of what the OS does.
  if (typeof window !== "undefined" && window.matchMedia) {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", () => {
      if (getThemeMode() !== "system") return;
      applyTheme("system", true);
    });
  }
};
