const THEME_KEY = "fleet-theme";
const LEGACY_DARK_KEY = "fleet-dark-mode";
const TRANSITION_MS = 300;

export type ThemeMode = "system" | "light" | "dark";

const isThemeMode = (value: unknown): value is ThemeMode =>
  value === "system" || value === "light" || value === "dark";

const systemPrefersDark = (): boolean => {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  );
};

// Migrate the old boolean dark-mode key to the new tri-state key the first
// time we encounter it, so existing users keep their explicit choice.
const readStoredMode = (): ThemeMode => {
  const stored = localStorage.getItem(THEME_KEY);
  if (isThemeMode(stored)) {
    return stored;
  }

  const legacy = localStorage.getItem(LEGACY_DARK_KEY);
  if (legacy === "true" || legacy === "false") {
    const migrated: ThemeMode = legacy === "true" ? "dark" : "light";
    localStorage.setItem(THEME_KEY, migrated);
    localStorage.removeItem(LEGACY_DARK_KEY);
    return migrated;
  }

  return "system";
};

export const getThemeMode = (): ThemeMode => readStoredMode();

const resolveDark = (mode: ThemeMode): boolean =>
  mode === "system" ? systemPrefersDark() : mode === "dark";

export const isDarkMode = (): boolean => resolveDark(readStoredMode());

// Apply a theme change to the DOM and notify listeners. `animate` adds a
// blanket transition class so the whole UI cross-fades instead of snapping.
const applyDarkMode = (dark: boolean, animate: boolean): void => {
  if (animate) {
    document.body.classList.add("theme-transition");
    setTimeout(() => {
      document.body.classList.remove("theme-transition");
    }, TRANSITION_MS);
  }
  document.body.classList.toggle("dark-mode", dark);
  window.dispatchEvent(
    new CustomEvent("fleet-theme-change", { detail: { dark } })
  );
};

export const setThemeMode = (mode: ThemeMode): void => {
  if (mode === "system") {
    localStorage.removeItem(THEME_KEY);
  } else {
    localStorage.setItem(THEME_KEY, mode);
  }
  applyDarkMode(resolveDark(mode), true);
};

export const initTheme = (): void => {
  const mode = readStoredMode();
  if (resolveDark(mode)) {
    document.body.classList.add("dark-mode");
  }

  // Follow OS theme changes live — but only while the user is on "system".
  // Once they've picked light or dark explicitly, their choice sticks
  // regardless of what the OS does.
  if (typeof window !== "undefined" && window.matchMedia) {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", (e) => {
      if (readStoredMode() !== "system") return;
      applyDarkMode(e.matches, true);
    });
  }
};
