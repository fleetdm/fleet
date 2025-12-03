// Note: if parts of a icon have a clip path, mask, or gradient, the IDs must be unique
// across all icons to avoid conflicts in the DOM. See uniqueId usage within icon components.

import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";

import AcrobatReader from "./AcrobatReader";
import CreativeCloud from "./AdobeCreativeCloud";
import ChromeApp from "./ChromeApp";
import Excel from "./Excel";
import Extension from "./Extension";
import Firefox from "./Firefox";
import AppleApp from "./AppleApp";
import MacOS from "./MacOS";
import Package from "./Package";
import Safari from "./Safari";
import Slack from "./Slack";
import Teams from "./Teams";
import VisualStudioCode from "./VisualStudioCode";
import WindowsApp from "./WindowsApp";
import WindowsOS from "./WindowsOS";
import Word from "./Word";
import Zoom from "./Zoom";
import ChromeOS from "./ChromeOS";
import LinuxOS from "./LinuxOS";
import Falcon from "./Falcon";
import AppleAppStore from "./AppleAppStore";
import AndroidPlayStore from "./AndroidPlayStore";
import AndroidApp from "./AndroidApp";
import TeamViewer from "./TeamViewer";
import Box from "./Box";
import Brave from "./Brave";
import Cloudflare from "./Cloudflare";
import Docker from "./Docker";
import Edge from "./Edge";
import Figma from "./Figma";
import Notion from "./Notion";
import WindowsDefender from "./WindowsDefender";
import WhatsApp from "./WhatsApp";
import P4V from "./P4V";
import Postman from "./Postman";
import OnePassword from "./OnePassword";
import OmnissaHorizonClient from "./OmnissaHorizonClient";
import AmazonDCV from "./AmazonDCV";
import IntuneCompanyPortal from "./IntuneCompanyPortal";
import Santa from "./Santa";
import YubikeyManager from "./YubikeyManager";
import BeyondCompare from "./BeyondCompare";
import ITerm from "./ITerm";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import VncViewer from "./VncViewer";
import WindowsAppRemote from "./WindowsAppRemote";
import ZeroOneZeroEditor from "./010Editor";
import EightXEightWork from "./8X8Work";
import Abstract from "./Abstract";
import AdobeAcrobat from "./AdobeAcrobat";
import AdobeDigitalEditions45 from "./AdobeDigitalEditions45";
import AndroidStudio from "./AndroidStudio";
import AnyDesk from "./AnyDesk";
import Asana from "./Asana";
import BBEdit from "./BBEdit";
import CLion from "./CLion";
import Camtasia from "./Camtasia";
import Canva from "./Canva";
import ChatGpt from "./ChatGpt";
import ChatGptAtlas from "./ChatGptAtlas";
import CiscoJabber from "./CiscoJabber";
import CitrixWorkspace from "./CitrixWorkspace";
import Claude from "./Claude";
import ClickUp from "./ClickUp";
import Cursor from "./Cursor";
import Cyberduck from "./Cyberduck";
import DataGrip from "./DataGrip";
import Discord from "./Discord";
import DisplayLinkManager from "./DisplayLinkManager";
import Drawio from "./DrawIo";
import Dropbox from "./Dropbox";
import GitHubDesktop from "./GitHubDesktop";
import GitKraken from "./GitKraken";
import GoLand from "./GoLand";
import GoogleDrive from "./GoogleDrive";
import GrammarlyDesktop from "./GrammarlyDesktop";
import Granola from "./Granola";
import IMazingProfileEditor from "./IMazingProfileEditor";
import Insomnia from "./Insomnia";
import IntelliJIdea from "./IntelliJIdea";
import IntelliJIdeaCe from "./IntelliJIdeaCe";
import JetBrainsToolbox from "./JetBrainsToolbox";
import Linear from "./Linear";
import LittleSnitch from "./LittleSnitch";
import Logioptionsplus from "./Logioptionsplus";
import Loom from "./Loom";
import LuLu from "./LuLu";
import Messenger from "./Messenger";
import MicrosoftOneNote from "./MicrosoftOneNote";
import MicrosoftOutlook from "./MicrosoftOutlook";
import MicrosoftPowerPoint from "./MicrosoftPowerPoint";
import Miro from "./Miro";
import MySqlWorkbench from "./MySqlWorkbench";
import NordVpn from "./NordVpn";
import NotionCalendar from "./NotionCalendar";
import Nudge from "./Nudge";
import OmniGraffle from "./OmniGraffle";
import OneDrive from "./OneDrive";
import Opera from "./Opera";
import ParallelsDesktop from "./ParallelsDesktop";
import PhpStorm from "./PhpStorm";
import PodmanDesktop from "./PodmanDesktop";
import Pritunl from "./Pritunl";
import Privileges from "./Privileges";
import ProtonMail from "./ProtonMail";
import PyCharm from "./PyCharm";
import PyCharmCe from "./PyCharmCe";
import RancherDesktop from "./RancherDesktop";
import Rectangle from "./Rectangle";
import Rider from "./Rider";
import RubyMine from "./RubyMine";
import RustRover from "./RustRover";
import Signal from "./Signal";
import Spotify from "./Spotify";
import SublimeText from "./SublimeText";
import TablePlus from "./TablePlus";
import Tailscale from "./Tailscale";
import Telegram from "./Telegram";
import TeleportConnect from "./TeleportConnect";
import Terminal from "./Terminal";
import Thunderbird from "./Thunderbird";
import Todoist from "./Todoist";
import Tunnelblick from "./Tunnelblick";
import Twingate from "./Twingate";
import Vlc from "./Vlc";
import WebStorm from "./WebStorm";
import Webex from "./Webex";
import WrikeForMac from "./WrikeForMac";
import Zed from "./Zed";

