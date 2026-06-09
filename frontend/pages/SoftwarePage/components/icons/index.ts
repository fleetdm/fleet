// Note: if parts of a icon have a clip path, mask, or gradient, the IDs must be unique
// across all icons to avoid conflicts in the DOM. See uniqueId usage within icon components.

import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";
import { matchLoosePrefixToKey } from "utilities/strings/stringUtils";

import AbletonLive12Suite from "./AbletonLive12Suite";
import Affinity from "./Affinity";
import AmazonCorretto21 from "./AmazonCorretto21";
import AmazonCorretto24 from "./AmazonCorretto24";
import AmazonCorretto25 from "./AmazonCorretto25";
import AmazonCorretto26 from "./AmazonCorretto26";
import AmazonWorkspaces from "./AmazonWorkspaces";
import AnotherRedisDesktopManager from "./AnotherRedisDesktopManager";
import Antigravity from "./Antigravity";
import AntigravityIde from "./AntigravityIde";
import AzulZulu25Jdk from "./AzulZulu25Jdk";
import AzulZulu25Jre from "./AzulZulu25Jre";
import Backblaze from "./Backblaze";
import BeekeeperStudio from "./BeekeeperStudio";
import BetterDisplay from "./BetterDisplay";
import Bluej from "./Bluej";
import BurpSuiteCommunity from "./BurpSuiteCommunity";
import CapCut from "./CapCut";
import Cavalry from "./Cavalry";
import Charles from "./Charles";
import ChromeRemoteDesktop from "./ChromeRemoteDesktop";
import Cinc from "./Cinc";
import ClaudeDevtools from "./ClaudeDevtools";
import ClickShare from "./ClickShare";
import Comet from "./Comet";
import ConnectFonts from "./ConnectFonts";
import CrashPlan from "./CrashPlan";
import Cryptomator from "./Cryptomator";
import DellCommandUpdate from "./DellCommandUpdate";
import DellDisplayManager from "./DellDisplayManager";
import DevinDesktop from "./DevinDesktop";
import DfuBlasterPro from "./DfuBlasterPro";
import DruvaInSync from "./DruvaInSync";
import DuoDesktop from "./DuoDesktop";
import FleetDesktop from "./FleetDesktop";
import Gemini from "./Gemini";
import GenesysCloud from "./GenesysCloud";
import Git from "./Git";
import GoogleCredentialProviderForWindows from "./GoogleCredentialProviderForWindows";
import GoToMeeting from "./GoToMeeting";
import GrooveOmniDialer from "./GrooveOmniDialer";
import IbmNotifier from "./IbmNotifier";
import IconComposer from "./IconComposer";
import Iina from "./Iina";
import Joplin from "./Joplin";
import Kitty from "./Kitty";
import Krita from "./Krita";
import LastPass from "./LastPass";
import LenovoDockManager from "./LenovoDockManager";
import Marvel from "./Marvel";
import MicrosoftOffice from "./MicrosoftOffice";
import Max from "./Max";
import Microsoft365Copilot from "./Microsoft365Copilot";
import MicrosoftDotnetRuntime from "./MicrosoftDotnetRuntime";
import MicrosoftRemoteHelp from "./MicrosoftRemoteHelp";
import MindManager from "./MindManager";
import NessusAgent from "./NessusAgent";
import Nextcloud from "./Nextcloud";
import Nodejs from "./Nodejs";
import Notepad from "./Notepad++";
import OktaVerify from "./OktaVerify";
import Ollama from "./Ollama";
import OpenvpnConnect from "./OpenvpnConnect";
import Pd from "./Pd";
import PlantronicsHub from "./PlantronicsHub";
import Postgresql15 from "./Postgresql15";
import Postgresql16 from "./Postgresql16";
import Postgresql17 from "./Postgresql17";
import Postgresql18 from "./Postgresql18";
import PowerAutomate from "./PowerAutomate";
import PowerBi from "./PowerBi";
import Plugdata from "./Plugdata";
import PowerMonitor from "./PowerMonitor";
import Powershell from "./Powershell";
import Powertoys from "./Powertoys";
import Prisma from "./Prisma";
import Proxifier from "./Proxifier";
import Proxyman from "./Proxyman";
import Putty from "./Putty";
import R from "./R";
import RealVncServer from "./RealVncServer";
import Reaper from "./Reaper";
import Rstudio from "./Rstudio";
import RustDesk from "./RustDesk";
import Secretive from "./Secretive";
import SequelAce from "./SequelAce";
import SevenZip from "./7Zip";
import Abstract from "./Abstract";
import AcrobatReader from "./AcrobatReader";
import AdobeDigitalEditions45 from "./AdobeDigitalEditions45";
import AdobeDngConverter from "./AdobeDngConverter";
import Aircall from "./Aircall";
import Airtame from "./Airtame";
import AmazonChime from "./AmazonChime";
import AmazonDCV from "./AmazonDCV";
import AndroidApp from "./AndroidApp";
import AndroidOS from "./AndroidOS";
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
import ArduinoIde from "./ArduinoIde";
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
import Python313 from "./Python313";
import Python314 from "./Python314";
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
import SfSymbols from "./SfSymbols";
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
import Swiftdialog from "./Swiftdialog";
import TableauDesktop from "./TableauDesktop";
import TablePlus from "./TablePlus";
import Tailscale from "./Tailscale";
import TeamViewer from "./TeamViewer";
import Teams from "./Teams";
import Telegram from "./Telegram";
import TeleportConnect from "./TeleportConnect";
import Terminal from "./Terminal";
import TextExpander from "./TextExpander";
import TheUnarchiver from "./TheUnarchiver";
import Thunderbird from "./Thunderbird";
import Todoist from "./Todoist";
import TorBrowser from "./TorBrowser";
import Tortoisegit from "./Tortoisegit";
import Tower from "./Tower";
import Transmit from "./Transmit";
import Tunnelblick from "./Tunnelblick";
import Twingate from "./Twingate";
import Utm from "./Utm";
import VcRedistX64 from "./VcRedistX64";
import VirtualBox from "./VirtualBox";
import VirtualBuddy from "./VirtualBuddy";
import Viscosity from "./Viscosity";
import VisualStudioCode from "./VisualStudioCode";
import Vlc from "./Vlc";
import VncViewer from "./VncViewer";
import VsCodium from "./VsCodium";
import WacomCenter from "./WacomCenter";
import Warp from "./Warp";
import Waterfox from "./Waterfox";
import Wave from "./Wave";
import Wavebox from "./Wavebox";
import Wealthfolio from "./Wealthfolio";
import Weasis from "./Weasis";
import WebStorm from "./WebStorm";
import Webcatalog from "./Webcatalog";
import Webex from "./Webex";
import Wechat from "./Wechat";
import Weektodo from "./Weektodo";
import Wezterm from "./Wezterm";
import Whatroute from "./Whatroute";
import WhatsApp from "./WhatsApp";
import Whisky from "./Whisky";
import Whispering from "./Whispering";
import Wifiman from "./Wifiman";
import Windowkeys from "./Windowkeys";
import WindowsApp from "./WindowsApp";
import WindowsAppRemote from "./WindowsAppRemote";
import WindowsDefender from "./WindowsDefender";
import WindowsOS from "./WindowsOS";
import Windsurf from "./Windsurf";
import Winrar from "./Winrar";
import Wins from "./Wins";
import Winscp from "./Winscp";
import Wireshark from "./Wireshark";
import WisprFlow from "./WisprFlow";
import Witch from "./Witch";
import WondershareEdrawmax from "./WondershareEdrawmax";
import WondershareFilmora from "./WondershareFilmora";
import Word from "./Word";
import Wordservice from "./Wordservice";
import Workflowy from "./Workflowy";
import WorksheetCrafter from "./WorksheetCrafter";
import Workspaces from "./Workspaces";
import WrikeForMac from "./WrikeForMac";
import XCreds from "./XCreds";
import Yaak from "./Yaak";
import Yacreader from "./Yacreader";
import Yattee from "./Yattee";
import Yippy from "./Yippy";
import YubicoAuthenticator from "./YubicoAuthenticator";
import YubikeyManager from "./YubikeyManager";
import Zappy from "./Zappy";
import Zed from "./Zed";
import Zen from "./Zen";
import Zeplin from "./Zeplin";
import ZeroOneZeroEditor from "./010Editor";
import Zettlr from "./Zettlr";
import Zight from "./Zight";
import Zoom from "./Zoom";
import ZoomRooms from "./ZoomRooms";
import Zotero from "./Zotero";
import Zulip from "./Zulip";
import Zwift from "./Zwift";
// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "010 editor": ZeroOneZeroEditor,
  "7 zip": SevenZip,
  "7-zip": SevenZip,
  "8x8 work": EightXEightWork,
  "1password": OnePassword,
  "ableton live suite": AbletonLive12Suite,
  abstract: Abstract,
  "adobe acrobat": AcrobatReader,
  "adobe acrobat reader": AcrobatReader,
  "adobe creative cloud": CreativeCloud,
  "adobe digital editions": AdobeDigitalEditions45,
  "adobe dng converter": AdobeDngConverter,
  affinity: Affinity,
  aircall: Aircall,
  airtame: Airtame,
  "amazon chime": AmazonChime,
  "amazon corretto 21": AmazonCorretto21,
  "amazon corretto 24": AmazonCorretto24,
  "amazon corretto 25": AmazonCorretto25,
  "amazon corretto 26": AmazonCorretto26,
  "amazon dcv": AmazonDCV,
  "amazon workspaces": AmazonWorkspaces,
  androidPlayStore: AndroidPlayStore,
  "android studio": AndroidStudio,
  anka: Anka,
  "another redis desktop manager": AnotherRedisDesktopManager,
  antigravity: Antigravity,
  "antigravity ide": AntigravityIde,
  capcut: CapCut,
  "dfu blaster pro": DfuBlasterPro,
  "google antigravity ide": AntigravityIde,
  anydesk: AnyDesk,
  apparency: Apparency,
  appcleaner: AppCleaner,
  appleAppStore: AppleAppStore,
  arc: Arc,
  archaeology: Archaeology,
  "arduino ide": ArduinoIde,
  asana: Asana,
  audacity: Audacity,
  avast: AvastSecureBrowser,
  "aws vpn client": AwsVpnClient,
  "aws client vpn": AwsVpnClient,
  "azul zulu 25 jdk": AzulZulu25Jdk,
  "azul zulu 25 jre": AzulZulu25Jre,
  backblaze: Backblaze,
  balenaetcher: BalenaEtcher,
  bbedit: BBEdit,
  "beekeeper studio": BeekeeperStudio,
  betterdisplay: BetterDisplay,
  "beyond compare": BeyondCompare,
  bitwarden: Bitwarden,
  blender: Blender,
  bluej: Bluej,
  box: Box,
  brave: Brave,
  bruno: Bruno,
  "burp suite community": BurpSuiteCommunity,
  calibre: Calibre,
  camtasia: Camtasia,
  canva: Canva,
  cavalry: Cavalry,
  charles: Charles,
  "chatgpt atlas": ChatGptAtlas,
  chatgpt: ChatGpt,
  "chrome remote desktop": ChromeRemoteDesktop,
  "cinc workstation": Cinc,
  "cisco jabber": CiscoJabber,
  "citrix workspace": CitrixWorkspace,
  claude: Claude,
  "claude-devtools": ClaudeDevtools,
  cleanmymac_5: CleanMyMac,
  cleanmymac: CleanMyMac,
  "cleanshot x": CleanShotX,
  clickshare: ClickShare,
  clion: CLion,
  clickup: ClickUp,
  "clockify desktop": ClockifyDesktop,
  cloudflare: Cloudflare,
  code: VisualStudioCode,
  comet: Comet,
  "company portal": IntuneCompanyPortal,
  "connect fonts": ConnectFonts,
  crashplan: CrashPlan,
  cryptomator: Cryptomator,
  "dell command update": DellCommandUpdate,
  "dell display manager": DellDisplayManager,
  "devin desktop": DevinDesktop,
  "duo desktop": DuoDesktop,
  "fleet desktop": FleetDesktop,
  gemini: Gemini,
  "genesys cloud": GenesysCloud,
  git: Git,
  "google credential provider for windows": GoogleCredentialProviderForWindows,
  gotomeeting: GoToMeeting,
  "groove omnidialer": GrooveOmniDialer,
  "ibm notifier": IbmNotifier,
  "icon composer": IconComposer,
  iina: Iina,
  insyncclient: DruvaInSync,
  joplin: Joplin,
  kitty: Kitty,
  krita: Krita,
  lastpass: LastPass,
  "lenovo dock manager": LenovoDockManager,
  marvel: Marvel,
  "microsoft office": MicrosoftOffice,
  max: Max,
  "microsoft .net runtime": MicrosoftDotnetRuntime,
  "microsoft 365 copilot": Microsoft365Copilot,
  "microsoft remote help": MicrosoftRemoteHelp,
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
  mindmanager: MindManager,
  "mongodb compass": MongoDbCompass,
  "mozilla firefox": Firefox,
  "github desktop": GitHubDesktop,
  gitkraken: GitKraken,
  goland: GoLand,
  "google antigravity": Antigravity,
  "google chrome": ChromeApp,
  "google drive": GoogleDrive,
  grammarly: GrammarlyDesktop,
  granola: Granola,
  imazing: IMazingProfileEditor,
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
  "microsoft edge": Edge,
  "microsoft excel": Excel,
  "microsoft onenote": MicrosoftOneNote,
  "microsoft outlook": MicrosoftOutlook,
  "microsoft powerpoint": MicrosoftPowerPoint,
  "microsoft teams": Teams,
  "microsoft visual c++": VcRedistX64,
  "microsoft visual studio code": VisualStudioCode,
  "microsoft word": Word,
  miro: Miro,
  "mysql workbench": MySqlWorkbench,
  "nessus agent": NessusAgent,
  nextcloud: Nextcloud,
  "node.js": Nodejs,
  "nord vpn": NordVpn,
  nordpass: Nordpass,
  nordvpn: NordVpn,
  "notepad++": Notepad,
  "notion calendar": NotionCalendar,
  notion: Notion,
  nova: Nova,
  nudge: Nudge,
  obs: Obs,
  obsidian: Obsidian,
  "okta verify": OktaVerify,
  ollama: Ollama,
  omnigraffle: OmniGraffle,
  "omnissa horizon client": OmnissaHorizonClient,
  onedrive: OneDrive,
  "openvpn connect": OpenvpnConnect,
  opera: Opera,
  orbstack: OrbStack,
  package: Package,
  "parallels desktop": ParallelsDesktop,
  p4v: P4V,
  pd: Pd,
  "pgadmin 4": PgAdmin4,
  pgadmin4: PgAdmin4,
  phpstorm: PhpStorm,
  "plantronics hub": PlantronicsHub,
  plugdata: Plugdata,
  "podman desktop": PodmanDesktop,
  "postgresql 15": Postgresql15,
  "postgresql 16": Postgresql16,
  "postgresql 17": Postgresql17,
  "postgresql 18": Postgresql18,
  postman: Postman,
  "power automate": PowerAutomate,
  "power bi": PowerBi,
  "power monitor": PowerMonitor,
  powershell: Powershell,
  powertoys: Powertoys,
  prisma: Prisma,
  privileges: Privileges,
  pritunl: Pritunl,
  "proton mail": ProtonMail,
  protonvpn: ProtonVpn,
  proxifier: Proxifier,
  proxyman: Proxyman,
  putty: Putty,
  "pycharm ce": PyCharmCe,
  pycharm: PyCharm,
  "python 3.13": Python313,
  "python 3.14": Python314,
  quip: Quip,
  "r for windows": R,
  "rancher desktop": RancherDesktop,
  rapidapi: RapidApi,
  raycast: Raycast,
  "realvnc server": RealVncServer,
  reaper: Reaper,
  rectangle: Rectangle,
  rider: Rider,
  "royal tsx": RoyalTsx,
  rstudio: Rstudio,
  rubymine: RubyMine,
  rustdesk: RustDesk,
  rustrover: RustRover,
  safari: Safari,
  santa: Santa,
  secretive: Secretive,
  "sequel ace": SequelAce,
  "sf symbols": SfSymbols,
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
  swiftdialog: Swiftdialog,
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
  "the unarchiver": TheUnarchiver,
  thunderbird: Thunderbird,
  todoist: Todoist,
  "tor browser": TorBrowser,
  tortoisegit: Tortoisegit,
  tower: Tower,
  transmit: Transmit,
  tunnelblick: Tunnelblick,
  twingate: Twingate,
  utm: Utm,
  virtualbox: VirtualBox,
  virtualbuddy: VirtualBuddy,
  viscosity: Viscosity,
  "vnc viewer": VncViewer,
  "visual studio code": VisualStudioCode,
  vlc: Vlc,
  vscodium: VsCodium,
  "wacom center": WacomCenter,
  "wacom tablet": WacomCenter,
  warp: Warp,
  waterfox: Waterfox,
  "wave terminal": Wave,
  wavebox: Wavebox,
  wealthfolio: Wealthfolio,
  weasis: Weasis,
  webstorm: WebStorm,
  webcatalog: Webcatalog,
  webex: Webex,
  "wechat for mac": Wechat,
  weektodo: Weektodo,
  wezterm: Wezterm,
  whatroute: Whatroute,
  whatsapp: WhatsApp,
  whisky: Whisky,
  whispering: Whispering,
  "wifiman desktop": Wifiman,
  windowkeys: Windowkeys,
  "windows app": WindowsApp,
  "windows app remote": WindowsAppRemote,
  "windows defender": WindowsDefender,
  windsurf: Windsurf,
  winrar: Winrar,
  wins: Wins,
  winscp: Winscp,
  wireshark: Wireshark,
  "wispr flow": WisprFlow,
  witch: Witch,
  edrawmax: WondershareEdrawmax,
  "wondershare filmora": WondershareFilmora,
  wordservice: Wordservice,
  workflowy: Workflowy,
  "worksheet crafter": WorksheetCrafter,
  workspaces: Workspaces,
  "wrike for mac": WrikeForMac,
  wrike: WrikeForMac,
  xcreds: XCreds,
  yaak: Yaak,
  yacreader: Yacreader,
  yattee: Yattee,
  yippy: Yippy,
  "yubico authenticator": YubicoAuthenticator,
  "yubikey manager": YubikeyManager,
  zappy: Zappy,
  zed: Zed,
  zen: Zen,
  zeplin: Zeplin,
  zettlr: Zettlr,
  zight: Zight,
  "zoom rooms": ZoomRooms,
  zotero: Zotero,
  zulip: Zulip,
  zwift: Zwift,
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
  android: AndroidOS,
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
