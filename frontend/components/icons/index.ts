import Alert from "./Alert";
import CalendarCheck from "./CalendarCheck";
import Check from "./Check";
import Chevron from "./Chevron";
import Ex from "./Ex";
import EmptyHosts from "./EmptyHosts";
import EmptyIntegrations from "./EmptyIntegrations";
import EmptyMembers from "./EmptyMembers";
import EmptyPacks from "./EmptyPacks";
import EmptyPolicies from "./EmptyPolicies";
import EmptyQueries from "./EmptyQueries";
import EmptySchedule from "./EmptySchedule";
import EmptySoftware from "./EmptySoftware";
import EmptyTeams from "./EmptyTeams";
import ExternalLink from "./ExternalLink";
import Issue from "./Issue";
import Plus from "./Plus";

import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import MissingHosts from "./MissingHosts";

import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";
import M1 from "./M1";
import Centos from "./Centos";
import Ubuntu from "./Ubuntu";

// Encircled
import ApplePurple from "./ApplePurple";
import LinuxGreen from "./LinuxGreen";
import WindowsBlue from "./WindowsBlue";

import Error from "./Error";
import Success from "./Success";

import Clipboard from "./Clipboard";
import Eye from "./Eye";
import Pencil from "./Pencil";
import TrashCan from "./TrashCan";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  alert: Alert,
  "calendar-check": CalendarCheck,
  chevron: Chevron,
  check: Check,
  ex: Ex,
  "empty-hosts": EmptyHosts,
  "empty-integrations": EmptyIntegrations,
  "empty-members": EmptyMembers,
  "empty-packs": EmptyPacks,
  "empty-policies": EmptyPolicies,
  "empty-queries": EmptyQueries,
  "empty-schedule": EmptySchedule,
  "empty-software": EmptySoftware,
  "empty-teams": EmptyTeams,
  "external-link": ExternalLink,
  "low-disk-space-hosts": LowDiskSpaceHosts,
  "missing-hosts": MissingHosts,
  issue: Issue,
  plus: Plus,
  clipboard: Clipboard,
  eye: Eye,
  pencil: Pencil,
  trash: TrashCan,
  success: Success,
  error: Error,
  darwin: Apple,
  macOS: Apple,
  windows: Windows,
  Windows,
  linux: Linux,
  Linux,
  m1: M1,
  centos: Centos,
  ubuntu: Ubuntu,
  "darwin-purple": ApplePurple,
  "windows-blue": WindowsBlue,
  "linux-green": LinuxGreen,
};

export type IconNames = keyof typeof ICON_MAP;
