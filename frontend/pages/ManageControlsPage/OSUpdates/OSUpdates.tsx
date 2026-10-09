import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";

import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import SectionHeader from "components/SectionHeader";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import { IConfig } from "interfaces/config";
import { ApplePlatform } from "interfaces/platform";
import { ITeamConfig } from "interfaces/team";
import PATHS from "router/paths";
import configAPI from "services/entities/config";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";
import { getPathWithQueryParams } from "utilities/url";

import CurrentVersionSection from "./components/CurrentVersionSection";
import { parseOSUpdatesCurrentVersionsQueryParams } from "./components/CurrentVersionSection/CurrentVersionSection";
import TargetSection from "./components/TargetSection";

export type OSUpdatesSupportedPlatform = ApplePlatform | "windows";

// "android" and "linux" only display empty-state messaging
export type OSUpdatesTargetPlatform =
  | OSUpdatesSupportedPlatform
  | "android"
  | "linux";

const baseClass = "os-updates";

const getDefaultSelectedPlatform = (
  appConfig: IConfig | null
): OSUpdatesTargetPlatform => {
  // We dont have the data ready yet so we default to mac.
  // This is usually when the users first comes to this page.
  if (appConfig === null) return "darwin";

  // Default to the first tab (macOS) unless only Windows MDM is turned on.
  return !appConfig.mdm.enabled_and_configured &&
    appConfig.mdm.windows_enabled_and_configured
    ? "windows"
    : "darwin";
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

  if (!config || isLoadingConfig || isLoadingTeam || isFetchingTeamConfig) {
    return <Spinner />;
  }

  // Only global or team admins have access to the OS updates settings.
  // This check needs to come after the team config is loaded so that we have the correct
  // values for isGlobalAdmin and isTeamAdmin. These values are updated after
  // the team config is loaded.
  if (!isGlobalAdmin && !isTeamAdmin) {
    router.replace(
      getPathWithQueryParams(PATHS.CONTROLS_OS_SETTINGS, {
        fleet_id: teamIdForApi,
      })
    );
  }

  // FIXME: Handle error states for app config and team config (need specifications for this).

  // If the user has not selected a platform yet, we default to the platform that
  // is enabled and configured.
  const selectedPlatform =
    selectedPlatformTab || getDefaultSelectedPlatform(config);

  return (
    <div className={baseClass}>
      <PageDescription
        variant="tab-panel"
        content="Remotely enforce software updates."
      />
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
