import PATHS from "router/paths";

import { IMdmProfile } from "interfaces/mdm";
import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import BootstrapPackage from "./cards/BootstrapPackage";

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
    title: "Bootstrap package",
    urlSection: "bootstrap-package",
    path: PATHS.CONTROLS_BOOTSTRAP_PACKAGE,
    Card: BootstrapPackage,
  },
];

export default MAC_OS_SETTINGS_NAV_ITEMS;
