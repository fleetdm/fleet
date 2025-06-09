import React, { useContext } from "react";

import { AppContext } from "context/app";

import { InjectedRouter, Params } from "react-router/lib/Router";

import paths from "router/paths";

import SideNav from "../components/SideNav";
import getIntegrationSettingsNavItems from "./IntegrationNavItems";

const baseClass = "integrations";

interface IIntegrationSettingsPageProps {
  router: InjectedRouter;
  params: Params;
}

const IntegrationsPage = ({
  router,
  params,
}: IIntegrationSettingsPageProps) => {
  const { config } = useContext(AppContext);
  if (!config) return <></>;

  const isManagedCloud = config.license.managed_cloud;

  const { section } = params;

  if (
    section?.includes("conditional-access") &&
    (!isManagedCloud || featureFlags.allowConditionalAccess !== "true")
  ) {
    router.push(paths.ADMIN_SETTINGS);
  }
  const navItems = getIntegrationSettingsNavItems(isManagedCloud);
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
