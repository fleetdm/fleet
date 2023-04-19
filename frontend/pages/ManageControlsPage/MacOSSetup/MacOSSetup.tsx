import React, { useContext, useState } from "react";
import PATHS from "router/paths";
import { useQuery } from "react-query";
import { Params } from "react-router/lib/Router";

import {
  API_NO_TEAM_ID,
  APP_CONTEXT_NO_TEAM_ID,
  ITeamConfig,
} from "interfaces/team";
import { AppContext } from "context/app";
import teamsAPI, { ILoadTeamResponse } from "services/entities/teams";

import SideNav from "pages/admin/components/SideNav";
import Button from "components/buttons/Button/Button";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import MAC_OS_SETUP_NAV_ITEMS from "./MacOSSetupNavItems";

const baseClass = "macos-setup";

interface ISetupEmptyState {
  router: any;
}

const SetupEmptyState = ({ router }: ISetupEmptyState) => {
  const onClickEmptyConnect = () => {
    router.push(PATHS.CONTROLS_MAC_SETTINGS);
  };

  return (
    <div className={`${baseClass}__empty-state`}>
      <h2>Setup experience for macOS hosts</h2>
      <p>Connect Fleet to the Apple Business Manager to get started.</p>
      <Button variant="brand" onClick={onClickEmptyConnect}>
        Connect
      </Button>
    </div>
  );
};

interface IMacOSSetupProps {
  params: Params;
  location: { search: string };
  router: any;
}

const MacOSSetup = ({
  params,
  location: { search: queryString },
  router,
}: IMacOSSetupProps) => {
  const { section } = params;
  const { currentTeam, isPremiumTier } = useContext(AppContext);
  const [isConfigured, setIsConfigured] = useState(false);

  // TODO: consider using useTeamIdParam hook here instead in the future
  const teamId =
    currentTeam?.id === undefined || currentTeam.id < APP_CONTEXT_NO_TEAM_ID
      ? API_NO_TEAM_ID // coerce undefined and -1 to 0 for 'No team'
      : currentTeam.id;

  const { data: teamConfig, isLoading } = useQuery<
    ILoadTeamResponse,
    Error,
    ITeamConfig
  >(["teamConfig", teamId], () => teamsAPI.load(teamId), {
    refetchOnWindowFocus: false,
    retry: false,
    enabled: Boolean(teamId),
    select: (res) => res.team,
  });

  const DEFAULT_SETTINGS_SECTION = MAC_OS_SETUP_NAV_ITEMS[0];

  const currentFormSection =
    MAC_OS_SETUP_NAV_ITEMS.find((item) => item.urlSection === section) ??
    DEFAULT_SETTINGS_SECTION;

  const CurrentCard = currentFormSection.Card;

  if (isConfigured) return <SetupEmptyState router={router} />;

  return (
    <div className={baseClass}>
      <p>
        Customize the setup experience for hosts that automatically enroll to
        this team.
      </p>
      {!isPremiumTier ? (
        <PremiumFeatureMessage />
      ) : (
        <SideNav
          className={`${baseClass}__side-nav`}
          navItems={MAC_OS_SETUP_NAV_ITEMS.map((navItem) => ({
            ...navItem,
            path: navItem.path.concat(queryString),
          }))}
          activeItem={currentFormSection.urlSection}
          CurrentCard={
            <CurrentCard
              key={teamId}
              currentTeamId={teamId}
              bootstrapConfigured={false}
            />
          }
        />
      )}
    </div>
  );
};

export default MacOSSetup;
