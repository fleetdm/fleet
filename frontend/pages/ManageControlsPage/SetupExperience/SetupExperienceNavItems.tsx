import { InjectedRouter } from "react-router";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";
import PATHS from "router/paths";

import BootstrapPackage from "./cards/BootstrapPackage";
import InstallSoftware from "./cards/InstallSoftware";
import RunScript from "./cards/RunScript";
import SetupAssistant from "./cards/SetupAssistant";
import Users from "./cards/Users/Users";

export interface ISetupExperienceCardProps {
  currentTeamId: number;
  router: InjectedRouter;
  urlPlatformParam?: string; // not yet guaranteed to be a valid platform
}

const SETUP_EXPERIENCE_NAV_ITEMS: ISideNavItem<ISetupExperienceCardProps>[] = [
  {
    title: "Users",
    urlSection: "users",
    path: PATHS.CONTROLS_USERS,
    Card: Users,
  },
  {
    title: "Bootstrap package",
    urlSection: "bootstrap-package",
    path: PATHS.CONTROLS_BOOTSTRAP_PACKAGE,
    Card: BootstrapPackage,
  },
  {
    title: "Install software",
    urlSection: "install-software",
    path: PATHS.CONTROLS_INSTALL_SOFTWARE("macos"),
    Card: InstallSoftware,
  },
  {
    title: "Run script",
    urlSection: "run-script",
    path: PATHS.CONTROLS_RUN_SCRIPT,
    Card: RunScript,
    exclude: true, // Excluded from the side navigation but still available via direct route - Fleet 5 deprecation
  },
  {
    title: "Setup Assistant",
    urlSection: "setup-assistant",
    path: PATHS.CONTROLS_SETUP_ASSISTANT,
    Card: SetupAssistant,
  },
];

export default SETUP_EXPERIENCE_NAV_ITEMS;
