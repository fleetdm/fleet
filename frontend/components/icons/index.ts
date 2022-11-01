import CalendarCheck from "./CalendarCheck";
import Chevron from "./Chevron";
import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";
import MissingHosts from "./MissingHosts";
import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import ApplePurple from "./ApplePurple";
import LinuxGreen from "./LinuxGreen";
import WindowsBlue from "./WindowsBlue";
import ExternalLink from "./ExternalLink";
import Error from "./Error";
import Success from "./Success";
import Check from "./Check";
import Plus from "./Plus";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  "calendar-check": CalendarCheck,
  chevron: Chevron,
  check: Check,
  plus: Plus,
  success: Success,
  error: Error,
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
  "external-link": ExternalLink,
};

export type IconNames = keyof typeof ICON_MAP;
