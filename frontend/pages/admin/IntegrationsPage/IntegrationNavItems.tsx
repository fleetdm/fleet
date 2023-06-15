import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";
import Mdm from "./cards/MdmSettings/MdmSettings";
import AutomaticEnrollment from "./cards/AutomaticEnrollment/AutomaticEnrollment";

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
    {
      title: "Automatic enrollment",
      urlSection: "automatic-enrollment",
      path: PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT,
      Card: AutomaticEnrollment,
    },
  ].filter((navItem) => !navItem.exclude);
};

export default getFilteredIntegrationSettingsNavItems;
