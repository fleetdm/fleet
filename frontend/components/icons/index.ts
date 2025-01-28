import Arrow from "./Arrow";
import ArrowInternalLink from "./ArrowInternalLink";
import Calendar from "./Calendar";
import CalendarCheck from "./CalendarCheck";
import Check from "./Check";
import Checkbox from "./Checkbox";
import CheckboxIndeterminate from "./CheckboxIndeterminate";
import CheckboxUnchecked from "./CheckboxUnchecked";
import ChevronLeft from "./ChevronLeft";
import ChevronRight from "./ChevronRight";
import ChevronUp from "./ChevronUp";
import ChevronDown from "./ChevronDown";
import Columns from "./Columns";
import Disable from "./Disable";
import Close from "./Close";
import CloseFilled from "./CloseFilled";
import ExternalLink from "./ExternalLink";
import Filter from "./Filter";
import FilterAlt from "./FilterAlt";
import FilterFunnel from "./FilterFunnel";
import Info from "./Info";
import More from "./More";
import Plus from "./Plus";
import PremiumFeature from "./PremiumFeature";
import Policy from "./Policy";
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
import iPadOS from "./iPadOS";
import iOS from "./iOS";

// Status Icons
import Success from "./Success";
import SuccessOutline from "./SuccessOutline";
import Pending from "./Pending";
import PendingOutline from "./PendingOutline";
import ErrorOutline from "./ErrorOutline";
import Error from "./Error";
import Warning from "./Warning";
import Clock from "./Clock";

import Copy from "./Copy";
import Eye from "./Eye";
import Pencil from "./Pencil";
import Sparkles from "./Sparkles";
import Text from "./Text";
import Transfer from "./Transfer";
import TrashCan from "./TrashCan";
import Profile from "./Profile";
import Download from "./Download";
import Upload from "./Upload";
import Refresh from "./Refresh";
import Install from "./Install";
import InstallSelfService from "./InstallSelfService";
import Settings from "./Settings";
import AutomaticSelfService from "./AutomaticSelfService";
import User from "./User";
import InfoOutline from "./InfoOutline";

// a mapping of the usable names of icons to the icon source.
export const ICON_MAP = {
  arrow: Arrow,
  "arrow-internal-link": ArrowInternalLink,
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
  "premium-feature": PremiumFeature,
  profile: Profile,
  download: Download,
  upload: Upload,
  refresh: Refresh,
  install: Install,
  "install-self-service": InstallSelfService,
  settings: Settings,
  "automatic-self-service": AutomaticSelfService,
  user: User,
};

export type IconNames = keyof typeof ICON_MAP;
