import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import { IAppConfigFormProps } from "./cards/constants";

import Info from "./cards/Info";
import WebAddress from "./cards/WebAddress";
import Sso from "./cards/Sso";
import Smtp from "./cards/Smtp";
import GlobalHostStatusWebhook from "./cards/GlobalHostStatusWebhook";
import Statistics from "./cards/Statistics";
import FleetDesktop from "./cards/FleetDesktop";
import Advanced from "./cards/Advanced";
import Agents from "./cards/Agents";

const ORG_SETTINGS_NAV_ITEMS: ISideNavItem<IAppConfigFormProps>[] = [
  {
    title: "Organization info",
    urlSection: "organization",
    path: PATHS.ADMIN_ORGANIZATION_INFO,
    Card: Info,
  },
  {
    title: "Fleet web address",
    urlSection: "webaddress",
    path: PATHS.ADMIN_ORGANIZATION_WEBADDRESS,
    Card: WebAddress,
  },
  {
    title: "Single sign-on options",
    urlSection: "sso",
    path: PATHS.ADMIN_ORGANIZATION_SSO,
    Card: Sso,
  },
  {
    title: "SMTP options",
    urlSection: "smtp",
    path: PATHS.ADMIN_ORGANIZATION_SMTP,
    Card: Smtp,
  },
  {
    title: "Agent options",
    urlSection: "agents",
    path: PATHS.ADMIN_ORGANIZATION_AGENTS,
    Card: Agents,
  },
  {
    title: "Host status webhook",
    urlSection: "host-status-webhook",
    path: PATHS.ADMIN_ORGANIZATION_HOST_STATUS_WEBHOOK,
    Card: GlobalHostStatusWebhook,
  },
  {
    title: "Usage statistics",
    urlSection: "statistics",
    path: PATHS.ADMIN_ORGANIZATION_STATISTICS,
    Card: Statistics,
  },
  {
    title: "Fleet Desktop",
    urlSection: "fleet-desktop",
    path: PATHS.ADMIN_ORGANIZATION_FLEET_DESKTOP,
    // isPremium: true,
    Card: FleetDesktop,
  },
  {
    title: "Advanced options",
    urlSection: "advanced",
    path: PATHS.ADMIN_ORGANIZATION_ADVANCED,
    Card: Advanced,
  },
];

export default ORG_SETTINGS_NAV_ITEMS;
