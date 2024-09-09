import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";
import MdmSettings from "./cards/MdmSettings";
import Calendars from "./cards/Calendars";

const integrationSettingsNavItems: ISideNavItem<any>[] = [
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
    Card: MdmSettings,
  },
  {
    title: "Calendars",
    urlSection: "calendars",
    path: PATHS.ADMIN_INTEGRATIONS_CALENDARS,
    Card: Calendars,
  },
];

export default integrationSettingsNavItems;
