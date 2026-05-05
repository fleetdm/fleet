// Static fallback values (used during SSR or if CSS custom property is missing)
const STATIC_COLORS = {
  // 2025 branding
  "core-fleet-black": "#192147",
  "core-fleet-green": "#009A7D",
  "core-fleet-white": "#FFFFFF",
  "ui-fleet-black-75": "#515774",
  "ui-fleet-black-50": "#8B8FA2",
  "ui-fleet-black-33": "#B3B6C1",
  "ui-fleet-black-25": "#C5C7D1",
  "ui-fleet-black-10": "#E2E4EA",
  "ui-fleet-black-5": "#F4F4F6",

  // 2025 secondary colors
  "ui-fleet-black-75-over": "#454C66",
  "ui-fleet-black-75-down": "#3A3E59",
  "core-fleet-green-over": "#00886C",
  "core-fleet-green-down": "#00775F",

  // core colors
  "core-fleet-blue": "#3E4771",
  "core-fleet-red": "#FF5C83",
  "core-fleet-purple": "#AE6DDF",

  // ui colors
  "core-vibrant-blue": "#6A67FE",
  "core-vibrant-red": "#FF5C83",
  "ui-off-white": "#F9FAFC",
  "ui-blue-hover": "#5D5AE7",
  "ui-blue-pressed": "#4B4AB4",
  "ui-blue-50": "#B4B2FE",
  "ui-blue-25": "#D9D9FE",
  "ui-blue-10": "#F1F0FF",
  "tooltip-bg": "#3E4771",
  "ui-light-grey": "#FAFAFA",
  "ui-error": "#d66c7b",
  "ui-warning": "#ebbc43",
  "ui-fleet-black-5-down": "#F0F1F4",

  // Notifications & status
  "status-success": "#3DB67B",
  "status-warning": "#F8CD6B",
  "status-error": "#ED6E85",

  "core-vibrant-blue-over": "#5d5ae7",
  "core-vibrant-blue-down": "#4b4ab4",
  "ui-vibrant-blue-25": "#d9d9fe",
  "ui-vibrant-blue-10": "#f1f0ff",

  // Static (un-themed): same value in light AND dark mode. Use for foreground
  // on always-colored surfaces (flash toasts, tooltips, etc.)
  "static-white": "#e8eaf0",
  "static-black": "#192147",
} as const;

export type Colors = keyof typeof STATIC_COLORS;

// Proxy returns CSS var() references so inline JS styles automatically adapt
// to light/dark mode without calling getComputedStyle. Falls back to the
// static value when running outside a browser (e.g. SSR or tests in jsdom).
// eslint-disable-next-line import/prefer-default-export
export const COLORS: Record<Colors, string> = new Proxy(
  STATIC_COLORS as Record<Colors, string>,
  {
    get(target, key, receiver) {
      if (typeof key === "symbol") {
        return Reflect.get(target, key, receiver);
      }
      const fallback = target[key as Colors] ?? "";
      if (typeof document !== "undefined") {
        return fallback ? `var(--${key}, ${fallback})` : `var(--${key})`;
      }
      return fallback;
    },
  }
);
