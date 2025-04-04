import React, { useContext } from "react";

import { AppContext } from "context/app";

import { InjectedRouter, Params } from "react-router/lib/Router";

import paths from "router/paths";
import { IRouterLocation } from "interfaces/routing";

import SideNav from "../components/SideNav";
import getIntegrationSettingsNavItems from "./IntegrationNavItems";

const baseClass = "integrations";

interface IIntegrationSettingsPageProps {
  router: InjectedRouter;
  params: Params;
  location: IRouterLocation; // no type in react-router v3
}

const IntegrationsPage = ({
  router,
  params,
  location: { pathname },
}: IIntegrationSettingsPageProps) => {
  const { config } = useContext(AppContext);
  if (!config) return <></>;

  const isManagedCloud = config.license.managed_cloud;

  if (!isManagedCloud && pathname.includes("conditional-access")) {
    router.push(paths.ADMIN_SETTINGS);
  }
  const navItems = getIntegrationSettingsNavItems(isManagedCloud);
  const { section } = params;
  const DEFAULT_SETTINGS_SECTION = navItems[0];
  const currentSection =
    navItems.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentSection.Card;

  return (
    <div className={`${baseClass}`}>
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={navItems}
        activeItem={currentSection.urlSection}
        CurrentCard={<CurrentCard router={router} />}
      />
    </div>
  );
};

export default IntegrationsPage;
