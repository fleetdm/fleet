import AppStore from "./AppStore";
import Calendar from "./Calendar";
import CollectingResults from "./CollectingResults";
import DataError from "./DataError";
import EmptyHosts from "./EmptyHosts";
import EmptyIntegrations from "./EmptyIntegrations";
import EmptyPacks from "./EmptyPacks";
import EmptyPolicies from "./EmptyPolicies";
import EmptyQueries from "./EmptyQueries";
import EmptySchedule from "./EmptySchedule";
import EmptySearchCheck from "./EmptySearchCheck";
import EmptySearchExclamation from "./EmptySearchExclamation";
import EmptySearchQuestion from "./EmptySearchQuestion";
import EmptySoftware from "./EmptySoftware";
import EmptyTeams from "./EmptyTeams";
import EmptyUsers from "./EmptyUsers";
import FileCertificate from "./FileCertificate";
import FileConfigurationProfile from "./FileConfigurationProfile";
import FileJson from "./FileJson";
import FileP7m from "./FileP7m";
import FilePdf from "./FilePdf";
import FilePem from "./FilePem";
import FilePkg from "./FilePkg";
import FilePng from "./FilePng";
import FilePs1 from "./FilePs1";
import FilePy from "./FilePy";
import FileScript from "./FileScript";
import FileSh from "./FileSh";
import FileVpp from "./FileVpp";
import FleetLogo from "./FleetLogo";
import Lock from "./Lock";
import Settings from "./Settings";

export const GRAPHIC_MAP = {
  // Empty state graphics
  "empty-queries": EmptyQueries,
  "empty-integrations": EmptyIntegrations,
  "empty-users": EmptyUsers,
  "empty-policies": EmptyPolicies,
  "empty-software": EmptySoftware,
  "empty-hosts": EmptyHosts,
  "empty-teams": EmptyTeams,
  "empty-packs": EmptyPacks,
  "empty-schedule": EmptySchedule,
  "empty-search-exclamation": EmptySearchExclamation,
  "empty-search-check": EmptySearchCheck,
  "empty-search-question": EmptySearchQuestion,
  // File type graphics
  "file-configuration-profile": FileConfigurationProfile,
  "file-sh": FileSh,
  "file-ps1": FilePs1,
  "file-py": FilePy,
  "file-script": FileScript,
  "file-pdf": FilePdf,
  "file-pkg": FilePkg,
  "file-png": FilePng,
  "file-p7m": FileP7m,
  "file-pem": FilePem,
  "file-json": FileJson,
  "file-vpp": FileVpp,
  "file-certificate": FileCertificate,
  "app-store": AppStore, // Used in non-editable file uploader for vpp apps edit modal
  // Other graphics
  "collecting-results": CollectingResults,
  "data-error": DataError,
  calendar: Calendar,
  lock: Lock,
  settings: Settings,
  "fleet-logo": FleetLogo,
};

export type GraphicNames = keyof typeof GRAPHIC_MAP;