import Bitwarden from "./Bitwarden";

// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "010 editor": ZeroOneZeroEditor,
  "8x8 work": EightXEightWork,
  "1password": OnePassword,
  abstract: Abstract,
  "adobe acrobat": AdobeAcrobat,
  "adobe acrobat reader": AcrobatReader,
  "adobe creative cloud": CreativeCloud,
  "adobe digital editions": AdobeDigitalEditions45,
  "amazon dcv": AmazonDCV,
  androidPlayStore: AndroidPlayStore,
  "android studio": AndroidStudio,
  anydesk: AnyDesk,
  appleAppStore: AppleAppStore,
  asana: Asana,
  bbedit: BBEdit,
  "beyond compare": BeyondCompare,
  bitwarden: Bitwarden,
  box: Box,
  brave: Brave,
  camtasia: Camtasia,
  canva: Canva,
  chatgpt: ChatGpt,
  "chatgpt atlas": ChatGptAtlas,
  "cisco jabber": CiscoJabber,
  "citrix workspace": CitrixWorkspace,
  claude: Claude,
  clion: CLion,
  clickup: ClickUp,
  cloudflare: Cloudflare,
  code: VisualStudioCode,
  "company portal": IntuneCompanyPortal,
  cursor: Cursor,
  cyberduck: Cyberduck,
  datagrip: DataGrip,
  discord: Discord,
  "displaylink manager": DisplayLinkManager,
  docker: Docker,
  "draw.io": Drawio,
  dropbox: Dropbox,
  falcon: Falcon,
  figma: Figma,
  firefox: Firefox,
  "github desktop": GitHubDesktop,
  gitkraken: GitKraken,
  goland: GoLand,
  "google chrome": ChromeApp,
  "google drive": GoogleDrive,
  grammarly: GrammarlyDesktop,
  granola: Granola,
  "imazing profile editor": IMazingProfileEditor,
  insomnia: Insomnia,
  "intellij idea": IntelliJIdea,
  "intellij idea ce": IntelliJIdeaCe,
  iterm2: ITerm,
  "jetbrains toolbox": JetBrainsToolbox,
  linear: Linear,
  "little snitch": LittleSnitch,
  "logi options plus": Logioptionsplus,
  loom: Loom,
  lulu: LuLu,
  messenger: Messenger,
  "microsoft edge": Edge,
  "microsoft excel": Excel,
  "microsoft onenote": MicrosoftOneNote,
  "microsoft outlook": MicrosoftOutlook,
  "microsoft powerpoint": MicrosoftPowerPoint,
  "microsoft teams": Teams,
  "microsoft visual studio code": VisualStudioCode,
  "microsoft word": Word,
  miro: Miro,
  "mysql workbench": MySqlWorkbench,
  "nord vpn": NordVpn,
  notion: Notion,
  "notion calendar": NotionCalendar,
  nudge: Nudge,
  omnigraffle: OmniGraffle,
  "omnissa horizon client": OmnissaHorizonClient,
  onedrive: OneDrive,
  opera: Opera,
  package: Package,
  "parallels desktop": ParallelsDesktop,
  p4v: P4V,
  phpstorm: PhpStorm,
  "podman desktop": PodmanDesktop,
  postman: Postman,
  privileges: Privileges,
  pritunl: Pritunl,
  "proton mail": ProtonMail,
  pycharm: PyCharm,
  "pycharm ce": PyCharmCe,
  "rancher desktop": RancherDesktop,
  rectangle: Rectangle,
  rider: Rider,
  rubymine: RubyMine,
  rustrover: RustRover,
  safari: Safari,
  santa: Santa,
  signal: Signal,
  slack: Slack,
  spotify: Spotify,
  "sublime text": SublimeText,
  "table plus": TablePlus,
  tailscale: Tailscale,
  telegram: Telegram,
  "teleport connect": TeleportConnect,
  terminal: Terminal,
  teamviewer: TeamViewer,
  thunderbird: Thunderbird,
  todoist: Todoist,
  tunnelblick: Tunnelblick,
  twingate: Twingate,
  "vnc viewer": VncViewer,
  "visual studio code": VisualStudioCode,
  vlc: Vlc,
  webstorm: WebStorm,
  webex: Webex,
  whatsapp: WhatsApp,
  "windows app": WindowsAppRemote,
  "windows defender": WindowsDefender,
  "wrike for mac": WrikeForMac,
  "yubikey manager": YubikeyManager,
  zed: Zed,
} as const;

