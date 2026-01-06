// Note: if parts of a icon have a clip path, mask, or gradient, the IDs must be unique
// across all icons to avoid conflicts in the DOM. See uniqueId usage within icon components.

import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";

import CrashPlan from "./CrashPlan";
import SevenZip from "./7Zip";
import Abstract from "./Abstract";
import AcrobatReader from "./AcrobatReader";
import AdobeAcrobat from "./AdobeAcrobat";
import AdobeDigitalEditions45 from "./AdobeDigitalEditions45";
import AdobeDngConverter from "./AdobeDngConverter";
import Aircall from "./Aircall";
import Airtame from "./Airtame";
import AmazonChime from "./AmazonChime";
import AmazonDCV from "./AmazonDCV";
import AndroidApp from "./AndroidApp";
import AndroidPlayStore from "./AndroidPlayStore";
import AndroidStudio from "./AndroidStudio";
import Anka from "./Anka";
import AnyDesk from "./AnyDesk";
import Apparency from "./Apparency";
import AppCleaner from "./AppCleaner";
import AppleApp from "./AppleApp";
import AppleAppStore from "./AppleAppStore";
import Arc from "./Arc";
import Archaeology from "./Archaeology";
import Asana from "./Asana";
import Audacity from "./Audacity";
import AvastSecureBrowser from "./AvastSecureBrowser";
import AwsVpnClient from "./AwsVpnClient";
import BalenaEtcher from "./BalenaEtcher";
import BBEdit from "./BBEdit";
import BeyondCompare from "./BeyondCompare";
import Bitwarden from "./Bitwarden";
import Blender from "./Blender";
import Box from "./Box";
import Brave from "./Brave";
import Bruno from "./Bruno";
import Calibre from "./Calibre";
import CleanMyMac from "./CleanMyMac";
import CleanShotX from "./CleanShotX";
import CLion from "./CLion";
import Camtasia from "./Camtasia";
import Canva from "./Canva";
import ChatGpt from "./ChatGpt";
import ChatGptAtlas from "./ChatGptAtlas";
import ChromeApp from "./ChromeApp";
import ChromeOS from "./ChromeOS";
import CiscoJabber from "./CiscoJabber";
import CitrixWorkspace from "./CitrixWorkspace";
import Claude from "./Claude";
import ClickUp from "./ClickUp";
import ClockifyDesktop from "./ClockifyDesktop";
import Cloudflare from "./Cloudflare";
import CompanyPortal from "./CompanyPortal";
import CotEditor from "./CotEditor";
import CreativeCloud from "./AdobeCreativeCloud";
import Cursor from "./Cursor";
import Cyberduck from "./Cyberduck";
import Dash from "./Dash";
import DataGrip from "./DataGrip";
import DbBrowserForSqLite from "./DbBrowserForSqLite";
import DBeaver from "./DBeaver";
import DBeaverEe from "./DBeaverEe";
import DBeaverLite from "./DBeaverLite";
import DBeaverUltimate from "./DBeaverUltimate";
import DeepL from "./DeepL";
import Dialpad from "./Dialpad";
import Discord from "./Discord";
import DisplayLinkManager from "./DisplayLinkManager";
import Docker from "./Docker";
import Drawio from "./DrawIo";
import Dropbox from "./Dropbox";
import Eclipse from "./Eclipse";
import Edge from "./Edge";
import Egnyte from "./Egnyte";
import EightXEightWork from "./8X8Work";
import ElgatoControlCenter from "./ElgatoControlCenter";
import ElgatoStreamDeck from "./ElgatoStreamDeck";
import Evernote from "./Evernote";
import Excel from "./Excel";
import ExpressVpn from "./ExpressVpn";
import Extension from "./Extension";
import Falcon from "./Falcon";
import Figma from "./Figma";
import FileMakerPro from "./FileMakerPro";
import Firefox from "./Firefox";
import Fork from "./Fork";
import Front from "./Front";
import Ghostty from "./Ghostty";
import Gimp from "./Gimp";
import GitHubDesktop from "./GitHubDesktop";
import GitKraken from "./GitKraken";
import GoLand from "./GoLand";
import GoogleDrive from "./GoogleDrive";
import GpgKeychain from "./GpgKeychain";
import GrammarlyDesktop from "./GrammarlyDesktop";
import Granola from "./Granola";
import Hyper from "./Hyper";
import IMazingProfileEditor from "./IMazingProfileEditor";
import Inkscape from "./Inkscape";
import ITerm from "./ITerm";
import Insomnia from "./Insomnia";
import IntelliJIdea from "./IntelliJIdea";
import IntelliJIdeaCe from "./IntelliJIdeaCe";
import IntuneCompanyPortal from "./IntuneCompanyPortal";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import JabraDirect from "./JabraDirect";
import JetBrainsToolbox from "./JetBrainsToolbox";
import KeePassXc from "./KeePassXc";
import KeeperPasswordManager from "./KeeperPasswordManager";
import Keka from "./Keka";
import Lens from "./Lens";
import LibreOffice from "./LibreOffice";
import Linear from "./Linear";
import LinuxOS from "./LinuxOS";
import LittleSnitch from "./LittleSnitch";
import Logioptionsplus from "./Logioptionsplus";
import Loom from "./Loom";
import LuLu from "./LuLu";
import Maccy from "./Maccy";
import MacOS from "./MacOS";
import Mattermost from "./Mattermost";
import Messenger from "./Messenger";
import MicrosoftAutoUpdate from "./MicrosoftAutoUpdate";
import MicrosoftEdge from "./MicrosoftEdge";
import MicrosoftOneNote from "./MicrosoftOneNote";
import MicrosoftOutlook from "./MicrosoftOutlook";
import MicrosoftPowerPoint from "./MicrosoftPowerPoint";
import Miro from "./Miro";
import MongoDbCompass from "./MongoDbCompass";
import MySqlWorkbench from "./MySqlWorkbench";
import Nordpass from "./Nordpass";
import NordVpn from "./NordVpn";
import Notion from "./Notion";
import NotionCalendar from "./NotionCalendar";
import Nova from "./Nova";
import Nudge from "./Nudge";
import Obs from "./Obs";
import Obsidian from "./Obsidian";
import OmniGraffle from "./OmniGraffle";
import OmnissaHorizonClient from "./OmnissaHorizonClient";
import OneDrive from "./OneDrive";
import OnePassword from "./OnePassword";
import Opera from "./Opera";
import OrbStack from "./OrbStack";
import P4V from "./P4V";
import Package from "./Package";
import ParallelsDesktop from "./ParallelsDesktop";
import PgAdmin4 from "./PgAdmin4";
import PhpStorm from "./PhpStorm";
import PodmanDesktop from "./PodmanDesktop";
import Postman from "./Postman";
import Pritunl from "./Pritunl";
import Privileges from "./Privileges";
import ProtonMail from "./ProtonMail";
import ProtonVpn from "./ProtonVpn";
import PyCharm from "./PyCharm";
import PyCharmCe from "./PyCharmCe";
import Quip from "./Quip";
import RancherDesktop from "./RancherDesktop";
import RapidApi from "./RapidApi";
import Raycast from "./Raycast";
import Rectangle from "./Rectangle";
import Rider from "./Rider";
import RoyalTsx from "./RoyalTsx";
import RubyMine from "./RubyMine";
import RustRover from "./RustRover";
import Safari from "./Safari";
import Santa from "./Santa";
import Shottr from "./Shottr";
import Signal from "./Signal";
import Sketch from "./Sketch";
import Slack from "./Slack";
import Snagit from "./Snagit";
import Sourcetree from "./Sourcetree";
import SplashtopBusiness from "./SplashtopBusiness";
import SplashtopStreamer from "./SplashtopStreamer";
import Spotify from "./Spotify";
import Stats from "./Stats";
import Steam from "./Steam";
import SublimeMerge from "./SublimeMerge";
import SublimeText from "./SublimeText";
import Surfshark from "./Surfshark";
import SuspiciousPackage from "./SuspiciousPackage";
import TableauDesktop from "./TableauDesktop";
import TablePlus from "./TablePlus";
import Tailscale from "./Tailscale";
import TeamViewer from "./TeamViewer";
import Teams from "./Teams";
import Telegram from "./Telegram";
import TeleportConnect from "./TeleportConnect";
import Terminal from "./Terminal";
import TextExpander from "./TextExpander";
import Thunderbird from "./Thunderbird";
import Todoist from "./Todoist";
import Tower from "./Tower";
import Transmit from "./Transmit";
import Tunnelblick from "./Tunnelblick";
import Twingate from "./Twingate";
import Utm from "./Utm";
import VirtualBox from "./VirtualBox";
import Viscosity from "./Viscosity";
import VisualStudioCode from "./VisualStudioCode";
import Vlc from "./Vlc";
import VncViewer from "./VncViewer";
import WacomCenter from "./WacomCenter";
import Warp from "./Warp";
import WebStorm from "./WebStorm";
import Webex from "./Webex";
import WhatsApp from "./WhatsApp";
import WindowsApp from "./WindowsApp";
import WindowsAppRemote from "./WindowsAppRemote";
import WindowsDefender from "./WindowsDefender";
import WindowsOS from "./WindowsOS";
import Windsurf from "./Windsurf";
import Wireshark from "./Wireshark";
import Word from "./Word";
import WrikeForMac from "./WrikeForMac";
import YubicoAuthenticator from "./YubicoAuthenticator";
import YubikeyManager from "./YubikeyManager";
import Zed from "./Zed";
import Zeplin from "./Zeplin";
import ZeroOneZeroEditor from "./010Editor";
import Zoom from "./Zoom";

// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "010 editor": ZeroOneZeroEditor,
  "7 zip": SevenZip,
  "7-zip": SevenZip,
  "8x8 work": EightXEightWork,
  "1password": OnePassword,
  abstract: Abstract,
  "adobe acrobat": AdobeAcrobat,
  "adobe acrobat reader": AcrobatReader,
  "adobe creative cloud": CreativeCloud,
  "adobe digital editions": AdobeDigitalEditions45,
  "adobe dng converter": AdobeDngConverter,
  aircall: Aircall,
  airtame: Airtame,
  "amazon chime": AmazonChime,
  "amazon dcv": AmazonDCV,
  androidPlayStore: AndroidPlayStore,
  "android studio": AndroidStudio,
  anka: Anka,
  anydesk: AnyDesk,
  apparency: Apparency,
  appcleaner: AppCleaner,
  appleAppStore: AppleAppStore,
  arc: Arc,
  archaeology: Archaeology,
  asana: Asana,
  audacity: Audacity,
  avast: AvastSecureBrowser,
  "aws vpn client": AwsVpnClient,
  "aws client vpn": AwsVpnClient,
  balenaetcher: BalenaEtcher,
  bbedit: BBEdit,
  "beyond compare": BeyondCompare,
  bitwarden: Bitwarden,
  blender: Blender,
  box: Box,
  brave: Brave,
  bruno: Bruno,
  calibre: Calibre,
  camtasia: Camtasia,
  canva: Canva,
  "chatgpt atlas": ChatGptAtlas,
  chatgpt: ChatGpt,
  "cisco jabber": CiscoJabber,
  "citrix workspace": CitrixWorkspace,
  claude: Claude,
  cleanmymac_5: CleanMyMac,
  cleanmymac: CleanMyMac,
  "cleanshot x": CleanShotX,
  clion: CLion,
  clickup: ClickUp,
  "clockify desktop": ClockifyDesktop,
  cloudflare: Cloudflare,
  code: VisualStudioCode,
  "company portal": IntuneCompanyPortal,
  crashplan: CrashPlan,
  "microsoft.companyportal": IntuneCompanyPortal,
  coteditor: CotEditor,
  cursor: Cursor,
  cyberduck: Cyberduck,
  dash: Dash,
  datagrip: DataGrip,
  "db browser for sqlite": DbBrowserForSqLite,
  "dbeaver community": DBeaver,
  dbeaver: DBeaver,
  "dbeaver enterprise edition": DBeaverEe,
  dbeaveree: DBeaverEe,
  "dbeaver lite edition": DBeaverLite,
  dbeaverlite: DBeaverLite,
  "dbeaver ultimate edition": DBeaverUltimate,
  dbeaverultimate: DBeaverUltimate,
  deepl: DeepL,
  dialpad: Dialpad,
  discord: Discord,
  "DisplayLink USB Graphics Software": DisplayLinkManager,
  "dng converter": AdobeDngConverter,
  docker: Docker,
  "draw.io": Drawio,
  dropbox: Dropbox,
  eclipse: Eclipse,
  edge: MicrosoftEdge,
  egnyte: Egnyte,
  "elgato control center": ElgatoControlCenter,
  "elgato stream deck": ElgatoStreamDeck,
  evernote: Evernote,
  expressvpn: ExpressVpn,
  falcon: Falcon,
  figma: Figma,
  "filemaker pro": FileMakerPro,
  firefox: Firefox,
  fork: Fork,
  front: Front,
  ghostty: Ghostty,
  gimp: Gimp,
  "gpg keychain": GpgKeychain,
  "gpg suite": GpgKeychain,
  hyper: Hyper,
  inkscape: Inkscape,
  "jabra direct": JabraDirect,
  keepassxc: KeePassXc,
  "keeper password manager": KeeperPasswordManager,
  keka: Keka,
  lens: Lens,
  libreoffice: LibreOffice,
  maccy: Maccy,
  mattermost: Mattermost,
  "microsoft autoupdate": MicrosoftAutoUpdate,
  "microsoft auto update": MicrosoftAutoUpdate,
  "mongodb compass": MongoDbCompass,
  "mozilla firefox": Firefox,
  "github desktop": GitHubDesktop,
  gitkraken: GitKraken,
  goland: GoLand,
  "google chrome": ChromeApp,
  "google drive": GoogleDrive,
  grammarly: GrammarlyDesktop,
  granola: Granola,
  "imazing profile editor": IMazingProfileEditor,
  insomnia: Insomnia,
  "intellij idea ce": IntelliJIdeaCe,
  "intellij idea": IntelliJIdea,
  iterm2: ITerm,
  "jetbrains toolbox": JetBrainsToolbox,
  linear: Linear,
  "little snitch": LittleSnitch,
  "logi options+": Logioptionsplus,
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
  nordpass: Nordpass,
  nordvpn: NordVpn,
  "notion calendar": NotionCalendar,
  notion: Notion,
  nova: Nova,
  nudge: Nudge,
  obs: Obs,
  obsidian: Obsidian,
  omnigraffle: OmniGraffle,
  "omnissa horizon client": OmnissaHorizonClient,
  onedrive: OneDrive,
  opera: Opera,
  orbstack: OrbStack,
  package: Package,
  "parallels desktop": ParallelsDesktop,
  p4v: P4V,
  "pgadmin 4": PgAdmin4,
  pgadmin4: PgAdmin4,
  phpstorm: PhpStorm,
  "podman desktop": PodmanDesktop,
  postman: Postman,
  privileges: Privileges,
  pritunl: Pritunl,
  "proton mail": ProtonMail,
  protonvpn: ProtonVpn,
  "pycharm ce": PyCharmCe,
  pycharm: PyCharm,
  quip: Quip,
  "rancher desktop": RancherDesktop,
  rapidapi: RapidApi,
  raycast: Raycast,
  rectangle: Rectangle,
  rider: Rider,
  "royal tsx": RoyalTsx,
  rubymine: RubyMine,
  rustrover: RustRover,
  safari: Safari,
  santa: Santa,
  shottr: Shottr,
  signal: Signal,
  sketch: Sketch,
  slack: Slack,
  snagit: Snagit,
  sourcetree: Sourcetree,
  "splashtop business": SplashtopBusiness,
  "splashtop streamer": SplashtopStreamer,
  spotify: Spotify,
  stats: Stats,
  steam: Steam,
  "stream deck": ElgatoStreamDeck,
  "sublime merge": SublimeMerge,
  "sublime text": SublimeText,
  surfshark: Surfshark,
  "suspicious package": SuspiciousPackage,
  tableau: TableauDesktop,
  tableplus: TablePlus,
  tailscale: Tailscale,
  telegram: Telegram,
  "teleport connect": TeleportConnect,
  "teleport suite": TeleportConnect,
  teleport: TeleportConnect,
  terminal: Terminal,
  teamviewer: TeamViewer,
  textexpander: TextExpander,
  thunderbird: Thunderbird,
  todoist: Todoist,
  tower: Tower,
  transmit: Transmit,
  tunnelblick: Tunnelblick,
  twingate: Twingate,
  utm: Utm,
  virtualbox: VirtualBox,
  viscosity: Viscosity,
  "vnc viewer": VncViewer,
  "visual studio code": VisualStudioCode,
  vlc: Vlc,
  "wacom center": WacomCenter,
  "wacom tablet": WacomCenter,
  warp: Warp,
  webstorm: WebStorm,
  webex: Webex,
  whatsapp: WhatsApp,
  "windows app": WindowsApp,
  "windows app remote": WindowsAppRemote,
  "windows defender": WindowsDefender,
  windsurf: Windsurf,
  wireshark: Wireshark,
  "wrike for mac": WrikeForMac,
  wrike: WrikeForMac,
  "yubico authenticator": YubicoAuthenticator,
  "yubikey manager": YubikeyManager,
  zed: Zed,
  zeplin: Zeplin,
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
  programs: WindowsOS,
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
 * provided string starts with the key or undefined otherwise. Keys are sorted by length (longest first) to ensure
 * more specific matches are checked before shorter, more general ones (e.g., "archaeology" before "arc").
 */
const matchLoosePrefixToKey = <T extends Record<string, unknown>>(
  dict: T,
  s: string
) => {
  s = s.trim().toLowerCase();
  if (!s) {
    return undefined;
  }
  // Sort keys by length (longest first) to prioritize more specific matches
  const sortedKeys = Object.keys(dict).sort((a, b) => b.length - a.length);
  const match = sortedKeys.find((k) => s.startsWith(k.trim().toLowerCase()));

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
    case name === "microsoft.companyportal":
      return IntuneCompanyPortal;
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
