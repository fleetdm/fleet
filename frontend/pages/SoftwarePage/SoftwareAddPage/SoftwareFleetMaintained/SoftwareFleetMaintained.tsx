import React from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";

import { DEFAULT_QUERY } from "utilities/constants";

import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";

const baseClass = "software-fleet-maintained";

interface ISoftwareFleetMaintainedProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: Location<ISoftwareAddPageQueryParams>;
}

// default values for query params used on this page if not provided
const DEFAULT_SORT_DIRECTION = "desc";
const DEFAULT_SORT_HEADER = "hosts_count";
const DEFAULT_PAGE_SIZE = 20;
const DEFAULT_PAGE = 0;

const SoftwareFleetMaintained = ({
  currentTeamId,
  router,
  location,
}: ISoftwareFleetMaintainedProps) => {
  const {
    order_key = DEFAULT_SORT_HEADER,
    order_direction = DEFAULT_SORT_DIRECTION,
    query = DEFAULT_QUERY,
    page,
  } = location.query;
  const currentPage = page ? parseInt(page, 10) : DEFAULT_PAGE;

  return <div className={baseClass}>Maintained Page</div>;
};

export default SoftwareFleetMaintained;
