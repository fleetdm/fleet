import React, { useContext, useState } from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import { IConfig } from "interfaces/config";
import { ITeamConfig } from "interfaces/team";
import { ApplePlatform } from "interfaces/platform";

import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";

import TurnOnMdmMessage from "../../../components/TurnOnMdmMessage/TurnOnMdmMessage";
import CurrentVersionSection from "./components/CurrentVersionSection";
import TargetSection from "./components/TargetSection";
import { parseOSUpdatesCurrentVersionsQueryParams } from "./components/CurrentVersionSection/CurrentVersionSection";

export type OSUpdatesSupportedPlatform = ApplePlatform | "windows";

export type OSUpdatesTargetPlatform = OSUpdatesSupportedPlatform | "android"; // used for displaying "coming soon" messaging

const baseClass = "os-updates";

const getDefaultSelectedPlatform = (
  appConfig: IConfig | null
): OSUpdatesTargetPlatform => {
  // We dont have the data ready yet so we default to mac.
  // This is usually when the users first comes to this page.
  if (appConfig === null) return "darwin";

  // if the mac mdm is enable and configured we check the app config to see if
  // the mdm for mac is enabled. If it is, it does not matter if windows is
  // enabled and configured and we will always return "mac".
  return appConfig.mdm.enabled_and_configured ? "darwin" : "windows"; // TODO(android): adjust this when android is supported
};

interface IOSUpdates {
  router: InjectedRouter;
  teamIdForApi: number;
  queryParams: ReturnType<typeof parseOSUpdatesCurrentVersionsQueryParams>;
}

const OSUpdates = ({ router, teamIdForApi, queryParams }: IOSUpdates) => {
  const {
    isPremiumTier,
    isGlobalAdmin,
    isTeamAdmin,
    config,
    setConfig,
  } = useContext(AppContext);

  const [
    selectedPlatformTab,
    setSelectedPlatformTab,
  ] = useState<OSUpdatesTargetPlatform | null>(null);

  const {
    isFetching: isFetchingConfig,
    isLoading: isLoadingConfig,
    refetch: refetchAppConfig,
  } = useQuery<IConfig, Error>(["config"], () => configAPI.loadAll(), {
    refetchOnWindowFocus: false,
    onSuccess: (data) => setConfig(data), // update the app context with the refetched config
    enabled: false, // this is disabled as the config is already fetched in App.tsx
  });

  const {
    data: teamConfig,
    isFetching: isFetchingTeamConfig,
    isLoading: isLoadingTeam,
    refetch: refetchTeamConfig,
  } = useQuery<ILoadTeamResponse, Error, ITeamConfig>(
    ["team-config", teamIdForApi],
    () => teamsAPI.load(teamIdForApi),
    {
      refetchOnWindowFocus: false,
      enabled: !!teamIdForApi,
      select: (data) => data.team,
    }
  );

  // Not premium shows premium message
  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage
        className={`${baseClass}__premium-feature-message`}
      />
    );
  }

  if (isLoadingConfig || isLoadingTeam || isFetchingTeamConfig) {
    return <Spinner />;
  }

  // Only global or team admins have access to the OS updates settings.
  // This check needs to come after the team config is loaded so that we have the correct
  // values for isGlobalAdmin and isTeamAdmin. These values are updated after
  // the team config is loaded.
  if (!isGlobalAdmin && !isTeamAdmin) {
    router.replace(
      getPathWithQueryParams(PATHS.CONTROLS_OS_SETTINGS, {
        team_id: teamIdForApi,
      })
    );
  }

  // FIXME: Handle error states for app config and team config (need specifications for this).
  // mdm is not enabled for mac or windows.
  if (
    !config?.mdm.enabled_and_configured &&
    !config?.mdm.windows_enabled_and_configured
  ) {
    return <TurnOnMdmMessage router={router} />;
  }

  // If the user has not selected a platform yet, we default to the platform that
  // is enabled and configured.
  const selectedPlatform =
    selectedPlatformTab || getDefaultSelectedPlatform(config);

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely encourage the installation of software updates on hosts
        assigned to this team.
      </p>
      <>
        <div className={`${baseClass}__current-version-container`}>
          <CurrentVersionSection
            router={router}
            currentTeamId={teamIdForApi}
            queryParams={queryParams}
          />
        </div>
        <div className={`${baseClass}__target-container`}>
          <SectionHeader
            title="Target"
            wrapperCustomClass={`${baseClass}__header`}
          />
          <TargetSection
            key={teamIdForApi} // if the team changes, remount the target section
            appConfig={config}
            currentTeamId={teamIdForApi}
            isFetching={isFetchingConfig || isFetchingTeamConfig}
            selectedPlatform={selectedPlatform}
            teamConfig={teamConfig}
            onSelectPlatform={setSelectedPlatformTab}
            refetchAppConfig={refetchAppConfig}
            refetchTeamConfig={refetchTeamConfig}
          />
        </div>
      </>
    </div>
  );
};

export default OSUpdates;
