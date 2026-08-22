import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";

import deepDifference from "utilities/deep_difference";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import { AppContext } from "context/app";

import configAPI from "services/entities/config";
import useUpdateAppConfig from "hooks/useUpdateAppConfig";

import { IConfig } from "interfaces/config";

import Spinner from "components/Spinner";
import { notify } from "components/ToastNotification";

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
  const { isPremiumTier } = useContext(AppContext);

  let { section } = params;
  const { subsection } = params;
  if (!section && !!subsection) {
    section = "sso";
  }
  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);
  const updateAppConfig = useUpdateAppConfig();

  // // // settings that live under the integrations page

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    isFetching: isFetchingAppConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    ...DEFAULT_USE_QUERY_OPTIONS,
  });

  /** The common submission logic for settings that are rendered on the Integrations page, but use
   * the common configAPI.update method, the same one used by cards of the OrgSettingsPage */
  const onUpdateSettings = useCallback(
    async (formUpdates: DeepPartial<IConfig>) => {
      if (!appConfig) {
        return false;
      }

      const diff = deepDifference(formUpdates, appConfig);

      // If there's no actual change, don't make the API call to update config.
      // Cards that make their own writes update the cache directly via
      // useUpdateAppConfig, so no refetch is needed here.
      if (Object.keys(diff).length === 0) {
        return true;
      }

      setIsUpdatingSettings(true);

      // send all formUpdates.agent_options because diff overrides all agent options
      diff.agent_options = formUpdates.agent_options;

      try {
        const updatedConfig = await configAPI.update(diff);
        updateAppConfig(updatedConfig);
        notify.success("Successfully updated settings.");
        return true;
      } catch (err: unknown) {
        notify.error("Could not update settings", { response: err });
        return false;
      } finally {
        setIsUpdatingSettings(false);
      }
    },
    [appConfig, updateAppConfig]
  );

  if (!appConfig) return <></>;

  const navItems = getIntegrationSettingsNavItems();
  const DEFAULT_SETTINGS_SECTION = navItems[0];
  const currentSection =
    navItems.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentSection.Card;
  const isLoading = isLoadingAppConfig || isFetchingAppConfig;

  return (
    <div className={`${baseClass}`}>
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={navItems}
        activeItem={currentSection.urlSection}
        CurrentCard={
          !isLoading && appConfig ? (
            <CurrentCard
              router={router}
              appConfig={appConfig}
              handleSubmit={onUpdateSettings}
              isPremiumTier={isPremiumTier}
              isUpdatingSettings={isUpdatingSettings}
              subsection={subsection}
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
