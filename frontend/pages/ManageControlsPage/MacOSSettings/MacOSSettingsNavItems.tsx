import PATHS from "router/paths";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import CustomSettings from "./cards/CustomSettings";

const MAC_OS_SETTINGS_NAV_ITEMS: ISideNavItem<any>[] = [
  {
    title: "Custom settings",
    urlSection: "custom-settings",
    path: PATHS.CONTROLS_CUSTOM_SETTINGS,
    Card: CustomSettings,
  },
];

export default MAC_OS_SETTINGS_NAV_ITEMS;
