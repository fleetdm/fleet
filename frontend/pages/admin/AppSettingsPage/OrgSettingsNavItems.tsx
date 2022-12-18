import React from "react";

import PATHS from "router/paths";

import { ISideNavItem } from "../components/SideNav/SideNav";
import { IAppConfigFormProps } from "./cards/constants";

import Info from "./cards/Info";
import WebAddress from "./cards/WebAddress";
import Sso from "./cards/Sso";
import Smtp from "./cards/Smtp";
import HostStatusWebhook from "./cards/HostStatusWebhook";
import Statistics from "./cards/Statistics";
import FleetDesktop from "./cards/FleetDesktop";
import Advanced from "./cards/Advanced";
import Agents from "./cards/Agents";

const ORG_SETTINGS_NAV_ITEMS: ISideNavItem<IAppConfigFormProps>[] = [
  {
    title: "Organization info",
    urlSection: "organization",
    path: PATHS.ADMIN_SETTINGS_INFO,
    Card: (props) => <Info {...props} />,
  },
  {
    title: "Fleet web address",
    urlSection: "webaddress",
    path: PATHS.ADMIN_SETTINGS_WEBADDRESS,
    Card: (props) => <WebAddress {...props} />,
  },
  {
    title: "Single sign-on options",
    urlSection: "sso",
    path: PATHS.ADMIN_SETTINGS_SSO,
    Card: (props) => <Sso {...props} />,
  },
  {
    title: "SMTP options",
    urlSection: "smtp",
    path: PATHS.ADMIN_SETTINGS_SMTP,
    Card: (props) => <Smtp {...props} />,
  },
  {
    title: "Agent options",
    urlSection: "agents",
    path: PATHS.ADMIN_SETTINGS_AGENTS,
    Card: (props) => <Agents {...props} />,
  },
  {
    title: "Host status webhook",
    urlSection: "host-status-webhook",
    path: PATHS.ADMIN_SETTINGS_HOST_STATUS_WEBHOOK,
    Card: (props) => <HostStatusWebhook {...props} />,
  },
  {
    title: "Usage statistics",
    urlSection: "statistics",
    path: PATHS.ADMIN_SETTINGS_STATISTICS,
    Card: (props) => <Statistics {...props} />,
  },
  {
    title: "Fleet Desktop",
    urlSection: "fleet-desktop",
    path: PATHS.ADMIN_SETTINGS_FLEET_DESKTOP,
    // isPremium: true,
    Card: (props) => <FleetDesktop {...props} />,
  },
  {
    title: "Advanced options",
    urlSection: "advanced",
    path: PATHS.ADMIN_SETTINGS_ADVANCED,
    Card: (props) => <Advanced {...props} />,
  },
];

export default ORG_SETTINGS_NAV_ITEMS;
