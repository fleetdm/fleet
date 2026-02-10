import React, { useContext, useMemo } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import SideNav from "pages/admin/components/SideNav";
import { API_NO_TEAM_ID, APP_CONTEXT_NO_TEAM_ID } from "interfaces/team";
import mdmAPI from "services/entities/mdm";

import OS_SETTINGS_NAV_ITEMS from "./OSSettingsNavItems";
import ProfileStatusAggregate from "./ProfileStatusAggregate";

const baseClass = "os-settings";

interface IOSSettingsProps {
  params: Params;
  router: InjectedRouter;
  currentPage: number;
  location: {
    search: string;
  };
}

const OSSettings = ({
  router,
  currentPage,
  location: { search: queryString },
  params,
}: IOSSettingsProps) => {
  const { section } = params;
  const { currentTeam, isTeamTechnician, isGlobalTechnician } = useContext(
    AppContext
  );

  // TODO: consider using useTeamIdParam hook here instead in the future
  const teamId =
    currentTeam?.id === undefined || currentTeam.id < APP_CONTEXT_NO_TEAM_ID
      ? API_NO_TEAM_ID // coerce undefined and -1 to 0 for 'No team'
      : currentTeam.id;

  const {
    data: aggregateProfileStatusData,
    refetch: refetchAggregateProfileStatus,
    isError: isErrorAggregateProfileStatus,
    isLoading: isLoadingAggregateProfileStatus,
  } = useQuery(
    ["aggregateProfileStatuses", teamId],
    () => mdmAPI.getProfilesStatusSummary(teamId),
    {
      refetchOnWindowFocus: false,
      retry: false,
    }
  );

  const filteredNavItems = useMemo(() => {
    if (isTeamTechnician || isGlobalTechnician) {
      return OS_SETTINGS_NAV_ITEMS.filter(
        (item) => item.title !== "Certificates"
      );
    }
    return OS_SETTINGS_NAV_ITEMS;
  }, [isTeamTechnician, isGlobalTechnician]);

  const DEFAULT_SETTINGS_SECTION = filteredNavItems[0];

  const currentFormSection =
    filteredNavItems.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  // Redirect to the default section if the URL section is not in the filtered list
  if (
    section &&
    currentFormSection === DEFAULT_SETTINGS_SECTION &&
    section !== DEFAULT_SETTINGS_SECTION.urlSection
  ) {
    router.replace(DEFAULT_SETTINGS_SECTION.path.concat(queryString));
    return null;
  }

  const CurrentCard = currentFormSection.Card;

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Remotely enforce OS settings on hosts assigned to this team.
      </p>
      <ProfileStatusAggregate
        isLoading={isLoadingAggregateProfileStatus}
        isError={isErrorAggregateProfileStatus}
        teamId={teamId}
        aggregateProfileStatusData={aggregateProfileStatusData}
      />
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={filteredNavItems.map((navItem) => ({
          ...navItem,
          path: navItem.path.concat(queryString),
        }))}
        activeItem={currentFormSection.urlSection}
        CurrentCard={
          <CurrentCard
            key={teamId}
            currentTeamId={teamId}
            onMutation={refetchAggregateProfileStatus}
            router={router}
            currentPage={currentPage}
          />
        }
      />
    </div>
  );
};

export default OSSettings;
