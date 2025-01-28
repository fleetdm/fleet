import React, { useCallback, useContext, useState } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";
import { useErrorHandler } from "react-error-boundary";

import { IConfig } from "interfaces/config";
import { IApiError } from "interfaces/errors";
import configAPI from "services/entities/config";
import { AppContext } from "context/app";
import { NotificationContext } from "context/notification";
import deepDifference from "utilities/deep_difference";
import Spinner from "components/Spinner";
import paths from "router/paths";

import SideNav from "../components/SideNav";
import ORG_SETTINGS_NAV_ITEMS from "./OrgSettingsNavItems";
import { DeepPartial } from "./cards/constants";

interface IOrgSettingsPageProps {
  params: Params;
  router: InjectedRouter; // v3
}

export const baseClass = "org-settings";

const OrgSettingsPage = ({ params, router }: IOrgSettingsPageProps) => {
  const { section } = params;
  const DEFAULT_SETTINGS_SECTION = ORG_SETTINGS_NAV_ITEMS[0];

  const [isUpdatingSettings, setIsUpdatingSettings] = useState(false);
  const { isFreeTier, isPremiumTier, setConfig, isSandboxMode } = useContext(
    AppContext
  );

  if (isSandboxMode) {
    // redirect to Integrations page in sandbox mode
    router.push(paths.ADMIN_INTEGRATIONS);
  }
  const { renderFlash } = useContext(NotificationContext);
  const handlePageError = useErrorHandler();

  const {
    data: appConfig,
    isLoading: isLoadingAppConfig,
    refetch: refetchConfig,
  } = useQuery<IConfig, Error, IConfig>(["config"], () => configAPI.loadAll(), {
    select: (data: IConfig) => data,
    onSuccess: (data) => {
      setConfig(data);
    },
  });

  const onFormSubmit = useCallback(
    (formUpdates: DeepPartial<IConfig>) => {
      if (!appConfig) {
        return false;
      }

      setIsUpdatingSettings(true);

      const diff = deepDifference(formUpdates, appConfig);
      // send all formUpdates.agent_options because diff overrides all agent options
      diff.agent_options = formUpdates.agent_options;

      configAPI
        .update(diff)
        .then(() => {
          renderFlash("success", "Successfully updated settings.");
          refetchConfig();
        })
        .catch((response: { data: IApiError }) => {
          if (
            response?.data.errors[0].reason.includes("could not dial smtp host")
          ) {
            renderFlash(
              "error",
              "Could not connect to SMTP server. Please try again."
            );
          } else if (response?.data.errors) {
            const reason = response?.data.errors[0].reason;
            const agentOptionsInvalid =
              reason.includes("unsupported key provided") ||
              reason.includes("invalid value type");
            const isAgentOptionsError =
              agentOptionsInvalid ||
              reason.includes("script_execution_timeout' value exceeds limit.");
            renderFlash(
              "error",
              <>
                Couldn&apos;t update{" "}
                {isAgentOptionsError ? "agent options" : "settings"}: {reason}
                {agentOptionsInvalid && (
                  <>
                    <br />
                    If you&apos;re not using the latest osquery, use the
                    fleetctl apply --force command to override validation.
                  </>
                )}
              </>
            );
          }
        })
        .finally(() => {
          setIsUpdatingSettings(false);
        });
    },
    [appConfig, refetchConfig, renderFlash]
  );

  // filter out non-premium options
  let navItems = ORG_SETTINGS_NAV_ITEMS;
  if (!isPremiumTier) {
    navItems = ORG_SETTINGS_NAV_ITEMS.filter(
      (item) => item.urlSection !== "fleet-desktop"
    );
  }

  const currentFormSection =
    navItems.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentFormSection.Card;

  if (isFreeTier && section === "fleet-desktop") {
    handlePageError({ status: 403 });
    return null;
  }

  return (
    <div className={`${baseClass}`}>
      <p className={`${baseClass}__page-description`}>
        Set your organization information and configure SSO and SMTP.
      </p>
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={navItems}
        activeItem={currentFormSection.urlSection}
        CurrentCard={
          !isLoadingAppConfig && appConfig ? (
            <CurrentCard
              appConfig={appConfig}
              handleSubmit={onFormSubmit}
              isUpdatingSettings={isUpdatingSettings}
              isPremiumTier={isPremiumTier}
            />
          ) : (
            <Spinner />
          )
        }
      />
    </div>
  );
};

export default OrgSettingsPage;
