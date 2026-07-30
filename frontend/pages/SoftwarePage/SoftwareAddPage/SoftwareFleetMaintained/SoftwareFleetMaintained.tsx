import React, { useContext } from "react";
import { InjectedRouter } from "react-router";
import { Location } from "history";
import { useQuery } from "react-query";
import { AxiosError } from "axios";
import { omit } from "lodash";

import softwareAPI, {
  ISoftwareFleetMaintainedAppsQueryParams,
  ISoftwareFleetMaintainedAppsResponse,
} from "services/entities/software";
import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { AppContext } from "context/app";

import Spinner from "components/Spinner";
import DataError from "components/DataError";
import PremiumFeatureMessage from "components/PremiumFeatureMessage";

import FleetMaintainedAppsTable from "./FleetMaintainedAppsTable";
import { ISoftwareAddPageQueryParams } from "../SoftwareAddPage";

const baseClass = "software-fleet-maintained";

const DATA_STALE_TIME = 30000;
const QUERY_OPTIONS = {
  keepPreviousData: true,
  staleTime: DATA_STALE_TIME,
};

interface IQueryKey extends ISoftwareFleetMaintainedAppsQueryParams {
  scope?: "fleet-maintained-apps";
}

interface ISoftwareFleetMaintainedProps {
  currentTeamId: number;
  router: InjectedRouter;
  location: Location<ISoftwareAddPageQueryParams>;
}

// default values for query params used on this page if not provided
const DEFAULT_SORT_DIRECTION = "asc";
const DEFAULT_SORT_HEADER = "name";
// The list is paginated server-side by app (an app's macOS and Windows entries
// are combined into a single row). 100 apps per page keeps the full library
// reachable without an unbounded response.
const DEFAULT_PAGE_SIZE = 100;
const DEFAULT_PAGE = 0;

const SoftwareFleetMaintained = ({
  currentTeamId,
  router,
  location,
}: ISoftwareFleetMaintainedProps) => {
  const { isPremiumTier } = useContext(AppContext);

  const {
    order_key = DEFAULT_SORT_HEADER,
    order_direction = DEFAULT_SORT_DIRECTION,
    query = "",
    page,
    platform,
    status,
  } = location.query;
  const currentPage = page ? parseInt(page, 10) : DEFAULT_PAGE;

  // Platform and "hide added apps" are filtered server-side. Map the UI's
  // platform value ("macos"/"windows") to the API's ("darwin"/"windows") and
  // the status toggle to the `available` flag. Undefined values are omitted
  // from the request.
  let apiPlatform: "darwin" | "windows" | undefined;
  if (platform === "macos") {
    apiPlatform = "darwin";
  } else if (platform === "windows") {
    apiPlatform = "windows";
  }
  const availableOnly = status === "available" ? true : undefined;

  const { data, isLoading, isFetching, isError } = useQuery<
    ISoftwareFleetMaintainedAppsResponse,
    AxiosError,
    ISoftwareFleetMaintainedAppsResponse,
    [IQueryKey]
  >(
    [
      {
        scope: "fleet-maintained-apps",
        page: currentPage,
        per_page: DEFAULT_PAGE_SIZE,
        query,
        order_direction,
        order_key,
        team_id: currentTeamId,
        platform: apiPlatform,
        available: availableOnly,
      },
    ],
    ({ queryKey: [queryKey] }) => {
      return softwareAPI.getFleetMaintainedApps(omit(queryKey, "scope"));
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      ...QUERY_OPTIONS,
    }
  );

  if (!isPremiumTier) {
    return (
      <PremiumFeatureMessage className={`${baseClass}__premium-message`} />
    );
  }

  if (isLoading) {
    return <Spinner />;
  }

  if (isError) {
    return <DataError verticalPaddingSize="pad-xxxlarge" />;
  }

  return (
    <div className={baseClass}>
      <FleetMaintainedAppsTable
        data={data}
        isLoading={isFetching}
        router={router}
        query={query}
        teamId={currentTeamId}
        orderDirection={order_direction}
        orderKey={order_key}
        perPage={DEFAULT_PAGE_SIZE}
        currentPage={currentPage}
        platformParam={platform}
        statusParam={status}
      />
    </div>
  );
};

export default SoftwareFleetMaintained;
