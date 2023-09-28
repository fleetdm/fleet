import EmptyQueries from "./EmptyQueries";
import EmptyIntegrations from "./EmptyIntegrations";
import EmptyMembers from "./EmptyMembers";
import EmptyPolicies from "./EmptyPolicies";
import EmptySoftware from "./EmptySoftware";
import FileConfigurationProfile from "./FileConfigurationProfile";
import FileSh from "./FileSh";
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

export const GRAPHIC_MAP = {
  // Empty state graphics
  "empty-queries": EmptyQueries,
  "empty-integrations": EmptyIntegrations,
  "empty-members": EmptyMembers,
  "empty-policies": EmptyPolicies,
  "empty-software": EmptySoftware,
  "empty-hosts": EmptyHosts,
  "empty-teams": EmptyTeams,
  "empty-packs": EmptyPacks,
  "empty-schedule": EmptySchedule,
  // File type graphics
  "file-configuration-profile": FileConfigurationProfile,
  "file-sh": FileSh,
  "file-py": FilePy,
  "file-script": FileScript,
  "file-pdf": FilePdf,
  "file-pkg": FilePkg,
  "file-p7m": FileP7m,
  "file-pem": FilePem,
};

export type GraphicNames = keyof typeof GRAPHIC_MAP;
