import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";
import MdmSettings from "./cards/MdmSettings";
import AutomaticEnrollment from "./cards/AutomaticEnrollment";
import Calendars from "./cards/Calendars";
import Vpp from "./cards/Vpp";

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
    title: "Automatic enrollment",
    urlSection: "automatic-enrollment",
    path: PATHS.ADMIN_INTEGRATIONS_AUTOMATIC_ENROLLMENT,
    Card: AutomaticEnrollment,
  },
  {
    title: "Calendars",
    urlSection: "calendars",
    path: PATHS.ADMIN_INTEGRATIONS_CALENDARS,
    Card: Calendars,
  },
  {
    title: "Volume Purchasing Program (VPP)",
    urlSection: "vpp",
    path: PATHS.ADMIN_INTEGRATIONS_VPP,
    Card: Vpp,
  },
];

export default integrationSettingsNavItems;
