const THEME_KEY = "fleet-dark-mode";
const TRANSITION_MS = 300;

export const isDarkMode = (): boolean => {
  return localStorage.getItem(THEME_KEY) === "true";
};

export const toggleDarkMode = (): boolean => {
  const dark = !isDarkMode();
  localStorage.setItem(THEME_KEY, String(dark));

  // Add a temporary class that applies a blanket transition to all elements
  // so the entire UI fades smoothly instead of individual pieces snapping.
  document.body.classList.add("theme-transition");
  document.body.classList.toggle("dark-mode", dark);

  setTimeout(() => {
    document.body.classList.remove("theme-transition");
  }, TRANSITION_MS);

  window.dispatchEvent(
    new CustomEvent("fleet-theme-change", { detail: { dark } })
  );

  return dark;
};

export const initTheme = (): void => {
  if (isDarkMode()) {
    document.body.classList.add("dark-mode");
  }
};
