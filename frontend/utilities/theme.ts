const THEME_KEY = "fleet-theme";
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

export const getThemeMode = (): ThemeMode => {
  let stored: string | null = null;
  try {
    stored = localStorage.getItem(THEME_KEY);
  } catch {
    // localStorage can throw in restricted environments; fall back to system.
  }
  return isThemeMode(stored) ? stored : "system";
};

const resolveDark = (mode: ThemeMode): boolean =>
  mode === "system" ? systemPrefersDark() : mode === "dark";

export const isDarkMode = (): boolean => resolveDark(getThemeMode());

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
  applyDarkMode(resolveDark(mode), true);
};

export const initTheme = (): void => {
  const mode = getThemeMode();
  if (resolveDark(mode)) {
    document.body.classList.add("dark-mode");
  }

  // Follow OS theme changes live — but only while the user is on "system".
  // Once they've picked light or dark explicitly, their choice sticks
  // regardless of what the OS does.
  if (typeof window !== "undefined" && window.matchMedia) {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", (e) => {
      if (getThemeMode() !== "system") return;
      applyDarkMode(e.matches, true);
    });
  }
};
