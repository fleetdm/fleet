import React from "react";

import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";

const INTEGRATION_SETTINGS_NAV_ITEMS: ISideNavItem<any>[] = [
  // TODO: types
  {
    title: "Ticket destinations",
    urlSection: "ticket-destinations",
    path: PATHS.ADMIN_INTEGRATIONS_TICKET_DESTINATIONS,
    Card: Integrations,
  },
  {
    title: "Mobile Device Management (MDM)",
    urlSection: "mdm",
    path: PATHS.ADMIN_INTEGRATIONS_MDM,
    Card: () => <p>INTEGRATE WITH MDM PAGE HERE</p>,
  },
];

export default INTEGRATION_SETTINGS_NAV_ITEMS;
