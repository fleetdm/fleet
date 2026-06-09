// Note: if parts of a icon have a clip path, mask, or gradient, the IDs must be unique
// across all icons to avoid conflicts in the DOM. See uniqueId usage within icon components.

import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";
import { matchLoosePrefixToKey } from "utilities/strings/stringUtils";

import FourKSlideshowMaker from "./FourKSlideshowMaker";
import FourKStogram from "./FourKStogram";
import FourKVideoDownloader from "./FourKVideoDownloader";
import FourKVideoToMp3 from "./FourKVideoToMp3";
import FourKYoutubeToMp3 from "./FourKYoutubeToMp3";
import ABetterFinderRename from "./ABetterFinderRename";
import AbletonLive12Suite from "./AbletonLive12Suite";
import Acorn from "./Acorn";
import Activedock from "./Activedock";
import Activitywatch from "./Activitywatch";
import Actual from "./Actual";
import Adguard from "./Adguard";
import Adlock from "./Adlock";
import AdvancedRenamer from "./AdvancedRenamer";
import Affinity from "./Affinity";
import AffinityDesigner from "./AffinityDesigner";
import AffinityDesigner1 from "./AffinityDesigner1";
import AffinityPhoto from "./AffinityPhoto";
import AffinityPhoto1 from "./AffinityPhoto1";
import AffinityPublisher from "./AffinityPublisher";
import AffinityPublisher1 from "./AffinityPublisher1";
import Airbuddy from "./Airbuddy";
import Airdroid from "./Airdroid";
import Airparrot from "./Airparrot";
import Airserver from "./Airserver";
import Airtable from "./Airtable";
import Airy from "./Airy";
import Akiflow from "./Akiflow";
import Alcove from "./Alcove";
import Aldente from "./Aldente";
import Alloy from "./Alloy";
import AltairGraphqlClient from "./AltairGraphqlClient";
import AltTab from "./AltTab";
import AmadeusPro from "./AmadeusPro";
import Amadine from "./Amadine";
import AmazonCorretto21 from "./AmazonCorretto21";
import AmazonCorretto24 from "./AmazonCorretto24";
import AmazonCorretto25 from "./AmazonCorretto25";
import AmazonCorretto26 from "./AmazonCorretto26";
import AmazonWorkspaces from "./AmazonWorkspaces";
import Amethyst from "./Amethyst";
import Amie from "./Amie";
import AngryIpScanner from "./AngryIpScanner";
import AnotherRedisDesktopManager from "./AnotherRedisDesktopManager";
import Antigravity from "./Antigravity";
import AntigravityIde from "./AntigravityIde";
import Antinote from "./Antinote";
import Anydo from "./Anydo";
import Anytype from "./Anytype";
import Apidog from "./Apidog";
import AppFair from "./AppFair";
import AppiumInspector from "./AppiumInspector";
import Applite from "./Applite";
import AssetCatalogTinkerer from "./AssetCatalogTinkerer";
import Atext from "./Atext";
import AudioHijack from "./AudioHijack";
import AviatrixVpnClient from "./AviatrixVpnClient";
import AxureRp from "./AxureRp";
import AzulZulu25Jdk from "./AzulZulu25Jdk";
import AzulZulu25Jre from "./AzulZulu25Jre";
import Backblaze from "./Backblaze";
import BackgroundMusic from "./BackgroundMusic";
import Badgeify from "./Badgeify";
import BalsamiqWireframes from "./BalsamiqWireframes";
import BambuStudio from "./BambuStudio";
import Bartender from "./Bartender";
import Batfi from "./Batfi";
import Bdash from "./Bdash";
import BeaverNotes from "./BeaverNotes";
import BeekeeperStudio from "./BeekeeperStudio";
import Beeper from "./Beeper";
import BetterDisplay from "./BetterDisplay";
import Bettermouse from "./Bettermouse";
import Bettertouchtool from "./Bettertouchtool";
import Betterzip from "./Betterzip";
import Bezel from "./Bezel";
import Bibdesk from "./Bibdesk";
import Binance from "./Binance";
import Biscuit from "./Biscuit";
import Bitbox from "./Bitbox";
import Bitrix24 from "./Bitrix24";
import BitwigStudio from "./BitwigStudio";
import Bleunlock from "./Bleunlock";
import Blip from "./Blip";
import Bluej from "./Bluej";
import Bluewallet from "./Bluewallet";
import Blurscreen from "./Blurscreen";
import Boltai from "./Boltai";
import BomeNetwork from "./BomeNetwork";
import Boom3D from "./Boom3D";
import Boop from "./Boop";
import BoostNote from "./BoostNote";
import Breaktimer from "./Breaktimer";
import BricklinkStudio from "./BricklinkStudio";
import Bunch from "./Bunch";
import BurpSuiteCommunity from "./BurpSuiteCommunity";
import Busycontacts from "./Busycontacts";
import Buttercup from "./Buttercup";
import Buzz from "./Buzz";
import Cacher from "./Cacher";
import Caffeine from "./Caffeine";
import CalibriteProfiler from "./CalibriteProfiler";
import CamoStudio from "./CamoStudio";
import CamundaModeler from "./CamundaModeler";
import Capacities from "./Capacities";
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
import Chatwise from "./Chatwise";
import Chatwork from "./Chatwork";
import Cheetah3D from "./Cheetah3D";
import CherryStudio from "./CherryStudio";
import Chime from "./Chime";
import Choosy from "./Choosy";
import ChromeRemoteDesktop from "./ChromeRemoteDesktop";
import Cinc from "./Cinc";
import ClaudeDevtools from "./ClaudeDevtools";
import Cleanclip from "./Cleanclip";
import ClickShare from "./ClickShare";
import Clipbook from "./Clipbook";
import Clipgrab from "./Clipgrab";
import Clipy from "./Clipy";
import Clocker from "./Clocker";
import Clop from "./Clop";
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
import CrashPlan from "./CrashPlan";
import Crossover from "./Crossover";
import Cryptomator from "./Cryptomator";
import Crystalfetch from "./Crystalfetch";
import Cursorsense from "./Cursorsense";
import Cursr from "./Cursr";
import Customshortcuts from "./Customshortcuts";
import Daisydisk from "./Daisydisk";
import Dangerzone from "./Dangerzone";
import Darkmodebuddy from "./Darkmodebuddy";
import Darktable from "./Darktable";
import Dataflare from "./Dataflare";
import Dataspell from "./Dataspell";
import Dayflow from "./Dayflow";
import Dbgate from "./Dbgate";
import Dbvisualizer from "./Dbvisualizer";
import Debookee from "./Debookee";
import Deckset from "./Deckset";
import Deezer from "./Deezer";
import DefaultFolderX from "./DefaultFolderX";
import DellCommandUpdate from "./DellCommandUpdate";
import DellDisplayManager from "./DellDisplayManager";
import Descript from "./Descript";
import Deskpad from "./Deskpad";
import Desktime from "./Desktime";
import DevinDesktop from "./DevinDesktop";
import Devknife from "./Devknife";
import DevonsphereExpress from "./DevonsphereExpress";
import Devonthink from "./Devonthink";
import Devtoys from "./Devtoys";
import Devutils from "./Devutils";
import DfuBlasterPro from "./DfuBlasterPro";
import Dictionaries from "./Dictionaries";
import Diffusionbee from "./Diffusionbee";
import Digikam from "./Digikam";
import DiskDrill from "./DiskDrill";
import Dockdoor from "./Dockdoor";
import Dockfix from "./Dockfix";
import Dockside from "./Dockside";
import Dockview from "./Dockview";
import Dot from "./Dot";
import Doughnut from "./Doughnut";
import Downie from "./Downie";
import DrataAgent from "./DrataAgent";
import Drawbot from "./Drawbot";
import Dropdmg from "./Dropdmg";
import Droplr from "./Droplr";
import Dropshare from "./Dropshare";
import Dropzone from "./Dropzone";
import DruvaInSync from "./DruvaInSync";
import Duckduckgo from "./Duckduckgo";
import Duet from "./Duet";
import DuoDesktop from "./DuoDesktop";
import Dupeguru from "./Dupeguru";
import DymoConnect from "./DymoConnect";
import Dynalist from "./Dynalist";
import Eaglefiler from "./Eaglefiler";
import Easydict from "./Easydict";
import Easyfind from "./Easyfind";
import Electronmail from "./Electronmail";
import Electrum from "./Electrum";
import Element from "./Element";
import Elephas from "./Elephas";
import ElgatoCameraHub from "./ElgatoCameraHub";
import ElgatoCaptureDeviceUtility from "./ElgatoCaptureDeviceUtility";
import ElgatoGameCaptureHd from "./ElgatoGameCaptureHd";
import ElgatoWaveLink from "./ElgatoWaveLink";
import ElmediaPlayer from "./ElmediaPlayer";
import Emclient from "./Emclient";
import Enpass from "./Enpass";
import EnteAuth from "./EnteAuth";
import EpicGames from "./EpicGames";
import Equinox from "./Equinox";
import Etrecheckpro from "./Etrecheckpro";
import Exifcleaner from "./Exifcleaner";
import Exifrenamer from "./Exifrenamer";
import Extradock from "./Extradock";
import Fantastical from "./Fantastical";
import Far2L from "./Far2L";
import Farrago from "./Farrago";
import Fastmail from "./Fastmail";
import Fastscripts from "./Fastscripts";
import Fellow from "./Fellow";
import Ferdium from "./Ferdium";
import FetchApp from "./FetchApp";
import Fig from "./Fig";
import FileJuicer from "./FileJuicer";
import Filen from "./Filen";
import Fing from "./Fing";
import Firealpaca from "./Firealpaca";
import FireflyIotaDesktop from "./FireflyIotaDesktop";
import FireflyShimmer from "./FireflyShimmer";
import Fission from "./Fission";
import FleetDesktop from "./FleetDesktop";
import Flexoptix from "./Flexoptix";
import Flowvision from "./Flowvision";
import Fluid from "./Fluid";
import FluxApp from "./FluxApp";
import FocusriteControl2 from "./FocusriteControl2";
import Folx from "./Folx";
import Fontbase from "./Fontbase";
import Fontlab from "./Fontlab";
import Forecast from "./Forecast";
import Forklift from "./Forklift";
import Framer from "./Framer";
import Franz from "./Franz";
import FreeDownloadManager from "./FreeDownloadManager";
import Freefilesync from "./Freefilesync";
import Fsmonitor from "./Fsmonitor";
import Funter from "./Funter";
import GarminExpress from "./GarminExpress";
import Gather from "./Gather";
import Gdevelop from "./Gdevelop";
import Geany from "./Geany";
import Geekbench from "./Geekbench";
import Gemini from "./Gemini";
import GenesysCloud from "./GenesysCloud";
import Gephi from "./Gephi";
import Git from "./Git";
import Gitfinder from "./Gitfinder";
import GithubCopilotForXcode from "./GithubCopilotForXcode";
import Gitify from "./Gitify";
import GitupApp from "./GitupApp";
import Glyphs from "./Glyphs";
import Go2Shell from "./Go2Shell";
import Godot from "./Godot";
import Godspeed from "./Godspeed";
import GogGalaxy from "./GogGalaxy";
import Goodsync from "./Goodsync";
import GoogleCredentialProviderForWindows from "./GoogleCredentialProviderForWindows";
import GoogleEarthPro from "./GoogleEarthPro";
import GoToMeeting from "./GoToMeeting";
import Gpodder from "./Gpodder";
import Grandperspective from "./Grandperspective";
import Grids from "./Grids";
import GrooveOmniDialer from "./GrooveOmniDialer";
import Gyazo from "./Gyazo";
import Hammerspoon from "./Hammerspoon";
import HandbrakeApp from "./HandbrakeApp";
import Hazel from "./Hazel";
import Hazeover from "./Hazeover";
import Helium from "./Helium";
import HexFiend from "./HexFiend";
import HeyDesktop from "./HeyDesktop";
import Hiddenbar from "./Hiddenbar";
import Hides from "./Hides";
import Hidock from "./Hidock";
import HighlightAi from "./HighlightAi";
import HiveApp from "./HiveApp";
import HomeAssistant from "./HomeAssistant";
import Homerow from "./Homerow";
import Hot from "./Hot";
import Houdahspot from "./Houdahspot";
import HpEasyAdmin from "./HpEasyAdmin";
import Hubstaff from "./Hubstaff";
import Huly from "./Huly";
import Hyperkey from "./Hyperkey";
import I1Profiler from "./I1Profiler";
import IbmAsperaConnect from "./IbmAsperaConnect";
import IbmNotifier from "./IbmNotifier";
import IconComposer from "./IconComposer";
import Iconjar from "./Iconjar";
import Idagio from "./Idagio";
import Iexplorer from "./Iexplorer";
import Iina from "./Iina";
import ImazingConverter from "./ImazingConverter";
import Imhex from "./Imhex";
import InputSourcePro from "./InputSourcePro";
import Intellidock from "./Intellidock";
import Invesalius from "./Invesalius";
import Istherenet from "./Istherenet";
import Itsycal from "./Itsycal";
import Jami from "./Jami";
import Jamovi from "./Jamovi";
import Jasp from "./Jasp";
import Jellyfin from "./Jellyfin";
import Jiggler from "./Jiggler";
import JitsiMeet from "./JitsiMeet";
import Joplin from "./Joplin";
import JordanbairdIce from "./JordanbairdIce";
import JuliaApp from "./JuliaApp";
import JumpDesktop from "./JumpDesktop";
import Kaleidoscope from "./Kaleidoscope";
import Kap from "./Kap";
import Kdenlive from "./Kdenlive";
import Keepingyouawake from "./Keepingyouawake";
import Keeweb from "./Keeweb";
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
import Lapce from "./Lapce";
import LassoApp from "./LassoApp";
import LastPass from "./LastPass";
import LastWindowQuits from "./LastWindowQuits";
import Latest from "./Latest";
import Launchbar from "./Launchbar";
import LenovoDockManager from "./LenovoDockManager";
import Lightburn from "./Lightburn";
import Linearmouse from "./Linearmouse";
import LingonX from "./LingonX";
import Local from "./Local";
import Localsend from "./Localsend";
import Locationsimulator from "./Locationsimulator";
import Logseq from "./Logseq";
import Lookaway from "./Lookaway";
import Loop from "./Loop";
import Loopback from "./Loopback";
import LoRain from "./LoRain";
import Losslesscut from "./Losslesscut";
import LowProfile from "./LowProfile";
import Lunacy from "./Lunacy";
import Lunar from "./Lunar";
import Lunasea from "./Lunasea";
import Lunatask from "./Lunatask";
import Lycheeslicer from "./Lycheeslicer";
import Macdown from "./Macdown";
import Mace from "./Mace";
import Macjournal from "./Macjournal";
import MacMouseFix from "./MacMouseFix";
import Macpacker from "./Macpacker";
import Macpass from "./Macpass";
import Macpilot from "./Macpilot";
import MacsFanControl from "./MacsFanControl";
import Macsyzones from "./Macsyzones";
import Mactracker from "./Mactracker";
import MacvimApp from "./MacvimApp";
import Macwhisper from "./Macwhisper";
import Maestral from "./Maestral";
import Magicquit from "./Magicquit";
import Mailspring from "./Mailspring";
import Malwarebytes from "./Malwarebytes";
import MarkedApp from "./MarkedApp";
import Markedit from "./Markedit";
import MarkText from "./MarkText";
import Marsedit from "./Marsedit";
import Marta from "./Marta";
import Marvel from "./Marvel";
import Masscode from "./Masscode";
import Meetingbar from "./Meetingbar";
import Megasync from "./Megasync";
import Mellel from "./Mellel";
import Melodics from "./Melodics";
import Memory from "./Memory";
import Memoryanalyzer from "./Memoryanalyzer";
import MemoryCleaner from "./MemoryCleaner";
import MendeleyReferenceManager from "./MendeleyReferenceManager";
import MenubarStats from "./MenubarStats";
import Menubarx from "./Menubarx";
import MerlinProject from "./MerlinProject";
import MicrosoftAzureStorageExplorer from "./MicrosoftAzureStorageExplorer";
import MicrosoftOffice from "./MicrosoftOffice";
import Max from "./Max";
import Microsoft365Copilot from "./Microsoft365Copilot";
import MicrosoftDotnetRuntime from "./MicrosoftDotnetRuntime";
import MicrosoftRemoteHelp from "./MicrosoftRemoteHelp";
import Middle from "./Middle";
import Middleclick from "./Middleclick";
import Milanote from "./Milanote";
import Mimecast from "./Mimecast";
import Mimestream from "./Mimestream";
import Mindmac from "./Mindmac";
import MindManager from "./MindManager";
import Minisim from "./Minisim";
import Minstaller from "./Minstaller";
import Missive from "./Missive";
import Mist from "./Mist";
import Mixxx from "./Mixxx";
import Mobirise from "./Mobirise";
import Mockoon from "./Mockoon";
import ModernCsv from "./ModernCsv";
import Monitorcontrol from "./Monitorcontrol";
import Moom from "./Moom";
import Moonlight from "./Moonlight";
import Morgen from "./Morgen";
import Mos from "./Mos";
import MountainDuck from "./MountainDuck";
import Mqttx from "./Mqttx";
import MullvadBrowser from "./MullvadBrowser";
import MullvadVpn from "./MullvadVpn";
import Multitouch from "./Multitouch";
import Mural from "./Mural";
import Museeks from "./Museeks";
import Musescore from "./Musescore";
import MxPowerGadget from "./MxPowerGadget";
import Nagstamon from "./Nagstamon";
import NameMangler from "./NameMangler";
import Naps2 from "./Naps2";
import NativeAccess from "./NativeAccess";
import NdiTools from "./NdiTools";
import Neofinder from "./Neofinder";
import NessusAgent from "./NessusAgent";
import Netiquette from "./Netiquette";
import Netnewswire from "./Netnewswire";
import Netron from "./Netron";
import Netspot from "./Netspot";
import Nextcloud from "./Nextcloud";
import NextcloudTalk from "./NextcloudTalk";
import Nightfall from "./Nightfall";
import NitroPdfPro from "./NitroPdfPro";
import Nocturnal from "./Nocturnal";
import Nodejs from "./Nodejs";
import Nordlayer from "./Nordlayer";
import NosqlWorkbench from "./NosqlWorkbench";
import Notchnook from "./Notchnook";
import Notepad from "./Notepad++";
import Notepadexe from "./Notepadexe";
import Notesnook from "./Notesnook";
import Notesollama from "./Notesollama";
import NounProject from "./NounProject";
import Novabench from "./Novabench";
import Nucleo from "./Nucleo";
import Numi from "./Numi";
import NvidiaGeforceNow from "./NvidiaGeforceNow";
import Ocenaudio from "./Ocenaudio";
import OkJson from "./OkJson";
import OktaVerify from "./OktaVerify";
import Ollama from "./Ollama";
import Omnidisksweeper from "./Omnidisksweeper";
import Omnifocus from "./Omnifocus";
import Omnioutliner from "./Omnioutliner";
import Omniplan from "./Omniplan";
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
import OptimusPlayer from "./OptimusPlayer";
import OrigamiStudio from "./OrigamiStudio";
import Osquery from "./Osquery";
import Outset from "./Outset";
import Overflow from "./Overflow";
import Oversight from "./Oversight";
import Owncloud from "./Owncloud";
import Pacifist from "./Pacifist";
import Packages from "./Packages";
import PaleMoon from "./PaleMoon";
import Paletro from "./Paletro";
import ParallelsClient from "./ParallelsClient";
import Parsec from "./Parsec";
import Pastebot from "./Pastebot";
import PathFinder from "./PathFinder";
import Pcoipclient from "./Pcoipclient";
import Pd from "./Pd";
import PdfExpert from "./PdfExpert";
import PdfPals from "./PdfPals";
import PdfsamBasic from "./PdfsamBasic";
import Pearcleaner from "./Pearcleaner";
import Perimeter81 from "./Perimeter81";
import Permute from "./Permute";
import PhilipsHueSync from "./PhilipsHueSync";
import PhoenixSlides from "./PhoenixSlides";
import Photosrevive from "./Photosrevive";
import Photostickies from "./Photostickies";
import Pibar from "./Pibar";
import Picview from "./Picview";
import Piezo from "./Piezo";
import Pika from "./Pika";
import Pingplotter from "./Pingplotter";
import Piphero from "./Piphero";
import Pitch from "./Pitch";
import Pixelsnap from "./Pixelsnap";
import PlantronicsHub from "./PlantronicsHub";
import Platypus from "./Platypus";
import Plex from "./Plex";
import PlexHtpc from "./PlexHtpc";
import PlexMediaServer from "./PlexMediaServer";
import PlisteditPro from "./PlisteditPro";
import Popchar from "./Popchar";
import Popclip from "./Popclip";
import Popsql from "./Popsql";
import Portfolioperformance from "./Portfolioperformance";
import Positron from "./Positron";
import PostgresApp from "./PostgresApp";
import Postgresql15 from "./Postgresql15";
import Postgresql16 from "./Postgresql16";
import Postgresql17 from "./Postgresql17";
import Postgresql18 from "./Postgresql18";
import Postico from "./Postico";
import PowerAutomate from "./PowerAutomate";
import PowerBi from "./PowerBi";
import Plugdata from "./Plugdata";
import PowerMonitor from "./PowerMonitor";
import Powerphotos from "./Powerphotos";
import Powershell from "./Powershell";
import Powertoys from "./Powertoys";
import Preform from "./Preform";
import Principle from "./Principle";
import Prism from "./Prism";
import Prisma from "./Prisma";
import PrivateInternetAccess from "./PrivateInternetAccess";
import Prizmo from "./Prizmo";
import Processing from "./Processing";
import Processspy from "./Processspy";
import Pronotes from "./Pronotes";
import ProtonDrive from "./ProtonDrive";
import ProtonMailBridge from "./ProtonMailBridge";
import ProtonMeet from "./ProtonMeet";
import ProtonPass from "./ProtonPass";
import Protopie from "./Protopie";
import Proxifier from "./Proxifier";
import Proxyman from "./Proxyman";
import Pulsar from "./Pulsar";
import Purevpn from "./Purevpn";
import Putty from "./Putty";
import Qlab from "./Qlab";
import Qlmarkdown from "./Qlmarkdown";
import Qobuz from "./Qobuz";
import QspacePro from "./QspacePro";
import Qview from "./Qview";
import R from "./R";
import RadioSilence from "./RadioSilence";
import Raindropio from "./Raindropio";
import Rapidweaver from "./Rapidweaver";
import RApp from "./RApp";
import RaspberryPiImager from "./RaspberryPiImager";
import Readest from "./Readest";
import RealVncServer from "./RealVncServer";
import Reaper from "./Reaper";
import Recents from "./Recents";
import RectanglePro from "./RectanglePro";
import Recut from "./Recut";
import RedcineXPro from "./RedcineXPro";
import RedisPro from "./RedisPro";
import Reflector from "./Reflector";
import RemindersMenubar from "./RemindersMenubar";
import RemoteBuddy from "./RemoteBuddy";
import RemoteDesktopManager from "./RemoteDesktopManager";
import Reqable from "./Reqable";
import Requestly from "./Requestly";
import Retcon from "./Retcon";
import Retroarch from "./Retroarch";
import Retrobatch from "./Retrobatch";
import Rewritebar from "./Rewritebar";
import Rightfont from "./Rightfont";
import Ringcentral from "./Ringcentral";
import Rive from "./Rive";
import RiversideStudio from "./RiversideStudio";
import Rize from "./Rize";
import Robofont from "./Robofont";
import Roboform from "./Roboform";
import Rocket from "./Rocket";
import RocketChat from "./RocketChat";
import RocketmanChoicesPackager from "./RocketmanChoicesPackager";
import RocketTypist from "./RocketTypist";
import Rodecaster from "./Rodecaster";
import RodeConnect from "./RodeConnect";
import Roon from "./Roon";
import Rstudio from "./Rstudio";
import Rsyncui from "./Rsyncui";
import Runjs from "./Runjs";
import RustDesk from "./RustDesk";
import Sabnzbd from "./Sabnzbd";
import SafeExamBrowser from "./SafeExamBrowser";
import SalesforceCli from "./SalesforceCli";
import Sanesidebuttons from "./Sanesidebuttons";
import ScMenu from "./ScMenu";
import Scratch from "./Scratch";
import ScreamingFrogSeoSpider from "./ScreamingFrogSeoSpider";
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
import Servo from "./Servo";
import Session from "./Session";
import Setapp from "./Setapp";
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
import Shapr3D from "./Shapr3D";
import Sharefile from "./Sharefile";
import Shift from "./Shift";
import Shifty from "./Shifty";
import Shortcat from "./Shortcat";
import Shotcut from "./Shotcut";
import Shottr from "./Shottr";
import ShureplusMotiv from "./ShureplusMotiv";
import Sidenotes from "./Sidenotes";
import Sigmaos from "./Sigmaos";
import Signal from "./Signal";
import Silentknight from "./Silentknight";
import SilhouetteStudio from "./SilhouetteStudio";
import SimpleComic from "./SimpleComic";
import Simpledemviewer from "./Simpledemviewer";
import Sirimote from "./Sirimote";
import Sketch from "./Sketch";
import Sketchup from "./Sketchup";
import Skim from "./Skim";
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
import Sonos from "./Sonos";
import SonosS1Controller from "./SonosS1Controller";
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
import Studio3T from "./Studio3T";
import Subethaedit from "./Subethaedit";
import SublimeMerge from "./SublimeMerge";
import SublimeText from "./SublimeText";
import Sunsama from "./Sunsama";
import Supercollider from "./Supercollider";
import Superhuman from "./Superhuman";
import Superkey from "./Superkey";
import Superlist from "./Superlist";
import SuperProductivity from "./SuperProductivity";
import Superwhisper from "./Superwhisper";
import Supportcompanion from "./Supportcompanion";
import Surfshark from "./Surfshark";
import Surge from "./Surge";
import SuspiciousPackage from "./SuspiciousPackage";
import Swiftbar from "./Swiftbar";
import Swiftdialog from "./Swiftdialog";
import SwiftQuit from "./SwiftQuit";
import SwiftShift from "./SwiftShift";
import Swifty from "./Swifty";
import Swish from "./Swish";
import Switch from "./Switch";
import Sync from "./Sync";
import Syncmate from "./Syncmate";
import Syncovery from "./Syncovery";
import SyncthingApp from "./SyncthingApp";
import Synologyassistant from "./Synologyassistant";
import SyntaxHighlight from "./SyntaxHighlight";
import Systhist from "./Systhist";
import Tabby from "./Tabby";
import TableauDesktop from "./TableauDesktop";
import TableauPrep from "./TableauPrep";
import TableauPublic from "./TableauPublic";
import TableauReader from "./TableauReader";
import TablePlus from "./TablePlus";
import Tabtab from "./Tabtab";
import Tabula from "./Tabula";
import Taccy from "./Taccy";
import Tageditor from "./Tageditor";
import Tailscale from "./Tailscale";
import Taskade from "./Taskade";
import Taskbar from "./Taskbar";
import Teacode from "./Teacode";
import TeamViewer from "./TeamViewer";
import Teams from "./Teams";
import TeamviewerHost from "./TeamviewerHost";
import TeamviewerQuicksupport from "./TeamviewerQuicksupport";
import Telegram from "./Telegram";
import TeleportConnect from "./TeleportConnect";
import Terminal from "./Terminal";
import Termius from "./Termius";
import TexLiveUtility from "./TexLiveUtility";
import Texshop from "./Texshop";
import TextExpander from "./TextExpander";
import Thaw from "./Thaw";
import TheUnarchiver from "./TheUnarchiver";
import Thorium from "./Thorium";
import Threema from "./Threema";
import Thumbsup from "./Thumbsup";
import Thunderbird from "./Thunderbird";
import Ticktick from "./Ticktick";
import Tidal from "./Tidal";
import Tiles from "./Tiles";
import Timer from "./Timer";
import Timescribe from "./Timescribe";
import Timing from "./Timing";
import Todoist from "./Todoist";
import Tofu from "./Tofu";
import Tomatobar from "./Tomatobar";
import TopazGigapixelAi from "./TopazGigapixelAi";
import TopazPhotoAi from "./TopazPhotoAi";
import TopazVideoAi from "./TopazVideoAi";
import Topnotch from "./Topnotch";
import TorBrowser from "./TorBrowser";
import Tortoisegit from "./Tortoisegit";
import Tower from "./Tower";
import Tradingview from "./Tradingview";
import Transfer from "./Transfer";
import Transmission from "./Transmission";
import Transmit from "./Transmit";
import Tresorit from "./Tresorit";
import Trex from "./Trex";
import TrezorSuite from "./TrezorSuite";
import Tribler from "./Tribler";
import Tripmode from "./Tripmode";
import Tunnelblick from "./Tunnelblick";
import Tuple from "./Tuple";
import TutaMail from "./TutaMail";
import TwineApp from "./TwineApp";
import Twingate from "./Twingate";
import Twobird from "./Twobird";
import Typeface from "./Typeface";
import Typinator from "./Typinator";
import Typora from "./Typora";
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
import Veracrypt from "./Veracrypt";
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
import Vnote from "./Vnote";
import Voiceink from "./Voiceink";
import VpnTracker365 from "./VpnTracker365";
import VsCodium from "./VsCodium";
import Vuescan from "./Vuescan";
import Vyprvpn from "./Vyprvpn";
import Vysor from "./Vysor";
import WacomCenter from "./WacomCenter";
import Warp from "./Warp";
import Waterfox from "./Waterfox";
import Wave from "./Wave";
import Wavebox from "./Wavebox";
import Wealthfolio from "./Wealthfolio";
import Weasis from "./Weasis";
import Webcatalog from "./Webcatalog";
import WebStorm from "./WebStorm";
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
import Xattred from "./Xattred";
import Xca from "./Xca";
import XcodesApp from "./XcodesApp";
import XCreds from "./XCreds";
import Xld from "./Xld";
import Xmenu from "./Xmenu";
import Xmind from "./Xmind";
import Xmplify from "./Xmplify";
import Xnapper from "./Xnapper";
import Xnconvert from "./Xnconvert";
import Xnviewmp from "./Xnviewmp";
import Xquartz from "./Xquartz";
import Yaak from "./Yaak";
import Yacreader from "./Yacreader";
import Yattee from "./Yattee";
import Yed from "./Yed";
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
  "3d slicer": Slicer,
  "4k slideshow maker": FourKSlideshowMaker,
  "4k stogram": FourKStogram,
  "4k video downloader": FourKVideoDownloader,
  "4k video to mp3": FourKVideoToMp3,
  "4k youtube to mp3": FourKYoutubeToMp3,
  "7 zip": SevenZip,
  "7-zip": SevenZip,
  "8x8 work": EightXEightWork,
  "1password": OnePassword,
  "a better finder rename": ABetterFinderRename,
  "ableton live suite": AbletonLive12Suite,
  abstract: Abstract,
  acorn: Acorn,
  activedock: Activedock,
  activitywatch: Activitywatch,
  actual: Actual,
  adguard: Adguard,
  adlock: Adlock,
  "adobe acrobat": AcrobatReader,
  "adobe acrobat reader": AcrobatReader,
  "adobe creative cloud": CreativeCloud,
  "adobe digital editions": AdobeDigitalEditions45,
  "adobe dng converter": AdobeDngConverter,
  "advanced renamer": AdvancedRenamer,
  affinity: Affinity,
  "affinity designer": AffinityDesigner1,
  "affinity designer 2": AffinityDesigner,
  "affinity photo": AffinityPhoto1,
  "affinity photo 2": AffinityPhoto,
  "affinity publisher": AffinityPublisher1,
  "affinity publisher 2": AffinityPublisher,
  airbuddy: Airbuddy,
  aircall: Aircall,
  airdroid: Airdroid,
  airparrot: Airparrot,
  airserver: Airserver,
  airtable: Airtable,
  airtame: Airtame,
  airy: Airy,
  akiflow: Akiflow,
  alcove: Alcove,
  aldente: Aldente,
  alloy: Alloy,
  "altair graphql client": AltairGraphqlClient,
  alttab: AltTab,
  "amadeus pro": AmadeusPro,
  amadine: Amadine,
  "amazon chime": AmazonChime,
  "amazon corretto 21": AmazonCorretto21,
  "amazon corretto 24": AmazonCorretto24,
  "amazon corretto 25": AmazonCorretto25,
  "amazon corretto 26": AmazonCorretto26,
  "amazon dcv": AmazonDCV,
  "amazon workspaces": AmazonWorkspaces,
  amethyst: Amethyst,
  amie: Amie,
  androidPlayStore: AndroidPlayStore,
  "android studio": AndroidStudio,
  "angry ip scanner": AngryIpScanner,
  anka: Anka,
  "another redis desktop manager": AnotherRedisDesktopManager,
  antigravity: Antigravity,
  "antigravity ide": AntigravityIde,
  antinote: Antinote,
  "any.do": Anydo,
  anytype: Anytype,
  apidog: Apidog,
  "app fair": AppFair,
  "appium inspector gui": AppiumInspector,
  applite: Applite,
  "asset catalog tinkerer": AssetCatalogTinkerer,
  atext: Atext,
  "audio hijack": AudioHijack,
  "aviatrix vpn client": AviatrixVpnClient,
  "axure rp": AxureRp,
  "background music": BackgroundMusic,
  badgeify: Badgeify,
  "balsamiq wireframes": BalsamiqWireframes,
  "bambu studio": BambuStudio,
  bartender: Bartender,
  batfi: Batfi,
  bdash: Bdash,
  "beaver notes": BeaverNotes,
  beeper: Beeper,
  bettermouse: Bettermouse,
  bettertouchtool: Bettertouchtool,
  betterzip: Betterzip,
  bezel: Bezel,
  bibdesk: Bibdesk,
  binance: Binance,
  biscuit: Biscuit,
  bitbox: Bitbox,
  "bitfocus companion": Companion,
  bitrix24: Bitrix24,
  "bitwig studio": BitwigStudio,
  bleunlock: Bleunlock,
  blip: Blip,
  bluewallet: Bluewallet,
  blurscreen: Blurscreen,
  "boltai 2": Boltai,
  "bome network": BomeNetwork,
  "boom 3d": Boom3D,
  boop: Boop,
  "boost note": BoostNote,
  breaktimer: Breaktimer,
  "bricklink studio": BricklinkStudio,
  bunch: Bunch,
  busycontacts: Busycontacts,
  buttercup: Buttercup,
  buzz: Buzz,
  cacher: Cacher,
  caffeine: Caffeine,
  "calibrite profiler": CalibriteProfiler,
  "camo studio": CamoStudio,
  "camunda modeler": CamundaModeler,
  capacities: Capacities,
  capcut: CapCut,
  captain: Captain,
  captin: Captin,
  capto: Capto,
  "carbon copy cloner": CarbonCopyCloner,
  cardhop: Cardhop,
  cellprofiler: Cellprofiler,
  chalk: Chalk,
  charmstone: Charmstone,
  chatwise: Chatwise,
  chatwork: Chatwork,
  cheetah3d: Cheetah3D,
  "cherry studio": CherryStudio,
  chime: Chime,
  choosy: Choosy,
  cleanclip: Cleanclip,
  clipbook: Clipbook,
  clipgrab: Clipgrab,
  clipy: Clipy,
  clocker: Clocker,
  clop: Clop,
  cmake: CmakeApp,
  cmux: Cmux,
  coconutbattery: Coconutbattery,
  codeedit: Codeedit,
  coderunner: Coderunner,
  codexbar: Codexbar,
  cog: CogApp,
  "colorsnapper 2": Colorsnapper,
  "colour contrast analyser": ColourContrastAnalyser,
  "command-tab plus": CommandTabPlus,
  commander: Commander,
  "commander one": CommanderOne,
  copilot: CopilotMoney,
  cork: Cork,
  crossover: Crossover,
  crystalfetch: Crystalfetch,
  cursorsense: Cursorsense,
  cursr: Cursr,
  customshortcuts: Customshortcuts,
  daisydisk: Daisydisk,
  dangerzone: Dangerzone,
  darkmodebuddy: Darkmodebuddy,
  darktable: Darktable,
  dataflare: Dataflare,
  dataspell: Dataspell,
  dayflow: Dayflow,
  dbgate: Dbgate,
  dbvisualizer: Dbvisualizer,
  debookee: Debookee,
  deckset: Deckset,
  deezer: Deezer,
  "default folder x": DefaultFolderX,
  descript: Descript,
  deskpad: Deskpad,
  desktime: Desktime,
  devknife: Devknife,
  "devonsphere express": DevonsphereExpress,
  devonthink: Devonthink,
  devtoys: Devtoys,
  devutils: Devutils,
  "dfu blaster pro": DfuBlasterPro,
  dictionaries: Dictionaries,
  "diffusion bee": Diffusionbee,
  digikam: Digikam,
  "disk drill": DiskDrill,
  dockdoor: Dockdoor,
  dockfix: Dockfix,
  dockside: Dockside,
  dockview: Dockview,
  dot: Dot,
  doughnut: Doughnut,
  downie: Downie,
  "drata agent": DrataAgent,
  drawbot: Drawbot,
  dropdmg: Dropdmg,
  droplr: Droplr,
  dropshare: Dropshare,
  dropzone: Dropzone,
  duckduckgo: Duckduckgo,
  duet: Duet,
  dupeguru: Dupeguru,
  "dymo connect": DymoConnect,
  dynalist: Dynalist,
  eaglefiler: Eaglefiler,
  easydict: Easydict,
  easyfind: Easyfind,
  "eclipse memory analyzer": Memoryanalyzer,
  edrawmax: WondershareEdrawmax,
  electronmail: Electronmail,
  electrum: Electrum,
  element: Element,
  elephas: Elephas,
  "elgato camera hub": ElgatoCameraHub,
  "elgato capture device utility": ElgatoCaptureDeviceUtility,
  "elgato game capture hd": ElgatoGameCaptureHd,
  "elgato wave link": ElgatoWaveLink,
  "elmedia player": ElmediaPlayer,
  "eltima cloudmounter": Cloudmounter,
  "em client": Emclient,
  enpass: Enpass,
  "ente auth": EnteAuth,
  "epic games launcher": EpicGames,
  equinox: Equinox,
  etrecheck: Etrecheckpro,
  exifcleaner: Exifcleaner,
  exifrenamer: Exifrenamer,
  extradock: Extradock,
  "f.lux": FluxApp,
  fantastical: Fantastical,
  far2l: Far2L,
  farrago: Farrago,
  fastmail: Fastmail,
  fastscripts: Fastscripts,
  fellow: Fellow,
  ferdium: Ferdium,
  fetch: FetchApp,
  fig: Fig,
  "file juicer": FileJuicer,
  filen: Filen,
  "fing desktop": Fing,
  "fire alpaca": Firealpaca,
  firefly: FireflyIotaDesktop,
  "firefly shimmer": FireflyShimmer,
  fission: Fission,
  "flexoptix app": Flexoptix,
  flowvision: Flowvision,
  fluid: Fluid,
  "focusrite control 2": FocusriteControl2,
  folx: Folx,
  fontbase: Fontbase,
  fontlab: Fontlab,
  forecast: Forecast,
  forklift: Forklift,
  framer: Framer,
  franz: Franz,
  "free download manager": FreeDownloadManager,
  freefilesync: Freefilesync,
  fsmonitor: Fsmonitor,
  funter: Funter,
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
  "google earth pro": GoogleEarthPro,
  gotomeeting: GoToMeeting,
  gpodder: Gpodder,
  grandperspective: Grandperspective,
  "graphpad prism": Prism,
  grids: Grids,
  "groove omnidialer": GrooveOmniDialer,
  hammerspoon: Hammerspoon,
  handbrake: HandbrakeApp,
  hazel: Hazel,
  hazeover: Hazeover,
  helium: Helium,
  "hex fiend": HexFiend,
  hey: HeyDesktop,
  "hidden bar": Hiddenbar,
  hides: Hides,
  hidock: Hidock,
  highlight: HighlightAi,
  hive: HiveApp,
  "home assistant": HomeAssistant,
  homerow: Homerow,
  hot: Hot,
  houdahspot: Houdahspot,
  "hp easy admin": HpEasyAdmin,
  hubstaff: Hubstaff,
  huly: Huly,
  hyperkey: Hyperkey,
  i1profiler: I1Profiler,
  "ibm aspera connect": IbmAsperaConnect,
  "ibm notifier": IbmNotifier,
  ice: JordanbairdIce,
  "icon composer": IconComposer,
  iconjar: Iconjar,
  idagio: Idagio,
  iexplorer: Iexplorer,
  iina: Iina,
  "imazing converter": ImazingConverter,
  imhex: Imhex,
  "input source pro": InputSourcePro,
  insyncclient: DruvaInSync,
  intellidock: Intellidock,
  invesalius: Invesalius,
  istherenet: Istherenet,
  itsycal: Itsycal,
  jami: Jami,
  jamovi: Jamovi,
  jasp: Jasp,
  jellyfin: Jellyfin,
  jiggler: Jiggler,
  "jitsi meet": JitsiMeet,
  joplin: Joplin,
  julia: JuliaApp,
  "jump desktop": JumpDesktop,
  kaleidoscope: Kaleidoscope,
  kap: Kap,
  kdenlive: Kdenlive,
  keepingyouawake: Keepingyouawake,
  keeweb: Keeweb,
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
  lapce: Lapce,
  lasso: LassoApp,
  "last window quits": LastWindowQuits,
  lastpass: LastPass,
  latest: Latest,
  launchbar: Launchbar,
  "lenovo dock manager": LenovoDockManager,
  lightburn: Lightburn,
  linearmouse: Linearmouse,
  "lingon x": LingonX,
  "lo-rain": LoRain,
  local: Local,
  localsend: Localsend,
  locationsimulator: Locationsimulator,
  logseq: Logseq,
  lookaway: Lookaway,
  loop: Loop,
  loopback: Loopback,
  losslesscut: Losslesscut,
  "low profile": LowProfile,
  lunacy: Lunacy,
  lunar: Lunar,
  lunasea: Lunasea,
  lunatask: Lunatask,
  "lychee slicer": Lycheeslicer,
  "mac mouse fix": MacMouseFix,
  macdown: Macdown,
  mace: Mace,
  macjournal: Macjournal,
  macpacker: Macpacker,
  macpass: Macpass,
  macpilot: Macpilot,
  "macs fan control": MacsFanControl,
  macsyzones: Macsyzones,
  mactracker: Mactracker,
  macvim: MacvimApp,
  macwhisper: Macwhisper,
  maestral: Maestral,
  magicquit: Magicquit,
  mailspring: Mailspring,
  "malwarebytes for mac": Malwarebytes,
  marked: MarkedApp,
  markedit: Markedit,
  marktext: MarkText,
  marsedit: Marsedit,
  "marta file manager": Marta,
  marvel: Marvel,
  masscode: Masscode,
  meetingbar: Meetingbar,
  megasync: Megasync,
  mellel: Mellel,
  melodics: Melodics,
  "memory cleaner": MemoryCleaner,
  "memory tracker by timely": Memory,
  "mendeley reference manager": MendeleyReferenceManager,
  "menubar stats": MenubarStats,
  menubarx: Menubarx,
  "merlin project": MerlinProject,
  "microsoft azure storage explorer": MicrosoftAzureStorageExplorer,
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
  middle: Middle,
  middleclick: Middleclick,
  milanote: Milanote,
  "mimecast for mac": Mimecast,
  mimestream: Mimestream,
  mindmac: Mindmac,
  mindmanager: MindManager,
  minisim: Minisim,
  minstaller: Minstaller,
  missive: Missive,
  mist: Mist,
  mixxx: Mixxx,
  mobirise: Mobirise,
  mockoon: Mockoon,
  "modern csv": ModernCsv,
  "mongodb compass": MongoDbCompass,
  monitorcontrol: Monitorcontrol,
  moom: Moom,
  moonlight: Moonlight,
  morgen: Morgen,
  mos: Mos,
  "mountain duck": MountainDuck,
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
  mqttx: Mqttx,
  "mullvad browser": MullvadBrowser,
  "mullvad vpn": MullvadVpn,
  multitouch: Multitouch,
  mural: Mural,
  museeks: Museeks,
  musescore: Musescore,
  "mx power gadget": MxPowerGadget,
  "mysql workbench": MySqlWorkbench,
  nagstamon: Nagstamon,
  "name mangler": NameMangler,
  naps2: Naps2,
  "native access": NativeAccess,
  "ndi tools": NdiTools,
  neofinder: Neofinder,
  "nessus agent": NessusAgent,
  netiquette: Netiquette,
  netnewswire: Netnewswire,
  netron: Netron,
  netspot: Netspot,
  nextcloud: Nextcloud,
  "nextcloud talk desktop": NextcloudTalk,
  nightfall: Nightfall,
  "nitro pdf pro": NitroPdfPro,
  nocturnal: Nocturnal,
  "node.js": Nodejs,
  "nord vpn": NordVpn,
  nordlayer: Nordlayer,
  nordpass: Nordpass,
  nordvpn: NordVpn,
  "nosql workbench": NosqlWorkbench,
  "nota gyazo gif": Gyazo,
  notchnook: Notchnook,
  "notepad++": Notepad,
  "notepad.exe": Notepadexe,
  notesnook: Notesnook,
  notesollama: Notesollama,
  "notion calendar": NotionCalendar,
  notion: Notion,
  "noun project": NounProject,
  nova: Nova,
  novabench: Novabench,
  nucleo: Nucleo,
  nudge: Nudge,
  numi: Numi,
  "nvidia geforce now": NvidiaGeforceNow,
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
  osquery: Osquery,
  outset: Outset,
  overflow: Overflow,
  oversight: Oversight,
  owncloud: Owncloud,
  pacifist: Pacifist,
  package: Package,
  packages: Packages,
  "pale moon": PaleMoon,
  paletro: Paletro,
  "parallels client": ParallelsClient,
  "parallels desktop": ParallelsDesktop,
  p4v: P4V,
  parsec: Parsec,
  pastebot: Pastebot,
  "path finder": PathFinder,
  pd: Pd,
  "pdf expert": PdfExpert,
  "pdf pals": PdfPals,
  "pdfsam basic": PdfsamBasic,
  pearcleaner: Pearcleaner,
  "perimeter 81": Perimeter81,
  permute: Permute,
  "pgadmin 4": PgAdmin4,
  pgadmin4: PgAdmin4,
  "philips hue sync": PhilipsHueSync,
  "phoenix slides": PhoenixSlides,
  photosrevive: Photosrevive,
  photostickies: Photostickies,
  phpstorm: PhpStorm,
  pibar: Pibar,
  picview: Picview,
  piezo: Piezo,
  pika: Pika,
  pingplotter: Pingplotter,
  piphero: Piphero,
  pitch: Pitch,
  pixelsnap: Pixelsnap,
  "plantronics hub": PlantronicsHub,
  platypus: Platypus,
  plex: Plex,
  "plex htpc": PlexHtpc,
  "plex media server": PlexMediaServer,
  "plistedit pro": PlisteditPro,
  plugdata: Plugdata,
  "podman desktop": PodmanDesktop,
  "popchar x": Popchar,
  popclip: Popclip,
  popsql: Popsql,
  "portfolio performance": Portfolioperformance,
  positron: Positron,
  postgres: PostgresApp,
  "postgresql 15": Postgresql15,
  "postgresql 16": Postgresql16,
  "postgresql 17": Postgresql17,
  "postgresql 18": Postgresql18,
  postico: Postico,
  postman: Postman,
  "power automate": PowerAutomate,
  "power bi": PowerBi,
  "power monitor": PowerMonitor,
  powerphotos: Powerphotos,
  powershell: Powershell,
  powertoys: Powertoys,
  preform: Preform,
  principle: Principle,
  prisma: Prisma,
  "private internet access": PrivateInternetAccess,
  privileges: Privileges,
  pritunl: Pritunl,
  prizmo: Prizmo,
  processing: Processing,
  processspy: Processspy,
  pronotes: Pronotes,
  "proton drive": ProtonDrive,
  "proton mail": ProtonMail,
  "proton mail bridge": ProtonMailBridge,
  "proton meet": ProtonMeet,
  "proton pass": ProtonPass,
  protonvpn: ProtonVpn,
  protopie: Protopie,
  proxifier: Proxifier,
  proxyman: Proxyman,
  "ps remote play": SonyPsRemotePlay,
  pulsar: Pulsar,
  purevpn: Purevpn,
  putty: Putty,
  "pycharm ce": PyCharmCe,
  pycharm: PyCharm,
  "python 3.13": Python313,
  "python 3.14": Python314,
  qlab: Qlab,
  qobuz: Qobuz,
  "qspace pro": QspacePro,
  quip: Quip,
  qview: Qview,
  r: RApp,
  "r for windows": R,
  "radio silence": RadioSilence,
  "raindrop.io": Raindropio,
  "rancher desktop": RancherDesktop,
  rapidapi: RapidApi,
  rapidweaver: Rapidweaver,
  "raspberry pi imager": RaspberryPiImager,
  raycast: Raycast,
  readest: Readest,
  "realvnc server": RealVncServer,
  reaper: Reaper,
  recents: Recents,
  rectangle: Rectangle,
  "rectangle pro": RectanglePro,
  recut: Recut,
  "redcine-x pro": RedcineXPro,
  "redis-pro": RedisPro,
  reflector: Reflector,
  "reminders menubar": RemindersMenubar,
  "remote buddy": RemoteBuddy,
  "remote desktop manager": RemoteDesktopManager,
  reqable: Reqable,
  requestly: Requestly,
  retcon: Retcon,
  retroarch: Retroarch,
  retrobatch: Retrobatch,
  rewritebar: Rewritebar,
  rider: Rider,
  rightfont: Rightfont,
  ringcentral: Ringcentral,
  rive: Rive,
  "riverside studio": RiversideStudio,
  rize: Rize,
  robofont: Robofont,
  roboform: Roboform,
  rocket: Rocket,
  "rocket typist": RocketTypist,
  "rocket.chat": RocketChat,
  "rocketman choices packager": RocketmanChoicesPackager,
  "rode connect": RodeConnect,
  "rodecaster app": Rodecaster,
  roon: Roon,
  "royal tsx": RoyalTsx,
  rstudio: Rstudio,
  rsyncui: Rsyncui,
  rubymine: RubyMine,
  runjs: Runjs,
  rustdesk: RustDesk,
  rustrover: RustRover,
  sabnzbd: Sabnzbd,
  safari: Safari,
  "safe exam browser": SafeExamBrowser,
  "salesforce cli": SalesforceCli,
  sanesidebuttons: Sanesidebuttons,
  santa: Santa,
  "sbarex qlmarkdown": Qlmarkdown,
  "sc menu": ScMenu,
  scratch: Scratch,
  "screaming frog seo spider": ScreamingFrogSeoSpider,
  "screen studio": ScreenStudio,
  screenflick: Screenflick,
  screenflow: Screenflow,
  screenfocus: Screenfocus,
  scribus: Scribus,
  scrivener: Scrivener,
  secretive: Secretive,
  securesafe: Securesafe,
  selfcontrol: Selfcontrol,
  "sempliva tiles": Tiles,
  sensei: Sensei,
  "sequel ace": SequelAce,
  servo: Servo,
  session: Session,
  setapp: Setapp,
  "sf symbols": SfSymbols,
  shapr3d: Shapr3D,
  sharefile: Sharefile,
  shift: Shift,
  shifty: Shifty,
  shotcut: Shotcut,
  shottr: Shottr,
  "shureplus motiv": ShureplusMotiv,
  sidenotes: Sidenotes,
  sigmaos: Sigmaos,
  signal: Signal,
  silentknight: Silentknight,
  "silhouette studio": SilhouetteStudio,
  "simple comic": SimpleComic,
  simpledemviewer: Simpledemviewer,
  sirimote: Sirimote,
  sketch: Sketch,
  sketchup: Sketchup,
  skim: Skim,
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
  "sonos s1": SonosS1Controller,
  "sonos s2": Sonos,
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
  "studio 3t": Studio3T,
  subethaedit: Subethaedit,
  "sublime merge": SublimeMerge,
  "sublime text": SublimeText,
  sunsama: Sunsama,
  "super productivity": SuperProductivity,
  supercollider: Supercollider,
  superhuman: Superhuman,
  superkey: Superkey,
  superlist: Superlist,
  superwhisper: Superwhisper,
  "support companion": Supportcompanion,
  surfshark: Surfshark,
  surge: Surge,
  "suspicious package": SuspiciousPackage,
  "swift quit": SwiftQuit,
  "swift shift": SwiftShift,
  swiftbar: Swiftbar,
  swiftdialog: Swiftdialog,
  swifty: Swifty,
  swish: Swish,
  "switch audio converter": Switch,
  sync: Sync,
  syncmate: Syncmate,
  syncovery: Syncovery,
  syncthing: SyncthingApp,
  "synology assistant": Synologyassistant,
  "syntax highlight": SyntaxHighlight,
  systhist: Systhist,
  tabby: Tabby,
  tableau: TableauDesktop,
  "tableau prep": TableauPrep,
  "tableau public": TableauPublic,
  "tableau reader": TableauReader,
  tableplus: TablePlus,
  tabtab: Tabtab,
  tabula: Tabula,
  taccy: Taccy,
  "tag editor": Tageditor,
  tailscale: Tailscale,
  taskade: Taskade,
  taskbar: Taskbar,
  teacode: Teacode,
  "teamviewer host": TeamviewerHost,
  "teamviewer quicksupport": TeamviewerQuicksupport,
  telegram: Telegram,
  "teleport connect": TeleportConnect,
  "teleport suite": TeleportConnect,
  teleport: TeleportConnect,
  "teradici pcoip software client for macos": Pcoipclient,
  terminal: Terminal,
  teamviewer: TeamViewer,
  termius: Termius,
  "tex live utility": TexLiveUtility,
  texshop: Texshop,
  textexpander: TextExpander,
  thaw: Thaw,
  "the unarchiver": TheUnarchiver,
  "thorium reader": Thorium,
  threema: Threema,
  thumbsup: Thumbsup,
  thunderbird: Thunderbird,
  ticktick: Ticktick,
  tidal: Tidal,
  timer: Timer,
  timescribe: Timescribe,
  timing: Timing,
  todoist: Todoist,
  tofu: Tofu,
  tomatobar: Tomatobar,
  "topaz gigapixel ai": TopazGigapixelAi,
  "topaz photo ai": TopazPhotoAi,
  "topaz video ai": TopazVideoAi,
  topnotch: Topnotch,
  "tor browser": TorBrowser,
  tortoisegit: Tortoisegit,
  tower: Tower,
  "tradingview desktop": Tradingview,
  transfer: Transfer,
  transmission: Transmission,
  transmit: Transmit,
  tresorit: Tresorit,
  trex: Trex,
  "trezor suite": TrezorSuite,
  tribler: Tribler,
  tripmode: Tripmode,
  tunnelblick: Tunnelblick,
  tuple: Tuple,
  "tuta mail": TutaMail,
  twine: TwineApp,
  twingate: Twingate,
  twobird: Twobird,
  typeface: Typeface,
  typinator: Typinator,
  typora: Typora,
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
  veracrypt: Veracrypt,
  "vernier spectral analysis": VernierSpectralAnalysis,
  versions: Versions,
  via: Via,
  vimcal: Vimcal,
  virtualbox: VirtualBox,
  virtualbuddy: VirtualBuddy,
  viscosity: Viscosity,
  "visual paradigm": VisualParadigm,
  vivid: VividApp,
  viz: Viz,
  "vnc viewer": VncViewer,
  "visual studio code": VisualStudioCode,
  vlc: Vlc,
  vnote: Vnote,
  voiceink: Voiceink,
  "vpn tracker 365": VpnTracker365,
  vscodium: VsCodium,
  vuescan: Vuescan,
  vyprvpn: Vyprvpn,
  vysor: Vysor,
  "wacom center": WacomCenter,
  "wacom tablet": WacomCenter,
  warp: Warp,
  waterfox: Waterfox,
  "wave terminal": Wave,
  wavebox: Wavebox,
  wealthfolio: Wealthfolio,
  weasis: Weasis,
  webcatalog: Webcatalog,
  webstorm: WebStorm,
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
  "wondershare filmora": WondershareFilmora,
  wordservice: Wordservice,
  workflowy: Workflowy,
  "worksheet crafter": WorksheetCrafter,
  workspaces: Workspaces,
  "wrike for mac": WrikeForMac,
  wrike: WrikeForMac,
  "x lossless decoder": Xld,
  xattred: Xattred,
  xca: Xca,
  xcodes: XcodesApp,
  xcreds: XCreds,
  xmenu: Xmenu,
  xmind: Xmind,
  xmplify: Xmplify,
  xnapper: Xnapper,
  "xnsoft xnconvert": Xnconvert,
  xnviewmp: Xnviewmp,
  xquartz: Xquartz,
  yaak: Yaak,
  yacreader: Yacreader,
  yattee: Yattee,
  yippy: Yippy,
  "yubico authenticator": YubicoAuthenticator,
  "yubikey manager": YubikeyManager,
  "yworks yed": Yed,
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
