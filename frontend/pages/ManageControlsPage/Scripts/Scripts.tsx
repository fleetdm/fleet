import React from "react";
import { InjectedRouter, Params } from "react-router/lib/Router";

import useTeamIdParam from "hooks/useTeamIdParam";

import { FLEET_WEBSITE_URL } from "utilities/constants";

import SideNav from "pages/admin/components/SideNav";
import CustomLink from "components/CustomLink";
import PageDescription from "components/PageDescription";
import Spinner from "components/Spinner";

import useScriptNavItems from "./ScriptsNavItems";

const baseClass = "scripts";

export interface ScriptsLocation {
  search: string;
  pathname: string;
  query: {
    fleet_id?: string;
    status?: string;
    page?: string;
    add_script?: string;
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

  // Redirect to the default section if the URL section is not in the filtered list
  if (
    section &&
    currentFormSection === DEFAULT_SCRIPTS_SECTION &&
    section !== DEFAULT_SCRIPTS_SECTION.urlSection
  ) {
    router.replace(DEFAULT_SCRIPTS_SECTION.path);
    return null;
  }

  const CurrentCard = currentFormSection.Card;

  // Hold render until useTeamIdParam has reconciled the URL fleet against
  // availableTeams. Coercing undefined to API_NO_TEAM_ID caused the cards
  // to fire team-0 requests before the correct fleet was known.
  if (teamIdForApi === undefined) {
    return <Spinner />;
  }

  return (
    <div className={baseClass}>
      <PageDescription
        variant="tab-panel"
        content={
          <>
            Change configuration and remediate issues on macOS, Windows, and
            Linux hosts.{" "}
            <CustomLink
              text="Learn more"
              url={`${FLEET_WEBSITE_URL}/docs/using-fleet/scripts`}
              newTab
            />
          </>
        }
      />
      <SideNav
        className={`${baseClass}__side-nav`}
        navItems={SCRIPTS_NAV_ITEMS}
        activeItem={currentFormSection.urlSection}
        CurrentCard={
          <CurrentCard
            key={teamIdForApi}
            teamId={teamIdForApi}
            router={router}
            location={location}
          />
        }
      />
    </div>
  );
};

export default Scripts;
