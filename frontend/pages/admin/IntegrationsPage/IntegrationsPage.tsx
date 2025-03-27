import React, { useContext } from "react";

import { AppContext } from "context/app";

import { InjectedRouter, Params } from "react-router/lib/Router";

import SideNav from "../components/SideNav";
import integrationSettingsNavItems from "./IntegrationNavItems";

const baseClass = "integrations";

interface IIntegrationSettingsPageProps {
  router: InjectedRouter;
  params: Params;
}

const IntegrationsPage = ({
  router,
  params,
}: IIntegrationSettingsPageProps) => {
  const { section } = params;
  const navItems = integrationSettingsNavItems;
  const DEFAULT_SETTINGS_SECTION = navItems[0];
  const currentSection =
    navItems.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  if (!useContext(AppContext)?.config?.license.managed_cloud) {
    navItems.splice(
      navItems.findIndex((item) => item.urlSection === "conditional-access"),
      1
    );
  }

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