// Maps all known Linux platforms to the LinuxOS icon
const LINUX_OS_NAME_TO_ICON_MAP = HOST_LINUX_PLATFORMS.reduce(
  (a, platform) => ({ ...a, [platform]: LinuxOS }),
  {}
);

export const PLATFORM_NAME_TO_ICON_MAP = {
  ...LINUX_OS_NAME_TO_ICON_MAP,
  darwin: MacOS,
  windows: WindowsOS,
  chrome: ChromeOS,
  ios: iOS,
  ipados: iPadOS,
} as const;

// SOFTWARE_SOURCE_TO_ICON_MAP maps different software sources to a defined
// icon.
export const SOFTWARE_SOURCE_TO_ICON_MAP = {
  package: Package,
  apt_sources: Package,
  deb_packages: Package,
  rpm_packages: Package,
  yum_sources: Package,
  npm_packages: Package,
  atom_packages: Package,
  python_packages: Package,
  homebrew_packages: Package,
  apps: AppleApp,
  ios_apps: AppleApp,
  ipados_apps: AppleApp,
  programs: WindowsApp,
  android_apps: AndroidApp,
  chrome_extensions: Extension,
  safari_extensions: Extension,
  firefox_addons: Extension,
  ie_extensions: Extension,
  chocolatey_packages: Package,
  pkg_packages: Package,
  vscode_extensions: Extension,
  jetbrains_plugins: Extension,
} as const;

/**
 * This attempts to loosely match the provided string to a key in a provided dictionary, returning the key if the
 * provided string starts with the key or undefined otherwise.
 */
const matchLoosePrefixToKey = <T extends Record<string, unknown>>(
  dict: T,
  s: string
) => {
  s = s.trim().toLowerCase();
  if (!s) {
    return undefined;
  }
  const match = Object.keys(dict).find((k) =>
    s.startsWith(k.trim().toLowerCase())
  );

  return match ? (match as keyof T) : undefined;
};

/**
 * This strictly matches the provided name and source to a software icon, returning the icon if a match is found or
 * null otherwise. It is intended to be used for special cases where a strict match is required
 * (e.g. Zoom). The caller should handle null cases by falling back to loose matching on name prefixes.
 */
const matchStrictNameSourceToIcon = ({
  name = "",
  source = "",
}: Pick<ISoftware, "name" | "source">) => {
  name = name.trim().toLowerCase();
  source = source.trim().toLowerCase();
  switch (true) {
    case name === "zoom.us.app" && source === "apps":
      return Zoom;
    case name === "zoom":
      return Zoom;
    case name.startsWith("zoom workplace"):
      return Zoom;
    default:
      return null;
  }
};

/**
 * This returns the icon component for a given software name and source. If a strict match is found,
 * it will be returned, otherwise it will fall back to loose matching on name and source prefixes.
 * If no match is found, the default package icon will be returned.
 */
export const getMatchedSoftwareIcon = ({
  name = "",
  source = "",
}: Pick<ISoftware, "name" | "source">) => {
  // Strip non-ascii, and non-printable characters
  name = name.replace(/[^\x20-\x7E]/g, "");
  // first, try strict matching on name and source
  let Icon = matchStrictNameSourceToIcon({
    name,
    source,
  });

  // if no match, try loose matching on name prefixes
  if (!Icon) {
    const matchedName = matchLoosePrefixToKey(SOFTWARE_NAME_TO_ICON_MAP, name);
    if (matchedName) {
      Icon = SOFTWARE_NAME_TO_ICON_MAP[matchedName];
    }
  }

  // if still no match, try loose matching on source prefixes
  if (!Icon) {
    const matchedSource = matchLoosePrefixToKey(
      SOFTWARE_SOURCE_TO_ICON_MAP,
      source
    );
    if (matchedSource) {
      Icon = SOFTWARE_SOURCE_TO_ICON_MAP[matchedSource];
    }
  }

  // if still no match, default to 'package'
  if (!Icon) {
    Icon = SOFTWARE_SOURCE_TO_ICON_MAP.package;
  }

  return Icon;
};

export const getMatchedOsIcon = ({ name = "" }) => {
  // Match only against platform names (never software/app maps)
  const matchedPlatform = matchLoosePrefixToKey(
    PLATFORM_NAME_TO_ICON_MAP,
    name
  );
  return matchedPlatform
    ? PLATFORM_NAME_TO_ICON_MAP[matchedPlatform]
    : SOFTWARE_SOURCE_TO_ICON_MAP.package; // TODO: Update default icon to something other than package icon >.<
};
