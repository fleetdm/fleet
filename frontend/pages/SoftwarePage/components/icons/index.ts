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
import Cacher from "./Cacher";
import Caffeine from "./Caffeine";
import Calibre from "./Calibre";
import CalibriteProfiler from "./CalibriteProfiler";
import CamoStudio from "./CamoStudio";
import Camtasia from "./Camtasia";
import CamundaModeler from "./CamundaModeler";
import Canva from "./Canva";
import Captain from "./Captain";
import Captin from "./Captin";
import Capto from "./Capto";
import CarbonCopyCloner from "./CarbonCopyCloner";
import Cardhop from "./Cardhop";
import Cellprofiler from "./Cellprofiler";
import Chalk from "./Chalk";
import Charmstone from "./Charmstone";
import ChatGpt from "./ChatGpt";
import ChatGptAtlas from "./ChatGptAtlas";
import Chatwise from "./Chatwise";
import Cheetah3D from "./Cheetah3D";
import CherryStudio from "./CherryStudio";
import Chime from "./Chime";
import Choosy from "./Choosy";
import ChromeApp from "./ChromeApp";
import ChromeOS from "./ChromeOS";
import CiscoJabber from "./CiscoJabber";
import CitrixWorkspace from "./CitrixWorkspace";
import Claude from "./Claude";
import Cleanclip from "./Cleanclip";
import CleanMyMac from "./CleanMyMac";
import CleanShotX from "./CleanShotX";
import ClickUp from "./ClickUp";
import CLion from "./CLion";
import Clipbook from "./Clipbook";
import Clipgrab from "./Clipgrab";
import Clipy from "./Clipy";
import Clocker from "./Clocker";
import ClockifyDesktop from "./ClockifyDesktop";
import Clop from "./Clop";
import Cloudflare from "./Cloudflare";
import Cloudmounter from "./Cloudmounter";
import CmakeApp from "./CmakeApp";
import Cmux from "./Cmux";
import Coconutbattery from "./Coconutbattery";
import Codeedit from "./Codeedit";
import Coderunner from "./Coderunner";
import Codexbar from "./Codexbar";
import CogApp from "./CogApp";
import Colorsnapper from "./Colorsnapper";
import ColourContrastAnalyser from "./ColourContrastAnalyser";
import Commander from "./Commander";
import CommanderOne from "./CommanderOne";
import CommandTabPlus from "./CommandTabPlus";
import Companion from "./Companion";
import CopilotMoney from "./CopilotMoney";
import Cork from "./Cork";
import CotEditor from "./CotEditor";
import CreativeCloud from "./AdobeCreativeCloud";
import Crossover from "./Crossover";
import Crystalfetch from "./Crystalfetch";
import Cursor from "./Cursor";
import Cursorsense from "./Cursorsense";
import Cursr from "./Cursr";
import Customshortcuts from "./Customshortcuts";
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
import GarminExpress from "./GarminExpress";
import Gather from "./Gather";
import Gdevelop from "./Gdevelop";
import Geany from "./Geany";
import Geekbench from "./Geekbench";
import Gephi from "./Gephi";
import Ghostty from "./Ghostty";
import Gimp from "./Gimp";
import Gitfinder from "./Gitfinder";
import GithubCopilotForXcode from "./GithubCopilotForXcode";
import GitHubDesktop from "./GitHubDesktop";
import Gitify from "./Gitify";
import GitKraken from "./GitKraken";
import GitupApp from "./GitupApp";
import Glyphs from "./Glyphs";
import Go2Shell from "./Go2Shell";
import Godot from "./Godot";
import Godspeed from "./Godspeed";
import GogGalaxy from "./GogGalaxy";
import GoLand from "./GoLand";
import Goodsync from "./Goodsync";
import GoogleDrive from "./GoogleDrive";
import GoogleEarthPro from "./GoogleEarthPro";
import GpgKeychain from "./GpgKeychain";
import Gpodder from "./Gpodder";
import GrammarlyDesktop from "./GrammarlyDesktop";
import Grandperspective from "./Grandperspective";
import Granola from "./Granola";
import Grids from "./Grids";
import Gyazo from "./Gyazo";
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
import Ocenaudio from "./Ocenaudio";
import OkJson from "./OkJson";
import Omnidisksweeper from "./Omnidisksweeper";
import Omnifocus from "./Omnifocus";
import OmniGraffle from "./OmniGraffle";
import Omnioutliner from "./Omnioutliner";
import Omniplan from "./Omniplan";
import OmnissaHorizonClient from "./OmnissaHorizonClient";
import OneDrive from "./OneDrive";
import OnePassword from "./OnePassword";
import OneSwitch from "./OneSwitch";
import Onionshare from "./Onionshare";
import Onlyoffice from "./Onlyoffice";
import OnlySwitch from "./OnlySwitch";
import OpalComposer from "./OpalComposer";
import Openaudible from "./Openaudible";
import Openboard from "./Openboard";
import Opencloud from "./Opencloud";
import OpencodeDesktop from "./OpencodeDesktop";
import Openinterminal from "./Openinterminal";
import Openlens from "./Openlens";
import Openmtp from "./Openmtp";
import Openrct2 from "./Openrct2";
import Openrefine from "./Openrefine";
import Opentoonz from "./Opentoonz";
import Opera from "./Opera";
import OptimusPlayer from "./OptimusPlayer";
import OrbStack from "./OrbStack";
import OrigamiStudio from "./OrigamiStudio";
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
import Qlab from "./Qlab";
import Qlmarkdown from "./Qlmarkdown";
import QspacePro from "./QspacePro";
import Quip from "./Quip";
import Qview from "./Qview";
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
import SqlServerManagementStudio from "./SqlServerManagementStudio";
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
import UaConnect from "./UaConnect";
import Ukelele from "./Ukelele";
import UltimakerCura from "./UltimakerCura";
import Unclutter from "./Unclutter";
import Unicodechecker from "./Unicodechecker";
import UnityHub from "./UnityHub";
import Updf from "./Updf";
import Upscayl from "./Upscayl";
import UsageApp from "./UsageApp";
import Utm from "./Utm";
import Vanilla from "./Vanilla";
import VcRedistX64 from "./VcRedistX64";
import Vellum from "./Vellum";
import VernierSpectralAnalysis from "./VernierSpectralAnalysis";
import Versions from "./Versions";
import Via from "./Via";
import Vimcal from "./Vimcal";
import VirtualBox from "./VirtualBox";
import VirtualBuddy from "./VirtualBuddy";
import Viscosity from "./Viscosity";
import VisualParadigm from "./VisualParadigm";
import VisualStudioCode from "./VisualStudioCode";
import VividApp from "./VividApp";
import Viz from "./Viz";
import Vlc from "./Vlc";
import VncViewer from "./VncViewer";
import Voiceink from "./Voiceink";
import VpnTracker365 from "./VpnTracker365";
import VsCodium from "./VsCodium";
import Vuescan from "./Vuescan";
import Vyprvpn from "./Vyprvpn";
import Vysor from "./Vysor";
import WacomCenter from "./WacomCenter";
import Warp from "./Warp";
import Wave from "./Wave";
import Wavebox from "./Wavebox";
import Wealthfolio from "./Wealthfolio";
import Weasis from "./Weasis";
import WebStorm from "./WebStorm";
import Webcatalog from "./Webcatalog";
import Webex from "./Webex";
import Wechat from "./Wechat";
import Weektodo from "./Weektodo";
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
import WondershareEdrawmax from "./WondershareEdrawmax";
import WondershareFilmora from "./WondershareFilmora";
import Word from "./Word";
import Wordservice from "./Wordservice";
import Workflowy from "./Workflowy";
import WorksheetCrafter from "./WorksheetCrafter";
import Workspaces from "./Workspaces";
import WrikeForMac from "./WrikeForMac";
import XCreds from "./XCreds";
import Xca from "./Xca";
import Xld from "./Xld";
import Xmenu from "./Xmenu";
import Xmplify from "./Xmplify";
import Xnapper from "./Xnapper";
import Xnconvert from "./Xnconvert";
import Xnviewmp from "./Xnviewmp";
import Xquartz from "./Xquartz";
import Yaak from "./Yaak";
import Yacreader from "./Yacreader";
import Yattee from "./Yattee";
import Yippy from "./Yippy";
import YtMusic from "./YtMusic";
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

