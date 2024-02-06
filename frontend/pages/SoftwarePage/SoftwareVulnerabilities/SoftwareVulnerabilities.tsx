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
  orderDirection: "asc" | "desc";
  orderKey: string;
  currentPage: number;
  teamId?: number;
}

const SoftwareVulnerabilities = ({
  router,
  isSoftwareEnabled,
  perPage,
  orderDirection,
  orderKey,
  currentPage,
  teamId,
}: ISoftwareVulnerabilitiesProps) => {
  const queryParams = {
    page: currentPage,
    per_page: perPage,
    order_direction: orderDirection,
    order_key: orderKey,
    teamId,
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
