import ABMIssueHosts from "./ABMIssueHosts";
import Android from "./Android";
import Apple from "./Apple";
import Arrow from "./Arrow";
import ArrowInternalLink from "./ArrowInternalLink";
import ArrowLeft from "./ArrowLeft";
import AutomaticSelfService from "./AutomaticSelfService";
import Calendar from "./Calendar";
import CalendarCheck from "./CalendarCheck";
import Centos from "./Centos";
import Check from "./Check";
import Checkbox from "./Checkbox";
import CheckboxIndeterminate from "./CheckboxIndeterminate";
import CheckboxUnchecked from "./CheckboxUnchecked";
import ChevronDown from "./ChevronDown";
import ChevronLeft from "./ChevronLeft";
import ChevronRight from "./ChevronRight";
import ChevronUp from "./ChevronUp";
import Chrome from "./Chrome";
import Clock from "./Clock";
import Close from "./Close";
import CloseFilled from "./CloseFilled";
import Columns from "./Columns";
import Copy from "./Copy";
import Disable from "./Disable";
import Download from "./Download";
import Error from "./Error";
import ErrorOutline from "./ErrorOutline";
import ExternalLink from "./ExternalLink";
import Eye from "./Eye";
import Filter from "./Filter";
import FilterAlt from "./FilterAlt";
import FilterFunnel from "./FilterFunnel";
import GitOpsMode from "./GitOpsMode";
import Info from "./Info";
import InfoOutline from "./InfoOutline";
import Install from "./Install";
import InstallSelfService from "./InstallSelfService";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import Lightbulb from "./Lightbulb";
import Linux from "./Linux";
import LowDiskSpaceHosts from "./LowDiskSpaceHosts";
import M1 from "./M1";
import MissingHosts from "./MissingHosts";
import More from "./More";
import Pencil from "./Pencil";
import Pending from "./Pending";
import PendingOutline from "./PendingOutline";
import Pin from "./Pin";
import Plus from "./Plus";
import Policy from "./Policy";
import PremiumFeature from "./PremiumFeature";
import Profile from "./Profile";
import Query from "./Query";
import Refresh from "./Refresh";
import Run from "./Run";
import Search from "./Search";
import Settings from "./Settings";
import Sparkles from "./Sparkles";
// Status Icons
import Success from "./Success";
import SuccessOutline from "./SuccessOutline";
import Tag from "./Tag";
import Text from "./Text";
import TotalHosts from "./TotalHosts";
import Transfer from "./Transfer";
import TrashCan from "./TrashCan";
import Ubuntu from "./Ubuntu";
import Upload from "./Upload";
import User from "./User";
import Warning from "./Warning";
import Windows from "./Windows";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  arrow: Arrow,
  "arrow-internal-link": ArrowInternalLink,
  "arrow-left": ArrowLeft,
  calendar: Calendar,
  "calendar-check": CalendarCheck,
  "chevron-left": ChevronLeft,
  "chevron-right": ChevronRight,
  "chevron-up": ChevronUp,
  "chevron-down": ChevronDown,
  check: Check,
  checkbox: Checkbox,
  "checkbox-indeterminate": CheckboxIndeterminate,
  "checkbox-unchecked": CheckboxUnchecked,
  columns: Columns,
  disable: Disable,
  close: Close,
  "close-filled": CloseFilled,
  "external-link": ExternalLink,
  filter: Filter,
  "filter-alt": FilterAlt,
  "filter-funnel": FilterFunnel,
  "low-disk-space-hosts": LowDiskSpaceHosts,
  "missing-hosts": MissingHosts,
  "total-hosts": TotalHosts,
  "abm-issue-hosts": ABMIssueHosts,
  lightbulb: Lightbulb,
  info: Info,
  "info-outline": InfoOutline,
  more: More,
  plus: Plus,
  policy: Policy,
  query: Query,
  copy: Copy,
  eye: Eye,
  pencil: Pencil,
  search: Search,
  sparkles: Sparkles,
  text: Text,
  transfer: Transfer,
  trash: TrashCan,
  success: Success,
  "success-outline": SuccessOutline,
  pending: Pending,
  "pending-outline": PendingOutline,
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
  run: Run,
  install: Install,
  "install-self-service": InstallSelfService,
  settings: Settings,
  "automatic-self-service": AutomaticSelfService,
  user: User,
  "gitops-mode": GitOpsMode,
  pin: Pin,
  tag: Tag,
};

export type IconNames = keyof typeof ICON_MAP;
