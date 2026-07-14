import React, { useContext, useMemo } from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useQuery } from "react-query";

import { AppContext } from "context/app";
import SideNav from "pages/admin/components/SideNav";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";
import mdmAPI from "services/entities/mdm";

import getOSSettingsNavItems from "./OSSettingsNavItems";
import ProfileStatusAggregate from "./ProfileStatusAggregate";

const baseClass = "os-settings";

interface IOSSettingsProps {
  params: Params;
  router: InjectedRouter;
  currentPage: number;
  // Injected by ManageControlsPage via React.cloneElement; undefined for one
  // render on refresh while useTeamIdParam reconciles the URL against
  // availableTeams. Rendering during that window fires queries against the
  // wrong fleet.
  teamIdForApi?: number;
  location: {
    search: string;
  };
}

const OSSettings = ({
  router,
  currentPage,
  teamIdForApi,
  location: { search: queryString },
  params,
}: IOSSettingsProps) => {
  const { section } = params;
  const { isTeamTechnician, isGlobalTechnician } = useContext(AppContext);

  const {
    data: aggregateProfileStatusData,
    refetch: refetchAggregateProfileStatus,
    isError: isErrorAggregateProfileStatus,
    isLoading: isLoadingAggregateProfileStatus,
  } = useQuery(
    ["aggregateProfileStatuses", teamIdForApi],
    () => mdmAPI.getProfilesStatusSummary(teamIdForApi as number),
    {
      enabled: teamIdForApi !== undefined,
      refetchOnWindowFocus: false,
      retry: false,
    }
  );

  const isTechnician = !!isTeamTechnician || !!isGlobalTechnician;

  const filteredNavItems = useMemo(() => {
    return getOSSettingsNavItems(isTechnician);
  }, [isTechnician]);

  const DEFAULT_SETTINGS_SECTION = filteredNavItems[0];

  // The "assets" route renders the Configuration profiles card's Assets
  // sub-tab, so it resolves to (and keeps the side nav on) that same section.
  const isAssetsSubTab = section === "assets";
  const effectiveSection = isAssetsSubTab ? "configuration-profiles" : section;

  const currentFormSection =
    filteredNavItems.find((item) => item.urlSection === effectiveSection) ??
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

  // Hold render until useTeamIdParam in the parent has reconciled the URL
  // fleet against availableTeams. Mounting cards with an undefined/coerced
  // team id fires API requests against the wrong fleet.
  if (teamIdForApi === undefined) {
    return <Spinner />;
  }

  return (
    <div className={baseClass}>
      <PageDescription
        variant="tab-panel"
        content="Remotely enforce OS settings on hosts assigned to this fleet."
      />
      <ProfileStatusAggregate
        isLoading={isLoadingAggregateProfileStatus}
        isError={isErrorAggregateProfileStatus}
        teamId={teamIdForApi}
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
            key={teamIdForApi}
            currentTeamId={teamIdForApi}
            onMutation={refetchAggregateProfileStatus}
            router={router}
            currentPage={currentPage}
            activeTab={isAssetsSubTab ? "assets" : "profiles"}
          />
        }
      />
    </div>
  );
};

export default OSSettings;
