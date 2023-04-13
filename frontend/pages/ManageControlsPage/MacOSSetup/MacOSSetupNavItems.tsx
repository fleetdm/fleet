import PATHS from "router/paths";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import BootstrapPackage from "./cards/BootstrapPackage";

interface IMacOSSetupCardProps {
  currentTeamId?: number;
}

// TODO: types
const MAC_OS_SETUP_NAV_ITEMS: ISideNavItem<IMacOSSetupCardProps | any>[] = [
  {
    title: "Bootstrap package",
    urlSection: "bootstrap-package",
    path: PATHS.CONTROLS_BOOTSTRAP_PACKAGE,
    Card: BootstrapPackage,
  },
];

export default MAC_OS_SETUP_NAV_ITEMS;
