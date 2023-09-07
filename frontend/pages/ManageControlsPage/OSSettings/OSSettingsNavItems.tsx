import PATHS from "router/paths";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import { IMdmProfile } from "interfaces/mdm";

import DiskEncryption from "./cards/DiskEncryption";
import CustomSettings from "./cards/CustomSettings";

interface IOSSettingsCardProps {
  currentTeamId?: number;
  profiles?: IMdmProfile[];
  onProfileUpload?: () => void;
  onProfileDelete?: () => void;
}

// TODO: types
const OS_SETTINGS_NAV_ITEMS: ISideNavItem<IOSSettingsCardProps | any>[] = [
  {
    title: "Disk encryption",
    urlSection: "disk-encryption",
    path: PATHS.CONTROLS_DISK_ENCRYPTION,
    Card: DiskEncryption,
  },
  {
    title: "Custom settings",
    urlSection: "custom-settings",
    path: PATHS.CONTROLS_CUSTOM_SETTINGS,
    Card: CustomSettings,
  },
];

export default OS_SETTINGS_NAV_ITEMS;
