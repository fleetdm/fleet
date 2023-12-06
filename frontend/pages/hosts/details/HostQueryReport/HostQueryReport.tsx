import BackLink from "components/BackLink";
import EmptyTable from "components/EmptyTable";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import { AppContext } from "context/app";
import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { browserHistory, InjectedRouter, Link } from "react-router";
import { Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import hqrAPI, { IGetHQRResponse } from "services/entities/host_query_report";
import queryAPI from "services/entities/queries";

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
  //   isLoading: isQueryLoading,
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

  const {
    host_name: hostName,
    host_team_id: hostTeamId,
    report_clipped: clipped,
    last_fetched: lastFetched,
    results,
  } = hqrResponse || {};

  const {
    query: querySQL,
    discard_data: queryDiscardData,
    interval: queryInterval,
  } = queryResponse || {};

  // TODO - finalize local setting reroute conditions
  // previous reroute can be done before API call, not this one, hence 2
  if (queryDiscardData) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  const onCancel = () => {
    setShowQuery(false);
  };

  const fullReportPath = PATHS.QUERY_DETAILS(queryId, hostTeamId);

  const onFullReportClick = () => {
    browserHistory.push(fullReportPath);
  };

  const renderHQR = () => {
    //  TODO
  };

  const renderHeader = () => (
    //  TODO - style this with CSS grid?
    <div className={`${baseClass}__header`}>
      <BackLink text="Back to host details" path={PATHS.HOST_QUERIES(hostId)} />
      {!hqrLoading && !hqrError && (
        <h1 className={`${baseClass}__host-name`}>{hostName}</h1>
      )}
      {/* TODO - how should teamId work here? */}
      <Link to={fullReportPath} onClick={onFullReportClick}>
        <>
          <span>View full query report</span>
          <Icon
            name="chevron-right"
            // className={`${baseClass}__forward-icon`}
            color="core-fleet-blue"
          />
        </>
      </Link>
    </div>
  );
  const renderContent = () => {
    // if query hasn't run on this host
    if (!lastFetched) {
      // collecting results
      return (
        <EmptyTable
          className={`${baseClass}__collecting-results`}
          graphicName="collecting-results"
          header="Collecting results..."
          info={`Fleet is collecting query results from ${hostName}. Check back later.`}
        />
      );
    }
    if (results.length === 0) {
      if (clipped) {
        // report clipped
        return (
          <EmptyTable
            className={`${baseClass}__report-clipped`}
            graphicName="empty-software"
            header="Report clipped"
            info="This query has paused reporting in Fleet, and no results were saved for this host."
          />
        );
      }
      return (
        // nothing to report
        <EmptyTable
          className={`${baseClass}__nothing-to-report`}
          graphicName="empty-software"
          header="Nothing to report"
          info={`This query has run on ${hostName}, but returned no data for this host.`}
        />
      );
    }
    // render the report
    renderHQR();
  };

  return (
    <MainContent className={baseClass}>
      <>
        {renderHeader()}
        {renderContent()}
        {showQuery && (
          <ShowQueryModal {...{ querySQL: queryResponse, onCancel }} />
        )}
      </>
    </MainContent>
  );
};

export default HostQueryReport;
