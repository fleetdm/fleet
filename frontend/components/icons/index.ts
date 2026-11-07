import ABMIssueHosts from "./ABMIssueHosts";
import Android from "./Android";
import Apple from "./Apple";
import Arrow from "./Arrow";
import ArrowLeft from "./ArrowLeft";
import Calendar from "./Calendar";
import CalendarCheck from "./CalendarCheck";
import Check from "./Check";
import Checkbox from "./Checkbox";
import CheckboxIndeterminate from "./CheckboxIndeterminate";
import CheckboxUnchecked from "./CheckboxUnchecked";
import ChevronDown from "./ChevronDown";
import ChevronLeft from "./ChevronLeft";
import ChevronRight from "./ChevronRight";
import Chrome from "./Chrome";
import Clock from "./Clock";
import Close from "./Close";
import Columns from "./Columns";
import Copy from "./Copy";
import Disable from "./Disable";
import Download from "./Download";
import Error from "./Error";
import ExternalLink from "./ExternalLink";
import ExternalLinkSmall from "./ExternalLinkSmall";
import Eye from "./Eye";
import EyeSlash from "./EyeSlash";
import Filter from "./Filter";
import GitOpsMode from "./GitOpsMode";
import Info from "./Info";
import Install from "./Install";
import InstallAuto from "./InstallAuto";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import Linux from "./Linux";
import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import MissingHosts from "./MissingHosts";
import More from "./More";
import Pencil from "./Pencil";
import Pending from "./Pending";
import Pin from "./Pin";
import Plus from "./Plus";
import Policy from "./Policy";
import PremiumFeature from "./PremiumFeature";
import Profile from "./Profile";
import Query from "./Query";
import Refresh from "./Refresh";
import Retry from "./Retry";
import Run from "./Run";
import Search from "./Search";
import SelfService from "./SelfService";
import Settings from "./Settings";
import Sort from "./Sort";
import Sparkles from "./Sparkles";
// Status Icons
import Success from "./Success";
import SuccessFilled from "./SuccessFilled";
import Tag from "./Tag";
import Text from "./Text";
import TotalHosts from "./TotalHosts";
import Transfer from "./Transfer";
import TrashCan from "./TrashCan";
import Update from "./Update";
import Upload from "./Upload";
import User from "./User";
import Warning from "./Warning";
import Windows from "./Windows";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  arrow: Arrow,
  "arrow-left": ArrowLeft,
  calendar: Calendar,
  "calendar-check": CalendarCheck,
  "chevron-left": ChevronLeft,
  "chevron-right": ChevronRight,
  "chevron-down": ChevronDown,
  check: Check,
  checkbox: Checkbox,
  "checkbox-indeterminate": CheckboxIndeterminate,
  "checkbox-unchecked": CheckboxUnchecked,
  columns: Columns,
  disable: Disable,
  close: Close,
  "external-link": ExternalLink,
  "external-link-small": ExternalLinkSmall,
  filter: Filter,
  sort: Sort,
  "low-disk-space-hosts": LowDiskSpaceHosts,
  "missing-hosts": MissingHosts,
  "total-hosts": TotalHosts,
  "abm-issue-hosts": ABMIssueHosts,
  info: Info,
  more: More,
  plus: Plus,
  policy: Policy,
  query: Query,
  copy: Copy,
  eye: Eye,
  "eye-slash": EyeSlash,
  pencil: Pencil,
  search: Search,
  sparkles: Sparkles,
  text: Text,
  transfer: Transfer,
  trash: TrashCan,
  success: Success,
  "success-filled": SuccessFilled,
  pending: Pending,
  error: Error,
  warning: Warning,
  clock: Clock,
  darwin: Apple,
  macOS: Apple,
  windows: Windows,
  Windows,
  linux: Linux,
  Linux,
  chrome: Chrome,
  ChromeOS: Chrome,
  ipados: iPadOS,
  iPadOS,
  ios: iOS,
  iOS,
  android: Android,
  "premium-feature": PremiumFeature,
  profile: Profile,
  download: Download,
  upload: Upload,
  refresh: Refresh,
  retry: Retry,
  update: Update,
  run: Run,
  install: Install,
  "install-auto": InstallAuto,
  settings: Settings,
  "self-service": SelfService,
  user: User,
  "gitops-mode": GitOpsMode,
  pin: Pin,
  tag: Tag,
};

export type IconNames = keyof typeof ICON_MAP;
