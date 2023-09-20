import React, { useContext, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import useTeamIdParam from "hooks/useTeamIdParam";

import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import queryAPI from "services/entities/queries";

import Spinner from "components/Spinner/Spinner";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import QueryAutomationsStatusIndicator from "pages/queries/ManageQueriesPage/components/QueryAutomationsStatusIndicator/QueryAutomationsStatusIndicator";
import EmptyTable from "components/EmptyTable/EmptyTable";
import LogDestinationIndicator from "components/LogDestinationIndicator/LogDestinationIndicator";
import CachedDetails from "../components/CachedDetails/CachedDetails";
import NoResults from "../components/NoResults/NoResults";
import DataError from "components/DataError/DataError";

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
    config,
    filteredQueriesPath,
  } = useContext(AppContext);
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryObserverCanRun,
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

  // Title that shows up on browser tabs (e.g., Query details | Discover TLS certificates | Fleet for osquery)
  document.title = `Query details | ${lastEditedQueryName} | Fleet for osquery`;

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

  const isLoading = isStoredQueryLoading; // TODO: Add || isCachedResultsLoading for new API response
  const isApiError = storedQueryError || true; // TODO: Add || isCachedResultsError for new API response

  const renderHeader = () => {
    const canEditQuery =
      isGlobalAdmin || isGlobalMaintainer || isAnyTeamMaintainerOrTeamAdmin;

    // Function instead of constant eliminates race condition with filteredQueriesPath
    const backToQueriesPath = () => {
      return filteredQueriesPath || PATHS.MANAGE_QUERIES;
    };

    return (
      <>
        <div className={`${baseClass}__header-links`}>
          <BackLink text="Back to queries" path={backToQueriesPath()} />
        </div>
        {!isLoading && !isApiError && (
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
                    onClick={() => {
                      queryId && router.push(PATHS.RUN_QUERY(queryId));
                    }}
                  >
                    Live query
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
        {!isLoading && !isApiError && (
          <div className={`${baseClass}__settings`}>
            <div className={`${baseClass}__automations`}>
              <TooltipWrapper tipContent="Query automations let you send data to your log destination on a schedule. When automations are on, data is sent according to a query’s frequency.">
                Automations:
              </TooltipWrapper>
              <QueryAutomationsStatusIndicator
                automationsEnabled={storedQuery?.automations_enabled || false}
                interval={storedQuery?.interval || 0}
              />
            </div>
            <div className={`${baseClass}__log-destination`}>
              <strong>Log destination:</strong>{" "}
              <LogDestinationIndicator
                logDestination={config?.logging.result.plugin || ""}
              />
            </div>
          </div>
        )}
      </>
    );
  };

  const renderReport = () => {
    const disabledCachingGlobally = true; // TODO: Update accordingly to config?.server_settings.query_reports_disabled
    const discardDataEnabled = true; // TODO: Update accordingly to storedQuery?.discard_data
    const loggingSnapshot = storedQuery?.logging === "snapshot";
    const disabledCaching =
      disabledCachingGlobally || discardDataEnabled || !loggingSnapshot;
    const emptyCache = true; // TODO: Update with API response
    const errorsOnly = true; // TODO: Update with API response

    // Loading state
    if (isLoading) {
      return <Spinner />;
    }

    // Error state
    if (isApiError) {
      return <DataError />;
    }

    // Empty state with varying messages explaining why there's no results
    if (emptyCache) {
      return (
        <NoResults
          queryInterval={storedQuery?.interval}
          queryUpdatedAt={storedQuery?.updated_at}
          disabledCaching={disabledCaching}
          disabledCachingGlobally={disabledCachingGlobally}
          discardDataEnabled={discardDataEnabled}
          loggingSnapshot={loggingSnapshot}
          errorsOnly={errorsOnly}
        />
      );
    }
    return <CachedDetails />; // TODO: Everything related to new APIs including surfacing errorsOnly
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
