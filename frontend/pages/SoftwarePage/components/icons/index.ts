// Note: if parts of a icon have a clip path, mask, or gradient, the IDs must be unique
// across all icons to avoid conflicts in the DOM. See uniqueId usage within icon components.

import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";
import { matchLoosePrefixToKey } from "utilities/strings/stringUtils";

import AbletonLive12Suite from "./AbletonLive12Suite";
import Abstract from "./Abstract";
import AcrobatReader from "./AcrobatReader";
import AdobeDigitalEditions45 from "./AdobeDigitalEditions45";
import AdobeDngConverter from "./AdobeDngConverter";
import Affinity from "./Affinity";
import Aircall from "./Aircall";
import Airtame from "./Airtame";
import AmazonChime from "./AmazonChime";
import AmazonCorretto21 from "./AmazonCorretto21";
import AmazonCorretto24 from "./AmazonCorretto24";
import AmazonCorretto25 from "./AmazonCorretto25";
import AmazonCorretto26 from "./AmazonCorretto26";
import AmazonDCV from "./AmazonDCV";
import AmazonWorkspaces from "./AmazonWorkspaces";
import AndroidApp from "./AndroidApp";
import AndroidOS from "./AndroidOS";
import AndroidPlayStore from "./AndroidPlayStore";
import AndroidStudio from "./AndroidStudio";
import Anka from "./Anka";
import AnotherRedisDesktopManager from "./AnotherRedisDesktopManager";
import Antigravity from "./Antigravity";
import AntigravityIde from "./AntigravityIde";
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
import AzulZulu25Jdk from "./AzulZulu25Jdk";
import AzulZulu25Jre from "./AzulZulu25Jre";
import Backblaze from "./Backblaze";
import BalenaEtcher from "./BalenaEtcher";
import BBEdit from "./BBEdit";
import BeekeeperStudio from "./BeekeeperStudio";
import BetterDisplay from "./BetterDisplay";
import BeyondCompare from "./BeyondCompare";
import Bitwarden from "./Bitwarden";
import Blender from "./Blender";
import Bluej from "./Bluej";
import Box from "./Box";
import Brave from "./Brave";
import Bruno from "./Bruno";
import BurpSuiteCommunity from "./BurpSuiteCommunity";
import Cacher from "./Cacher";
import Caffeine from "./Caffeine";
import Calibre from "./Calibre";
import CalibriteProfiler from "./CalibriteProfiler";
import CamoStudio from "./CamoStudio";
import Camtasia from "./Camtasia";
import CamundaModeler from "./CamundaModeler";
import Canva from "./Canva";
import CapCut from "./CapCut";
import Captain from "./Captain";
import Captin from "./Captin";
import Capto from "./Capto";
import CarbonCopyCloner from "./CarbonCopyCloner";
import Cardhop from "./Cardhop";
import Cavalry from "./Cavalry";
import Cellprofiler from "./Cellprofiler";
import Chalk from "./Chalk";
import Charles from "./Charles";
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
import ChromeRemoteDesktop from "./ChromeRemoteDesktop";
import Cinc from "./Cinc";
import CiscoJabber from "./CiscoJabber";
import CitrixWorkspace from "./CitrixWorkspace";
import Claude from "./Claude";
import ClaudeDevtools from "./ClaudeDevtools";
import Cleanclip from "./Cleanclip";
import CleanMyMac from "./CleanMyMac";
import CleanShotX from "./CleanShotX";
import ClickShare from "./ClickShare";
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
import Comet from "./Comet";
import Commander from "./Commander";
import CommanderOne from "./CommanderOne";
import CommandTabPlus from "./CommandTabPlus";
import Companion from "./Companion";
import ConnectFonts from "./ConnectFonts";
import CopilotMoney from "./CopilotMoney";
import Cork from "./Cork";
import CotEditor from "./CotEditor";
import CrashPlan from "./CrashPlan";
import CreativeCloud from "./AdobeCreativeCloud";
import Crossover from "./Crossover";
import Cryptomator from "./Cryptomator";
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
import DellCommandUpdate from "./DellCommandUpdate";
import DellDisplayManager from "./DellDisplayManager";
import DevinDesktop from "./DevinDesktop";
import DfuBlasterPro from "./DfuBlasterPro";
import Dialpad from "./Dialpad";
import Discord from "./Discord";
import DisplayLinkManager from "./DisplayLinkManager";
import Docker from "./Docker";
import Drawio from "./DrawIo";
import Dropbox from "./Dropbox";
import DruvaInSync from "./DruvaInSync";
import DuoDesktop from "./DuoDesktop";
import Eaglefiler from "./Eaglefiler";
import Easydict from "./Easydict";
import Easyfind from "./Easyfind";
import Eclipse from "./Eclipse";
import Edge from "./Edge";
import Egnyte from "./Egnyte";
import EightXEightWork from "./8X8Work";
import Electronmail from "./Electronmail";
import Electrum from "./Electrum";
import Element from "./Element";
import Elephas from "./Elephas";
import ElgatoCameraHub from "./ElgatoCameraHub";
import ElgatoCaptureDeviceUtility from "./ElgatoCaptureDeviceUtility";
import ElgatoControlCenter from "./ElgatoControlCenter";
import ElgatoGameCaptureHd from "./ElgatoGameCaptureHd";
import ElgatoStreamDeck from "./ElgatoStreamDeck";
import ElgatoWaveLink from "./ElgatoWaveLink";
import ElmediaPlayer from "./ElmediaPlayer";
import Emclient from "./Emclient";
import Enpass from "./Enpass";
import EnteAuth from "./EnteAuth";
import EpicGames from "./EpicGames";
import Equinox from "./Equinox";
import Etrecheckpro from "./Etrecheckpro";
import Evernote from "./Evernote";
import Excel from "./Excel";
import Exifcleaner from "./Exifcleaner";
import Exifrenamer from "./Exifrenamer";
import ExpressVpn from "./ExpressVpn";
import Extension from "./Extension";
import Extradock from "./Extradock";
import Falcon from "./Falcon";
import Figma from "./Figma";
import FileMakerPro from "./FileMakerPro";
import Firefox from "./Firefox";
import FleetDesktop from "./FleetDesktop";
import Fork from "./Fork";
import Front from "./Front";
import GarminExpress from "./GarminExpress";
import Gather from "./Gather";
import Gdevelop from "./Gdevelop";
import Geany from "./Geany";
import Geekbench from "./Geekbench";
import Gemini from "./Gemini";
import GenesysCloud from "./GenesysCloud";
import Gephi from "./Gephi";
import Ghostty from "./Ghostty";
import Gimp from "./Gimp";
import Git from "./Git";
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
import GoogleCredentialProviderForWindows from "./GoogleCredentialProviderForWindows";
import GoogleDrive from "./GoogleDrive";
import GoogleEarthPro from "./GoogleEarthPro";
import GoToMeeting from "./GoToMeeting";
import GpgKeychain from "./GpgKeychain";
import Gpodder from "./Gpodder";
import GrammarlyDesktop from "./GrammarlyDesktop";
import Grandperspective from "./Grandperspective";
import Granola from "./Granola";
import Grids from "./Grids";
import GrooveOmniDialer from "./GrooveOmniDialer";
import Gyazo from "./Gyazo";
import Hyper from "./Hyper";
import IbmNotifier from "./IbmNotifier";
import IconComposer from "./IconComposer";
import Iina from "./Iina";
import IMazingProfileEditor from "./IMazingProfileEditor";
import Inkscape from "./Inkscape";
import Insomnia from "./Insomnia";
import IntelliJIdea from "./IntelliJIdea";
import IntelliJIdeaCe from "./IntelliJIdeaCe";
import IntuneCompanyPortal from "./IntuneCompanyPortal";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import ITerm from "./ITerm";
import JabraDirect from "./JabraDirect";
import Jami from "./Jami";
import Jamovi from "./Jamovi";
import Jasp from "./Jasp";
import Jellyfin from "./Jellyfin";
import JetBrainsToolbox from "./JetBrainsToolbox";
import Jiggler from "./Jiggler";
import JitsiMeet from "./JitsiMeet";
import Joplin from "./Joplin";
import JordanbairdIce from "./JordanbairdIce";
import JuliaApp from "./JuliaApp";
import JumpDesktop from "./JumpDesktop";
import Kaleidoscope from "./Kaleidoscope";
import Kap from "./Kap";
import Kdenlive from "./Kdenlive";
import KeePassXc from "./KeePassXc";
import KeeperPasswordManager from "./KeeperPasswordManager";
import Keepingyouawake from "./Keepingyouawake";
import Keeweb from "./Keeweb";
import Keka from "./Keka";
import Keyboardcleantool from "./Keyboardcleantool";
import KeyboardCowboy from "./KeyboardCowboy";
import KeyboardMaestro from "./KeyboardMaestro";
import Keycastr from "./Keycastr";
import Keyclu from "./Keyclu";
import KeystoreExplorer from "./KeystoreExplorer";
import Kiro from "./Kiro";
import KiroCli from "./KiroCli";
import Kitty from "./Kitty";
import Klokki from "./Klokki";
import Knime from "./Knime";
import Knockknock from "./Knockknock";
import Krisp from "./Krisp";
import Krita from "./Krita";
import LastPass from "./LastPass";
import LenovoDockManager from "./LenovoDockManager";
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
import Marvel from "./Marvel";
import Mattermost from "./Mattermost";
import Max from "./Max";
import Microsoft365Copilot from "./Microsoft365Copilot";
import MicrosoftAutoUpdate from "./MicrosoftAutoUpdate";
import MicrosoftDotnetRuntime from "./MicrosoftDotnetRuntime";
import MicrosoftEdge from "./MicrosoftEdge";
import MicrosoftOffice from "./MicrosoftOffice";
import MicrosoftOneNote from "./MicrosoftOneNote";
import MicrosoftOutlook from "./MicrosoftOutlook";
import MicrosoftPowerPoint from "./MicrosoftPowerPoint";
import MicrosoftRemoteHelp from "./MicrosoftRemoteHelp";
import MindManager from "./MindManager";
import Miro from "./Miro";
import MongoDbCompass from "./MongoDbCompass";
import MySqlWorkbench from "./MySqlWorkbench";
import NessusAgent from "./NessusAgent";
import Nextcloud from "./Nextcloud";
import Nodejs from "./Nodejs";
import Nordpass from "./Nordpass";
import NordVpn from "./NordVpn";
import Notepad from "./Notepad++";
import Notion from "./Notion";
import NotionCalendar from "./NotionCalendar";
import Nova from "./Nova";
import Nudge from "./Nudge";
import Obs from "./Obs";
import Obsidian from "./Obsidian";
import Ocenaudio from "./Ocenaudio";
import OkJson from "./OkJson";
import OktaVerify from "./OktaVerify";
import Ollama from "./Ollama";
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
import OpenvpnConnect from "./OpenvpnConnect";
import Opera from "./Opera";
import OptimusPlayer from "./OptimusPlayer";
import OrbStack from "./OrbStack";
import OrigamiStudio from "./OrigamiStudio";
import P4V from "./P4V";
import Package from "./Package";
import ParallelsDesktop from "./ParallelsDesktop";
import Pd from "./Pd";
import PgAdmin4 from "./PgAdmin4";
import PhpStorm from "./PhpStorm";
import PlantronicsHub from "./PlantronicsHub";
import Plugdata from "./Plugdata";
import PodmanDesktop from "./PodmanDesktop";
import Postgresql15 from "./Postgresql15";
import Postgresql16 from "./Postgresql16";
import Postgresql17 from "./Postgresql17";
import Postgresql18 from "./Postgresql18";
import Postman from "./Postman";
import PowerAutomate from "./PowerAutomate";
import PowerBi from "./PowerBi";
import PowerMonitor from "./PowerMonitor";
import Powershell from "./Powershell";
import Powertoys from "./Powertoys";
import Prisma from "./Prisma";
import Pritunl from "./Pritunl";
import Privileges from "./Privileges";
import ProtonMail from "./ProtonMail";
import ProtonVpn from "./ProtonVpn";
import Proxifier from "./Proxifier";
import Proxyman from "./Proxyman";
import Putty from "./Putty";
import PyCharm from "./PyCharm";
import PyCharmCe from "./PyCharmCe";
import Python313 from "./Python313";
import Python314 from "./Python314";
import Qlab from "./Qlab";
import Qlmarkdown from "./Qlmarkdown";
import QspacePro from "./QspacePro";
import Quip from "./Quip";
import Qview from "./Qview";
import R from "./R";
import RancherDesktop from "./RancherDesktop";
import RapidApi from "./RapidApi";
import Raycast from "./Raycast";
import RealVncServer from "./RealVncServer";
import Reaper from "./Reaper";
import Rectangle from "./Rectangle";
import Rider from "./Rider";
import RoyalTsx from "./RoyalTsx";
import Rstudio from "./Rstudio";
import RubyMine from "./RubyMine";
import RustDesk from "./RustDesk";
import RustRover from "./RustRover";
import Sabnzbd from "./Sabnzbd";
import Safari from "./Safari";
import SafeExamBrowser from "./SafeExamBrowser";
import Sanesidebuttons from "./Sanesidebuttons";
import Santa from "./Santa";
import ScMenu from "./ScMenu";
import Scratch from "./Scratch";
import Screenflick from "./Screenflick";
import Screenflow from "./Screenflow";
import Screenfocus from "./Screenfocus";
import ScreenStudio from "./ScreenStudio";
import Scribus from "./Scribus";
import Scrivener from "./Scrivener";
import Secretive from "./Secretive";
import Securesafe from "./Securesafe";
import Selfcontrol from "./Selfcontrol";
import Sensei from "./Sensei";
import SequelAce from "./SequelAce";
import Session from "./Session";
import Setapp from "./Setapp";
import SevenZip from "./7Zip";
import SfSymbols from "./SfSymbols";
import Shapr3D from "./Shapr3D";
import Sharefile from "./Sharefile";
import Shift from "./Shift";
import Shifty from "./Shifty";
import Shortcat from "./Shortcat";
import Shotcut from "./Shotcut";
import Shottr from "./Shottr";
import Sidenotes from "./Sidenotes";
import Sigmaos from "./Sigmaos";
import Signal from "./Signal";
import SimpleComic from "./SimpleComic";
import Sirimote from "./Sirimote";
import Sketch from "./Sketch";
import Slab from "./Slab";
import Slack from "./Slack";
import Slicer from "./Slicer";
import Slidepad from "./Slidepad";
import Sloth from "./Sloth";
import Smartsheet from "./Smartsheet";
import Smoothscroll from "./Smoothscroll";
import Smultron from "./Smultron";
import Snagit from "./Snagit";
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
import Sourcetree from "./Sourcetree";
import Spamsieve from "./Spamsieve";
import SpectraApp from "./SpectraApp";
import SpitfireAudio from "./SpitfireAudio";
import SplashtopBusiness from "./SplashtopBusiness";
import SplashtopStreamer from "./SplashtopStreamer";
import Splice from "./Splice";
import Spotify from "./Spotify";
import Spyder from "./Spyder";
import Sqlectron from "./Sqlectron";
import SqlproForMssql from "./SqlproForMssql";
import SqlproForMysql from "./SqlproForMysql";
import SqlproForPostgres from "./SqlproForPostgres";
import SqlproForSqlite from "./SqlproForSqlite";
import SqlproStudio from "./SqlproStudio";
import SqlServerManagementStudio from "./SqlServerManagementStudio";
import Squash from "./Squash";
import SshConfigEditor from "./SshConfigEditor";
import StandardNotes from "./StandardNotes";
import Staruml from "./Staruml";
import Stats from "./Stats";
import Steam from "./Steam";
import Steermouse from "./Steermouse";
import Stellarium from "./Stellarium";
import Stillcolor from "./Stillcolor";
import Stretchly from "./Stretchly";
import SublimeMerge from "./SublimeMerge";
import SublimeText from "./SublimeText";
import Supercollider from "./Supercollider";
import Superhuman from "./Superhuman";
import Superkey from "./Superkey";
import SuperProductivity from "./SuperProductivity";
import Superwhisper from "./Superwhisper";
import Supportcompanion from "./Supportcompanion";
import Surfshark from "./Surfshark";
import Surge from "./Surge";
import SuspiciousPackage from "./SuspiciousPackage";
import Swiftbar from "./Swiftbar";
import Swiftdialog from "./Swiftdialog";
import Swifty from "./Swifty";
import Swish from "./Swish";
import Sync from "./Sync";
import Syncmate from "./Syncmate";
import Syncovery from "./Syncovery";
import SyncthingApp from "./SyncthingApp";
import SyntaxHighlight from "./SyntaxHighlight";
import TableauDesktop from "./TableauDesktop";
import TablePlus from "./TablePlus";
import Tailscale from "./Tailscale";
import Teams from "./Teams";
import TeamViewer from "./TeamViewer";
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
import Webcatalog from "./Webcatalog";
import Webex from "./Webex";
import WebStorm from "./WebStorm";
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
import Xca from "./Xca";
import XCreds from "./XCreds";
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
// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "010 editor": ZeroOneZeroEditor,
  "1password": OnePassword,
  "3d slicer": Slicer,
  "7 zip": SevenZip,
  "7-zip": SevenZip,
  "8x8 work": EightXEightWork,
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
  "android studio": AndroidStudio,
  androidPlayStore: AndroidPlayStore,
  anka: Anka,
  "another redis desktop manager": AnotherRedisDesktopManager,
  antigravity: Antigravity,
  "antigravity ide": AntigravityIde,
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
  "aws client vpn": AwsVpnClient,
  "aws vpn client": AwsVpnClient,
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
  capcut: CapCut,
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
  coteditor: CotEditor,
  crashplan: CrashPlan,
  crossover: Crossover,
  cryptomator: Cryptomator,
  crystalfetch: Crystalfetch,
  cursor: Cursor,
  cursorsense: Cursorsense,
  cursr: Cursr,
  customshortcuts: Customshortcuts,
  cyberduck: Cyberduck,
  dash: Dash,
  datagrip: DataGrip,
  "db browser for sqlite": DbBrowserForSqLite,
  dbeaver: DBeaver,
  "dbeaver community": DBeaver,
  "dbeaver enterprise edition": DBeaverEe,
  "dbeaver lite edition": DBeaverLite,
  "dbeaver ultimate edition": DBeaverUltimate,
  dbeaveree: DBeaverEe,
  dbeaverlite: DBeaverLite,
  dbeaverultimate: DBeaverUltimate,
  deepl: DeepL,
  "dell command update": DellCommandUpdate,
  "dell display manager": DellDisplayManager,
  "devin desktop": DevinDesktop,
  "dfu blaster pro": DfuBlasterPro,
  dialpad: Dialpad,
  discord: Discord,
  "DisplayLink USB Graphics Software": DisplayLinkManager,
  "dng converter": AdobeDngConverter,
  docker: Docker,
  "draw.io": Drawio,
  dropbox: Dropbox,
  "duo desktop": DuoDesktop,
  eaglefiler: Eaglefiler,
  easydict: Easydict,
  easyfind: Easyfind,
  eclipse: Eclipse,
  edge: MicrosoftEdge,
  edrawmax: WondershareEdrawmax,
  egnyte: Egnyte,
  electronmail: Electronmail,
  electrum: Electrum,
  element: Element,
  elephas: Elephas,
  "elgato camera hub": ElgatoCameraHub,
  "elgato capture device utility": ElgatoCaptureDeviceUtility,
  "elgato control center": ElgatoControlCenter,
  "elgato game capture hd": ElgatoGameCaptureHd,
  "elgato stream deck": ElgatoStreamDeck,
  "elgato wave link": ElgatoWaveLink,
  "elmedia player": ElmediaPlayer,
  "eltima cloudmounter": Cloudmounter,
  "em client": Emclient,
  enpass: Enpass,
  "ente auth": EnteAuth,
  "epic games launcher": EpicGames,
  equinox: Equinox,
  etrecheck: Etrecheckpro,
  evernote: Evernote,
  exifcleaner: Exifcleaner,
  exifrenamer: Exifrenamer,
  expressvpn: ExpressVpn,
  extradock: Extradock,
  falcon: Falcon,
  figma: Figma,
  "filemaker pro": FileMakerPro,
  firefox: Firefox,
  "fleet desktop": FleetDesktop,
  fork: Fork,
  front: Front,
  "garmin express": GarminExpress,
  "gather town": Gather,
  gdevelop: Gdevelop,
  geany: Geany,
  geekbench: Geekbench,
  gemini: Gemini,
  "genesys cloud": GenesysCloud,
  gephi: Gephi,
  ghostty: Ghostty,
  gimp: Gimp,
  git: Git,
  gitfinder: Gitfinder,
  "github copilot for xcode": GithubCopilotForXcode,
  "github desktop": GitHubDesktop,
  gitify: Gitify,
  gitkraken: GitKraken,
  gitup: GitupApp,
  glyphs: Glyphs,
  go2shell: Go2Shell,
  "godot engine": Godot,
  godspeed: Godspeed,
  "gog galaxy": GogGalaxy,
  goland: GoLand,
  goodsync: Goodsync,
  "google antigravity": Antigravity,
  "google antigravity ide": AntigravityIde,
  "google chrome": ChromeApp,
  "google credential provider for windows": GoogleCredentialProviderForWindows,
  "google drive": GoogleDrive,
  "google earth pro": GoogleEarthPro,
  gotomeeting: GoToMeeting,
  "gpg keychain": GpgKeychain,
  "gpg suite": GpgKeychain,
  gpodder: Gpodder,
  grammarly: GrammarlyDesktop,
  grandperspective: Grandperspective,
  granola: Granola,
  grids: Grids,
  "groove omnidialer": GrooveOmniDialer,
  hyper: Hyper,
  "ibm notifier": IbmNotifier,
  ice: JordanbairdIce,
  "icon composer": IconComposer,
  iina: Iina,
  imazing: IMazingProfileEditor,
  "imazing profile editor": IMazingProfileEditor,
  inkscape: Inkscape,
  insomnia: Insomnia,
  insyncclient: DruvaInSync,
  "intellij idea": IntelliJIdea,
  "intellij idea ce": IntelliJIdeaCe,
  iterm2: ITerm,
  "jabra direct": JabraDirect,
  jami: Jami,
  jamovi: Jamovi,
  jasp: Jasp,
  jellyfin: Jellyfin,
  "jetbrains toolbox": JetBrainsToolbox,
  jiggler: Jiggler,
  "jitsi meet": JitsiMeet,
  joplin: Joplin,
  julia: JuliaApp,
  "jump desktop": JumpDesktop,
  kaleidoscope: Kaleidoscope,
  kap: Kap,
  kdenlive: Kdenlive,
  keepassxc: KeePassXc,
  "keeper password manager": KeeperPasswordManager,
  keepingyouawake: Keepingyouawake,
  keeweb: Keeweb,
  keka: Keka,
  "keyboard cowboy": KeyboardCowboy,
  "keyboard maestro": KeyboardMaestro,
  keyboardcleantool: Keyboardcleantool,
  keycastr: Keycastr,
  keyclu: Keyclu,
  "keystore explorer": KeystoreExplorer,
  kiro: Kiro,
  "kiro cli": KiroCli,
  kitty: Kitty,
  klokki: Klokki,
  "knime analytics platform": Knime,
  knockknock: Knockknock,
  krisp: Krisp,
  krita: Krita,
  lastpass: LastPass,
  "lenovo dock manager": LenovoDockManager,
  lens: Lens,
  libreoffice: LibreOffice,
  linear: Linear,
  "little snitch": LittleSnitch,
  "logi options+": Logioptionsplus,
  loom: Loom,
  lulu: LuLu,
  maccy: Maccy,
  marvel: Marvel,
  mattermost: Mattermost,
  max: Max,
  "microsoft .net runtime": MicrosoftDotnetRuntime,
  "microsoft 365 copilot": Microsoft365Copilot,
  "microsoft auto update": MicrosoftAutoUpdate,
  "microsoft autoupdate": MicrosoftAutoUpdate,
  "microsoft edge": Edge,
  "microsoft excel": Excel,
  "microsoft office": MicrosoftOffice,
  "microsoft onenote": MicrosoftOneNote,
  "microsoft outlook": MicrosoftOutlook,
  "microsoft powerpoint": MicrosoftPowerPoint,
  "microsoft remote help": MicrosoftRemoteHelp,
  "microsoft teams": Teams,
  "microsoft visual c++": VcRedistX64,
  "microsoft visual studio code": VisualStudioCode,
  "microsoft word": Word,
  "microsoft.companyportal": IntuneCompanyPortal,
  mindmanager: MindManager,
  miro: Miro,
  "mongodb compass": MongoDbCompass,
  "mozilla firefox": Firefox,
  "mysql workbench": MySqlWorkbench,
  "nessus agent": NessusAgent,
  nextcloud: Nextcloud,
  "node.js": Nodejs,
  "nord vpn": NordVpn,
  nordpass: Nordpass,
  nordvpn: NordVpn,
  "nota gyazo gif": Gyazo,
  "notepad++": Notepad,
  notion: Notion,
  "notion calendar": NotionCalendar,
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
  p4v: P4V,
  package: Package,
  "parallels desktop": ParallelsDesktop,
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
  pritunl: Pritunl,
  privileges: Privileges,
  "proton mail": ProtonMail,
  protonvpn: ProtonVpn,
  proxifier: Proxifier,
  proxyman: Proxyman,
  "ps remote play": SonyPsRemotePlay,
  putty: Putty,
  pycharm: PyCharm,
  "pycharm ce": PyCharmCe,
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
  sabnzbd: Sabnzbd,
  safari: Safari,
  "safe exam browser": SafeExamBrowser,
  sanesidebuttons: Sanesidebuttons,
  santa: Santa,
  "sbarex qlmarkdown": Qlmarkdown,
  "sc menu": ScMenu,
  scratch: Scratch,
  "screen studio": ScreenStudio,
  screenflick: Screenflick,
  screenflow: Screenflow,
  screenfocus: Screenfocus,
  scribus: Scribus,
  scrivener: Scrivener,
  secretive: Secretive,
  securesafe: Securesafe,
  selfcontrol: Selfcontrol,
  sensei: Sensei,
  "sequel ace": SequelAce,
  session: Session,
  setapp: Setapp,
  "sf symbols": SfSymbols,
  shapr3d: Shapr3D,
  sharefile: Sharefile,
  shift: Shift,
  shifty: Shifty,
  shotcut: Shotcut,
  shottr: Shottr,
  sidenotes: Sidenotes,
  sigmaos: Sigmaos,
  signal: Signal,
  "simple comic": SimpleComic,
  sirimote: Sirimote,
  sketch: Sketch,
  slab: Slab,
  slack: Slack,
  slidepad: Slidepad,
  sloth: Sloth,
  smartsheet: Smartsheet,
  smoothscroll: Smoothscroll,
  smultron: Smultron,
  snagit: Snagit,
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
  sourcetree: Sourcetree,
  spamsieve: Spamsieve,
  spectra: SpectraApp,
  "spitfire audio": SpitfireAudio,
  "splashtop business": SplashtopBusiness,
  "splashtop streamer": SplashtopStreamer,
  splice: Splice,
  spotify: Spotify,
  "sproutcube shortcat": Shortcat,
  spyder: Spyder,
  "sql server management studio": SqlServerManagementStudio,
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
  stats: Stats,
  steam: Steam,
  steermouse: Steermouse,
  stellarium: Stellarium,
  stillcolor: Stillcolor,
  "stream deck": ElgatoStreamDeck,
  stretchly: Stretchly,
  "sublime merge": SublimeMerge,
  "sublime text": SublimeText,
  "super productivity": SuperProductivity,
  supercollider: Supercollider,
  superhuman: Superhuman,
  superkey: Superkey,
  superwhisper: Superwhisper,
  "support companion": Supportcompanion,
  surfshark: Surfshark,
  surge: Surge,
  "suspicious package": SuspiciousPackage,
  swiftbar: Swiftbar,
  swiftdialog: Swiftdialog,
  swifty: Swifty,
  swish: Swish,
  sync: Sync,
  syncmate: Syncmate,
  syncovery: Syncovery,
  syncthing: SyncthingApp,
  "syntax highlight": SyntaxHighlight,
  tableau: TableauDesktop,
  tableplus: TablePlus,
  tailscale: Tailscale,
  teamviewer: TeamViewer,
  telegram: Telegram,
  teleport: TeleportConnect,
  "teleport connect": TeleportConnect,
  "teleport suite": TeleportConnect,
  terminal: Terminal,
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
  webcatalog: Webcatalog,
  webex: Webex,
  webstorm: WebStorm,
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
  "wondershare filmora": WondershareFilmora,
  wordservice: Wordservice,
  workflowy: Workflowy,
  "worksheet crafter": WorksheetCrafter,
  workspaces: Workspaces,
  wrike: WrikeForMac,
  "wrike for mac": WrikeForMac,
  "x lossless decoder": Xld,
  xca: Xca,
  xcreds: XCreds,
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
