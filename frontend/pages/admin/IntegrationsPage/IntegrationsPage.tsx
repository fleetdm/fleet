import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";

import deepDifference from "utilities/deep_difference";

import { NotificationContext } from "context/notification";
import { AppContext } from "context/app";

import paths from "router/paths";

import configAPI from "services/entities/config";

import { IConfig } from "interfaces/config";

import Spinner from "components/Spinner";

import SideNav from "../components/SideNav";
import getIntegrationSettingsNavItems from "./IntegrationNavItems";
import { DeepPartial } from "../OrgSettingsPage/cards/constants";

const baseClass = "integrations";

interface IIntegrationSettingsPageProps {
  router: InjectedRouter;
  params: Params;
}

const IntegrationsPage = ({
  router,
  params,
}: IIntegrationSettingsPageProps) => {
  const { renderFlash } = useContext(NotificationContext);
  const { isPremiumTier } = useContext(AppContext);

  const { section } = params;
  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);

  // // // settings that live under the integrations page

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error, IConfig>(
    ["config"],
    () => configAPI.loadAll(),
    {}
  );

  console.log("appConfig from call: ", appConfig);

  /** The common submission logic for settings that are rendered on the Integrations page, but use
   * the common configAPI.update method, the same one used by cards of the OrgSettingsPage */
  const onUpdateSettings = useCallback(
    async (formUpdates: DeepPartial<IConfig>) => {
      if (!appConfig) {
        return false;
      }

      setIsUpdatingSettings(true);

      const diff = deepDifference(formUpdates, appConfig);
      // send all formUpdates.agent_options because diff overrides all agent options
      diff.agent_options = formUpdates.agent_options;

      try {
        await configAPI.update(diff);
        renderFlash("success", "Successfully updated settings.");
        refetchConfig();
      } catch (err: unknown) {
        renderFlash("error", "Could not update settings");
      } finally {
        setIsUpdatingSettings(false);
      }
    },
    [appConfig, refetchConfig, renderFlash]
  );

  if (!appConfig) return <></>;
  const isManagedCloud = appConfig.license.managed_cloud;
  if (section?.includes("conditional-access") && !isManagedCloud) {
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
        CurrentCard={
          !isLoadingAppConfig && appConfig ? (
            <CurrentCard
              router={router}
              // below props used only by settings-related cards e.g. SSO
              appConfig={appConfig}
              handleSubmit={onUpdateSettings}
              isPremiumTier={isPremiumTier}
              isUpdatingSettings={isUpdatingSettings}
            />
          ) : (
            <Spinner />
          )
        }
      />
    </div>
  );
};

export default IntegrationsPage;
