import React from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import {
  IGetVulnerabilitiesQueryKey,
  IVulnerabilitiesResponse,
  getVulnerabilities,
} from "services/entities/vulnerabilities";

import TableDataError from "components/DataError";

import SoftwareVulnerabilitiesTable from "./SoftwareVulnerabilitiesTable";

const baseClass = "software-vulnerabilities";

interface ISoftwareVulnerabilitiesProps {
  router: InjectedRouter;
  isSoftwareEnabled: boolean;
  perPage: number;
  query?: string;
  orderDirection: "asc" | "desc";
  orderKey: string;
  currentPage: number;
  teamId?: number;
  exploited?: string;
}

const SoftwareVulnerabilities = ({
  router,
  isSoftwareEnabled,
  query,
  perPage,
  orderDirection,
  orderKey,
  currentPage,
  teamId,
  exploited,
}: ISoftwareVulnerabilitiesProps) => {
  const queryParams = {
    page: currentPage,
    per_page: perPage,
    order_direction: orderDirection,
    order_key: orderKey,
    teamId,
    query,
    showExploitedVulnerabilitiesOnly: Boolean(exploited),
  };

  const { data, isFetching, isError } = useQuery<
    IVulnerabilitiesResponse,
    Error,
    IVulnerabilitiesResponse,
    IGetVulnerabilitiesQueryKey[]
  >(
    [
      {
        scope: "software-vulnerabilities",
        ...queryParams,
      },
    ],
    () => getVulnerabilities(queryParams),
    {
      keepPreviousData: true,
      staleTime: 30000,
    }
  );

  if (isError) {
    return <TableDataError className={`${baseClass}__table-error`} />;
  }

  return (
    <div className={baseClass}>
      <SoftwareVulnerabilitiesTable
        router={router}
        data={data}
        query={query}
        showExploitedVulnerabilitiesOnly={
          queryParams.showExploitedVulnerabilitiesOnly
        }
        isSoftwareEnabled={isSoftwareEnabled}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={isFetching}
      />
    </div>
  );
};

export default SoftwareVulnerabilities;
