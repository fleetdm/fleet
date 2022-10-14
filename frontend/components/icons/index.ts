import CalendarCheck from "./CalendarCheck";
import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  "calendar-check": CalendarCheck,
  darwin: Apple,
  macOS: Apple,
  windows: Windows,
  Windows,
  linux: Linux,
  Linux,
};

export type IconNames = keyof typeof ICON_MAP;
