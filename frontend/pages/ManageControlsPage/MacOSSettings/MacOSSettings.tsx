import React, { useContext } from "react";
import { Params } from "react-router/lib/Router";

import { AppContext } from "context/app";
import SideNav from "pages/admin/components/SideNav";
import { useQuery } from "react-query";
import { ProfileSummaryResponse } from "interfaces/mdm";
import { API_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import mdmAPI from "services/entities/mdm";

import MAC_OS_SETTINGS_NAV_ITEMS from "./MacOSSettingsNavItems";
import AggregateMacSettingsIndicators from "./AggregateMacSettingsIndicators";

const baseClass = "mac-os-settings";

interface IMacOSSettingsProps {
  params: Params;
  location: {
    search: string;
  };
}

const MacOSSettings = ({
  location: { search: queryString },
  params,
}: IMacOSSettingsProps) => {
  const { section } = params;
  const { currentTeam } = useContext(AppContext);

  // TODO: consider using useTeamIdParam hook here instead in the future
  const teamId =
    currentTeam?.id === undefined || currentTeam.id < APP_CONTEXT_NO_TEAM_ID
      ? API_NO_TEAM_ID // coerce undefined and -1 to 0 for 'No team'
      : currentTeam.id;

  const {
    data: aggregateProfileStatusData,
    refetch: refetchAggregateProfileStatus,
    isLoading: isLoadingAggregateProfileStatus,
  } = useQuery<ProfileSummaryResponse>(
    ["aggregateProfileStatuses", teamId],
    () => mdmAPI.getAggregateProfileStatuses(teamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
    }
  );

  const DEFAULT_SETTINGS_SECTION = MAC_OS_SETTINGS_NAV_ITEMS[0];

  const currentFormSection =
    MAC_OS_SETTINGS_NAV_ITEMS.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentFormSection.Card;

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely enforce settings on macOS hosts assigned to this team.
      </p>
      <AggregateMacSettingsIndicators
        isLoading={isLoadingAggregateProfileStatus}
        teamId={teamId}
        aggregateProfileStatusData={aggregateProfileStatusData}
      />
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={MAC_OS_SETTINGS_NAV_ITEMS.map((navItem) => ({
          ...navItem,
          path: navItem.path.concat(queryString),
        }))}
        activeItem={currentFormSection.urlSection}
        CurrentCard={
          <CurrentCard
            key={teamId}
            currentTeamId={teamId}
            onMutation={refetchAggregateProfileStatus}
          />
        }
      />
    </div>
  );
};

export default MacOSSettings;
