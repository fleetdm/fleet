import PATHS from "router/paths";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import EndUserAuthentication from "./cards/EndUserAuthentication/EndUserAuthentication";
import BootstrapPackage from "./cards/BootstrapPackage";

interface ISetupExperienceCardProps {
  currentTeamId?: number;
}

// TODO: types
const SETUP_EXPERIENCE_NAV_ITEMS: ISideNavItem<
  ISetupExperienceCardProps | any
>[] = [
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

export default SETUP_EXPERIENCE_NAV_ITEMS;
