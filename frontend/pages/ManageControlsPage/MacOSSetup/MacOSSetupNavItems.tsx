import PATHS from "router/paths";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import EndUserAuthentication from "./cards/EndUserAuthentication/EndUserAuthentication";
import BootstrapPackage from "./cards/BootstrapPackage";

interface IMacOSSetupCardProps {
  currentTeamId?: number;
}

// TODO: types
const MAC_OS_SETUP_NAV_ITEMS: ISideNavItem<IMacOSSetupCardProps | any>[] = [
  {
    title: "End user authentication",
    urlSection: "end-user-auth",
    path: PATHS.CONTROLS_END_USER_AUTHENTICATION,
    Card: EndUserAuthentication,
  },
  {
    title: "Bootstrap package",
    urlSection: "bootstrap-package",
    path: PATHS.CONTROLS_BOOTSTRAP_PACKAGE,
    Card: BootstrapPackage,
  },
];

export default MAC_OS_SETUP_NAV_ITEMS;
