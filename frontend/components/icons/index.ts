import Alert from "./Alert";
import CalendarCheck from "./CalendarCheck";
import Check from "./Check";
import Chevron from "./Chevron";
import ExternalLink from "./ExternalLink";
import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import MissingHosts from "./MissingHosts";
import Plus from "./Plus";
import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";
import ApplePurple from "./ApplePurple";
import LinuxGreen from "./LinuxGreen";
import WindowsBlue from "./WindowsBlue";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  alert: Alert,
  "calendar-check": CalendarCheck,
  chevron: Chevron,
  check: Check,
  "external-link": ExternalLink,
  "low-disk-space-hosts": LowDiskSpaceHosts,
  "missing-hosts": MissingHosts,
  plus: Plus,
  darwin: Apple,
  macOS: Apple,
  windows: Windows,
  Windows,
  linux: Linux,
  Linux,
  "darwin-purple": ApplePurple,
  "windows-blue": WindowsBlue,
  "linux-green": LinuxGreen,
};

export type IconNames = keyof typeof ICON_MAP;
