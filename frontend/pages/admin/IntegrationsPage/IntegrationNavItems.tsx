import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import Integrations from "./cards/Integrations";
import MdmSettings from "./cards/MdmSettings";
import Calendars from "./cards/Calendars";
import ChangeManagement from "./cards/ChangeManagement";
import CertificateAuthorities from "./cards/CertificateAuthorities";
import ConditionalAccess from "./cards/ConditionalAccess";
import IdentityProviders from "./cards/IdentityProviders";
import Sso from "../OrgSettingsPage/cards/Sso";
import GlobalHostStatusWebhook from "../OrgSettingsPage/cards/GlobalHostStatusWebhook";

const getIntegrationSettingsNavItems = (
  isManagedCloud: boolean
): ISideNavItem<any>[] => {
  const items: ISideNavItem<any>[] = [
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
    {
      title: "Single sign-on options",
      urlSection: "sso",
      path: PATHS.ADMIN_INTEGRATIONS_SSO,
      Card: Sso,
    },

    {
      title: "Certificates",
      urlSection: "certificates",
      path: PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
      Card: CertificateAuthorities,
    },
    {
      title: "Identity provider (IdP)",
      urlSection: "identity-provider",
      path: PATHS.ADMIN_INTEGRATIONS_IDENTITY_PROVIDER,
      Card: IdentityProviders,
    },
    {
      title: "Host status webhook",
      urlSection: "host-status-webhook",
      path: PATHS.ADMIN_INTEGRATIONS_HOST_STATUS_WEBHOOK,
      Card: GlobalHostStatusWebhook,
    },
  ];

  if (isManagedCloud) {
    items.push({
      title: "Conditional access",
      urlSection: "conditional-access",
      path: PATHS.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS,
      Card: ConditionalAccess,
    });
  }
  return items;
};

export default getIntegrationSettingsNavItems;
