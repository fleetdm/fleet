import PATHS from "router/paths";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import { IMdmProfile } from "interfaces/mdm";
import CustomSettings from "./cards/CustomSettings";

interface IMacOSSettingsCardProps {
  profiles?: IMdmProfile[];
  onProfileUpload?: () => void;
  onProfileDelete?: () => void;
}

const MAC_OS_SETTINGS_NAV_ITEMS: ISideNavItem<
  IMacOSSettingsCardProps | any
>[] = [
  {
    title: "Custom settings",
    urlSection: "custom-settings",
    path: PATHS.CONTROLS_CUSTOM_SETTINGS,
    Card: CustomSettings,
  },
];

export default MAC_OS_SETTINGS_NAV_ITEMS;
