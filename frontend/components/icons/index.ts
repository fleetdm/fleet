import CalendarCheck from "./CalendarCheck";
import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";
import MissingHosts from "./MissingHosts";
import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import ApplePurple from "./ApplePurple";
import LinuxGreen from "./LinuxGreen";
import WindowsBlue from "./WindowsBlue";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  "calendar-check": CalendarCheck,
  darwin: Apple,
  macOS: Apple,
  windows: Windows,
  Windows,
  linux: Linux,
  Linux,
  "darwin-purple": ApplePurple,
  "windows-blue": WindowsBlue,
  "linux-green": LinuxGreen,
  "missing-hosts": MissingHosts,
  "low-disk-space-hosts": LowDiskSpaceHosts,
};

export type IconNames = keyof typeof ICON_MAP;
