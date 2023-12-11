import BackLink from "components/BackLink";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import Spinner from "components/Spinner";
import { AppContext } from "context/app";
import { ISchedulableQuery } from "interfaces/schedulable_query";
import React, { useCallback, useContext, useState } from "react";
import { useQuery } from "react-query";
import { browserHistory, InjectedRouter, Link } from "react-router";
import { Params } from "react-router/lib/Router";
import PATHS from "router/paths";
import hqrAPI, { IGetHQRResponse } from "services/entities/host_query_report";
import queryAPI from "services/entities/queries";
import { DOCUMENT_TITLE_SUFFIX } from "utilities/constants";
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
  const { config } = useContext(AppContext);
  const globalReportsDisabled = config?.server_settings.query_reports_disabled;
  const hostId = Number(host_id);
  const queryId = Number(query_id);

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
    //     report_clipped: false,
    //     last_fetched: "2021-01-01T00:00:00.000Z",
    //     results: [
    //       {
    //         columns: {
    //           username: "user1",
    //           email: "e@mail",
    //           ausername: "user1",
    //           aemail: "e@mail",
    //           aausername: "user1",
    //           aaemail: "e@mail",
    //           aaausername: "user1",
    //           aaaemail: "e@mail",
    //           aaaausername: "user1",
    //           aaaaemail: "e@mail",
    //           aaaaausername: "user1",
    //           aaaaaemail: "e@mail",
    //           aaaaaausername: "user1",
    //           aaaaaaemail: "e@mail",
    //           aaaaaaausername: "user1",
    //           aaaaaaaemail: "e@mail",
    //           aaaaaaaausername: "user1",
    //           aaaaaaaaemail: "e@mail",
    //           aaaaaaaaausername: "user1",
    //           aaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaausername: "user1",
    //           aaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //         },
    //       },
    //       {
    //         columns: {
    //           username: "zser1",
    //           email: "e@mail",
    //           ausername: "user1",
    //           aemail: "e@mail",
    //           aausername: "user1",
    //           aaemail: "e@mail",
    //           aaausername: "user1",
    //           aaaemail: "e@mail",
    //           aaaausername: "user1",
    //           aaaaemail: "e@mail",
    //           aaaaausername: "user1",
    //           aaaaaemail: "e@mail",
    //           aaaaaausername: "user1",
    //           aaaaaaemail: "e@mail",
    //           aaaaaaausername: "user1",
    //           aaaaaaaemail: "e@mail",
    //           aaaaaaaausername: "user1",
    //           aaaaaaaaemail: "e@mail",
    //           aaaaaaaaausername: "user1",
    //           aaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaausername: "user1",
    //           aaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //         },
    //       },
    //       {
    //         columns: {
    //           username: "aser1",
    //           email: "e@mail",
    //           ausername: "user1",
    //           aemail: "e@mail",
    //           aausername: "user1",
    //           aaemail: "e@mail",
    //           aaausername: "user1",
    //           aaaemail: "e@mail",
    //           aaaausername: "user1",
    //           aaaaemail: "e@mail",
    //           aaaaausername: "user1",
    //           aaaaaemail: "e@mail",
    //           aaaaaausername: "user1",
    //           aaaaaaemail: "e@mail",
    //           aaaaaaausername: "user1",
    //           aaaaaaaemail: "e@mail",
    //           aaaaaaaausername: "user1",
    //           aaaaaaaaemail: "e@mail",
    //           aaaaaaaaausername: "user1",
    //           aaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaausername: "user1",
    //           aaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //           aaaaaaaaaaaaaaaaaaaaaaausername: "user1",
    //           aaaaaaaaaaaaaaaaaaaaaaaemail: "e@mail",
    //         },
    //       },
    //     ],
    //   },
    //   {
    //     name: "Test Query",
    //     description: "A great query",
    //     query: "SELECT * FROM users",
    //     discard_data: false,
    //     interval: 20,
    //   },
    // ],

    // collecting results (A)
    [
      {
        host_name: "Haley's Macbook Air",
        report_clipped: false,
        last_fetched: null,
        results: [],
      },
      {
        name: "Test Query",
        description: "a great query",
        query: "SELECT * FROM users",
        discard_data: false,
        inverval: 20,
      },
    ],

    // // nothing to report (B)
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
    //     report_clipped: false,
    //     last_fetched: "2021-01-01T00:00:00.000Z",
    //     results: [],
    //   },
    //   {
    //     name: "Test Query",
    //     description: "a great query",
    //     query: "SELECT * FROM users",
    //     discard_data: false,
    //     inverval: 20,
    //   },
    // ],

    // // report clipped (C)
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
    //     report_clipped: true,
    //     last_fetched: "2021-01-01T00:00:00.000Z",
    //     results: [],
    //   },
    //   {
    //     name: "Test Query",
    //     description: "a great query",
    //     query: "SELECT * FROM users",
    //     interval: 20,
    //     discard_data: false,
    //   },
    // ],

    // // reroute (local setting)
    // [
    //   {
    //     host_name: "Haley's Macbook Air",
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
    //     name: 'Test Query',
    //     description: "a great query",
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

  // TODO - remove mock loading state
  const queryLoading = false;

  const isLoading = queryLoading || hqrLoading;

  const {
    host_name: hostName,
    report_clipped: reportClipped,
    last_fetched: lastFetched,
    results,
    // TODO - remove below casting, just for testing
  } = (hqrResponse || {}) as Partial<IGetHQRResponse>;

  // API response is nested this way to mirror that of the full Query Reports response (IQueryReport)
  const rows = results?.map((row) => row.columns) ?? [];

  const {
    name: queryName,
    description: queryDescription,
    query: querySQL,
    discard_data: queryDiscardData,
  } = (queryResponse || {}) as Partial<ISchedulableQuery>;

  // TODO - finalize local setting reroute conditions
  // previous reroute can be done before API call, not this one, hence 2
  if (queryDiscardData) {
    router.push(PATHS.HOST_QUERIES(hostId));
  }

  document.title = `Host query report | ${queryName} | ${hostName} | ${DOCUMENT_TITLE_SUFFIX}`;

  const HQRHeader = useCallback(() => {
    const fullReportPath = PATHS.QUERY_DETAILS(queryId);
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
