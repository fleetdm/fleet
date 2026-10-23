import React, { useContext, useEffect } from "react";
import { InjectedRouter } from "react-router";
import { Params } from "react-router/lib/Router";

import BackButton from "components/BackButton";
import Card from "components/Card";
import FleetsDropdown from "components/FleetsDropdown";
import MainContent from "components/MainContent";
import PageDescription from "components/PageDescription";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";
import { AppContext } from "context/app";
import useGitOpsMode from "hooks/useGitOpsMode";
import useTeamIdParam from "hooks/useTeamIdParam";
import SideNav from "pages/admin/components/SideNav";
import PATHS from "router/paths";
import { getPathWithQueryParams } from "utilities/url";

import getLibrarySettingsNavItems from "./SoftwareLibrarySettingsNavItems";
import { ILibrarySettingsLocation } from "./types";

const baseClass = "software-library-settings-page";

interface ISoftwareLibrarySettingsPageProps {
  router: InjectedRouter;
  location: ILibrarySettingsLocation;
  params: Params;
}

const SoftwareLibrarySettingsPage = ({
  router,
  location,
  params,
}: ISoftwareLibrarySettingsPageProps) => {
  const {
    config,
    isPremiumTier,
    isGlobalAdmin,
    isGlobalMaintainer,
  } = useContext(AppContext);
  const isPrimoMode = config?.partnerships?.enable_primo || false;
  const { gitOpsModeEnabled } = useGitOpsMode("software");

  const {
    currentTeamId,
    teamIdForApi,
    userTeams,
    handleTeamChange,
    isRouteOk,
    isTeamAdmin,
    isTeamMaintainer,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: true,
  });

  const fleetId = teamIdForApi ?? 0;
  const canManage =
    !!isGlobalAdmin ||
    !!isGlobalMaintainer ||
    !!isTeamAdmin ||
    !!isTeamMaintainer;

  const backToLibraryPath = getPathWithQueryParams(PATHS.SOFTWARE_LIBRARY, {
    fleet_id: teamIdForApi,
  });

  const navItems = getLibrarySettingsNavItems();
  const DEFAULT_SECTION = navItems[0];
  const currentSection =
    navItems.find((item) => item.urlSection === params.section) ??
    DEFAULT_SECTION;

  // Bare /library/settings (and any unknown :section) land here. Normalize the
  // URL to the default section so refreshes, shares, and browser history are
  // stable. router.replace keeps Back usable, and location.search preserves
  // fleet_id and any deep-link params (e.g. add_category=1 from the palette).
  useEffect(() => {
    if (!isPremiumTier) return;
    if (params.section && currentSection.urlSection === params.section) return;
    router.replace({
      pathname: DEFAULT_SECTION.path,
      search: location.search,
    });
  }, [
    isPremiumTier,
    params.section,
    currentSection.urlSection,
    DEFAULT_SECTION.path,
    location.search,
    router,
  ]);

  const CurrentCard = currentSection.Card;

  const renderHeader = () => (
    <>
      <BackButton text="Back to software library" path={backToLibraryPath} />
      {isPremiumTier && !isPrimoMode ? (
        <div className={`${baseClass}__fleet-row`}>
          <FleetsDropdown
            currentUserFleets={userTeams ?? []}
            selectedFleetId={currentTeamId}
            onChange={handleTeamChange}
            includeAllFleets={false}
            includeUnassigned
          />
        </div>
      ) : (
        <h1>Software library settings</h1>
      )}
      <PageDescription content="Customize self-service and patching." />
    </>
  );

  const renderBody = () => {
    if (!isPremiumTier) {
      return (
        <Card
          color="grey"
          paddingSize="xxlarge"
          className={`${baseClass}__premium-card`}
        >
          <PremiumFeatureMessage />
        </Card>
      );
    }

    return (
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={navItems.map((item) => ({
          ...item,
          path: getPathWithQueryParams(item.path, { fleet_id: teamIdForApi }),
        }))}
        activeItem={currentSection.urlSection}
        CurrentCard={
          <CurrentCard
            router={router}
            location={location}
            fleetId={fleetId}
            isRouteOk={isRouteOk}
            canManage={canManage}
            gitOpsModeEnabled={gitOpsModeEnabled}
          />
        }
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <>
        {renderHeader()}
        {renderBody()}
      </>
    </MainContent>
  );
};

export default SoftwareLibrarySettingsPage;
