import PATHS from "router/paths";

import { InjectedRouter } from "react-router";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import EndUserAuthentication from "./cards/EndUserAuthentication/EndUserAuthentication";
import BootstrapPackage from "./cards/BootstrapPackage";
import SetupAssistant from "./cards/SetupAssistant";
import InstallSoftware from "./cards/InstallSoftware";
import RunScript from "./cards/RunScript";

export interface ISetupExperienceCardProps {
  currentTeamId: number;
  router: InjectedRouter;
  urlPlatformParam?: string; // not yet guaranteed to be a valid platform
}

const SETUP_EXPERIENCE_NAV_ITEMS: ISideNavItem<ISetupExperienceCardProps>[] = [
  {
    title: "1. End user authentication",
    urlSection: "end-user-auth",
    path: PATHS.CONTROLS_END_USER_AUTHENTICATION,
    Card: EndUserAuthentication,
  },
  {
    title: "2. Bootstrap package",
    urlSection: "bootstrap-package",
    path: PATHS.CONTROLS_BOOTSTRAP_PACKAGE,
    Card: BootstrapPackage,
  },
  {
    title: "3. Install software",
    urlSection: "install-software",
    path: PATHS.CONTROLS_INSTALL_SOFTWARE("macos"),
    Card: InstallSoftware,
  },
  {
    title: "4. Run script",
    urlSection: "run-script",
    path: PATHS.CONTROLS_RUN_SCRIPT,
    Card: RunScript,
  },
  {
    title: "5. Setup assistant",
    urlSection: "setup-assistant",
    path: PATHS.CONTROLS_SETUP_ASSISTANT,
    Card: SetupAssistant,
  },
];

export default SETUP_EXPERIENCE_NAV_ITEMS;