import Sabnzbd from "./Sabnzbd";
import SafeExamBrowser from "./SafeExamBrowser";
import Sanesidebuttons from "./Sanesidebuttons";
import ScMenu from "./ScMenu";
import Scratch from "./Scratch";
import Screenflick from "./Screenflick";
import Screenflow from "./Screenflow";
import Screenfocus from "./Screenfocus";
import ScreenStudio from "./ScreenStudio";
import Scribus from "./Scribus";
import Scrivener from "./Scrivener";
import Securesafe from "./Securesafe";
import Selfcontrol from "./Selfcontrol";
import Sensei from "./Sensei";
import Session from "./Session";
import Setapp from "./Setapp";
import Shapr3D from "./Shapr3D";
import Sharefile from "./Sharefile";
import Shift from "./Shift";
import Shifty from "./Shifty";
import Shortcat from "./Shortcat";
import Shotcut from "./Shotcut";
import Sidenotes from "./Sidenotes";
import Sigmaos from "./Sigmaos";
import SimpleComic from "./SimpleComic";
import Sirimote from "./Sirimote";
import Slab from "./Slab";
import Slicer from "./Slicer";
import Slidepad from "./Slidepad";
import Sloth from "./Sloth";
import Smartsheet from "./Smartsheet";
import Smoothscroll from "./Smoothscroll";
import Smultron from "./Smultron";
import Snapmotion from "./Snapmotion";
import SnowflakeSnowsql from "./SnowflakeSnowsql";
import Sococo from "./Sococo";
import SonicVisualiser from "./SonicVisualiser";
import Sonobus from "./Sonobus";
import SonyPsRemotePlay from "./SonyPsRemotePlay";
import Soulver from "./Soulver";
import Soundanchor from "./Soundanchor";
import SoundControl from "./SoundControl";
import SoundSiphon from "./SoundSiphon";
import Soundsource from "./Soundsource";
import Spamsieve from "./Spamsieve";
import SpectraApp from "./SpectraApp";
import SpitfireAudio from "./SpitfireAudio";
import Splice from "./Splice";
import Spyder from "./Spyder";
import Sqlectron from "./Sqlectron";
import SqlproForMssql from "./SqlproForMssql";
import SqlproForMysql from "./SqlproForMysql";
import SqlproForPostgres from "./SqlproForPostgres";
import SqlproForSqlite from "./SqlproForSqlite";
import SqlproStudio from "./SqlproStudio";
import Squash from "./Squash";
import SshConfigEditor from "./SshConfigEditor";
import StandardNotes from "./StandardNotes";
import Staruml from "./Staruml";
import Steermouse from "./Steermouse";
import Stellarium from "./Stellarium";
import Stillcolor from "./Stillcolor";
import Stretchly from "./Stretchly";
import Supercollider from "./Supercollider";
import Superhuman from "./Superhuman";
import Superkey from "./Superkey";
import SuperProductivity from "./SuperProductivity";
import Superwhisper from "./Superwhisper";
import Supportcompanion from "./Supportcompanion";
import Surge from "./Surge";
import Swiftbar from "./Swiftbar";
import Swifty from "./Swifty";
import Swish from "./Swish";
import Sync from "./Sync";
import Syncmate from "./Syncmate";
import Syncovery from "./Syncovery";
import SyncthingApp from "./SyncthingApp";
import SyntaxHighlight from "./SyntaxHighlight";
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
  "bitfocus companion": Companion,
  bitwarden: Bitwarden,
  blender: Blender,
  bluej: Bluej,
  box: Box,
  brave: Brave,
  bruno: Bruno,
  "burp suite community": BurpSuiteCommunity,
  cacher: Cacher,
  caffeine: Caffeine,
  calibre: Calibre,
  "calibrite profiler": CalibriteProfiler,
  "camo studio": CamoStudio,
  camtasia: Camtasia,
  "camunda modeler": CamundaModeler,
  canva: Canva,
  captain: Captain,
  captin: Captin,
  capto: Capto,
  "carbon copy cloner": CarbonCopyCloner,
  cardhop: Cardhop,
  cavalry: Cavalry,
  cellprofiler: Cellprofiler,
  chalk: Chalk,
  charles: Charles,
  charmstone: Charmstone,
  chatgpt: ChatGpt,
  "chatgpt atlas": ChatGptAtlas,
  chatwise: Chatwise,
  cheetah3d: Cheetah3D,
  "cherry studio": CherryStudio,
  chime: Chime,
  choosy: Choosy,
  "chrome remote desktop": ChromeRemoteDesktop,
  "cinc workstation": Cinc,
  "cisco jabber": CiscoJabber,
  "citrix workspace": CitrixWorkspace,
  claude: Claude,
  "claude-devtools": ClaudeDevtools,
  cleanclip: Cleanclip,
  cleanmymac: CleanMyMac,
  cleanmymac_5: CleanMyMac,
  "cleanshot x": CleanShotX,
  clickshare: ClickShare,
  clickup: ClickUp,
  clion: CLion,
  clipbook: Clipbook,
  clipgrab: Clipgrab,
  clipy: Clipy,
  clocker: Clocker,
  "clockify desktop": ClockifyDesktop,
  clop: Clop,
  cloudflare: Cloudflare,
  cmake: CmakeApp,
  cmux: Cmux,
  coconutbattery: Coconutbattery,
  code: VisualStudioCode,
  codeedit: Codeedit,
  coderunner: Coderunner,
  codexbar: Codexbar,
  cog: CogApp,
  "colorsnapper 2": Colorsnapper,
  "colour contrast analyser": ColourContrastAnalyser,
  comet: Comet,
  "command-tab plus": CommandTabPlus,
  commander: Commander,
  "commander one": CommanderOne,
  "company portal": IntuneCompanyPortal,
  "connect fonts": ConnectFonts,
  copilot: CopilotMoney,
  cork: Cork,
  crashplan: CrashPlan,
  crossover: Crossover,
  cryptomator: Cryptomator,
  crystalfetch: Crystalfetch,
  cursorsense: Cursorsense,
  cursr: Cursr,
  customshortcuts: Customshortcuts,
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
  "eltima cloudmounter": Cloudmounter,
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
  "garmin express": GarminExpress,
  "gather town": Gather,
  gdevelop: Gdevelop,
  geany: Geany,
  geekbench: Geekbench,
  gephi: Gephi,
  gitfinder: Gitfinder,
  "github copilot for xcode": GithubCopilotForXcode,
  gitify: Gitify,
  gitup: GitupApp,
  glyphs: Glyphs,
  go2shell: Go2Shell,
  "godot engine": Godot,
  godspeed: Godspeed,
  "gog galaxy": GogGalaxy,
  goodsync: Goodsync,
  "google earth pro": GoogleEarthPro,
  gpodder: Gpodder,
  grandperspective: Grandperspective,
  grids: Grids,
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
  "nota gyazo gif": Gyazo,
  "notepad++": Notepad,
  "notion calendar": NotionCalendar,
  notion: Notion,
  nova: Nova,
  nudge: Nudge,
  obs: Obs,
  obsidian: Obsidian,
  ocenaudio: Ocenaudio,
  "ok json": OkJson,
  "okta verify": OktaVerify,
  ollama: Ollama,
  omnidisksweeper: Omnidisksweeper,
  omnifocus: Omnifocus,
  omnigraffle: OmniGraffle,
  omnioutliner: Omnioutliner,
  omniplan: Omniplan,
  "omnissa horizon client": OmnissaHorizonClient,
  "one switch": OneSwitch,
  onedrive: OneDrive,
  onionshare: Onionshare,
  onlyoffice: Onlyoffice,
  onlyswitch: OnlySwitch,
  "opal composer": OpalComposer,
  openaudible: Openaudible,
  openboard: Openboard,
  "opencloud desktop": Opencloud,
  opencode: OpencodeDesktop,
  openinterminal: Openinterminal,
  openlens: Openlens,
  openmtp: Openmtp,
  openrct2: Openrct2,
  openrefine: Openrefine,
  opentoonz: Opentoonz,
  "openvpn connect": OpenvpnConnect,
  opera: Opera,
  "optimus player": OptimusPlayer,
  orbstack: OrbStack,
  "origami studio": OrigamiStudio,
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
  qlab: Qlab,
  "qspace pro": QspacePro,
  quip: Quip,
  qview: Qview,
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
  "sbarex qlmarkdown": Qlmarkdown,
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
  "sql server management studio": SqlServerManagementStudio,
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
  "ua connect": UaConnect,
  ukelele: Ukelele,
  "ultimaker cura": UltimakerCura,
  unclutter: Unclutter,
  unicodechecker: Unicodechecker,
  "unity hub": UnityHub,
  updf: Updf,
  upscayl: Upscayl,
  usage: UsageApp,
  utm: Utm,
  vanilla: Vanilla,
  vellum: Vellum,
  "vernier spectral analysis": VernierSpectralAnalysis,
  versions: Versions,
  via: Via,
  vimcal: Vimcal,
  virtualbox: VirtualBox,
  virtualbuddy: VirtualBuddy,
  viscosity: Viscosity,
  "visual paradigm": VisualParadigm,
  "visual studio code": VisualStudioCode,
  vivid: VividApp,
  viz: Viz,
  vlc: Vlc,
  "vnc viewer": VncViewer,
  voiceink: Voiceink,
  "vpn tracker 365": VpnTracker365,
  vscodium: VsCodium,
  vuescan: Vuescan,
  vyprvpn: Vyprvpn,
  vysor: Vysor,
  "wacom center": WacomCenter,
  "wacom tablet": WacomCenter,
  warp: Warp,
  "wave terminal": Wave,
  wavebox: Wavebox,
  wealthfolio: Wealthfolio,
  weasis: Weasis,
  webstorm: WebStorm,
  webcatalog: Webcatalog,
  webex: Webex,
  "wechat for mac": Wechat,
  weektodo: Weektodo,
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
  edrawmax: WondershareEdrawmax,
  "wondershare filmora": WondershareFilmora,
  wordservice: Wordservice,
  workflowy: Workflowy,
  "worksheet crafter": WorksheetCrafter,
  workspaces: Workspaces,
  "wrike for mac": WrikeForMac,
  wrike: WrikeForMac,
  xcreds: XCreds,
  xca: Xca,
  "x lossless decoder": Xld,
  xmenu: Xmenu,
  xmplify: Xmplify,
  xnapper: Xnapper,
  "xnsoft xnconvert": Xnconvert,
  xnviewmp: Xnviewmp,
  xquartz: Xquartz,
  yaak: Yaak,
  yacreader: Yacreader,
  yattee: Yattee,
  yippy: Yippy,
  "youtube music": YtMusic,
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
  "3d slicer": Slicer,
  "ps remote play": SonyPsRemotePlay,
  sabnzbd: Sabnzbd,
  "safe exam browser": SafeExamBrowser,
  sanesidebuttons: Sanesidebuttons,
  "sc menu": ScMenu,
  scratch: Scratch,
  "screen studio": ScreenStudio,
  screenflick: Screenflick,
  screenflow: Screenflow,
  screenfocus: Screenfocus,
  scribus: Scribus,
  scrivener: Scrivener,
  securesafe: Securesafe,
  selfcontrol: Selfcontrol,
  sensei: Sensei,
  session: Session,
  setapp: Setapp,
  shapr3d: Shapr3D,
  sharefile: Sharefile,
  shift: Shift,
  shifty: Shifty,
  shotcut: Shotcut,
  sidenotes: Sidenotes,
  sigmaos: Sigmaos,
  "simple comic": SimpleComic,
  sirimote: Sirimote,
  slab: Slab,
  slidepad: Slidepad,
  sloth: Sloth,
  smartsheet: Smartsheet,
  smoothscroll: Smoothscroll,
  smultron: Smultron,
  snapmotion: Snapmotion,
  snowsql: SnowflakeSnowsql,
  sococo: Sococo,
  "sonic visualiser": SonicVisualiser,
  sonobus: Sonobus,
  soulver: Soulver,
  "sound control": SoundControl,
  soundanchor: Soundanchor,
  soundsiphon: SoundSiphon,
  soundsource: Soundsource,
  spamsieve: Spamsieve,
  spectra: SpectraApp,
  "spitfire audio": SpitfireAudio,
  splice: Splice,
  "sproutcube shortcat": Shortcat,
  spyder: Spyder,
  sqlectron: Sqlectron,
  "sqlpro for mssql": SqlproForMssql,
  "sqlpro for mysql": SqlproForMysql,
  "sqlpro for postgres": SqlproForPostgres,
  "sqlpro for sqlite": SqlproForSqlite,
  "sqlpro studio": SqlproStudio,
  squash: Squash,
  "ssh config editor": SshConfigEditor,
  "standard notes": StandardNotes,
  staruml: Staruml,
  steermouse: Steermouse,
  stellarium: Stellarium,
  stillcolor: Stillcolor,
  stretchly: Stretchly,
  "super productivity": SuperProductivity,
  supercollider: Supercollider,
  superhuman: Superhuman,
  superkey: Superkey,
  superwhisper: Superwhisper,
  "support companion": Supportcompanion,
  surge: Surge,
  swiftbar: Swiftbar,
  swifty: Swifty,
  swish: Swish,
  sync: Sync,
  syncmate: Syncmate,
  syncovery: Syncovery,
  syncthing: SyncthingApp,
  "syntax highlight": SyntaxHighlight,
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
