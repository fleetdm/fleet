export type Colors = keyof typeof COLORS;

export const COLORS = {
  // 2025 branding
  "core-fleet-black": "#192147", // Headers, thead, Field :focus outline, keyboard :focus-visible outline
  "core-fleet-green": "#009A7D",
  "core-fleet-white": "#FFFFFF",
  "ui-fleet-black-75": "#515774",
  "ui-fleet-black-50": "#8B8FA2", // Field :hover borders
  "ui-fleet-black-33": "#B3B6C1",
  "ui-fleet-black-25": "#C5C7D1",
  "ui-fleet-black-10": "#E2E4EA", // Field borders, card borders
  "ui-fleet-black-5": "#F4F4F6",

  // 2025 secondary colors
  // Sass functions only work in SCSS, not in runtime TypeScript or JavaScript files
  "ui-fleet-black-75-over": "#454C66", // darken(#515774, 5%) or color.adjust(#515774, $lightness: -5%)
  "ui-fleet-black-75-down": "#3A3E59", // darken(#515774, 10%) or color.adjust(#515774, $lightness: -10%)
  "core-fleet-green-over": "#00886C", // "darken(#009A7D, 5%)" or color.adjust(#009A7D, $lightness: -5%)
  "core-fleet-green-down": "#00775F", // "darken(#009A7D, 10%)" or color.adjust(#009A7D, $lightness: -10%)

  // core colors
  "core-fleet-blue": "#6A67FE", // TODO: lots of work to correctly match scss core-fleet-blue and not ui-vibrant-blue
  "core-fleet-red": "#FF5C83",
  "core-fleet-purple": "#AE6DDF",

  // ui colors
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

  // Notifications & status
  "status-success": "#3DB67B",
  "status-warning": "#F8CD6B",
  "status-error": "#ED6E85",

  "core-vibrant-blue-over": "#5d5ae7",
  "core-vibrant-blue-down": "#4b4ab4",
  "ui-vibrant-blue-25": "#d9d9fe",
  "ui-vibrant-blue-10": "#f1f0ff",
};
