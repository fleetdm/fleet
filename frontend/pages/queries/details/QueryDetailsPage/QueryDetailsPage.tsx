import React, { useContext, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";
import differenceInSeconds from "date-fns/differenceInSeconds";
import formatDistance from "date-fns/formatDistance";
import add from "date-fns/add";

import PATHS from "router/paths";
import { AppContext } from "context/app";
import { QueryContext } from "context/query";
import useTeamIdParam from "hooks/useTeamIdParam";

import {
  IGetQueryResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";

import queryAPI from "services/entities/queries";

import Button from "components/buttons/Button";
import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import EmptyTable from "components/EmptyTable/EmptyTable";
import CachedDetails from "../components/CachedDetails/CachedDetails";

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

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Run live query | Discover TLS certificates | Fleet for osquery
    document.title = `Query details | ${lastEditedQueryName} | Fleet for osquery`;
  }, [location.pathname, lastEditedQueryName]);

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

  const errorsOnly = true; // TODO

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
      </>
    );
  };

  const renderReport = () => {
    const disabledSaving = true; // TODO: Update accordingly
    const disabledSavingGlobally = true; // TODO: Update accordingly
    const discardDataEnabled = true; // TODO: Update accordingly
    const loggingSnapshot = storedQuery?.logging === "snapshot";

    const secondsCheckbackTime = () => {
      const secondsSinceUpdate = storedQuery?.updated_at
        ? differenceInSeconds(new Date(), new Date(storedQuery?.updated_at))
        : 0;
      const secondsUpdateWaittime = lastEditedQueryFrequency + 60;
      return secondsUpdateWaittime - secondsSinceUpdate;
    };

    const collectingResults = secondsCheckbackTime() > 0;
    const noResults = secondsCheckbackTime() <= 0;

    const readableCheckbackTime = formatDistance(
      add(new Date(), { seconds: secondsCheckbackTime() }),
      new Date()
    );

    const collectingResultsInfo = () =>
      `Fleet is collecting query results. Check back in about ${readableCheckbackTime}.`;

    const noResultsInfo = () => {
      if (!storedQuery?.interval) {
        return (
          <>
            This query does not collect data on a schedule. Add a{" "}
            <strong>frequency</strong> or run this as a live query to see
            results.
          </>
        );
      }
      if (disabledSaving) {
        // TODO: Where's the tooltip underline?
        const tipContent = () => {
          if (disabledSavingGlobally) {
            return "The following setting prevents saving this query's results in Fleet:<ul><li>Query reports are globally disabled in organization settings.</li></ul>";
          }
          if (discardDataEnabled) {
            return "The following setting prevents saving this query's results in Fleet:<ul><li>This query has Discard data enabled.</li></ul>";
          }
          if (!loggingSnapshot) {
            return "The following setting prevents saving this query's results in Fleet:<ul><li>The logging setting for this query is not Snapshot.</li></ul>";
          }
          return "Unknown";
        };
        return (
          <>
            Results from this query are{" "}
            <TooltipWrapper tipContent={tipContent()}>
              not reported in Fleet
            </TooltipWrapper>
            .
          </>
        );
      }
      if (errorsOnly) {
        return (
          <>
            This query had trouble collecting data on some hosts. Check out the{" "}
            <strong>Errors</strong> tab to see why.
          </>
        );
      }
      return "This query has returned no data so far."; // TODO: Fix so it's correct
    };
    if (collectingResults) {
      return (
        <EmptyTable
          iconName="collecting-results"
          header={"Collecting results..."}
          info={collectingResultsInfo()}
        />
      );
    }
    if (noResults) {
      return (
        <EmptyTable
          iconName="empty-software"
          header={"Nothing to report yet"}
          info={noResultsInfo()}
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
