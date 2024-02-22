import EmptyQueries from "./EmptyQueries";
import EmptyIntegrations from "./EmptyIntegrations";
import EmptyUsers from "./EmptyUsers";
import EmptyPolicies from "./EmptyPolicies";
import EmptySoftware from "./EmptySoftware";
import FileConfigurationProfile from "./FileConfigurationProfile";
import FileSh from "./FileSh";
import FilePs1 from "./FilePs1";
import FilePy from "./FilePy";
import FileScript from "./FileScript";
import FilePdf from "./FilePdf";
import FilePkg from "./FilePkg";
import FileP7m from "./FileP7m";
import FilePem from "./FilePem";
import EmptyHosts from "./EmptyHosts";
import EmptyTeams from "./EmptyTeams";
import EmptyPacks from "./EmptyPacks";
import EmptySchedule from "./EmptySchedule";
import CollectingResults from "./CollectingResults";

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
  // File type graphics
  "file-configuration-profile": FileConfigurationProfile,
  "file-sh": FileSh,
  "file-ps1": FilePs1,
  "file-py": FilePy,
  "file-script": FileScript,
  "file-pdf": FilePdf,
  "file-pkg": FilePkg,
  "file-p7m": FileP7m,
  "file-pem": FilePem,
  // Other graphics
  "collecting-results": CollectingResults,
};

export type GraphicNames = keyof typeof GRAPHIC_MAP;
