// Note: if parts of a icon have a clip path, mask, or gradient, the IDs must be unique
// across all icons to avoid conflicts in the DOM. See uniqueId usage within icon components.

import { HOST_LINUX_PLATFORMS } from "interfaces/platform";
import { ISoftware } from "interfaces/software";
import { matchLoosePrefixToKey } from "utilities/strings/stringUtils";

import { TMatchedIcon } from "./MatchedIcon";

import ABetterFinderRename from "./png/ABetterFinderRename.png";
import AbletonLive12Suite from "./png/AbletonLive12Suite.png";
import Abstract from "./png/Abstract.png";
import Acorn from "./png/Acorn.png";
import AcrobatReader from "./AcrobatReader";
import Activedock from "./png/Activedock.png";
import Activitywatch from "./png/Activitywatch.png";
import Actual from "./png/Actual.png";
import Adguard from "./png/Adguard.png";
import Adlock from "./png/Adlock.png";
import AdobeDigitalEditions45 from "./png/AdobeDigitalEditions45.png";
import AdobeDngConverter from "./png/AdobeDngConverter.png";
import AdobePlugin from "./AdobePlugin";
import AdvancedInstaller from "./png/AdvancedInstaller.png";
import AdvancedRenamer from "./png/AdvancedRenamer.png";
import Affinity from "./png/Affinity.png";
import AffinityDesigner from "./png/AffinityDesigner.png";
import AffinityDesigner1 from "./png/AffinityDesigner1.png";
import AffinityPhoto from "./png/AffinityPhoto.png";
import AffinityPhoto1 from "./png/AffinityPhoto1.png";
import AffinityPublisher from "./png/AffinityPublisher.png";
import AffinityPublisher1 from "./png/AffinityPublisher1.png";
import AgentRansack from "./png/AgentRansack.png";
import Airbuddy from "./png/Airbuddy.png";
import Aircall from "./png/Aircall.png";
import Airdroid from "./png/Airdroid.png";
import AirExplorer from "./png/AirExplorer.png";
import Airparrot from "./png/Airparrot.png";
import Airserver from "./png/Airserver.png";
import Airtable from "./png/Airtable.png";
import Airtame from "./png/Airtame.png";
import Airy from "./png/Airy.png";
import Akiflow from "./png/Akiflow.png";
import Alacritty from "./png/Alacritty.png";
import Alcove from "./png/Alcove.png";
import Aldente from "./png/Aldente.png";
import Alfaview from "./png/Alfaview.png";
import Alloy from "./png/Alloy.png";
import AllwaySync from "./png/AllwaySync.png";
import AltairGraphqlClient from "./png/AltairGraphqlClient.png";
import AltTab from "./png/AltTab.png";
import AmadeusPro from "./png/AmadeusPro.png";
import Amadine from "./png/Amadine.png";
import AmazonChime from "./png/AmazonChime.png";
import AmazonCorretto21 from "./png/AmazonCorretto21.png";
import AmazonCorretto24 from "./png/AmazonCorretto24.png";
import AmazonCorretto25 from "./png/AmazonCorretto25.png";
import AmazonCorretto26 from "./png/AmazonCorretto26.png";
import AmazonDCV from "./AmazonDCV";
import AmazonRedshiftOdbcDriver from "./png/AmazonRedshiftOdbcDriver.png";
import AmazonWorkspaces from "./png/AmazonWorkspaces.png";
import Amethyst from "./png/Amethyst.png";
import Amie from "./png/Amie.png";
import AndroidApp from "./AndroidApp";
import AndroidOS from "./AndroidOS";
import AndroidPlayStore from "./AndroidPlayStore";
import AndroidStudio from "./png/AndroidStudio.png";
import AngryIpScanner from "./png/AngryIpScanner.png";
import Anka from "./png/Anka.png";
import AnotherRedisDesktopManager from "./png/AnotherRedisDesktopManager.png";
import Antigravity from "./png/Antigravity.png";
import AntigravityIde from "./png/AntigravityIde.png";
import Antinote from "./png/Antinote.png";
import Anyburn from "./png/Anyburn.png";
import AnyDesk from "./png/AnyDesk.png";
import Anydo from "./png/Anydo.png";
import Anytype from "./png/Anytype.png";
import AomeiBackupperStandard from "./png/AomeiBackupperStandard.png";
import Apidog from "./png/Apidog.png";
import Apparency from "./png/Apparency.png";
import AppCleaner from "./png/AppCleaner.png";
import AppFair from "./png/AppFair.png";
import AppiumInspector from "./png/AppiumInspector.png";
import AppleApp from "./AppleApp";
import AppleAppStore from "./AppleAppStore";
import Applite from "./png/Applite.png";
import Aptakube from "./png/Aptakube.png";
import Arc from "./png/Arc.png";
import Archaeology from "./png/Archaeology.png";
import ArduinoIde from "./png/ArduinoIde.png";
import Asana from "./png/Asana.png";
import AssetCatalogTinkerer from "./png/AssetCatalogTinkerer.png";
import Atext from "./png/Atext.png";
import Audacity from "./png/Audacity.png";
import AudioHijack from "./png/AudioHijack.png";
import Audiveris from "./png/Audiveris.png";
import Autopsy from "./png/Autopsy.png";
import AvastSecureBrowser from "./png/AvastSecureBrowser.png";
import AviatrixVpnClient from "./png/AviatrixVpnClient.png";
import AvsImageConverter from "./png/AvsImageConverter.png";
import AvsMediaPlayer from "./png/AvsMediaPlayer.png";
import AwsCli from "./png/AwsCli.png";
import AwsSamCli from "./png/AwsSamCli.png";
import AwsVpnClient from "./png/AwsVpnClient.png";
import AxureRp from "./png/AxureRp.png";
import AzulZulu25Jdk from "./png/AzulZulu25Jdk.png";
import AzulZulu25Jre from "./png/AzulZulu25Jre.png";
import AzureDataStudio from "./png/AzureDataStudio.png";
import AzureFunctionsCoreTools from "./png/AzureFunctionsCoreTools.png";
import Backblaze from "./png/Backblaze.png";
import BackgroundMusic from "./png/BackgroundMusic.png";
import Badgeify from "./png/Badgeify.png";
import BalenaEtcher from "./png/BalenaEtcher.png";
import BalsamiqWireframes from "./png/BalsamiqWireframes.png";
import BambuStudio from "./png/BambuStudio.png";
import Bandiview from "./png/Bandiview.png";
import Bartender from "./png/Bartender.png";
import Batfi from "./png/Batfi.png";
import BBEdit from "./png/BBEdit.png";
import Bdash from "./png/Bdash.png";
import BeaverNotes from "./png/BeaverNotes.png";
import BeekeeperStudio from "./png/BeekeeperStudio.png";
import Beeper from "./png/Beeper.png";
import BetterDisplay from "./png/BetterDisplay.png";
import Bettermouse from "./png/Bettermouse.png";
import Bettertouchtool from "./png/Bettertouchtool.png";
import Betterzip from "./png/Betterzip.png";
import BeyondCompare from "./png/BeyondCompare.png";
import Bezel from "./png/Bezel.png";
import Bibdesk from "./png/Bibdesk.png";
import Binance from "./png/Binance.png";
import Biscuit from "./png/Biscuit.png";
import Bitbox from "./png/Bitbox.png";
import Bitrix24 from "./png/Bitrix24.png";
import Bitwarden from "./png/Bitwarden.png";
import BitwigStudio from "./png/BitwigStudio.png";
import Bleachbit from "./png/Bleachbit.png";
import Blender from "./png/Blender.png";
import Bleunlock from "./png/Bleunlock.png";
import Blip from "./png/Blip.png";
import Bluej from "./png/Bluej.png";
import Bluewallet from "./png/Bluewallet.png";
import Blurscreen from "./png/Blurscreen.png";
import Boltai from "./png/Boltai.png";
import BomeNetwork from "./png/BomeNetwork.png";
import Boom3D from "./png/Boom3D.png";
import Boop from "./png/Boop.png";
import BoostNote from "./png/BoostNote.png";
import Box from "./Box";
import BoxTools from "./png/BoxTools.png";
import Brave from "./Brave";
import Breaktimer from "./png/Breaktimer.png";
import BricklinkStudio from "./png/BricklinkStudio.png";
import Browserstacklocal from "./png/Browserstacklocal.png";
import Bruno from "./png/Bruno.png";
import BulkCrapUninstaller from "./png/BulkCrapUninstaller.png";
import Bunch from "./png/Bunch.png";
import BurpSuiteCommunity from "./png/BurpSuiteCommunity.png";
import BurpSuiteProfessional from "./png/BurpSuiteProfessional.png";
import Busycontacts from "./png/Busycontacts.png";
import Buttercup from "./png/Buttercup.png";
import Buzz from "./png/Buzz.png";
import Cacher from "./png/Cacher.png";
import Caffeine from "./png/Caffeine.png";
import Calibre from "./png/Calibre.png";
import CalibriteProfiler from "./png/CalibriteProfiler.png";
import CamoStudio from "./png/CamoStudio.png";
import Camtasia from "./png/Camtasia.png";
import CamundaModeler from "./png/CamundaModeler.png";
import Canva from "./png/Canva.png";
import CapCut from "./png/CapCut.png";
import Captain from "./png/Captain.png";
import Capto from "./png/Capto.png";
import CarbonCopyCloner from "./png/CarbonCopyCloner.png";
import Cardhop from "./png/Cardhop.png";
import Cavalry from "./png/Cavalry.png";
import Cellprofiler from "./png/Cellprofiler.png";
import CertifyTheWeb from "./png/CertifyTheWeb.png";
import Chalk from "./png/Chalk.png";
import Charles from "./png/Charles.png";
import Charmstone from "./png/Charmstone.png";
import Chatbox from "./png/Chatbox.png";
import ChatGpt from "./png/ChatGpt.png";
import ChatGptAtlas from "./png/ChatGptAtlas.png";
import Chatwise from "./png/Chatwise.png";
import Cheetah3D from "./png/Cheetah3D.png";
import ChefWorkstation from "./png/ChefWorkstation.png";
import CherryKeys from "./png/CherryKeys.png";
import CherryStudio from "./png/CherryStudio.png";
import Chime from "./png/Chime.png";
import Choosy from "./png/Choosy.png";
import ChromeApp from "./ChromeApp";
import ChromeOS from "./ChromeOS";
import ChromeRemoteDesktop from "./png/ChromeRemoteDesktop.png";
import Cinc from "./png/Cinc.png";
import CiscoJabber from "./png/CiscoJabber.png";
import CiscoWebexRecorderAndPlayer from "./png/CiscoWebexRecorderAndPlayer.png";
import CitrixWorkspace from "./png/CitrixWorkspace.png";
import Claude from "./png/Claude.png";
import ClaudeDevtools from "./png/ClaudeDevtools.png";
import Cleanclip from "./png/Cleanclip.png";
import CleanMyMac from "./png/CleanMyMac.png";
import CleanShotX from "./png/CleanShotX.png";
import ClickShare from "./png/ClickShare.png";
import ClickUp from "./png/ClickUp.png";
import CLion from "./png/CLion.png";
import Clipboardfusion from "./png/Clipboardfusion.png";
import Clipbook from "./png/Clipbook.png";
import Clipgrab from "./png/Clipgrab.png";
import Clipy from "./png/Clipy.png";
import Clockassist from "./png/Clockassist.png";
import Clocker from "./png/Clocker.png";
import ClockifyDesktop from "./png/ClockifyDesktop.png";
import Clop from "./png/Clop.png";
import Cloudflare from "./Cloudflare";
import Cloudmounter from "./png/Cloudmounter.png";
import CmakeApp from "./png/CmakeApp.png";
import Cmux from "./png/Cmux.png";
import Coconutbattery from "./png/Coconutbattery.png";
import Codeedit from "./png/Codeedit.png";
import CodemeterRuntimeKit from "./png/CodemeterRuntimeKit.png";
import Coderunner from "./png/Coderunner.png";
import CodexApp from "./png/CodexApp.png";
import Codexbar from "./png/Codexbar.png";
import CogApp from "./png/CogApp.png";
import Colorsnapper from "./png/Colorsnapper.png";
import ColourContrastAnalyser from "./png/ColourContrastAnalyser.png";
import Comet from "./png/Comet.png";
import Commander from "./png/Commander.png";
import CommanderOne from "./png/CommanderOne.png";
import CommandTabPlus from "./png/CommandTabPlus.png";
import Companion from "./png/Companion.png";
import ConnectFonts from "./png/ConnectFonts.png";
import CopilotMoney from "./png/CopilotMoney.png";
import Cork from "./png/Cork.png";
import CotEditor from "./png/CotEditor.png";
import CpuZ from "./png/CpuZ.png";
import CrashPlan from "./png/CrashPlan.png";
import CreativeCloud from "./png/AdobeCreativeCloud.png";
import CreativeForceKelvin from "./png/CreativeForceKelvin.png";
import CreativeForceTriad from "./png/CreativeForceTriad.png";
import CrestronAirmedia from "./png/CrestronAirmedia.png";
import CrestronAirmediaPeripherals from "./png/CrestronAirmediaPeripherals.png";
import CriblEdge from "./png/CriblEdge.png";
import Crisisgo from "./png/Crisisgo.png";
import Crossover from "./png/Crossover.png";
import Cryptomator from "./png/Cryptomator.png";
import Crystaldiskmark from "./png/Crystaldiskmark.png";
import Crystalfetch from "./png/Crystalfetch.png";
import CubeBrowser from "./png/CubeBrowser.png";
import Cursor from "./png/Cursor.png";
import Cursorsense from "./png/Cursorsense.png";
import Cursr from "./png/Cursr.png";
import Customshortcuts from "./png/Customshortcuts.png";
import Cyberduck from "./png/Cyberduck.png";
import CyberduckCli from "./png/CyberduckCli.png";
import Daisydisk from "./png/Daisydisk.png";
import Dangerzone from "./png/Dangerzone.png";
import DanteController from "./png/DanteController.png";
import Darkmodebuddy from "./png/Darkmodebuddy.png";
import Darktable from "./png/Darktable.png";
import Dash from "./png/Dash.png";
import Dataflare from "./png/Dataflare.png";
import DataGrip from "./png/DataGrip.png";
import Dataspell from "./png/Dataspell.png";
import DaxStudio from "./png/DaxStudio.png";
import Dayflow from "./png/Dayflow.png";
import DbBrowserForSqLite from "./png/DbBrowserForSqLite.png";
import DBeaver from "./png/DBeaver.png";
import DBeaverEe from "./png/DBeaverEe.png";
import DBeaverLite from "./png/DBeaverLite.png";
import DBeaverUltimate from "./png/DBeaverUltimate.png";
import Dbgate from "./png/Dbgate.png";
import Dbvisualizer from "./png/Dbvisualizer.png";
import Debookee from "./png/Debookee.png";
import Deckset from "./png/Deckset.png";
import DeepL from "./png/DeepL.png";
import Deezer from "./png/Deezer.png";
import DefaultFolderX from "./png/DefaultFolderX.png";
import DelineaConnectionManager from "./png/DelineaConnectionManager.png";
import DellCommandUpdate from "./png/DellCommandUpdate.png";
import DellDisplayAndPeripheralManager from "./png/DellDisplayAndPeripheralManager.png";
import Descript from "./png/Descript.png";
import Deskpad from "./png/Deskpad.png";
import Desktime from "./png/Desktime.png";
import DevinDesktop from "./png/DevinDesktop.png";
import Devknife from "./png/Devknife.png";
import DevolutionsLauncher from "./png/DevolutionsLauncher.png";
import DevolutionsWorkspace from "./png/DevolutionsWorkspace.png";
import DevonsphereExpress from "./png/DevonsphereExpress.png";
import Devonthink from "./png/Devonthink.png";
import Devpod from "./png/Devpod.png";
import Devtoys from "./png/Devtoys.png";
import Devutils from "./png/Devutils.png";
import DfuBlasterPro from "./png/DfuBlasterPro.png";
import Dialpad from "./png/Dialpad.png";
import Dictionaries from "./png/Dictionaries.png";
import Diffusionbee from "./png/Diffusionbee.png";
import Digikam from "./png/Digikam.png";
import DigisealReader from "./png/DigisealReader.png";
import DirectoryOpus from "./png/DirectoryOpus.png";
import Discord from "./png/Discord.png";
import DiskDrill from "./png/DiskDrill.png";
import DisplayLinkManager from "./png/DisplayLinkManager.png";
import Dngrep from "./png/Dngrep.png";
import Dockdoor from "./png/Dockdoor.png";
import Docker from "./png/Docker.png";
import Dockfix from "./png/Dockfix.png";
import Dockside from "./png/Dockside.png";
import Dockview from "./png/Dockview.png";
import Dot from "./png/Dot.png";
import Doughnut from "./png/Doughnut.png";
import Downie from "./png/Downie.png";
import DraftableDesktop from "./png/DraftableDesktop.png";
import DrataAgent from "./png/DrataAgent.png";
import Drawbot from "./png/Drawbot.png";
import Drawio from "./png/DrawIo.png";
import Drofus from "./png/Drofus.png";
import Dropbox from "./png/Dropbox.png";
import Dropdmg from "./png/Dropdmg.png";
import Droplr from "./png/Droplr.png";
import Dropshare from "./png/Dropshare.png";
import Dropzone from "./png/Dropzone.png";
import DruvaInSync from "./png/DruvaInSync.png";
import Duckduckgo from "./png/Duckduckgo.png";
import Duet from "./png/Duet.png";
import DuoDesktop from "./png/DuoDesktop.png";
import Dupeguru from "./png/Dupeguru.png";
import DymoConnect from "./png/DymoConnect.png";
import DymoId from "./png/DymoId.png";
import Dynalist from "./png/Dynalist.png";
import Eaglefiler from "./png/Eaglefiler.png";
import Easydict from "./png/Easydict.png";
import Easyfind from "./png/Easyfind.png";
import Eclipse from "./png/Eclipse.png";
import EclipseTemurinJdk11 from "./png/EclipseTemurinJdk11.png";
import EclipseTemurinJdk17 from "./png/EclipseTemurinJdk17.png";
import EclipseTemurinJdk21 from "./png/EclipseTemurinJdk21.png";
import EclipseTemurinJdk8 from "./png/EclipseTemurinJdk8.png";
import EclipseTemurinJre11 from "./png/EclipseTemurinJre11.png";
import EclipseTemurinJre17 from "./png/EclipseTemurinJre17.png";
import EclipseTemurinJre21 from "./png/EclipseTemurinJre21.png";
import EclipseTemurinJre8 from "./png/EclipseTemurinJre8.png";
import Edge from "./Edge";
import Egnyte from "./png/Egnyte.png";
import EgnyteWebedit from "./png/EgnyteWebedit.png";
import EightXEightWork from "./png/8X8Work.png";
import Electronmail from "./png/Electronmail.png";
import Electrum from "./png/Electrum.png";
import Element from "./png/Element.png";
import Elephas from "./png/Elephas.png";
import ElevateUc from "./png/ElevateUc.png";
import ElgatoCameraHub from "./png/ElgatoCameraHub.png";
import ElgatoCaptureDeviceUtility from "./png/ElgatoCaptureDeviceUtility.png";
import ElgatoControlCenter from "./png/ElgatoControlCenter.png";
import ElgatoGameCaptureHd from "./png/ElgatoGameCaptureHd.png";
import ElgatoStreamDeck from "./png/ElgatoStreamDeck.png";
import ElgatoWaveLink from "./png/ElgatoWaveLink.png";
import ElmediaPlayer from "./png/ElmediaPlayer.png";
import Emclient from "./png/Emclient.png";
import Endnote from "./png/Endnote.png";
import Enpass from "./png/Enpass.png";
import EnteAuth from "./png/EnteAuth.png";
import EpicGames from "./png/EpicGames.png";
import Equinox from "./png/Equinox.png";
import Etrecheckpro from "./png/Etrecheckpro.png";
import Evernote from "./png/Evernote.png";
import Excel from "./Excel";
import Exifcleaner from "./png/Exifcleaner.png";
import Exifrenamer from "./png/Exifrenamer.png";
import ExpressVpn from "./png/ExpressVpn.png";
import Extension from "./Extension";
import Extradock from "./png/Extradock.png";
import Falcon from "./Falcon";
import Fantastical from "./png/Fantastical.png";
import Far2L from "./png/Far2L.png";
import Farrago from "./png/Farrago.png";
import Fastmail from "./png/Fastmail.png";
import Fastscripts from "./png/Fastscripts.png";
import Fellow from "./png/Fellow.png";
import Ferdium from "./png/Ferdium.png";
import FetchApp from "./png/FetchApp.png";
import Figma from "./Figma";
import Filebeat from "./png/Filebeat.png";
import FileJuicer from "./png/FileJuicer.png";
import FileMakerPro from "./png/FileMakerPro.png";
import Filen from "./png/Filen.png";
import Fing from "./png/Fing.png";
import Firealpaca from "./png/Firealpaca.png";
import FireflyIotaDesktop from "./png/FireflyIotaDesktop.png";
import FireflyShimmer from "./png/FireflyShimmer.png";
import Firefox from "./png/Firefox.png";
import FirefoxDeveloperEdition from "./png/FirefoxDeveloperEdition.png";
import FirefoxNightly from "./png/FirefoxNightly.png";
import Fission from "./png/Fission.png";
import FleetDesktop from "./png/FleetDesktop.png";
import Flexoptix from "./png/Flexoptix.png";
import Flexwhere from "./png/Flexwhere.png";
import Fluid from "./png/Fluid.png";
import FluxApp from "./png/FluxApp.png";
import FocusriteControl2 from "./png/FocusriteControl2.png";
import Folx from "./png/Folx.png";
import Fontbase from "./png/Fontbase.png";
import Fontlab from "./png/Fontlab.png";
import Forecast from "./png/Forecast.png";
import Fork from "./png/Fork.png";
import Forklift from "./png/Forklift.png";
import Fortify from "./png/Fortify.png";
import FourKSlideshowMaker from "./png/FourKSlideshowMaker.png";
import FourKStogram from "./png/FourKStogram.png";
import FourKVideoDownloader from "./png/FourKVideoDownloader.png";
import FourKVideoDownloaderPlus from "./png/FourKVideoDownloaderPlus.png";
import FourKVideoToMp3 from "./png/FourKVideoToMp3.png";
import FourKYoutubeToMp3 from "./png/FourKYoutubeToMp3.png";
import FoxitPdfEditor from "./png/FoxitPdfEditor.png";
import FoxitPdfReader from "./png/FoxitPdfReader.png";
import Framer from "./png/Framer.png";
import Franz from "./png/Franz.png";
import Freecad from "./png/Freecad.png";
import FreeDownloadManager from "./png/FreeDownloadManager.png";
import Freefilesync from "./png/Freefilesync.png";
import Front from "./png/Front.png";
import Fsmonitor from "./png/Fsmonitor.png";
import Funter from "./png/Funter.png";
import GalaxyModeler from "./png/GalaxyModeler.png";
import GarminBasecamp from "./png/GarminBasecamp.png";
import GarminExpress from "./png/GarminExpress.png";
import Gather from "./png/Gather.png";
import Gdevelop from "./png/Gdevelop.png";
import Geany from "./png/Geany.png";
import Geekbench from "./png/Geekbench.png";
import Gemini2 from "./png/Gemini2.png";
import GenesysCloud from "./png/GenesysCloud.png";
import GeogebraClassic from "./png/GeogebraClassic.png";
import Gephi from "./png/Gephi.png";
import Ghostty from "./png/Ghostty.png";
import Gimp from "./png/Gimp.png";
import Git from "./png/Git.png";
import GitExtensions from "./png/GitExtensions.png";
import Gitfinder from "./png/Gitfinder.png";
import GithubCopilotForXcode from "./png/GithubCopilotForXcode.png";
import GitHubDesktop from "./png/GitHubDesktop.png";
import Gitify from "./png/Gitify.png";
import GitKraken from "./png/GitKraken.png";
import GitupApp from "./png/GitupApp.png";
import Glyphs from "./png/Glyphs.png";
import Gnupg from "./png/Gnupg.png";
import Go from "./png/Go.png";
import Go2Shell from "./png/Go2Shell.png";
import GoanywhereOpenpgpStudio from "./png/GoanywhereOpenpgpStudio.png";
import Godot from "./png/Godot.png";
import Godspeed from "./png/Godspeed.png";
import GogGalaxy from "./png/GogGalaxy.png";
import GoLand from "./png/GoLand.png";
import GoldendictNg from "./png/GoldendictNg.png";
import Goodsync from "./png/Goodsync.png";
import GoogleAdsEditor from "./png/GoogleAdsEditor.png";
import GoogleCredentialProviderForWindows from "./png/GoogleCredentialProviderForWindows.png";
import GoogleDrive from "./png/GoogleDrive.png";
import GoogleEarthPro from "./png/GoogleEarthPro.png";
import GoogleGemini from "./png/GoogleGemini.png";
import GoogleWebDesigner from "./png/GoogleWebDesigner.png";
import GoToMeeting from "./png/GoToMeeting.png";
import GpgKeychain from "./png/GpgKeychain.png";
import Gpg4Win from "./png/Gpg4Win.png";
import Gpodder from "./png/Gpodder.png";
import GrammarlyDesktop from "./png/GrammarlyDesktop.png";
import Grandperspective from "./png/Grandperspective.png";
import Granola from "./png/Granola.png";
import Graphviz from "./png/Graphviz.png";
import Grepwin from "./png/Grepwin.png";
import Grids from "./png/Grids.png";
import GrooveOmniDialer from "./png/GrooveOmniDialer.png";
import Gyazo from "./png/Gyazo.png";
import Hammerspoon from "./png/Hammerspoon.png";
import HandbrakeApp from "./png/HandbrakeApp.png";
import HarmonySase from "./png/HarmonySase.png";
import Hazel from "./png/Hazel.png";
import Hazeover from "./png/Hazeover.png";
import Heidisql from "./png/Heidisql.png";
import Helium from "./png/Helium.png";
import HexFiend from "./png/HexFiend.png";
import HeyDesktop from "./png/HeyDesktop.png";
import Hiddenbar from "./png/Hiddenbar.png";
import Hides from "./png/Hides.png";
import Hidock from "./png/Hidock.png";
import HighlightAi from "./png/HighlightAi.png";
import HiveApp from "./png/HiveApp.png";
import HomeAssistant from "./png/HomeAssistant.png";
import Homerow from "./png/Homerow.png";
import Hot from "./png/Hot.png";
import Houdahspot from "./png/Houdahspot.png";
import HpEasyAdmin from "./png/HpEasyAdmin.png";
import HpPrimeVirtualCalculator from "./png/HpPrimeVirtualCalculator.png";
import Hubstaff from "./png/Hubstaff.png";
import Huly from "./png/Huly.png";
import Hwmonitor from "./png/Hwmonitor.png";
import Hyper from "./png/Hyper.png";
import Hyperkey from "./png/Hyperkey.png";
import I1Profiler from "./png/I1Profiler.png";
import IbmNotifier from "./png/IbmNotifier.png";
import IbmSemeruJdk11 from "./png/IbmSemeruJdk11.png";
import IbmSemeruJdk17 from "./png/IbmSemeruJdk17.png";
import IbmSemeruJdk21 from "./png/IbmSemeruJdk21.png";
import IbmSemeruJdk8 from "./png/IbmSemeruJdk8.png";
import IbmSemeruJre11 from "./png/IbmSemeruJre11.png";
import IbmSemeruJre17 from "./png/IbmSemeruJre17.png";
import IbmSemeruJre21 from "./png/IbmSemeruJre21.png";
import IbmSemeruJre8 from "./png/IbmSemeruJre8.png";
import IconComposer from "./png/IconComposer.png";
import Iconjar from "./png/Iconjar.png";
import Idagio from "./png/Idagio.png";
import Iexplorer from "./png/Iexplorer.png";
import Iina from "./png/Iina.png";
import Imageglass from "./png/Imageglass.png";
import ImazingConverter from "./png/ImazingConverter.png";
import ImazingHeicConverter from "./png/ImazingHeicConverter.png";
import IMazingProfileEditor from "./png/IMazingProfileEditor.png";
import Imhex from "./png/Imhex.png";
import Inkscape from "./png/Inkscape.png";
import InputSourcePro from "./png/InputSourcePro.png";
import Insomnia from "./png/Insomnia.png";
import Install4J from "./png/Install4J.png";
import Intellidock from "./png/Intellidock.png";
import IntelliJIdea from "./png/IntelliJIdea.png";
import IntelliJIdeaCe from "./png/IntelliJIdeaCe.png";
import IntuneCompanyPortal from "./IntuneCompanyPortal";
import Invesalius from "./png/Invesalius.png";
import iOS from "./iOS";
import iPadOS from "./iPadOS";
import Irfanview from "./png/Irfanview.png";
import Ironpython from "./png/Ironpython.png";
import Isobuster from "./png/Isobuster.png";
import Istherenet from "./png/Istherenet.png";
import ITerm from "./png/ITerm.png";
import Itsycal from "./png/Itsycal.png";
import Itunes from "./png/Itunes.png";
import JabraDirect from "./png/JabraDirect.png";
import Jami from "./png/Jami.png";
import Jamovi from "./png/Jamovi.png";
import Jasp from "./png/Jasp.png";
import Jellyfin from "./png/Jellyfin.png";
import JetBrainsToolbox from "./png/JetBrainsToolbox.png";
import Jiggler from "./png/Jiggler.png";
import JitsiMeet from "./png/JitsiMeet.png";
import Joplin from "./png/Joplin.png";
import JordanbairdIce from "./png/JordanbairdIce.png";
import JuliaApp from "./png/JuliaApp.png";
import JumpDesktop from "./png/JumpDesktop.png";
import Kaleidoscope from "./png/Kaleidoscope.png";
import Kap from "./png/Kap.png";
import Kdenlive from "./png/Kdenlive.png";
import Keepass from "./png/Keepass.png";
import KeePassXc from "./png/KeePassXc.png";
import KeeperPasswordManager from "./png/KeeperPasswordManager.png";
import Keepingyouawake from "./png/Keepingyouawake.png";
import Keeweb from "./png/Keeweb.png";
import Keka from "./png/Keka.png";
import Keyboardcleantool from "./png/Keyboardcleantool.png";
import KeyboardCowboy from "./png/KeyboardCowboy.png";
import KeyboardMaestro from "./png/KeyboardMaestro.png";
import Keycastr from "./png/Keycastr.png";
import Keyclu from "./png/Keyclu.png";
import KeystoreExplorer from "./png/KeystoreExplorer.png";
import Kiro from "./png/Kiro.png";
import KiroCli from "./png/KiroCli.png";
import Kitty from "./png/Kitty.png";
import Klokki from "./png/Klokki.png";
import Knime from "./png/Knime.png";
import Knockknock from "./png/Knockknock.png";
import Krisp from "./png/Krisp.png";
import Krita from "./png/Krita.png";
import Lapce from "./png/Lapce.png";
import LassoApp from "./png/LassoApp.png";
import LastPass from "./png/LastPass.png";
import LastWindowQuits from "./png/LastWindowQuits.png";
import Latest from "./png/Latest.png";
import Launchbar from "./png/Launchbar.png";
import LenovoDockManager from "./png/LenovoDockManager.png";
import LenovoSystemUpdate from "./png/LenovoSystemUpdate.png";
import Lens from "./png/Lens.png";
import LibreOffice from "./png/LibreOffice.png";
import Lightburn from "./png/Lightburn.png";
import Linear from "./png/Linear.png";
import Linearmouse from "./png/Linearmouse.png";
import LingonX from "./png/LingonX.png";
import LinuxOS from "./LinuxOS";
import LittleSnitch from "./png/LittleSnitch.png";
import Local from "./png/Local.png";
import Localsend from "./png/Localsend.png";
import Locationsimulator from "./png/Locationsimulator.png";
import Logioptionsplus from "./png/Logioptionsplus.png";
import LogiTune from "./png/LogiTune.png";
import LogitechUnifyingSoftware from "./png/LogitechUnifyingSoftware.png";
import Logseq from "./png/Logseq.png";
import Lookaway from "./png/Lookaway.png";
import Loom from "./png/Loom.png";
import Loop from "./png/Loop.png";
import Loopback from "./png/Loopback.png";
import LoRain from "./png/LoRain.png";
import Losslesscut from "./png/Losslesscut.png";
import LowProfile from "./png/LowProfile.png";
import LuLu from "./png/LuLu.png";
import Lunacy from "./png/Lunacy.png";
import Lunar from "./png/Lunar.png";
import Lunasea from "./png/Lunasea.png";
import Lunatask from "./png/Lunatask.png";
import Lycheeslicer from "./png/Lycheeslicer.png";
import Maccy from "./png/Maccy.png";
import Macdown from "./png/Macdown.png";
import Mace from "./png/Mace.png";
import Macjournal from "./png/Macjournal.png";
import MacMouseFix from "./png/MacMouseFix.png";
import MacOS from "./MacOS";
import Macpacker from "./png/Macpacker.png";
import Macpass from "./png/Macpass.png";
import Macpilot from "./png/Macpilot.png";
import MacsFanControl from "./png/MacsFanControl.png";
import Macsyzones from "./png/Macsyzones.png";
import Mactracker from "./png/Mactracker.png";
import MacvimApp from "./png/MacvimApp.png";
import Macwhisper from "./png/Macwhisper.png";
import Maestral from "./png/Maestral.png";
import Magicquit from "./png/Magicquit.png";
import Mailspring from "./png/Mailspring.png";
import Malwarebytes from "./png/Malwarebytes.png";
import MarkedApp from "./png/MarkedApp.png";
import Markedit from "./png/Markedit.png";
import MarkText from "./png/MarkText.png";
import Marsedit from "./png/Marsedit.png";
import Marta from "./png/Marta.png";
import Marvel from "./png/Marvel.png";
import Masscode from "./png/Masscode.png";
import Mattermost from "./png/Mattermost.png";
import Max from "./png/Max.png";
import Meetingbar from "./png/Meetingbar.png";
import Megasync from "./png/Megasync.png";
import Mellel from "./png/Mellel.png";
import Melodics from "./png/Melodics.png";
import Memory from "./png/Memory.png";
import Memoryanalyzer from "./png/Memoryanalyzer.png";
import MemoryCleaner from "./png/MemoryCleaner.png";
import MendeleyReferenceManager from "./png/MendeleyReferenceManager.png";
import MenubarStats from "./png/MenubarStats.png";
import Menubarx from "./png/Menubarx.png";
import MerlinProject from "./png/MerlinProject.png";
import Microsoft365Copilot from "./png/Microsoft365Copilot.png";
import MicrosoftAutoUpdate from "./png/MicrosoftAutoUpdate.png";
import MicrosoftAzureStorageExplorer from "./png/MicrosoftAzureStorageExplorer.png";
import MicrosoftDefender from "./png/MicrosoftDefender.png";
import MicrosoftDotnetRuntime from "./png/MicrosoftDotnetRuntime.png";
import MicrosoftEdge from "./png/MicrosoftEdge.png";
import MicrosoftOdbcDriver17 from "./png/MicrosoftOdbcDriver17.png";
import MicrosoftOdbcDriver18 from "./png/MicrosoftOdbcDriver18.png";
import MicrosoftOffice from "./png/MicrosoftOffice.png";
import MicrosoftOleDbDriver19 from "./png/MicrosoftOleDbDriver19.png";
import MicrosoftOneNote from "./png/MicrosoftOneNote.png";
import MicrosoftOutlook from "./png/MicrosoftOutlook.png";
import MicrosoftPowerPoint from "./png/MicrosoftPowerPoint.png";
import MicrosoftRemoteHelp from "./png/MicrosoftRemoteHelp.png";
import Middle from "./png/Middle.png";
import Middleclick from "./png/Middleclick.png";
import Milanote from "./png/Milanote.png";
import Mimecast from "./png/Mimecast.png";
import Mimestream from "./png/Mimestream.png";
import Mindmac from "./png/Mindmac.png";
import MindManager from "./png/MindManager.png";
import Minisim from "./png/Minisim.png";
import Minstaller from "./png/Minstaller.png";
import Miro from "./png/Miro.png";
import Missive from "./png/Missive.png";
import Mist from "./png/Mist.png";
import Mixxx from "./png/Mixxx.png";
import Mobirise from "./png/Mobirise.png";
import Mockoon from "./png/Mockoon.png";
import ModernCsv from "./png/ModernCsv.png";
import MongoDbCompass from "./png/MongoDbCompass.png";
import Monitorcontrol from "./png/Monitorcontrol.png";
import Moom from "./png/Moom.png";
import Moonlight from "./png/Moonlight.png";
import Morgen from "./png/Morgen.png";
import Mos from "./png/Mos.png";
import MountainDuck from "./png/MountainDuck.png";
import MozillaVpn from "./png/MozillaVpn.png";
import Mqttx from "./png/Mqttx.png";
import Mremoteng from "./png/Mremoteng.png";
import MullvadBrowser from "./png/MullvadBrowser.png";
import MullvadVpn from "./png/MullvadVpn.png";
import Multitouch from "./png/Multitouch.png";
import Mural from "./png/Mural.png";
import Museeks from "./png/Museeks.png";
import Musescore from "./png/Musescore.png";
import MxPowerGadget from "./png/MxPowerGadget.png";
import MySqlWorkbench from "./png/MySqlWorkbench.png";
import Nagstamon from "./png/Nagstamon.png";
import NameMangler from "./png/NameMangler.png";
import Naps2 from "./png/Naps2.png";
import NdiTools from "./png/NdiTools.png";
import Neofinder from "./png/Neofinder.png";
import NessusAgent from "./png/NessusAgent.png";
import Netiquette from "./png/Netiquette.png";
import Netnewswire from "./png/Netnewswire.png";
import Netron from "./png/Netron.png";
import Netspot from "./png/Netspot.png";
import Nextcloud from "./png/Nextcloud.png";
import NextcloudTalk from "./png/NextcloudTalk.png";
import Nightfall from "./png/Nightfall.png";
import NitroPdfPro from "./png/NitroPdfPro.png";
import Nodejs from "./png/Nodejs.png";
import Nordlayer from "./png/Nordlayer.png";
import Nordpass from "./png/Nordpass.png";
import NordVpn from "./png/NordVpn.png";
import NosqlWorkbench from "./png/NosqlWorkbench.png";
import Notchnook from "./png/Notchnook.png";
import Notepad from "./png/NotepadPlusPlus.png";
import Notepadexe from "./png/Notepadexe.png";
import Notesnook from "./png/Notesnook.png";
import Notesollama from "./png/Notesollama.png";
import Notion from "./png/Notion.png";
import NotionCalendar from "./png/NotionCalendar.png";
import NounProject from "./png/NounProject.png";
import Nova from "./png/Nova.png";
import Novabench from "./png/Novabench.png";
import Nucleo from "./png/Nucleo.png";
import Nudge from "./png/Nudge.png";
import Numi from "./png/Numi.png";
import Nvda from "./png/Nvda.png";
import NvidiaGeforceNow from "./png/NvidiaGeforceNow.png";
import Obs from "./png/Obs.png";
import Obsidian from "./png/Obsidian.png";
import Ocenaudio from "./png/Ocenaudio.png";
import OkJson from "./png/OkJson.png";
import OktaVerify from "./png/OktaVerify.png";
import Ollama from "./png/Ollama.png";
import Omnidisksweeper from "./png/Omnidisksweeper.png";
import Omnifocus from "./png/Omnifocus.png";
import OmniGraffle from "./png/OmniGraffle.png";
import Omnioutliner from "./png/Omnioutliner.png";
import Omniplan from "./png/Omniplan.png";
import OmnissaHorizonClient from "./png/OmnissaHorizonClient.png";
import OneDrive from "./png/OneDrive.png";
import OnePassword from "./OnePassword";
import OneSwitch from "./png/OneSwitch.png";
import Onionshare from "./png/Onionshare.png";
import Onlyoffice from "./png/Onlyoffice.png";
import OnlySwitch from "./png/OnlySwitch.png";
import OpalComposer from "./png/OpalComposer.png";
import Openaudible from "./png/Openaudible.png";
import Openboard from "./png/Openboard.png";
import Opencloud from "./png/Opencloud.png";
import OpencodeDesktop from "./png/OpencodeDesktop.png";
import Openinterminal from "./png/Openinterminal.png";
import Openlens from "./png/Openlens.png";
import Openmtp from "./png/Openmtp.png";
import Openrct2 from "./png/Openrct2.png";
import Openrefine from "./png/Openrefine.png";
import Opentoonz from "./png/Opentoonz.png";
import OpenvpnConnect from "./png/OpenvpnConnect.png";
import Opera from "./png/Opera.png";
import OptimusPlayer from "./png/OptimusPlayer.png";
import OrbStack from "./png/OrbStack.png";
import OrigamiStudio from "./png/OrigamiStudio.png";
import P4V from "./png/P4V.png";
import Pacifist from "./png/Pacifist.png";
import Package from "./Package";
import PaintDotNet from "./png/PaintDotNet.png";
import PaleMoon from "./png/PaleMoon.png";
import Paletro from "./png/Paletro.png";
import ParallelsDesktop from "./png/ParallelsDesktop.png";
import Pastebot from "./png/Pastebot.png";
import Pcoipclient from "./png/Pcoipclient.png";
import Pd from "./png/Pd.png";
import PdfExpert from "./png/PdfExpert.png";
import PdfPals from "./png/PdfPals.png";
import PdfsamBasic from "./png/PdfsamBasic.png";
import Pearcleaner from "./png/Pearcleaner.png";
import PgAdmin4 from "./png/PgAdmin4.png";
import PhoenixSlides from "./png/PhoenixSlides.png";
import Photosrevive from "./png/Photosrevive.png";
import Photostickies from "./png/Photostickies.png";
import PhpStorm from "./png/PhpStorm.png";
import Pibar from "./png/Pibar.png";
import Picview from "./png/Picview.png";
import Piezo from "./png/Piezo.png";
import Pika from "./png/Pika.png";
import Piphero from "./png/Piphero.png";
import Pixelsnap from "./png/Pixelsnap.png";
import PlantronicsHub from "./png/PlantronicsHub.png";
import Platypus from "./png/Platypus.png";
import Plex from "./png/Plex.png";
import PlexHtpc from "./png/PlexHtpc.png";
import PlexMediaServer from "./png/PlexMediaServer.png";
import PlisteditPro from "./png/PlisteditPro.png";
import Plugdata from "./png/Plugdata.png";
import PodmanDesktop from "./png/PodmanDesktop.png";
import Popchar from "./png/Popchar.png";
import Popclip from "./png/Popclip.png";
import Popsql from "./png/Popsql.png";
import Portfolioperformance from "./png/Portfolioperformance.png";
import PostgresApp from "./png/PostgresApp.png";
import Postgresql15 from "./png/Postgresql15.png";
import Postgresql16 from "./png/Postgresql16.png";
import Postgresql17 from "./png/Postgresql17.png";
import Postgresql18 from "./png/Postgresql18.png";
import Postico from "./png/Postico.png";
import Postman from "./png/Postman.png";
import PowerAutomate from "./png/PowerAutomate.png";
import PowerBi from "./png/PowerBi.png";
import PowerMonitor from "./png/PowerMonitor.png";
import Powerphotos from "./png/Powerphotos.png";
import Powershell from "./png/Powershell.png";
import Powertoys from "./png/Powertoys.png";
import PppcUtility from "./png/PppcUtility.png";
import Preform from "./png/Preform.png";
import Principle from "./png/Principle.png";
import Prism from "./png/Prism.png";
import Prisma from "./png/Prisma.png";
import Pritunl from "./png/Pritunl.png";
import Privileges from "./png/Privileges.png";
import Prizmo from "./png/Prizmo.png";
import Processing from "./png/Processing.png";
import Processspy from "./png/Processspy.png";
import Pronotes from "./png/Pronotes.png";
import ProtonDrive from "./png/ProtonDrive.png";
import ProtonMail from "./png/ProtonMail.png";
import ProtonMailBridge from "./png/ProtonMailBridge.png";
import ProtonMeet from "./png/ProtonMeet.png";
import ProtonPass from "./png/ProtonPass.png";
import ProtonVpn from "./png/ProtonVpn.png";
import Protopie from "./png/Protopie.png";
import Proxifier from "./png/Proxifier.png";
import Proxyman from "./png/Proxyman.png";
import Pulsar from "./png/Pulsar.png";
import Purevpn from "./png/Purevpn.png";
import Putty from "./png/Putty.png";
import PyCharm from "./png/PyCharm.png";
import PyCharmCe from "./png/PyCharmCe.png";
import Python313 from "./png/Python313.png";
import Python314 from "./png/Python314.png";
import Qemu from "./png/Qemu.png";
import Qlab from "./png/Qlab.png";
import Qlmarkdown from "./png/Qlmarkdown.png";
import QspacePro from "./png/QspacePro.png";
import Quip from "./png/Quip.png";
import Qview from "./png/Qview.png";
import R from "./png/R.png";
import RadioSilence from "./png/RadioSilence.png";
import Raindropio from "./png/Raindropio.png";
import RancherDesktop from "./png/RancherDesktop.png";
import RapidApi from "./png/RapidApi.png";
import Rapidweaver from "./png/Rapidweaver.png";
import RaspberryPiImager from "./png/RaspberryPiImager.png";
import Raycast from "./png/Raycast.png";
import Readest from "./png/Readest.png";
import RealVncServer from "./png/RealVncServer.png";
import Reaper from "./png/Reaper.png";
import Recents from "./png/Recents.png";
import Rectangle from "./png/Rectangle.png";
import RectanglePro from "./png/RectanglePro.png";
import Recut from "./png/Recut.png";
import RedcineXPro from "./png/RedcineXPro.png";
import RedisPro from "./png/RedisPro.png";
import Reflector from "./png/Reflector.png";
import RemindersMenubar from "./png/RemindersMenubar.png";
import RemoteBuddy from "./png/RemoteBuddy.png";
import RemoteDesktopManager from "./png/RemoteDesktopManager.png";
import Reqable from "./png/Reqable.png";
import Requestly from "./png/Requestly.png";
import Resharper from "./png/Resharper.png";
import Retcon from "./png/Retcon.png";
import Retroarch from "./png/Retroarch.png";
import Retrobatch from "./png/Retrobatch.png";
import Rewritebar from "./png/Rewritebar.png";
import Rider from "./png/Rider.png";
import Rightfont from "./png/Rightfont.png";
import Ringcentral from "./png/Ringcentral.png";
import Rive from "./png/Rive.png";
import Rize from "./png/Rize.png";
import Robofont from "./png/Robofont.png";
import Roboform from "./png/Roboform.png";
import Rocket from "./png/Rocket.png";
import RocketChat from "./png/RocketChat.png";
import RocketmanChoicesPackager from "./png/RocketmanChoicesPackager.png";
import RocketTypist from "./png/RocketTypist.png";
import RoyalTsx from "./png/RoyalTsx.png";
import Rstudio from "./png/Rstudio.png";
import Rsyncui from "./png/Rsyncui.png";
import Rtools from "./png/Rtools.png";
import RubyMine from "./png/RubyMine.png";
import Runjs from "./png/Runjs.png";
import RustDesk from "./png/RustDesk.png";
import RustRover from "./png/RustRover.png";
import Sabnzbd from "./png/Sabnzbd.png";
import Safari from "./Safari";
import SafeExamBrowser from "./png/SafeExamBrowser.png";
import Sanesidebuttons from "./png/Sanesidebuttons.png";
import Santa from "./png/Santa.png";
import ScaleFt from "./png/ScaleFt.png";
import ScMenu from "./png/ScMenu.png";
import Scratch from "./png/Scratch.png";
import Screenflick from "./png/Screenflick.png";
import Screenflow from "./png/Screenflow.png";
import Screenfocus from "./png/Screenfocus.png";
import ScreenStudio from "./png/ScreenStudio.png";
import Scribe from "./png/Scribe.png";
import Scribus from "./png/Scribus.png";
import Scrivener from "./png/Scrivener.png";
import Secretive from "./png/Secretive.png";
import Securesafe from "./png/Securesafe.png";
import Selfcontrol from "./png/Selfcontrol.png";
import Sensei from "./png/Sensei.png";
import SequelAce from "./png/SequelAce.png";
import Session from "./png/Session.png";
import Setapp from "./png/Setapp.png";
import SevenZip from "./png/7Zip.png";
import SfSymbols from "./png/SfSymbols.png";
import Shapr3D from "./png/Shapr3D.png";
import Sharefile from "./png/Sharefile.png";
import Shift from "./png/Shift.png";
import Shifty from "./png/Shifty.png";
import Shortcat from "./png/Shortcat.png";
import Shotcut from "./png/Shotcut.png";
import Shottr from "./png/Shottr.png";
import Sidenotes from "./png/Sidenotes.png";
import Sigmaos from "./png/Sigmaos.png";
import Signal from "./png/Signal.png";
import SimpleComic from "./png/SimpleComic.png";
import Sirimote from "./png/Sirimote.png";
import Sketch from "./png/Sketch.png";
import Slab from "./png/Slab.png";
import Slack from "./Slack";
import Slicer from "./png/Slicer.png";
import Slidepad from "./png/Slidepad.png";
import Sloth from "./png/Sloth.png";
import SmallstepAgent from "./png/SmallstepAgent.png";
import Smartsheet from "./png/Smartsheet.png";
import Smartsvn from "./png/Smartsvn.png";
import Smoothscroll from "./png/Smoothscroll.png";
import Smultron from "./png/Smultron.png";
import Snagit from "./png/Snagit.png";
import Snapmotion from "./png/Snapmotion.png";
import SnowflakeSnowsql from "./png/SnowflakeSnowsql.png";
import Sococo from "./png/Sococo.png";
import SonicVisualiser from "./png/SonicVisualiser.png";
import SonicwallNetextender from "./png/SonicwallNetextender.png";
import Sonobus from "./png/Sonobus.png";
import Sonos from "./png/Sonos.png";
import SonyPsRemotePlay from "./png/SonyPsRemotePlay.png";
import Soulver from "./png/Soulver.png";
import Soundanchor from "./png/Soundanchor.png";
import SoundControl from "./png/SoundControl.png";
import SoundSiphon from "./png/SoundSiphon.png";
import Soundsource from "./png/Soundsource.png";
import Sourcetree from "./png/Sourcetree.png";
import Spamsieve from "./png/Spamsieve.png";
import SpectraApp from "./png/SpectraApp.png";
import SpitfireAudio from "./png/SpitfireAudio.png";
import SplashtopBusiness from "./png/SplashtopBusiness.png";
import SplashtopStreamer from "./png/SplashtopStreamer.png";
import Splice from "./png/Splice.png";
import Spokenly from "./png/Spokenly.png";
import Spotify from "./png/Spotify.png";
import Spyder from "./png/Spyder.png";
import Sqlectron from "./png/Sqlectron.png";
import SqlproForMssql from "./png/SqlproForMssql.png";
import SqlproForMysql from "./png/SqlproForMysql.png";
import SqlproForPostgres from "./png/SqlproForPostgres.png";
import SqlproForSqlite from "./png/SqlproForSqlite.png";
import SqlproStudio from "./png/SqlproStudio.png";
import SqlServerManagementStudio from "./png/SqlServerManagementStudio.png";
import Squash from "./png/Squash.png";
import SshConfigEditor from "./png/SshConfigEditor.png";
import StandardNotes from "./png/StandardNotes.png";
import Staruml from "./png/Staruml.png";
import Stats from "./png/Stats.png";
import Steam from "./png/Steam.png";
import Steermouse from "./png/Steermouse.png";
import Stellarium from "./png/Stellarium.png";
import Stillcolor from "./png/Stillcolor.png";
import Stretchly from "./png/Stretchly.png";
import SublimeMerge from "./png/SublimeMerge.png";
import SublimeText from "./png/SublimeText.png";
import Supercollider from "./png/Supercollider.png";
import Superhuman from "./png/Superhuman.png";
import Superkey from "./png/Superkey.png";
import SuperProductivity from "./png/SuperProductivity.png";
import Superwhisper from "./png/Superwhisper.png";
import Supportcompanion from "./png/Supportcompanion.png";
import Surfshark from "./png/Surfshark.png";
import Surge from "./png/Surge.png";
import SuspiciousPackage from "./png/SuspiciousPackage.png";
import Swiftbar from "./png/Swiftbar.png";
import Swiftdialog from "./png/Swiftdialog.png";
import Swifty from "./png/Swifty.png";
import Swish from "./png/Swish.png";
import Sync from "./png/Sync.png";
import Syncmate from "./png/Syncmate.png";
import Syncovery from "./png/Syncovery.png";
import SyncthingApp from "./png/SyncthingApp.png";
import SyntaxHighlight from "./png/SyntaxHighlight.png";
import Tabby from "./png/Tabby.png";
import TableauDesktop from "./png/TableauDesktop.png";
import TableauPrep from "./png/TableauPrep.png";
import TablePlus from "./png/TablePlus.png";
import Tabtab from "./png/Tabtab.png";
import Tailscale from "./png/Tailscale.png";
import Taskade from "./png/Taskade.png";
import Taskbar from "./png/Taskbar.png";
import Teacode from "./png/Teacode.png";
import Teams from "./Teams";
import TeamViewer from "./TeamViewer";
import Telegram from "./png/Telegram.png";
import TeleportConnect from "./png/TeleportConnect.png";
import Terminal from "./png/Terminal.png";
import Termius from "./png/Termius.png";
import TexLiveUtility from "./png/TexLiveUtility.png";
import Texshop from "./png/Texshop.png";
import TextExpander from "./png/TextExpander.png";
import Thaw from "./png/Thaw.png";
import TheUnarchiver from "./png/TheUnarchiver.png";
import Thorium from "./png/Thorium.png";
import ThreeDfZephyrFree from "./png/3DfZephyrFree.png";
import Threema from "./png/Threema.png";
import Thumbsup from "./png/Thumbsup.png";
import Thunderbird from "./png/Thunderbird.png";
import Ticktick from "./png/Ticktick.png";
import Tidal from "./png/Tidal.png";
import Tightvnc from "./png/Tightvnc.png";
import Tiles from "./png/Tiles.png";
import Timescribe from "./png/Timescribe.png";
import Timing from "./png/Timing.png";
import Todoist from "./png/Todoist.png";
import TopazGigapixelAi from "./png/TopazGigapixelAi.png";
import TopazPhotoAi from "./png/TopazPhotoAi.png";
import TopazVideoAi from "./png/TopazVideoAi.png";
import Topnotch from "./png/Topnotch.png";
import TorBrowser from "./png/TorBrowser.png";
import Tortoisegit from "./png/Tortoisegit.png";
import Tower from "./png/Tower.png";
import Tradingview from "./png/Tradingview.png";
import Transfer from "./png/Transfer.png";
import Transmission from "./png/Transmission.png";
import Transmit from "./png/Transmit.png";
import Trex from "./png/Trex.png";
import TrezorSuite from "./png/TrezorSuite.png";
import Tripmode from "./png/Tripmode.png";
import Tunnelblick from "./png/Tunnelblick.png";
import Tuple from "./png/Tuple.png";
import TwineApp from "./png/TwineApp.png";
import Twingate from "./png/Twingate.png";
import Twobird from "./png/Twobird.png";
import Typeface from "./png/Typeface.png";
import Typinator from "./png/Typinator.png";
import Typora from "./png/Typora.png";
import UaConnect from "./png/UaConnect.png";
import Ukelele from "./png/Ukelele.png";
import UltimakerCura from "./png/UltimakerCura.png";
import Unclutter from "./png/Unclutter.png";
import Unicodechecker from "./png/Unicodechecker.png";
import UnityHub from "./png/UnityHub.png";
import Updf from "./png/Updf.png";
import Upscayl from "./png/Upscayl.png";
import UsageApp from "./png/UsageApp.png";
import Utm from "./png/Utm.png";
import Vagrant from "./png/Vagrant.png";
import Vanilla from "./png/Vanilla.png";
import VcRedistX64 from "./png/VcRedistX64.png";
import Vellum from "./png/Vellum.png";
import VernierSpectralAnalysis from "./png/VernierSpectralAnalysis.png";
import Versions from "./png/Versions.png";
import Via from "./png/Via.png";
import Vim from "./png/Vim.png";
import Vimcal from "./png/Vimcal.png";
import VirtualBox from "./png/VirtualBox.png";
import VirtualBuddy from "./png/VirtualBuddy.png";
import Viscosity from "./png/Viscosity.png";
import VisualParadigm from "./png/VisualParadigm.png";
import VisualStudio2022 from "./png/VisualStudio2022.png";
import VisualStudioCode from "./VisualStudioCode";
import Vivaldi from "./png/Vivaldi.png";
import VividApp from "./png/VividApp.png";
import Viz from "./png/Viz.png";
import Vlc from "./png/Vlc.png";
import VncViewer from "./png/VncViewer.png";
import Voiceink from "./png/Voiceink.png";
import VpnTracker365 from "./png/VpnTracker365.png";
import VsCodium from "./png/VsCodium.png";
import Vuescan from "./png/Vuescan.png";
import Vyprvpn from "./png/Vyprvpn.png";
import Vysor from "./png/Vysor.png";
import WacomCenter from "./png/WacomCenter.png";
import Warp from "./png/Warp.png";
import Wave from "./png/Wave.png";
import Wavebox from "./png/Wavebox.png";
import Wealthfolio from "./png/Wealthfolio.png";
import Weasis from "./png/Weasis.png";
import Webcatalog from "./png/Webcatalog.png";
import Webex from "./png/Webex.png";
import WebStorm from "./png/WebStorm.png";
import Wechat from "./png/Wechat.png";
import Weektodo from "./png/Weektodo.png";
import Whatroute from "./png/Whatroute.png";
import WhatsApp from "./WhatsApp";
import Whisky from "./png/Whisky.png";
import Whispering from "./png/Whispering.png";
import Wifiman from "./png/Wifiman.png";
import Windirstat from "./png/Windirstat.png";
import Windowkeys from "./png/Windowkeys.png";
import WindowsApp from "./png/WindowsApp.png";
import WindowsAppRemote from "./WindowsAppRemote";
import WindowsDefender from "./WindowsDefender";
import WindowsOS from "./WindowsOS";
import Windsurf from "./png/Windsurf.png";
import Winlogbeat from "./png/Winlogbeat.png";
import Winrar from "./png/Winrar.png";
import Wins from "./png/Wins.png";
import Winscp from "./png/Winscp.png";
import Wireshark from "./png/Wireshark.png";
import WisprFlow from "./png/WisprFlow.png";
import WondershareEdrawmax from "./png/WondershareEdrawmax.png";
import WondershareFilmora from "./png/WondershareFilmora.png";
import Word from "./Word";
import Wordservice from "./png/Wordservice.png";
import Workflowy from "./png/Workflowy.png";
import WorksheetCrafter from "./png/WorksheetCrafter.png";
import Workspaces from "./png/Workspaces.png";
import WrikeForMac from "./png/WrikeForMac.png";
import Xca from "./png/Xca.png";
import XCreds from "./png/XCreds.png";
import Xld from "./png/Xld.png";
import Xmenu from "./png/Xmenu.png";
import Xmplify from "./png/Xmplify.png";
import Xnapper from "./png/Xnapper.png";
import Xnconvert from "./png/Xnconvert.png";
import Xnviewmp from "./png/Xnviewmp.png";
import Xquartz from "./png/Xquartz.png";
import Yaak from "./png/Yaak.png";
import Yacreader from "./png/Yacreader.png";
import Yarn from "./png/Yarn.png";
import Yattee from "./png/Yattee.png";
import Yippy from "./png/Yippy.png";
import YtMusic from "./png/YtMusic.png";
import YubicoAuthenticator from "./png/YubicoAuthenticator.png";
import YubikeyManager from "./png/YubikeyManager.png";
import Zappy from "./png/Zappy.png";
import Zed from "./png/Zed.png";
import Zen from "./png/Zen.png";
import Zeplin from "./png/Zeplin.png";
import ZeroOneZeroEditor from "./png/010Editor.png";
import Zettlr from "./png/Zettlr.png";
import Zight from "./png/Zight.png";
import Zoom from "./Zoom";
import ZoomOutlookPlugin from "./png/ZoomOutlookPlugin.png";
import ZoomRooms from "./png/ZoomRooms.png";
import Zotero from "./png/Zotero.png";
import Zulip from "./png/Zulip.png";
import Zwift from "./png/Zwift.png";
// SOFTWARE_NAME_TO_ICON_MAP list "special" applications that have a defined
// icon for them, keys refer to application names, and are intended to be fuzzy
// matched in the application logic.
export const SOFTWARE_NAME_TO_ICON_MAP = {
  "010 editor": ZeroOneZeroEditor,
  "1password": OnePassword,
  "3d slicer": Slicer,
  "3df zephyr free": ThreeDfZephyrFree,
  "4k slideshow maker": FourKSlideshowMaker,
  "4k stogram": FourKStogram,
  "4k video downloader": FourKVideoDownloader,
  "4k video downloader+": FourKVideoDownloaderPlus,
  "4k video to mp3": FourKVideoToMp3,
  "4k youtube to mp3": FourKYoutubeToMp3,
  "7 zip": SevenZip,
  "7-zip": SevenZip,
  "8x8 work": EightXEightWork,
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
  "advanced installer": AdvancedInstaller,
  "advanced renamer": AdvancedRenamer,
  affinity: Affinity,
  "affinity designer": AffinityDesigner1,
  "affinity designer 2": AffinityDesigner,
  "affinity photo": AffinityPhoto1,
  "affinity photo 2": AffinityPhoto,
  "affinity publisher": AffinityPublisher1,
  "affinity publisher 2": AffinityPublisher,
  "agent ransack": AgentRansack,
  "air explorer": AirExplorer,
  airbuddy: Airbuddy,
  aircall: Aircall,
  airdroid: Airdroid,
  airparrot: Airparrot,
  airserver: Airserver,
  airtable: Airtable,
  airtame: Airtame,
  airy: Airy,
  akiflow: Akiflow,
  alacritty: Alacritty,
  alcove: Alcove,
  aldente: Aldente,
  alfaview: Alfaview,
  alloy: Alloy,
  "allway sync": AllwaySync,
  "altair graphql client": AltairGraphqlClient,
  alttab: AltTab,
  "amadeus pro": AmadeusPro,
  amadine: Amadine,
  "amazon chime": AmazonChime,
  "amazon corretto 11": AmazonCorretto21,
  "amazon corretto 17": AmazonCorretto21,
  "amazon corretto 21": AmazonCorretto21,
  "amazon corretto 24": AmazonCorretto24,
  "amazon corretto 25": AmazonCorretto25,
  "amazon corretto 26": AmazonCorretto26,
  "amazon corretto 8": AmazonCorretto21,
  "amazon corretto jre 8": AmazonCorretto21,
  "amazon dcv": AmazonDCV,
  "amazon redshift odbc driver": AmazonRedshiftOdbcDriver,
  "amazon workspaces": AmazonWorkspaces,
  amethyst: Amethyst,
  amie: Amie,
  "android studio": AndroidStudio,
  androidPlayStore: AndroidPlayStore,
  "angry ip scanner": AngryIpScanner,
  anka: Anka,
  "another redis desktop manager": AnotherRedisDesktopManager,
  antigravity: Antigravity,
  "antigravity ide": AntigravityIde,
  antinote: Antinote,
  "any.do": Anydo,
  anyburn: Anyburn,
  anydesk: AnyDesk,
  anytype: Anytype,
  "aomei backupper standard": AomeiBackupperStandard,
  apidog: Apidog,
  "app fair": AppFair,
  apparency: Apparency,
  appcleaner: AppCleaner,
  "appium inspector gui": AppiumInspector,
  appleAppStore: AppleAppStore,
  applite: Applite,
  aptakube: Aptakube,
  arc: Arc,
  archaeology: Archaeology,
  "arduino ide": ArduinoIde,
  asana: Asana,
  "asset catalog tinkerer": AssetCatalogTinkerer,
  atext: Atext,
  audacity: Audacity,
  "audio hijack": AudioHijack,
  audiveris: Audiveris,
  autopsy: Autopsy,
  avast: AvastSecureBrowser,
  "aviatrix vpn client": AviatrixVpnClient,
  "avs image converter": AvsImageConverter,
  "avs media player": AvsMediaPlayer,
  "aws client vpn": AwsVpnClient,
  "aws command line interface": AwsCli,
  "aws sam command line interface": AwsSamCli,
  "aws session manager plugin": AwsCli,
  "aws vpn client": AwsVpnClient,
  "axure rp": AxureRp,
  "azul zulu jdk": AzulZulu25Jdk,
  "azul zulu jre": AzulZulu25Jre,
  "azure data studio": AzureDataStudio,
  "azure functions core tools": AzureFunctionsCoreTools,
  backblaze: Backblaze,
  "background music": BackgroundMusic,
  badgeify: Badgeify,
  balenaetcher: BalenaEtcher,
  "balsamiq wireframes": BalsamiqWireframes,
  "bambu studio": BambuStudio,
  bandiview: Bandiview,
  bartender: Bartender,
  batfi: Batfi,
  bbedit: BBEdit,
  bdash: Bdash,
  "beaver notes": BeaverNotes,
  "beekeeper studio": BeekeeperStudio,
  beeper: Beeper,
  betterdisplay: BetterDisplay,
  bettermouse: Bettermouse,
  bettertouchtool: Bettertouchtool,
  betterzip: Betterzip,
  "beyond compare": BeyondCompare,
  bezel: Bezel,
  bibdesk: Bibdesk,
  binance: Binance,
  biscuit: Biscuit,
  bitbox: Bitbox,
  "bitfocus companion": Companion,
  bitrix24: Bitrix24,
  bitwarden: Bitwarden,
  "bitwig studio": BitwigStudio,
  bleachbit: Bleachbit,
  blender: Blender,
  bleunlock: Bleunlock,
  blip: Blip,
  bluej: Bluej,
  bluewallet: Bluewallet,
  blurscreen: Blurscreen,
  "boltai 2": Boltai,
  "bome network": BomeNetwork,
  "boom 3d": Boom3D,
  boop: Boop,
  "boost note": BoostNote,
  box: Box,
  "box tools": BoxTools,
  brave: Brave,
  breaktimer: Breaktimer,
  "bricklink studio": BricklinkStudio,
  browserstacklocal: Browserstacklocal,
  bruno: Bruno,
  "bulk crap uninstaller": BulkCrapUninstaller,
  bunch: Bunch,
  "burp suite community": BurpSuiteCommunity,
  "burp suite professional": BurpSuiteProfessional,
  busycontacts: Busycontacts,
  buttercup: Buttercup,
  buzz: Buzz,
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
  capto: Capto,
  "carbon copy cloner": CarbonCopyCloner,
  cardhop: Cardhop,
  cavalry: Cavalry,
  cellprofiler: Cellprofiler,
  "certify the web": CertifyTheWeb,
  chalk: Chalk,
  charles: Charles,
  charmstone: Charmstone,
  chatbox: Chatbox,
  chatgpt: ChatGpt,
  "chatgpt atlas": ChatGptAtlas,
  chatwise: Chatwise,
  cheetah3d: Cheetah3D,
  "chef workstation": ChefWorkstation,
  "cherry keys": CherryKeys,
  "cherry studio": CherryStudio,
  chime: Chime,
  choosy: Choosy,
  "chrome remote desktop": ChromeRemoteDesktop,
  "cinc workstation": Cinc,
  "cisco jabber": CiscoJabber,
  "cisco webex recorder and player": CiscoWebexRecorderAndPlayer,
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
  clipboardfusion: Clipboardfusion,
  clipbook: Clipbook,
  clipgrab: Clipgrab,
  clipy: Clipy,
  clockassist: Clockassist,
  clocker: Clocker,
  "clockify desktop": ClockifyDesktop,
  clop: Clop,
  cloudflare: Cloudflare,
  cmake: CmakeApp,
  cmux: Cmux,
  coconutbattery: Coconutbattery,
  code: VisualStudioCode,
  codeedit: Codeedit,
  "codemeter runtime kit": CodemeterRuntimeKit,
  coderunner: Coderunner,
  codex: CodexApp,
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
  "cpu-z": CpuZ,
  crashplan: CrashPlan,
  "creative force kelvin": CreativeForceKelvin,
  "creative force triad": CreativeForceTriad,
  "crestron airmedia": CrestronAirmedia,
  "crestron airmedia peripherals": CrestronAirmediaPeripherals,
  "cribl edge": CriblEdge,
  crisisgo: Crisisgo,
  crossover: Crossover,
  cryptomator: Cryptomator,
  crystaldiskmark: Crystaldiskmark,
  crystalfetch: Crystalfetch,
  "cube browser": CubeBrowser,
  cursor: Cursor,
  cursorsense: Cursorsense,
  cursr: Cursr,
  customshortcuts: Customshortcuts,
  cyberduck: Cyberduck,
  "cyberduck cli": CyberduckCli,
  daisydisk: Daisydisk,
  dangerzone: Dangerzone,
  "dante controller": DanteController,
  darkmodebuddy: Darkmodebuddy,
  darktable: Darktable,
  dash: Dash,
  dataflare: Dataflare,
  datagrip: DataGrip,
  dataspell: Dataspell,
  "dax studio": DaxStudio,
  dayflow: Dayflow,
  "db browser for sqlite": DbBrowserForSqLite,
  dbeaver: DBeaver,
  "dbeaver community": DBeaver,
  "dbeaver enterprise edition": DBeaverEe,
  "dbeaver lite edition": DBeaverLite,
  "dbeaver ultimate edition": DBeaverUltimate,
  dbeaveree: DBeaverEe,
  dbeaverlite: DBeaverLite,
  dbeaverultimate: DBeaverUltimate,
  dbgate: Dbgate,
  dbvisualizer: Dbvisualizer,
  debookee: Debookee,
  deckset: Deckset,
  deepl: DeepL,
  deezer: Deezer,
  "default folder x": DefaultFolderX,
  "delinea connection manager": DelineaConnectionManager,
  "dell command update": DellCommandUpdate,
  "dell display and peripheral manager": DellDisplayAndPeripheralManager,
  descript: Descript,
  deskpad: Deskpad,
  desktime: Desktime,
  "devin desktop": DevinDesktop,
  devknife: Devknife,
  "devolutions launcher": DevolutionsLauncher,
  "devolutions workspace": DevolutionsWorkspace,
  "devonsphere express": DevonsphereExpress,
  devonthink: Devonthink,
  devpod: Devpod,
  devtoys: Devtoys,
  devutils: Devutils,
  "dfu blaster pro": DfuBlasterPro,
  dialpad: Dialpad,
  dictionaries: Dictionaries,
  "diffusion bee": Diffusionbee,
  digikam: Digikam,
  "digiseal reader": DigisealReader,
  "directory opus": DirectoryOpus,
  discord: Discord,
  "disk drill": DiskDrill,
  "DisplayLink USB Graphics Software": DisplayLinkManager,
  "dng converter": AdobeDngConverter,
  dngrep: Dngrep,
  dockdoor: Dockdoor,
  docker: Docker,
  dockfix: Dockfix,
  dockside: Dockside,
  dockview: Dockview,
  dot: Dot,
  doughnut: Doughnut,
  downie: Downie,
  "draftable desktop": DraftableDesktop,
  "drata agent": DrataAgent,
  "draw.io": Drawio,
  drawbot: Drawbot,
  drofus: Drofus,
  dropbox: Dropbox,
  dropdmg: Dropdmg,
  droplr: Droplr,
  dropshare: Dropshare,
  dropzone: Dropzone,
  "druva insync": DruvaInSync,
  duckduckgo: Duckduckgo,
  duet: Duet,
  "duo desktop": DuoDesktop,
  dupeguru: Dupeguru,
  "dymo connect": DymoConnect,
  "dymo id": DymoId,
  dynalist: Dynalist,
  eaglefiler: Eaglefiler,
  easydict: Easydict,
  easyfind: Easyfind,
  eclipse: Eclipse,
  "eclipse memory analyzer": Memoryanalyzer,
  "eclipse temurin jdk 11": EclipseTemurinJdk11,
  "eclipse temurin jdk 17": EclipseTemurinJdk17,
  "eclipse temurin jdk 21": EclipseTemurinJdk21,
  "eclipse temurin jdk 8": EclipseTemurinJdk8,
  "eclipse temurin jre 11": EclipseTemurinJre11,
  "eclipse temurin jre 17": EclipseTemurinJre17,
  "eclipse temurin jre 21": EclipseTemurinJre21,
  "eclipse temurin jre 8": EclipseTemurinJre8,
  edge: MicrosoftEdge,
  edrawmax: WondershareEdrawmax,
  egnyte: Egnyte,
  "egnyte webedit": EgnyteWebedit,
  electronmail: Electronmail,
  electrum: Electrum,
  element: Element,
  elephas: Elephas,
  "elevate uc": ElevateUc,
  "elgato camera hub": ElgatoCameraHub,
  "elgato capture device utility": ElgatoCaptureDeviceUtility,
  "elgato control center": ElgatoControlCenter,
  "elgato game capture hd": ElgatoGameCaptureHd,
  "elgato stream deck": ElgatoStreamDeck,
  "elgato wave link": ElgatoWaveLink,
  "elmedia player": ElmediaPlayer,
  "eltima cloudmounter": Cloudmounter,
  "em client": Emclient,
  endnote: Endnote,
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
  "f.lux": FluxApp,
  falcon: Falcon,
  fantastical: Fantastical,
  far2l: Far2L,
  farrago: Farrago,
  fastmail: Fastmail,
  fastscripts: Fastscripts,
  fellow: Fellow,
  ferdium: Ferdium,
  fetch: FetchApp,
  figma: Figma,
  "file juicer": FileJuicer,
  filebeat: Filebeat,
  "filemaker pro": FileMakerPro,
  filen: Filen,
  "fing desktop": Fing,
  "fire alpaca": Firealpaca,
  firefly: FireflyIotaDesktop,
  "firefly shimmer": FireflyShimmer,
  firefox: Firefox,
  "firefox developer edition": FirefoxDeveloperEdition,
  "firefox nightly": FirefoxNightly,
  fission: Fission,
  "fleet desktop": FleetDesktop,
  "flexoptix app": Flexoptix,
  flexwhere: Flexwhere,
  fluid: Fluid,
  "focusrite control 2": FocusriteControl2,
  folx: Folx,
  fontbase: Fontbase,
  fontlab: Fontlab,
  forecast: Forecast,
  fork: Fork,
  forklift: Forklift,
  fortify: Fortify,
  "foxit pdf editor": FoxitPdfEditor,
  "foxit pdf reader": FoxitPdfReader,
  framer: Framer,
  franz: Franz,
  "free download manager": FreeDownloadManager,
  freecad: Freecad,
  freefilesync: Freefilesync,
  front: Front,
  fsmonitor: Fsmonitor,
  funter: Funter,
  "galaxy modeler": GalaxyModeler,
  "garmin basecamp": GarminBasecamp,
  "garmin express": GarminExpress,
  "gather town": Gather,
  gdevelop: Gdevelop,
  geany: Geany,
  geekbench: Geekbench,
  gemini: GoogleGemini,
  "gemini 2": Gemini2,
  "genesys cloud": GenesysCloud,
  "geogebra classic": GeogebraClassic,
  gephi: Gephi,
  ghostty: Ghostty,
  gimp: Gimp,
  git: Git,
  "git extensions": GitExtensions,
  gitfinder: Gitfinder,
  "github copilot for xcode": GithubCopilotForXcode,
  "github desktop": GitHubDesktop,
  gitify: Gitify,
  gitkraken: GitKraken,
  gitup: GitupApp,
  glyphs: Glyphs,
  "gnu privacy guard": Gnupg,
  go: Go,
  go2shell: Go2Shell,
  "goanywhere openpgp studio": GoanywhereOpenpgpStudio,
  "godot engine": Godot,
  godspeed: Godspeed,
  "gog galaxy": GogGalaxy,
  goland: GoLand,
  "goldendict-ng": GoldendictNg,
  goodsync: Goodsync,
  "google ads editor": GoogleAdsEditor,
  "google antigravity": Antigravity,
  "google antigravity ide": AntigravityIde,
  "google chrome": ChromeApp,
  "google credential provider for windows": GoogleCredentialProviderForWindows,
  "google drive": GoogleDrive,
  "google earth pro": GoogleEarthPro,
  "google gemini": GoogleGemini,
  "google web designer": GoogleWebDesigner,
  gotomeeting: GoToMeeting,
  "gpg keychain": GpgKeychain,
  "gpg suite": GpgKeychain,
  gpg4win: Gpg4Win,
  gpodder: Gpodder,
  grammarly: GrammarlyDesktop,
  grandperspective: Grandperspective,
  granola: Granola,
  "graphpad prism": Prism,
  graphviz: Graphviz,
  grepwin: Grepwin,
  grids: Grids,
  "groove omnidialer": GrooveOmniDialer,
  hammerspoon: Hammerspoon,
  handbrake: HandbrakeApp,
  "harmony sase": HarmonySase,
  hazel: Hazel,
  hazeover: Hazeover,
  heidisql: Heidisql,
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
  "hp prime virtual calculator": HpPrimeVirtualCalculator,
  hubstaff: Hubstaff,
  huly: Huly,
  hwmonitor: Hwmonitor,
  hyper: Hyper,
  hyperkey: Hyperkey,
  i1profiler: I1Profiler,
  "ibm notifier": IbmNotifier,
  "ibm semeru runtime open edition jdk 11": IbmSemeruJdk11,
  "ibm semeru runtime open edition jdk 17": IbmSemeruJdk17,
  "ibm semeru runtime open edition jdk 21": IbmSemeruJdk21,
  "ibm semeru runtime open edition jdk 8": IbmSemeruJdk8,
  "ibm semeru runtime open edition jre 11": IbmSemeruJre11,
  "ibm semeru runtime open edition jre 17": IbmSemeruJre17,
  "ibm semeru runtime open edition jre 21": IbmSemeruJre21,
  "ibm semeru runtime open edition jre 8": IbmSemeruJre8,
  ice: JordanbairdIce,
  "icon composer": IconComposer,
  iconjar: Iconjar,
  idagio: Idagio,
  iexplorer: Iexplorer,
  iina: Iina,
  imageglass: Imageglass,
  imazing: IMazingProfileEditor,
  "imazing converter": ImazingConverter,
  "imazing heic converter": ImazingHeicConverter,
  "imazing profile editor": IMazingProfileEditor,
  imhex: Imhex,
  inkscape: Inkscape,
  "input source pro": InputSourcePro,
  insomnia: Insomnia,
  install4j: Install4J,
  insyncclient: DruvaInSync,
  intellidock: Intellidock,
  "intellij idea": IntelliJIdea,
  "intellij idea ce": IntelliJIdeaCe,
  invesalius: Invesalius,
  irfanview: Irfanview,
  "ironpython 3": Ironpython,
  isobuster: Isobuster,
  istherenet: Istherenet,
  iterm2: ITerm,
  itsycal: Itsycal,
  itunes: Itunes,
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
  keepass: Keepass,
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
  lapce: Lapce,
  lasso: LassoApp,
  "last window quits": LastWindowQuits,
  lastpass: LastPass,
  latest: Latest,
  launchbar: Launchbar,
  "lenovo dock manager": LenovoDockManager,
  "lenovo system update": LenovoSystemUpdate,
  lens: Lens,
  libreoffice: LibreOffice,
  lightburn: Lightburn,
  linear: Linear,
  linearmouse: Linearmouse,
  "lingon x": LingonX,
  "little snitch": LittleSnitch,
  "lo-rain": LoRain,
  local: Local,
  localsend: Localsend,
  locationsimulator: Locationsimulator,
  "logi options+": Logioptionsplus,
  "logi tune": LogiTune,
  "logitech unifying software": LogitechUnifyingSoftware,
  logseq: Logseq,
  lookaway: Lookaway,
  loom: Loom,
  loop: Loop,
  loopback: Loopback,
  losslesscut: Losslesscut,
  "low profile": LowProfile,
  lulu: LuLu,
  lunacy: Lunacy,
  lunar: Lunar,
  lunasea: Lunasea,
  lunatask: Lunatask,
  "lychee slicer": Lycheeslicer,
  "mac mouse fix": MacMouseFix,
  maccy: Maccy,
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
  mattermost: Mattermost,
  max: Max,
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
  "microsoft .net desktop runtime": MicrosoftDotnetRuntime,
  "microsoft .net runtime": MicrosoftDotnetRuntime,
  "microsoft 365 copilot": Microsoft365Copilot,
  "microsoft auto update": MicrosoftAutoUpdate,
  "microsoft autoupdate": MicrosoftAutoUpdate,
  "microsoft azure storage explorer": MicrosoftAzureStorageExplorer,
  "microsoft defender": MicrosoftDefender,
  "microsoft edge": Edge,
  "microsoft excel": Excel,
  "microsoft odbc driver 17 for sql server": MicrosoftOdbcDriver17,
  "microsoft odbc driver 18 for sql server": MicrosoftOdbcDriver18,
  "microsoft office": MicrosoftOffice,
  "microsoft ole db driver 19 for sql server": MicrosoftOleDbDriver19,
  "microsoft onenote": MicrosoftOneNote,
  "microsoft outlook": MicrosoftOutlook,
  "microsoft powerpoint": MicrosoftPowerPoint,
  "microsoft remote help": MicrosoftRemoteHelp,
  "microsoft teams": Teams,
  "microsoft visual c++": VcRedistX64,
  "microsoft visual studio code": VisualStudioCode,
  "microsoft word": Word,
  "microsoft.companyportal": IntuneCompanyPortal,
  middle: Middle,
  middleclick: Middleclick,
  milanote: Milanote,
  "mimecast for mac": Mimecast,
  mimestream: Mimestream,
  mindmac: Mindmac,
  mindmanager: MindManager,
  minisim: Minisim,
  minstaller: Minstaller,
  miro: Miro,
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
  "mozilla firefox developer edition": FirefoxDeveloperEdition,
  "mozilla firefox nightly": FirefoxNightly,
  "mozilla vpn": MozillaVpn,
  mqttx: Mqttx,
  mremoteng: Mremoteng,
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
  notion: Notion,
  "notion calendar": NotionCalendar,
  "noun project": NounProject,
  nova: Nova,
  novabench: Novabench,
  nucleo: Nucleo,
  nudge: Nudge,
  numi: Numi,
  nvda: Nvda,
  "nvidia geforce now": NvidiaGeforceNow,
  obs: Obs,
  obsidian: Obsidian,
  ocenaudio: Ocenaudio,
  "ok json": OkJson,
  "okta advanced server access": ScaleFt,
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
  pacifist: Pacifist,
  package: Package,
  "paint.net": PaintDotNet,
  "pale moon": PaleMoon,
  paletro: Paletro,
  "parallels desktop": ParallelsDesktop,
  pastebot: Pastebot,
  pd: Pd,
  "pdf expert": PdfExpert,
  "pdf pals": PdfPals,
  "pdfsam basic": PdfsamBasic,
  pearcleaner: Pearcleaner,
  "pgadmin 4": PgAdmin4,
  pgadmin4: PgAdmin4,
  "phoenix slides": PhoenixSlides,
  photosrevive: Photosrevive,
  photostickies: Photostickies,
  phpstorm: PhpStorm,
  pibar: Pibar,
  picview: Picview,
  piezo: Piezo,
  pika: Pika,
  piphero: Piphero,
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
  "pppc utility": PppcUtility,
  preform: Preform,
  principle: Principle,
  prisma: Prisma,
  pritunl: Pritunl,
  privileges: Privileges,
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
  pycharm: PyCharm,
  "pycharm ce": PyCharmCe,
  "python 3.13": Python313,
  "python 3.14": Python314,
  qemu: Qemu,
  qlab: Qlab,
  "qspace pro": QspacePro,
  quip: Quip,
  qview: Qview,
  "r for windows": R,
  "radio silence": RadioSilence,
  "raindrop.io": Raindropio,
  "rancher desktop": RancherDesktop,
  rapidapi: RapidApi,
  rapidweaver: Rapidweaver,
  "raspberry pi imager": RaspberryPiImager,
  raycast: Raycast,
  readest: Readest,
  "realvnc connect viewer": VncViewer,
  "realvnc server": RealVncServer,
  "realvnc viewer": VncViewer,
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
  resharper: Resharper,
  retcon: Retcon,
  retroarch: Retroarch,
  retrobatch: Retrobatch,
  rewritebar: Rewritebar,
  rider: Rider,
  rightfont: Rightfont,
  ringcentral: Ringcentral,
  rive: Rive,
  rize: Rize,
  robofont: Robofont,
  roboform: Roboform,
  rocket: Rocket,
  "rocket typist": RocketTypist,
  "rocket.chat": RocketChat,
  "rocketman choices packager": RocketmanChoicesPackager,
  "royal tsx": RoyalTsx,
  rstudio: Rstudio,
  rsyncui: Rsyncui,
  rtools: Rtools,
  rubymine: RubyMine,
  runjs: Runjs,
  rustdesk: RustDesk,
  rustrover: RustRover,
  sabnzbd: Sabnzbd,
  safari: Safari,
  "safe exam browser": SafeExamBrowser,
  sanesidebuttons: Sanesidebuttons,
  santa: Santa,
  "sbarex qlmarkdown": Qlmarkdown,
  "sc menu": ScMenu,
  scaleft: ScaleFt,
  scratch: Scratch,
  "screen studio": ScreenStudio,
  screenflick: Screenflick,
  screenflow: Screenflow,
  screenfocus: Screenfocus,
  scribe: Scribe,
  scribus: Scribus,
  scrivener: Scrivener,
  secretive: Secretive,
  securesafe: Securesafe,
  selfcontrol: Selfcontrol,
  "sempliva tiles": Tiles,
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
  "smallstep agent": SmallstepAgent,
  smartsheet: Smartsheet,
  smartsvn: Smartsvn,
  smoothscroll: Smoothscroll,
  smultron: Smultron,
  snagit: Snagit,
  snapmotion: Snapmotion,
  snowsql: SnowflakeSnowsql,
  sococo: Sococo,
  "sonic visualiser": SonicVisualiser,
  "sonicwall netextender": SonicwallNetextender,
  sonobus: Sonobus,
  sonos: Sonos,
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
  spokenly: Spokenly,
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
  tabby: Tabby,
  tableau: TableauDesktop,
  "tableau prep": TableauPrep,
  tableplus: TablePlus,
  tabtab: Tabtab,
  tailscale: Tailscale,
  taskade: Taskade,
  taskbar: Taskbar,
  teacode: Teacode,
  teamviewer: TeamViewer,
  telegram: Telegram,
  teleport: TeleportConnect,
  "teleport connect": TeleportConnect,
  "teleport suite": TeleportConnect,
  "teradici pcoip software client for macos": Pcoipclient,
  terminal: Terminal,
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
  tightvnc: Tightvnc,
  timescribe: Timescribe,
  timing: Timing,
  todoist: Todoist,
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
  trex: Trex,
  "trezor suite": TrezorSuite,
  tripmode: Tripmode,
  tunnelblick: Tunnelblick,
  tuple: Tuple,
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
  vagrant: Vagrant,
  vanilla: Vanilla,
  vellum: Vellum,
  "vernier spectral analysis": VernierSpectralAnalysis,
  versions: Versions,
  via: Via,
  vim: Vim,
  vimcal: Vimcal,
  virtualbox: VirtualBox,
  virtualbuddy: VirtualBuddy,
  viscosity: Viscosity,
  "visual paradigm": VisualParadigm,
  "visual studio code": VisualStudioCode,
  "visual studio community 2022": VisualStudio2022,
  "visual studio enterprise 2022": VisualStudio2022,
  "visual studio professional 2022": VisualStudio2022,
  vivaldi: Vivaldi,
  vivid: VividApp,
  viz: Viz,
  vlc: Vlc,
  "vnc server": RealVncServer,
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
  wechat: Wechat,
  "wechat for mac": Wechat,
  weektodo: Weektodo,
  whatroute: Whatroute,
  whatsapp: WhatsApp,
  whisky: Whisky,
  whispering: Whispering,
  "wifiman desktop": Wifiman,
  windirstat: Windirstat,
  windowkeys: Windowkeys,
  "windows app": WindowsApp,
  "windows app remote": WindowsAppRemote,
  "windows defender": WindowsDefender,
  windsurf: Windsurf,
  winlogbeat: Winlogbeat,
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
  yarn: Yarn,
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
  "zoom outlook plugin": ZoomOutlookPlugin,
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
  adobe_plugins: AdobePlugin,
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
 * Sources whose own icon wins over any name match, strict or loose, because their names
 * collide with the application they extend. An Adobe plugin named "Adobe Creative Cloud
 * Libraries" is a plugin, not Creative Cloud, and one named "Zoom" is a plugin, not Zoom,
 * so showing the other application's icon would misrepresent the row. Other extension
 * sources keep matching on name first, so e.g. a VSCode extension named "Docker" still
 * gets the Docker icon.
 */
const SOURCE_ICON_OVERRIDES_NAME = ["adobe_plugins"];

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

  // for a few sources, the source icon wins over every name match below
  const overriddenSource = SOURCE_ICON_OVERRIDES_NAME.includes(
    source.trim().toLowerCase()
  )
    ? matchLoosePrefixToKey(SOFTWARE_SOURCE_TO_ICON_MAP, source)
    : undefined;

  // otherwise, try strict matching on name and source
  let Icon: TMatchedIcon | null = overriddenSource
    ? SOFTWARE_SOURCE_TO_ICON_MAP[overriddenSource]
    : matchStrictNameSourceToIcon({
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
