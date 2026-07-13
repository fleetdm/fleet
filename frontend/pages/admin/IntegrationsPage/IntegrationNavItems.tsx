import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import TicketDestinations from "./cards/Integrations";
import MdmSettings from "./cards/MdmSettings";
import Calendars from "./cards/Calendars";
import ChangeManagement from "./cards/ChangeManagement";
import CertificateAuthorities from "./cards/CertificateAuthorities";
import ConditionalAccess from "./cards/ConditionalAccess";
import IdentityProviders from "./cards/IdentityProviders";
import Sso from "./cards/Sso";
import AccountProvisioning from "./cards/AccountProvisioning";
import GlobalHostStatusWebhook from "../IntegrationsPage/cards/GlobalHostStatusWebhook";

const getIntegrationSettingsNavItems = (): ISideNavItem<any>[] => {
  const items: ISideNavItem<any>[] = [
    {
      title: "Ticketing",
      urlSection: "ticket-destinations",
      path: PATHS.ADMIN_INTEGRATIONS_TICKET_DESTINATIONS,
      Card: TicketDestinations,
    },
    {
      title: "MDM",
      urlSection: "mdm",
      path: PATHS.ADMIN_INTEGRATIONS_MDM,
      Card: MdmSettings,
    },
    {
      title: "Calendar events",
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
      title: "Authentication (SSO)",
      urlSection: "sso",
      path: PATHS.ADMIN_INTEGRATIONS_SSO_FLEET_USERS,
      Card: Sso,
    },
    {
      title: "Account provisioning",
      urlSection: "account-provisioning",
      path: PATHS.ADMIN_INTEGRATIONS_FPSSO,
      Card: AccountProvisioning,
    },
    {
      title: "User mapping",
      urlSection: "identity-provider",
      path: PATHS.ADMIN_INTEGRATIONS_IDENTITY_PROVIDER,
      Card: IdentityProviders,
    },
    {
      title: "Certificate enrollment",
      urlSection: "certificate-authorities",
      path: PATHS.ADMIN_INTEGRATIONS_CERTIFICATE_AUTHORITIES,
      Card: CertificateAuthorities,
    },
    {
      title: "Host status alerts",
      urlSection: "host-status-webhook",
      path: PATHS.ADMIN_INTEGRATIONS_HOST_STATUS_WEBHOOK,
      Card: GlobalHostStatusWebhook,
    },
    {
      title: "Conditional access",
      urlSection: "conditional-access",
      path: PATHS.ADMIN_INTEGRATIONS_CONDITIONAL_ACCESS,
      Card: ConditionalAccess,
    },
  ];

  return items;
};

export default getIntegrationSettingsNavItems;
