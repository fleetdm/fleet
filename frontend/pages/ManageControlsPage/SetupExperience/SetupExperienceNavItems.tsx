import PATHS from "router/paths";

import { ISideNavItem } from "pages/admin/components/SideNav/SideNav";

import EndUserAuthentication from "./cards/EndUserAuthentication/EndUserAuthentication";
import BootstrapPackage from "./cards/BootstrapPackage";
import SetupAssistant from "./cards/SetupAssistant";
import InstallSoftware from "./cards/InstallSoftware";
import SetupExperienceScript from "./cards/SetupExperienceScript";

interface ISetupExperienceCardProps {
  currentTeamId?: number;
}

// TODO: types
const SETUP_EXPERIENCE_NAV_ITEMS: ISideNavItem<
  ISetupExperienceCardProps | any
>[] = [
  {
    title: "1. End user authentication",
    urlSection: "end-user-auth",
    path: PATHS.CONTROLS_END_USER_AUTHENTICATION,
    Card: EndUserAuthentication,
  },
  {
    title: "2. Setup assistant",
    urlSection: "setup-assistant",
    path: PATHS.CONTROLS_SETUP_ASSITANT,
    Card: SetupAssistant,
  },
  {
    title: "3. Bootstrap package",
    urlSection: "bootstrap-package",
    path: PATHS.CONTROLS_BOOTSTRAP_PACKAGE,
    Card: BootstrapPackage,
  },
  {
    title: "4. Install software",
    urlSection: "install-software",
    path: PATHS.CONTROLS_INSTALL_SOFTWARE,
    Card: InstallSoftware,
  },
  {
    title: "5. Run script",
    urlSection: "run-script",
    path: PATHS.CONTROLS_RUN_SCRIPT,
    Card: SetupExperienceScript,
  },
];

export default SETUP_EXPERIENCE_NAV_ITEMS;
