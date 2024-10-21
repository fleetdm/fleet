/** software/vulnerabilities Vulnerabilities tab */

import React, { useState, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter } from "react-router";
import { AxiosError } from "axios";
import softwareVulnAPI, {
  IGetVulnerabilitiesQueryKey,
  IVulnerabilitiesResponse,
  IGetVulnerabilityQueryKey,
  IVulnerabilityResponse,
  getVulnerabilities,
  IVulnerabilitiesEmptyStateReason,
} from "services/entities/vulnerabilities";
import { IApiError } from "interfaces/errors";

import { DEFAULT_USE_QUERY_OPTIONS } from "utilities/constants";
import { stripQuotes } from "utilities/strings/stringUtils";

import TableDataError from "components/DataError";
import Spinner from "components/Spinner";

import SoftwareVulnerabilitiesTable from "./SoftwareVulnerabilitiesTable";
import { isValidCVEFormat } from "./SoftwareVulnerabilitiesTable/helpers";

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
  ] = useState<IVulnerabilitiesEmptyStateReason>();

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
      enabled: !isExactMatchQuery && isSoftwareEnabled,
      onSuccess: (data) => {
        setTableData(data);
        if (data.count === 0) {
          if (
            queryParams.exploit ||
            (queryParams.query && queryParams.query.length > 0)
          ) {
            setEmptyStateReason("no-matching-items");
          } else {
            setEmptyStateReason("no-vulns-detected");
          }
        }
      },
    }
  );

  // Calling software/vulnerabilities/:CVE endpoint when user searches with quotation marks
  const {
    isLoading: isLoadingExactMatch,
    isFetching: isFetchingExactMatch,
    refetch: refetchExactMatch,
  } = useQuery<
    IVulnerabilityResponse | null,
    AxiosError<IApiError>,
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
      // Revisit: Refactor to return status alongside data and check for 204 instead of !data
      return softwareVulnAPI.getVulnerability(queryKey[0]);
    },
    {
      ...DEFAULT_USE_QUERY_OPTIONS,
      retry: false,
      onSuccess: (data) => {
        // Handle 204 response which doesn't return data if it is a known CVE but doesn't exist in response
        if (!data) {
          setTableData({
            count: 0,
            counts_updated_at: "",
            vulnerabilities: [],
            meta: {
              has_next_results: false,
              has_previous_results: false,
            },
          });
          setEmptyStateReason("known-vuln");
        }
        // If filtering for exploited vulns, hide vulnerability if cisa_known_exploit is false
        else if (
          queryParams.exploit &&
          !data.vulnerability.cisa_known_exploit
        ) {
          setTableData({
            count: 0,
            counts_updated_at: "",
            vulnerabilities: [],
            meta: {
              has_next_results: false,
              has_previous_results: false,
            },
          });
          setEmptyStateReason("no-matching-items");
          // Otherwise return IVulnerabilityResponse as IVulnerabilitiesResponse format
        } else {
          setTableData({
            count: 1,
            counts_updated_at: data.vulnerability.hosts_count_updated_at,
            vulnerabilities: [data.vulnerability],
            meta: {
              has_next_results: false,
              has_previous_results: false,
            },
          });
        }
      },
      onError: (err) => {
        // Type assertion for failing "Property 'data' does not exist on type 'AxiosError<IApiError, any>"
        const error = err as AxiosError<IApiError> & {
          data?: IApiError;
        };

        // Handle 400 response which is an invalid CVE format
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
            });
            setEmptyStateReason("invalid-cve");
          }

          // Handle 404 response which is BE validated CVE string but not a known CVE
        } else if (error.status === 404) {
          if (
            error?.data?.errors &&
            (error.data.errors[0].reason.includes("This is not a known CVE.") ||
              error.data.errors[0].reason.includes(
                "was not found in the datastore"
              ))
          ) {
            // FE validatation for CVE string
            if (query && !isValidCVEFormat(stripQuotes(query))) {
              setEmptyStateReason("invalid-cve");
            } else {
              setEmptyStateReason("unknown-cve");
            }
          }
          setTableData({
            count: 0,
            counts_updated_at: "",
            vulnerabilities: [],
            meta: {
              has_next_results: false,
              has_previous_results: false,
            },
          });
        }
      },
      enabled: isExactMatchQuery && isSoftwareEnabled,
    }
  );

  // If a user toggles between exact exploit and non-exploit,
  // we need the table data to recheck cisa_known_exploit and populate accordingly
  useEffect(() => {
    if (isExactMatchQuery) {
      refetchExactMatch();
    }
  }, [queryParams.exploit, isExactMatchQuery]);

  // !tableData is used to show the Spinner only on the first render.
  // This prevents the Spinner from flashing on every data refresh, noticable
  // when going between search and exact match search deselects search box.
  if (!tableData && (isLoading || isLoadingExactMatch)) {
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
