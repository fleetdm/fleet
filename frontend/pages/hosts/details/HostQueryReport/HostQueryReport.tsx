import BackLink from "components/BackLink";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import { AppContext } from "context/app";
import { ISchedulableQuery } from "interfaces/schedulable_query";
import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { browserHistory, InjectedRouter, Link } from "react-router";
import { Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import hqrAPI, { IGetHQRResponse } from "services/entities/host_query_report";
import queryAPI from "services/entities/queries";
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
  // Need to know:

  // globalReportsDisabled (from app config)
  const { config } = useContext(AppContext);
  const globalReportsDisabled = config?.server_settings.query_reports_disabled;
  // queryDiscardData (from API, CONFIRM?) – need for rerouting
  // or use !lastFetched && (!interval || discardData) ?
  // !lastFechted && !iinterval –> redirect
  // last fetched only matters to differentiate between collecting results and nothing to report

  const hostId = Number(host_id);
  const queryId = Number(query_id);

  // teamId (from API? TODO?)

  // query sql (API)

  if (globalReportsDisabled) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  const [showQuery, setShowQuery] = useState(false);

  // TODO - remove dummy data, restore API call
  const [[hqrResponse, queryResponse], hqrLoading, hqrError] = [
    // // render report
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
    //     host_team_id: 1,
    //     report_clipped: false,
    //     last_fetched: "2021-01-01T00:00:00.000Z",
    //     results: [
    //       {
    //         columns: {
    //           username: "user1",
    //           email: "e@mail",
    //         },
    //       },
    //     ],
    //   },
    //   {
    //     name: 'test query',
    //     query: "SELECT * FROM users",
    //     discard_data: false,
    //     interval: 20,
    //   },
    // ],

    // // collecting results (A)
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
    //     host_team_id: 1,
    //     report_clipped: false,
    //     last_fetched: null,
    //     results: [],
    //   },
    //   {
    //     name: 'test query',
    //     query: "SELECT * FROM users",
    //     discard_data: false,
    //     inverval: 20,
    //   },
    // ],

    // // nothing to report (B)
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
    //     host_team_id: 1,
    //     report_clipped: false,
    //     last_fetched: "2021-01-01T00:00:00.000Z",
    //     results: [],
    //   },
    //   {
    //     name: 'test query',
    //     query: "SELECT * FROM users",
    //     discard_data: false,
    //     inverval: 20,
    //   },
    // ],

    // report clipped (C)
    [
      {
        host_name: "Haley's Macbook Air",
        host_team_id: 1,
        report_clipped: true,
        last_fetched: "2021-01-01T00:00:00.000Z",
        results: [],
      },
      {
        name: "test query",
        query: "SELECT * FROM users",
        interval: 20,
        discard_data: false,
      },
    ],

    // // reroute (local setting)
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
    //     host_team_id: 1,
    //     report_clipped: false,
    //     last_fetched: "2021-01-01T00:00:00.000Z",
    //     results: [
    //       {
    //         columns: {
    //           username: "user1",
    //           email: "e@mail",
    //         },
    //       },
    //     ],
    //   },
    //   {
    //     name: 'test query',
    //     query: "SELECT * FROM users",
    //     discard_data: true,
    //     inverval: 20,
    //   },
    // ],

    false,
    null,
  ];

  // const {
  //   data: hqrResponse,
  //   isLoading: hqrLoading,
  //   error: hqrError,
  // } = useQuery<IGetHQRResponse, Error>(
  //   [hostId, queryId],
  //   () => hqrAPI.load(hostId, queryId),
  //   {
  //     refetchOnMount: false,
  //     refetchOnReconnect: false,
  //     refetchOnWindowFocus: false,
  //   }
  // );

  // const {
  //   isLoading: queryLoading,
  //   data: queryResponse,
  //   error: queryError,
  // } = useQuery<IGetQueryResponse, Error, ISchedulableQuery>(
  //   ["query", queryId],
  //   () => queryAPI.load(queryId),
  //   {
  //     enabled: !!queryId,
  //     refetchOnMount: false,
  //     refetchOnReconnect: false,
  //     refetchOnWindowFocus: false,
  //   }
  // );

  const queryLoading = false;

  const {
    host_name: hostName,
    host_team_id: hostTeamId,
    report_clipped: reportClipped,
    last_fetched: lastFetched,
    results,
    // TODO - remove below casting, just for testing
  } = (hqrResponse || {}) as Partial<IGetHQRResponse>;

  const rows = results?.map((row) => row.columns) ?? [];

  const {
    name: queryName,
    query: querySQL,
    discard_data: queryDiscardData,
    interval: queryInterval,
  } = (queryResponse || {}) as Partial<ISchedulableQuery>;

  // TODO - finalize local setting reroute conditions
  // previous reroute can be done before API call, not this one, hence 2
  if (queryDiscardData) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  const fullReportPath = PATHS.QUERY_DETAILS(queryId, hostTeamId);

  const HQRHeader = () => (
    <div className={`${baseClass}__header`}>
      <span className="row1">
        <BackLink
          text="Back to host details"
          path={PATHS.HOST_QUERIES(hostId)}
        />
      </span>
      <span className="row2">
        {!hqrLoading && !hqrError && <h1 className="host-name">{hostName}</h1>}
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
      </span>
    </div>
  );

  return (
    <MainContent className={baseClass}>
      <>
        <HQRHeader />
        <HQRTable
          {...{
            queryName,
            hostName,
            rows: [],
            reportClipped,
            lastFetched,
            onShowQuery: () => setShowQuery(true),
            isLoading: queryLoading || hqrLoading,
          }}
        />
        {showQuery && (
          <ShowQueryModal
            query={querySQL}
            onCancel={() => setShowQuery(false)}
          />
        )}
      </>
    </MainContent>
  );
};

export default HostQueryReport;
