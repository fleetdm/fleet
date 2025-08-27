export type IconSizes = keyof typeof ICON_SIZES;

export const ICON_SIZES = {
  small: "12",
  "small-medium": "14",
  medium: "16",
  large: "24",
  "large-card": "36",
  "extra-large": "48",
};

export const ICON_SIZES_BASE14 = {
  small: "10.5",
  medium: "14",
  large: "21",
  "extra-large": "42",
};

export type SoftwareIconSizes =
  | "xsmall"
  | "small"
  | "medium"
  | "large"
  | "xlarge";

export const SOFTWARE_ICON_SIZES: Record<SoftwareIconSizes, string> = {
  xsmall: "20",
  small: "24",
  medium: "40",
  large: "64",
  xlarge: "96",
};
