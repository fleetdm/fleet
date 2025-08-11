import React from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";

import useTeamIdParam from "hooks/useTeamIdParam";

import { API_NO_TEAM_ID } from "interfaces/team";

import { FLEET_WEBSITE_URL } from "utilities/constants";

import SideNav from "pages/admin/components/SideNav";
import CustomLink from "components/CustomLink";

import useScriptNavItems from "./ScriptsNavItems";

const baseClass = "scripts";

export interface ScriptsLocation {
  search: string;
  pathname: string;
  query: {
    team_id?: string;
    status?: string;
  };
}
interface IScriptsProps {
  params: Params;
  router: InjectedRouter;
  location: ScriptsLocation;
}

const Scripts = ({ router, location, params }: IScriptsProps) => {
  const { section } = params;

  const { teamIdForApi } = useTeamIdParam({
    location,
    router,
    includeAllTeams: false,
    includeNoTeam: true,
  });

  const SCRIPTS_NAV_ITEMS = useScriptNavItems(teamIdForApi);

  const DEFAULT_SCRIPTS_SECTION = SCRIPTS_NAV_ITEMS[0];

  const currentFormSection =
    SCRIPTS_NAV_ITEMS.find((item) => item.urlSection === section) ??
    DEFAULT_SCRIPTS_SECTION;

  const CurrentCard = currentFormSection.Card;

  return (
    <div className={baseClass}>
      <p className={`${baseClass}__description`}>
        Change configuration and remediate issues on macOS, Windows, and Linux
        hosts.{" "}
        <CustomLink
          text="Learn more"
          url={`${FLEET_WEBSITE_URL}/docs/using-fleet/scripts`}
          newTab
        />
      </p>
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={SCRIPTS_NAV_ITEMS}
        activeItem={currentFormSection.urlSection}
        CurrentCard={
          <CurrentCard
            // potential undefined for teamIdForApi is an implemenation artifact - it can be assumed
            // to always be defined here
            key={teamIdForApi ?? API_NO_TEAM_ID}
            teamId={teamIdForApi ?? API_NO_TEAM_ID} // Scripts must be scoped to a team
            router={router}
            location={location}
          />
        }
      />
    </div>
  );
};

export default Scripts;
