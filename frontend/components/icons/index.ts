import CalendarCheck from "./CalendarCheck";
import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";

export const ICON_MAP = {
  "calendar-check": CalendarCheck,
  darwin: Apple,
  windows: Windows,
  linux: Linux,
};

export type IconNames = keyof typeof ICON_MAP;
