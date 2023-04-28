import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";
import Mdm from "./cards/MdmSettings/MdmSettings";

const getFilteredIntegrationSettingsNavItems = (
  isSandboxMode = false
): ISideNavItem<any>[] => {
  return [
    // TODO: types
    {
      title: "Ticket destinations",
      urlSection: "ticket-destinations",
      path: PATHS.ADMIN_INTEGRATIONS_TICKET_DESTINATIONS,
      Card: Integrations,
    },
    {
      title: "Mobile device management (MDM)",
      urlSection: "mdm",
      path: PATHS.ADMIN_INTEGRATIONS_MDM,
      Card: Mdm,
      exclude: isSandboxMode,
    },
  ].filter((navItem) => !navItem.exclude);
};

export default getFilteredIntegrationSettingsNavItems;
