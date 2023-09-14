import React, { useContext } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";

import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import queryAPI from "services/entities/queries";

import Button from "components/buttons/Button";
import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import useTeamIdParam from "hooks/useTeamIdParam";
import EmptyTable from "components/EmptyTable/EmptyTable";

interface IQueryDetailsPageProps {
  router: InjectedRouter; // v3
  params: Params;
  location: {
    pathname: string;
    query: { team_id?: string };
    search: string;
  };
}

const baseClass = "query-details-page";

const QueryDetailsPage = ({
  router,
  params: { id: paramsQueryId },
  location,
}: IQueryDetailsPageProps): JSX.Element => {
  const queryId = paramsQueryId ? parseInt(paramsQueryId, 10) : null;

  const {
    currentTeamName: teamNameForQuery,
    teamIdForApi: apiTeamIdForQuery,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const handlePageError = useErrorHandler();
  const {
    isGlobalAdmin,
    isGlobalMaintainer,
    isAnyTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    isAnyTeamObserverPlus,
  } = useContext(AppContext);
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryObserverCanRun,
    lastEditedQueryFrequency,
    setLastEditedQueryId,
    setLastEditedQueryName,
    setLastEditedQueryDescription,
    setLastEditedQueryBody,
    setLastEditedQueryObserverCanRun,
    setLastEditedQueryFrequency,
    setLastEditedQueryLoggingType,
    setLastEditedQueryMinOsqueryVersion,
    setLastEditedQueryPlatforms,
  } = useContext(QueryContext);

  // disabled on page load so we can control the number of renders
  // else it will re-populate the context on occasion
  const {
    isLoading: isStoredQueryLoading,
    data: storedQuery,
    error: storedQueryError,
  } = useQuery<IGetQueryResponse, Error, ISchedulableQuery>(
    ["query", queryId],
    () => queryAPI.load(queryId as number),
    {
      enabled: !!queryId,
      refetchOnWindowFocus: false,
      select: (data) => data.query,
      onSuccess: (returnedQuery) => {
        setLastEditedQueryId(returnedQuery.id);
        setLastEditedQueryName(returnedQuery.name);
        setLastEditedQueryDescription(returnedQuery.description);
        setLastEditedQueryBody(returnedQuery.query);
        setLastEditedQueryObserverCanRun(returnedQuery.observer_can_run);
        setLastEditedQueryFrequency(returnedQuery.interval);
        setLastEditedQueryPlatforms(returnedQuery.platform);
        setLastEditedQueryLoggingType(returnedQuery.logging);
        setLastEditedQueryMinOsqueryVersion(returnedQuery.min_osquery_version);
      },
      onError: (error) => handlePageError(error),
    }
  );

  const canEditQuery =
    isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin;

  const renderHeader = () => {
    return (
      <>
        {" "}
        <div className={`${baseClass}__header-links`}>
          <BackLink text="Back to queries" path={PATHS.MANAGE_QUERIES} />
        </div>
        <div className={`${baseClass}__title-bar`}>
          <div className="name-description">
            <h1 className={`${baseClass}__query-name`}>
              {lastEditedQueryName}
            </h1>
            <p className={`${baseClass}__query-description`}>
              {lastEditedQueryDescription}
            </p>
          </div>
          <div className={`${baseClass}__action-button-container`}>
            <Button
              onClick={() => {
                queryId && router.push(PATHS.EDIT_QUERY(queryId));
              }}
              className={`${baseClass}__manage-automations button`}
              variant="brand"
            >
              {canEditQuery ? "Edit query" : "More details"}
            </Button>
            {(lastEditedQueryObserverCanRun ||
              isObserverPlus ||
              isAnyTeamObserverPlus ||
              canEditQuery) && (
              <div
                className={`${baseClass}__button-wrap ${baseClass}__button-wrap--new-query`}
              >
                <Button
                  className={`${baseClass}__run`}
                  variant="blue-green"
                  // onClick={goToSelectTargets} // TODO
                  onClick={() => {
                    console.log("goToSelectTargets");
                  }}
                >
                  Live query
                </Button>
              </div>
            )}
          </div>
        </div>
      </>
    );
  };

  const renderReport = () => {
    const collectingResults = true; // TODO: Fix so it's correct
    const noResults = true; // TODO: Fix so it's correct
    const dynamicFrequency = lastEditedQueryFrequency; // TODO: Fix so it's correct
    const collectingResultsInfo = () =>
      `Fleet is collecting query results. Check back in about ${dynamicFrequency} hour.`;
    const noResultsInfo = () => {
      return "This query has returned no data so far."; // TODO: Fix so it's correct
    };
    if (collectingResults) {
      return (
        <EmptyTable
          header={"Collecting results..."}
          info={collectingResultsInfo()}
        />
      );
    }
    if (noResults) {
      return (
        <EmptyTable header={"Nothing to report yet"} info={noResultsInfo()} />
      );
    }
    return <>Report here</>; // TODO
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        {renderHeader()}
        {renderReport()}
      </div>
    </MainContent>
  );
};

export default QueryDetailsPage;
