/** software/vulnerabilities Vulnerabilities tab */

import React, { useState } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";
import softwareVulnAPI, {
  IGetVulnerabilitiesQueryKey,
  IVulnerabilitiesResponse,
  IGetVulnerabilityQueryKey,
  IVulnerabilityResponse,
  getVulnerabilities,
  IVulnerabiltiesEmptyStateReason,
} from "services/entities/vulnerabilities";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { stripQuotes } from "utilities/strings/stringUtils";

import TableDataError from "components/DataError";
import Spinner from "components/Spinner";

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
  showExploitedVulnerabilitiesOnly: boolean;
  resetPageIndex: boolean;
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
  showExploitedVulnerabilitiesOnly,
  resetPageIndex,
}: ISoftwareVulnerabilitiesProps) => {
  const [tableData, setTableData] = useState<IVulnerabilitiesResponse>();
  const [
    emptyStateReason,
    setEmptyStateReason,
  ] = useState<IVulnerabiltiesEmptyStateReason>();

  const queryParams = {
    page: currentPage,
    per_page: perPage,
    order_direction: orderDirection,
    order_key: orderKey,
    teamId,
    query,
    exploit: showExploitedVulnerabilitiesOnly,
  };

  const isExactMatchQuery = (() => {
    if (query) {
      const pattern = /^(['"]).*\1$/;
      return pattern.test(query);
    }
    return false;
  })();

  const { isFetching, isLoading, isError } = useQuery<
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
      enabled: !isExactMatchQuery,
      onSuccess: (data) => {
        setTableData(data);
        if (data.count === 0) {
          if (queryParams.exploit) {
            setEmptyStateReason("no-matching-items");
          }
          setEmptyStateReason("no-vulns-detected");
        }
      },
    }
  );

  // Calling software/vulnerabilities/:CVE endpoint when user searches with quotation marks
  const {
    isLoading: isLoadingExactMatch,
    isFetching: isFetchingExactMatch,
    isError: isExactMatchError,
  } = useQuery<
    IVulnerabilityResponse,
    any, // TODO: Fix error type
    IVulnerabilityResponse,
    IGetVulnerabilityQueryKey[]
  >(
    [
      {
        scope: "softwareVulnByCVE",
        vulnerability: (query && stripQuotes(query)) || "",
        teamId,
      },
    ],
    ({ queryKey }) => {
      return softwareVulnAPI.getVulnerability(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      onSuccess: (data) => {
        if (queryParams.exploit && !data.vulnerability.cisa_known_exploit) {
          setTableData({
            count: 0,
            counts_updated_at: "",
            vulnerabilities: [],
            meta: {
              has_next_results: false,
              has_previous_results: false,
            },
            known_vulnerability: false,
          });
          setEmptyStateReason("no-matching-items");
        }
        setTableData({
          count: 1,
          counts_updated_at: data.vulnerability.hosts_count_updated_at,
          vulnerabilities: [data.vulnerability],
          meta: {
            has_next_results: false,
            has_previous_results: false,
          },
          known_vulnerability: true,
        });
      },
      onError: (error) => {
        if (error.status === 400) {
          if (
            error?.data?.errors &&
            error.data.errors[0].reason.includes(
              "That vulnerability (CVE) is not valid."
            )
          ) {
            setTableData({
              count: 0,
              counts_updated_at: "",
              vulnerabilities: [],
              meta: {
                has_next_results: false,
                has_previous_results: false,
              },
              known_vulnerability: false,
            });
            setEmptyStateReason("invalid-cve");
          }
        } else if (error.status === 404) {
          if (
            error?.data?.errors &&
            error.data.errors[0].reason.includes("This is not known CVE.")
          ) {
            setTableData({
              count: 0,
              counts_updated_at: "",
              vulnerabilities: [],
              meta: {
                has_next_results: false,
                has_previous_results: false,
              },
              known_vulnerability: false,
            });
            setEmptyStateReason("unknown-cve");
          }
        }
      },
      enabled: isExactMatchQuery,
    }
  );

  if (isLoading || isLoadingExactMatch) {
    return <Spinner />;
  }

  if (isError) {
    return <TableDataError className={`${baseClass}__table-error`} />;
  }

  return (
    <div className={baseClass}>
      <SoftwareVulnerabilitiesTable
        router={router}
        data={tableData}
        emptyStateReason={emptyStateReason}
        query={query}
        showExploitedVulnerabilitiesOnly={showExploitedVulnerabilitiesOnly}
        isSoftwareEnabled={isSoftwareEnabled}
        perPage={perPage}
        orderDirection={orderDirection}
        orderKey={orderKey}
        currentPage={currentPage}
        teamId={teamId}
        isLoading={isFetching || isFetchingExactMatch}
        resetPageIndex={resetPageIndex}
      />
    </div>
  );
};

export default SoftwareVulnerabilities;
