import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";
import MdmSettings from "./cards/MdmSettings";
import Calendars from "./cards/Calendars";
import ChangeManagement from "./cards/ChangeManagement";
import CertificateAuthorities from "./cards/CertificateAuthorities";
import ConditionalAccess from "./cards/ConditionalAccess";

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
  {
    title: "Change management",
    urlSection: "change-management",
    path: PATHS.ADMIN_INTEGRATIONS_CHANGE_MANAGEMENT,
    Card: ChangeManagement,
  },
  // TODO: digicert update: add this back when the feature is ready
  {
    title: "Certificates",
    urlSection: "certificates",
    path: PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
    Card: CertificateAuthorities,
  },
  {
    title: "Conditional access",
    urlSection: "conditional-access",
    path: PATHS.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS,
    Card: ConditionalAccess,
  },
];

export default integrationSettingsNavItems;
