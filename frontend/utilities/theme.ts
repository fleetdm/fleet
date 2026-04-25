const THEME_KEY = "fleet-dark-mode";
const TRANSITION_MS = 300;

const systemPrefersDark = (): boolean => {
  return (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  );
};

export const isDarkMode = (): boolean => {
  // Explicit user choice wins; otherwise inherit the system preference so
  // first-time visitors match their OS theme without us persisting anything.
  const stored = localStorage.getItem(THEME_KEY);
  if (stored !== null) {
    return stored === "true";
  }
  return systemPrefersDark();
};

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

export const toggleDarkMode = (): boolean => {
  const dark = !isDarkMode();
  localStorage.setItem(THEME_KEY, String(dark));
  applyDarkMode(dark, true);
  return dark;
};

export const initTheme = (): void => {
  if (isDarkMode()) {
    document.body.classList.add("dark-mode");
  }

  // Follow OS theme changes live — but only while the user has no explicit
  // preference stored. Once they've toggled in-app, their choice sticks
  // regardless of what the OS does.
  if (typeof window !== "undefined" && window.matchMedia) {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", (e) => {
      if (localStorage.getItem(THEME_KEY) !== null) return;
      applyDarkMode(e.matches, true);
    });
  }
};
