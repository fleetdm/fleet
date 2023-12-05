import BackLink from "components/BackLink";
import EmptyTable from "components/EmptyTable";
import Icon from "components/Icon";
import MainContent from "components/MainContent";
import ShowQueryModal from "components/modals/ShowQueryModal";
import React, { useState } from "react";
import { browserHistory, InjectedRouter, Link, routerShape } from "react-router";
import PATHS from "router/paths";

const baseClass = "host-query-report";

interface IHostQueryReportProps {
  router: InjectedRouter;
}

const HostQueryReport = ({router}: IHostQueryReportProps) => {
  // Need to know:

  // globalReportsDisabled (from app config)
  // queryDiscardData (from API, CONFIRM?) – need for rerouting
    // or use !lastFetched && (!interval || discardData) ?
      // !lastFechted && !iinterval –> redirect
      // last fetched only matters to differentiate between collecting results and nothing to report
  // hostId (from path)
  // queryId (from path)
  // hostName (from API)
  // report clipped (from API)
  // query has run on this host (!!lastFetched, from API)
  // query has stored results (!!.results, from API)
  // teamId (TODO?)

  // GET /api/v1/fleet/hosts/{hostId}/queries/{queryId}

  if (globalReportsDisabled || queryDiscardData) {
    router.push(PATHS.HOST_QUERIES(hostId);
  }
  const [showQuery, setShowQuery] = useState(false);

  const onCancel = () => {
    setShowQuery(false);
  };

  const fullReportPath = PATHS.QUERY_DETAILS(queryId, teamId);

  const onFullReportClick = () => {
      browserHistory.push(fullReportPath);
  };

  const renderHeader = () => {
  //  TODO - style this with CSS grid?
   <div className={`${baseClass}__header`}> 
    {/* // Back to host details button */}
    <BackLink text="Back to host details" path={PATHS.HOST_QUERIES(hostId)} />
    {/* // Host name */}
     {!isLoading && !apiError && (
       <h1 className={`${baseClass}__host-name`}>
         {hostName}
       </h1>
    )}
    {/* // View full query report button */}
    {/* TODO - how should teamId work here? */}
    <Link  to={fullReportPath} onClick={onFullReportClick}>
      <>
      <span>View full query report</span>
        <Icon
          name="chevron-right"
          // className={`${baseClass}__forward-icon`}
          color="core-fleet-blue"
        />
      </>
    </Link>
  };

  const renderContent = () => {
    // Gabe – make sense to move this to its own component?
    if (!queryHasRunOnThisHost) {
      // A - collecting results state
      return <EmptyTable
        className={`${baseClass}__collecting-results`}
        graphicName="collecting-results"
        header="Collecting results..."
        info={`Fleet is collecting query results from ${hostName}. Check back later.`}
      />
    }
    if (!queryHasStoredResults) {
      if (reportClipped) {
        // C – Report clipped state
        return <EmptyTable
          className={`${baseClass}__report-clipped`}
          graphicName="empty-software"
          header="Report clipped"
          info="This query has paused reporting in Fleet, and no results were saved for this host."
        />
      } else {
        // B - Nothing to report state
        return <EmptyTable
          className={`${baseClass}__nothing-to-report`}
          graphicName="empty-software"
          header="Nothing to report"
          info={`This query has run on ${hostName}, but returned no data for this host.`}
        />
      }

    // render the report
    renderHQR();

  };

  const renderHQR = () => {
      //  TODO
  };

  return (
    <MainContent className={baseClass}>
      <>
        {renderHeader()}
        {renderContent()}
        {showQuery && <ShowQueryModal {...{ query, onCancel }} />}
      </>
    </MainContent>
  );
};

export default HostQueryReport;
