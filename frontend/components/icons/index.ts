import Alert from "./Alert";
import Arrow from "./Arrow";
import ArrowInternalLink from "./ArrowInternalLink";
import CalendarCheck from "./CalendarCheck";
import Check from "./Check";
import Chevron from "./Chevron";
import Columns from "./Columns";
import CriticalPolicy from "./CriticalPolicy";
import Disable from "./Disable";
import DownCaret from "./DownCaret";
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
import More from "./More";
import Plus from "./Plus";
import PremiumFeature from "./PremiumFeature";
import Query from "./Query";
import Search from "./Search";

import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import MissingHosts from "./MissingHosts";
import Lightbulb from "./Lightbulb";

import Apple from "./Apple";
import Windows from "./Windows";
import Linux from "./Linux";
import M1 from "./M1";
import Centos from "./Centos";
import Ubuntu from "./Ubuntu";
import Chrome from "./Chrome";

// Encircled
import ApplePurple from "./ApplePurple";
import LinuxGreen from "./LinuxGreen";
import WindowsBlue from "./WindowsBlue";
import RedEncircledChrome from "./RedEncircledChrome";

// Status Icons
import Success from "./Success";
import SuccessPartial from "./SuccessPartial";
import Pending from "./Pending";
import PendingPartial from "./PendingPartial";
import ErrorOutline from "./ErrorOutline";
import Error from "./Error";
import Warning from "./Warning";
import Clock from "./Clock";

import Copy from "./Copy";
import Eye from "./Eye";
import Pencil from "./Pencil";
import Transfer from "./Transfer";
import TrashCan from "./TrashCan";
import Profile from "./Profile";
import Download from "./Download";
import Files from "./Files";
import Refresh from "./Refresh";
import FilePython from "./FilePython";
import FileZsh from "./FileZsh";
import FileBash from "./FileBash";
import FileGeneric from "./FileGeneric";
import FilePkg from "./FilePkg";
import FilePdf from "./FilePdf";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  alert: Alert,
  arrow: Arrow,
  "arrow-internal-link": ArrowInternalLink,
  "calendar-check": CalendarCheck,
  chevron: Chevron,
  check: Check,
  columns: Columns,
  "critical-policy": CriticalPolicy,
  disable: Disable,
  "down-caret": DownCaret,
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
  lightbulb: Lightbulb,
  issue: Issue,
  more: More,
  plus: Plus,
  query: Query,
  copy: Copy,
  eye: Eye,
  pencil: Pencil,
  search: Search,
  transfer: Transfer,
  trash: TrashCan,
  success: Success,
  "success-partial": SuccessPartial,
  pending: Pending,
  "pending-partial": PendingPartial,
  error: Error,
  "error-outline": ErrorOutline,
  warning: Warning,
  clock: Clock,
  darwin: Apple,
  macOS: Apple,
  windows: Windows,
  Windows,
  linux: Linux,
  Linux,
  m1: M1,
  centos: Centos,
  ubuntu: Ubuntu,
  chrome: Chrome,
  ChromeOS: Chrome,
  "premium-feature": PremiumFeature,
  "darwin-purple": ApplePurple,
  "windows-blue": WindowsBlue,
  "linux-green": LinuxGreen,
  "chrome-red": RedEncircledChrome,
  profile: Profile,
  download: Download,
  files: Files,
  "file-python": FilePython,
  "file-zsh": FileZsh,
  "file-bash": FileBash,
  "file-pkg": FilePkg,
  "file-generic": FileGeneric,
  "file-pdf": FilePdf,
  refresh: Refresh,
};

export type IconNames = keyof typeof ICON_MAP;
