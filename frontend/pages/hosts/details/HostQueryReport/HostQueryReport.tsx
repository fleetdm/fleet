import BackLink from "components/BackLink";
import EmptyTable from "components/EmptyTable";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import { AppContext } from "context/app";
import React, { useContext, useState } from "react";
import { useQuery } from "react-query";
import { browserHistory, InjectedRouter, Link } from "react-router";
import PATHS from "router/paths";
import hqrAPI, { IGetHQRResponse } from "services/entities/host_query_report";

const baseClass = "host-query-report";

interface IHostQueryReportProps {
  location: {
    query: {
      host_id: string;
      query_id: string;
    };
  };
  router: InjectedRouter;
}

const HostQueryReport = ({ location, router }: IHostQueryReportProps) => {
  // Need to know:

  // globalReportsDisabled (from app config)
  const { config } = useContext(AppContext);
  const globalReportsDisabled = config?.server_settings.query_reports_disabled;
  // queryDiscardData (from API, CONFIRM?) – need for rerouting
  // or use !lastFetched && (!interval || discardData) ?
  // !lastFechted && !iinterval –> redirect
  // last fetched only matters to differentiate between collecting results and nothing to report

  // TODO - fix this, url params are coming through undefined
  const hostId = Number(location.query.host_id);
  const queryId = Number(location.query.query_id);

  // teamId (from API? TODO?)

  // query sql (API)

  if (globalReportsDisabled) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  const [showQuery, setShowQuery] = useState(false);

  // TODO - remove dummy data, restore API call
  const [hqrResponse, hqrLoading, hqrError] = [
    // // render report
    // {
    //   query: "SELECT * FROM users",
    //   discard_data: false,
    //   host_name: "Haley's Macbook Air",
    //   host_team_id: 1,
    //   report_clipped: false,
    //   last_fetched: "2021-01-01T00:00:00.000Z",
    //   results: [
    //     {
    //       columns: {
    //         username: "user1",
    //         email: "e@mail",
    //       },
    //     },
    //   ],
    // },

    // // empty A
    // {
    //   query: "SELECT * FROM users",
    //   discard_data: false,
    //   host_name: "Haley's Macbook Air",
    //   host_team_id: 1,
    //   report_clipped: false,
    //   last_fetched: null,
    //   results: [],
    // },

    // // empty B
    // {
    //   query: "SELECT * FROM users",
    //   discard_data: false,
    //   host_name: "Haley's Macbook Air",
    //   host_team_id: 1,
    //   report_clipped: false,
    //   last_fetched: "2021-01-01T00:00:00.000Z",
    //   results: [],
    // },

    // empty C
    {
      query: "SELECT * FROM users",
      discard_data: false,
      host_name: "Haley's Macbook Air",
      host_team_id: 1,
      report_clipped: true,
      last_fetched: "2021-01-01T00:00:00.000Z",
      results: [],
    },

    // // reroute (local setting)
    // {
    //   query: "SELECT * FROM users",
    //   discard_data: true,
    //   host_name: "Haley's Macbook Air",
    //   host_team_id: 1,
    //   report_clipped: false,
    //   last_fetched: "2021-01-01T00:00:00.000Z",
    //   results: [
    //     {
    //       columns: {
    //         username: "user1",
    //         email: "e@mail",
    //       },
    //     },
    //   ],
    // },

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

  const {
    query,
    host_name: hostName,
    host_team_id: hostTeamId,
    report_clipped: clipped,
    last_fetched: lastFetched,
    discard_data: queryDiscardData,
    results,
  } = hqrResponse || {};

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
        {showQuery && <ShowQueryModal {...{ querySQL: query, onCancel }} />}
      </>
    </MainContent>
  );
};

export default HostQueryReport;
