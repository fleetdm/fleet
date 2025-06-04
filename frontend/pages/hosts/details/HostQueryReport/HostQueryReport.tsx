import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { browserHistory, InjectedRouter, Link } from "react-router";
import { Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import { AppContext } from "context/app";

import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
import { getPathWithQueryParams } from "utilities/url";
import hqrAPI, { IGetHQRResponse } from "services/entities/host_query_report";
import queryAPI from "services/entities/queries";
import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import BackLink from "components/BackLink";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import Spinner from "components/Spinner";
import HQRTable from "./HQRTable";

const baseClass = "host-query-report";

interface IHostQueryReportProps {
  router: InjectedRouter;
  params: Params;
}

const HostQueryReport = ({
  router,
  params: { host_id, query_id },
}: IHostQueryReportProps) => {
  const { config, currentTeam } = useContext(AppContext);
  const globalReportsDisabled = config?.server_settings.query_reports_disabled;
  const hostId = Number(host_id);
  const queryId = Number(query_id);

  if (globalReportsDisabled) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  const [showQuery, setShowQuery] = useState(false);

  const {
    data: hqrResponse,
    isLoading: hqrLoading,
    error: hqrError,
  } = useQuery<IGetHQRResponse, Error>(
    [hostId, queryId],
    () => hqrAPI.load(hostId, queryId),
    {
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  const {
    isLoading: queryLoading,
    data: queryResponse,
    error: queryError,
  } = useQuery<IGetQueryResponse, Error, ISchedulableQuery>(
    ["query", queryId],
    () => queryAPI.load(queryId),

    {
      select: (data) => data.query,
      enabled: !!queryId,
      refetchOnMount: false,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
    }
  );

  const isLoading = queryLoading || hqrLoading;

  const {
    host_name: hostName,
    report_clipped: reportClipped,
    last_fetched: lastFetched,
    results,
  } = hqrResponse || {};

  // API response is nested this way to mirror that of the full Query Reports response (IQueryReport)
  const rows = results?.map((row) => row.columns) ?? [];

  const {
    name: queryName,
    description: queryDescription,
    query: querySQL,
    discard_data: queryDiscardData,
  } = queryResponse || {};

  // previous reroute can be done before API call, not this one, hence 2
  if (queryDiscardData) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  // Updates title that shows up on browser tabs
  if (queryName && hostName) {
    // e.g., Discover TLS certificates (Rachel's MacBook Pro) | Hosts | Fleet
    document.title = `${queryName} (${hostName}) |
   Hosts | ${DOCUMENT_TITLE_SUFFIX}`;
  } else {
    document.title = `Hosts | ${DOCUMENT_TITLE_SUFFIX}`;
  }

  const HQRHeader = useCallback(() => {
    const fullReportPath = getPathWithQueryParams(
      PATHS.QUERY_DETAILS(queryId),
      { team_id: currentTeam?.id }
    );
    return (
      <div className={`${baseClass}__header`}>
        <div className={`${baseClass}__header__row1`}>
          <BackLink
            text="Back to host details"
            path={PATHS.HOST_QUERIES(hostId)}
          />
        </div>
        <div className={`${baseClass}__header__row2`}>
          {!hqrError && <h1 className="host-name">{hostName}</h1>}
          <Link
            // to and onClick seem redundant
            to={fullReportPath}
            onClick={() => {
              browserHistory.push(fullReportPath);
            }}
            className={`${baseClass}__direction-link`}
          >
            <>
              <span>View full query report</span>
              <Icon name="chevron-right" color="core-fleet-blue" />
            </>
          </Link>
        </div>
      </div>
    );
  }, [queryId, hostId, hqrError, hostName]);

  return (
    <MainContent className={baseClass}>
      {isLoading ? (
        <Spinner />
      ) : (
        <>
          <HQRHeader />
          <HQRTable
            queryName={queryName}
            queryDescription={queryDescription}
            hostName={hostName}
            rows={rows}
            reportClipped={reportClipped}
            lastFetched={lastFetched}
            onShowQuery={() => setShowQuery(true)}
            isLoading={false}
          />
          {showQuery && (
            <ShowQueryModal
              query={querySQL}
              onCancel={() => setShowQuery(false)}
            />
          )}
        </>
      )}
    </MainContent>
  );
};

export default HostQueryReport;
