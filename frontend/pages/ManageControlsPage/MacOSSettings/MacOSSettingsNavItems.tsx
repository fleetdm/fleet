import PATHS from "router/paths";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import { IMdmProfile } from "interfaces/mdm";

import DiskEncryption from "./cards/DiskEncryption";
import CustomSettings from "./cards/CustomSettings";

interface IMacOSSettingsCardProps {
  currentTeamId?: number;
  profiles?: IMdmProfile[];
  onProfileUpload?: () => void;
  onProfileDelete?: () => void;
}

// TODO: types
const MAC_OS_SETTINGS_NAV_ITEMS: ISideNavItem<
  IMacOSSettingsCardProps | any
>[] = [
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

export default MAC_OS_SETTINGS_NAV_ITEMS;
