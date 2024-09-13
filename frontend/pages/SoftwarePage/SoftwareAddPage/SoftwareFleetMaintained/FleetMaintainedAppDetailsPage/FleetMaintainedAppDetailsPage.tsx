import React from "react";
import { Location } from "history";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";

import PATHS from "router/paths";
import { buildQueryStringFromParams } from "utilities/url";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";

import BackLink from "components/BackLink";
import MainContent from "components/MainContent";

const baseClass = "fleet-maintained-app-details-page";

export interface IFleetMaintainedAppDetailsQueryParams {
  team_id?: string;
}

interface IFleetMaintainedAppDetailsRouteParams {
  id: string;
}

interface IFleetMaintainedAppDetailsPageProps {
  location: Location<IFleetMaintainedAppDetailsQueryParams>;
  router: InjectedRouter;
  routeParams: IFleetMaintainedAppDetailsRouteParams;
}

const FleetMaintainedAppDetailsPage = ({
  location,
  router,
  routeParams,
}: IFleetMaintainedAppDetailsPageProps) => {
  const teamId = location.query.team_id;
  const id = parseInt(routeParams.id, 10);

  const { data } = useQuery(["fleet-maintained-app", id], () => {}, {
    ...DEFAULT_USE_QUERY_OPTIONS,
  });

  return (
    <MainContent className={baseClass}>
      <>
        <BackLink
          text="Back to add software"
          path={`${
            PATHS.SOFTWARE_ADD_FLEET_MAINTAINED
          }?${buildQueryStringFromParams({ team_id: teamId })}`}
          className={`${baseClass}__back-to-add-software`}
        />
        <h1>Add Software</h1>
      </>
    </MainContent>
  );
};

export default FleetMaintainedAppDetailsPage;
