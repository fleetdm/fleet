/**
 software/library Library tab — fleet-managed software available for installation
 */
import React from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { omit } from "lodash";

import PATHS from "router/paths";
import softwareAPI, {
  ISoftwareTitlesQueryKey,
  ISoftwareTitlesResponse,
} from "services/entities/software";

import Spinner from "components/Spinner";
import TableDataError from "components/DataError";

import SoftwareLibraryTable from "./SoftwareLibraryTable";

const baseClass = "software-library";

const DATA_STALE_TIME = 30000;
const QUERY_OPTIONS = {
  keepPreviousData: true,
  staleTime: DATA_STALE_TIME,
};

interface ISoftwareLibraryProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  query: string;
  perPage: number;
  orderDirection: "asc" | "desc";
  orderKey: string;
  selfServiceOnly: boolean;
  currentPage: number;
  teamId?: number;
}

const SoftwareLibrary = ({
  router,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  selfServiceOnly,
  currentPage,
  teamId,
}: ISoftwareLibraryProps) => {
  const {
    data: titlesData,
    isFetching: isTitlesFetching,
    isLoading: isTitlesLoading,
    isError: isTitlesError,
  } = useQuery<
    ISoftwareTitlesResponse,
    Error,
    ISoftwareTitlesResponse,
    [ISoftwareTitlesQueryKey]
  >(
    [
      {
        scope: "software-titles",
        page: currentPage,
        perPage,
        query,
        orderDirection,
        orderKey,
        teamId,
        availableForInstall: true,
        ...(selfServiceOnly ? { selfService: true } : {}),
      },
    ],
    ({ queryKey: [queryKey] }) =>
      softwareAPI.getSoftwareTitles(omit(queryKey, "scope")),
    {
      ...QUERY_OPTIONS,
      enabled: location.pathname === PATHS.SOFTWARE_LIBRARY,
    }
  );

  if (isTitlesLoading) {
    return <Spinner />;
  }

  if (isTitlesError) {
    return <TableDataError verticalPaddingSize="pad-xxxlarge" />;
  }

  return (
    <div className={baseClass}>
      <SoftwareLibraryTable
        router={router}
        data={titlesData}
        isSoftwareEnabled={isSoftwareEnabled}
        query={query}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        selfServiceOnly={selfServiceOnly}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={isTitlesFetching}
      />
    </div>
  );
};

export default SoftwareLibrary;
