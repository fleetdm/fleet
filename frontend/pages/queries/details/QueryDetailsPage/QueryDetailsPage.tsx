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
import { IQueryReport } from "interfaces/query_report";

import queryAPI from "services/entities/queries";
import queryReportAPI, { ISortOption } from "services/entities/query_report";

import Spinner from "components/Spinner/Spinner";
import Button from "components/buttons/Button";
import BackLink from "components/BackLink";
import MainContent from "components/MainContent";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import QueryAutomationsStatusIndicator from "pages/queries/ManageQueriesPage/components/QueryAutomationsStatusIndicator/QueryAutomationsStatusIndicator";
import DataError from "components/DataError/DataError";
import LogDestinationIndicator from "components/LogDestinationIndicator/LogDestinationIndicator";
import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";
import QueryReport from "../components/QueryReport/QueryReport";
import NoResults from "../components/NoResults/NoResults";

import {
  DEFAULT_SORT_HEADER,
  DEFAULT_SORT_DIRECTION,
  QUERY_REPORT_RESULTS_LIMIT,
} from "./QueryDetailsPageConfig";

interface IQueryDetailsPageProps {
  router: InjectedRouter; // v3
  params: Params;
  location: {
    pathname: string;
    query: { team_id?: string; order_key?: string; order_direction?: string };
    search: string;
  };
}

const baseClass = "query-details-page";

const QueryDetailsPage = ({
  router,
  params: { id: paramsQueryId },
  location,
}: IQueryDetailsPageProps): JSX.Element => {
  const queryId = parseInt(paramsQueryId, 10);
  const queryParams = location.query;

  // Functions to avoid race conditions
  const serverSortBy: ISortOption[] = (() => {
    return [
      {
        key: queryParams?.order_key ?? DEFAULT_SORT_HEADER,
        direction: queryParams?.order_direction ?? DEFAULT_SORT_DIRECTION,
      },
    ];
  })();

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
    setLastEditedQueryDiscardData,
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
    () => queryAPI.load(queryId),
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
        setLastEditedQueryDiscardData(returnedQuery.discard_data);
      },
      onError: (error) => handlePageError(error),
    }
  );

  const {
    isLoading: isQueryReportLoading,
    data: queryReport,
    error: queryReportError,
  } = useQuery<IQueryReport, Error, IQueryReport>(
    [],
    () =>
      queryReportAPI.load({
        sortBy: serverSortBy,
        id: queryId,
      }),
    {
      enabled: !!queryId,
      refetchOnWindowFocus: false,
      onError: (error) => handlePageError(error),
    }
  );

  const isLoading = isStoredQueryLoading || isQueryReportLoading;
  const isApiError = storedQueryError || queryReportError;
  const isClipped =
    (queryReport?.results?.length ?? 0) >= QUERY_REPORT_RESULTS_LIMIT;

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
        <div className={`${baseClass}__header-details`}>
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
                {canEditQuery && (
                  <Button
                    onClick={() => {
                      queryId && router.push(PATHS.EDIT_QUERY(queryId));
                    }}
                    className={`${baseClass}__manage-automations button`}
                    variant="brand"
                  >
                    Edit query
                  </Button>
                )}
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
                        queryId && router.push(PATHS.LIVE_QUERY(queryId));
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
                <TooltipWrapper
                  tipContent={`Query automations let you send data to your log destination on a schedule. When automations are <b>on</b>, data is sent according to a query’s frequency.`}
                >
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
        </div>
      </>
    );
  };

  const renderClippedBanner = () => (
    <InfoBanner
      color="yellow"
      cta={
        <CustomLink
          url="https://www.fleetdm.com/support"
          text="Get help"
          newTab
        />
      }
    >
      <div>
        <b>Report clipped.</b> A sample of this query&apos;s results is included
        below. You can still use query automations to complete this report in
        your log destination.
      </div>
    </InfoBanner>
  );

  const renderReport = () => {
    const disabledCachingGlobally = true; // TODO: Update accordingly to config?.server_settings.query_reports_disabled
    const discardDataEnabled = true; // TODO: Update accordingly to storedQuery?.discard_data
    const loggingSnapshot = storedQuery?.logging === "snapshot";
    const disabledCaching =
      disabledCachingGlobally || discardDataEnabled || !loggingSnapshot;
    const emptyCache = (queryReport?.results?.length ?? 0) === 0; // TODO: Update with API response

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
        />
      );
    }
    return <QueryReport queryReport={queryReport} isClipped={isClipped} />;
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        {renderHeader()}
        {isClipped && renderClippedBanner()}
        {renderReport()}
      </div>
    </MainContent>
  );
};

export default QueryDetailsPage;
