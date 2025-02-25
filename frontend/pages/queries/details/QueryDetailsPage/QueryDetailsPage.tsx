import React, { useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";
import ReactTooltip from "react-tooltip";
import { COLORS } from "styles/var/colors";

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
import {
  isGlobalObserver,
  isTeamObserver,
} from "utilities/permissions/permissions";
import { DOCUMENT_TITLE_SUFFIX, SUPPORT_LINK } from "utilities/constants";
import { buildQueryStringFromParams } from "utilities/url";
import useTeamIdParam from "hooks/useTeamIdParam";

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
import ShowQueryModal from "components/modals/ShowQueryModal";
import QueryReport from "../components/QueryReport/QueryReport";
import NoResults from "../components/NoResults/NoResults";

import {
  DEFAULT_SORT_HEADER,
  DEFAULT_SORT_DIRECTION,
} from "./QueryDetailsPageConfig";

interface IQueryDetailsPageProps {
  router: InjectedRouter; // v3
  params: Params;
  location: {
    pathname: string;
    query: {
      team_id?: string;
      order_key?: string;
      order_direction?: string;
      host_id?: string;
    };
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
  if (isNaN(queryId)) {
    router.push(PATHS.MANAGE_QUERIES);
  }
  const queryParams = location.query;

  // Present when observer is redirected from host details > query
  // since observer does not have access to edit page
  const hostId = queryParams?.host_id
    ? parseInt(queryParams.host_id, 10)
    : undefined;

  const { currentTeamId } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

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
    currentUser,
    isGlobalAdmin,
    isGlobalMaintainer,
    isTeamMaintainerOrTeamAdmin,
    isObserverPlus,
    config,
    filteredQueriesPath,
    availableTeams,
    setCurrentTeam,
    isOnGlobalTeam,
  } = useContext(AppContext);
  const {
    lastEditedQueryName,
    lastEditedQueryDescription,
    lastEditedQueryBody,
    lastEditedQueryObserverCanRun,
    lastEditedQueryDiscardData,
    lastEditedQueryLoggingType,
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

  const [showQueryModal, setShowQueryModal] = useState(false);
  const [disabledCachingGlobally, setDisabledCachingGlobally] = useState(true);

  useEffect(() => {
    if (config) {
      setDisabledCachingGlobally(config.server_settings.query_reports_disabled);
    }
  }, [config]);

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

  /** Pesky bug affecting team level users:
   - Navigating to queries/:id immediately defaults the user to the first team they're on
  with the most permissions, in the URL bar because of useTeamIdParam
  even if the queries/:id entity has a team attached to it
  Hacky fix:
   - Push entity's team id to url for team level users
  */
  if (
    !isOnGlobalTeam &&
    !isStoredQueryLoading &&
    storedQuery?.team_id &&
    !(storedQuery?.team_id?.toString() === location.query.team_id)
  ) {
    router.push(
      `${location.pathname}?team_id=${storedQuery?.team_id?.toString()}`
    );
  }

  const {
    isLoading: isQueryReportLoading,
    data: queryReport,
    error: queryReportError,
  } = useQuery<IQueryReport, Error, IQueryReport>(
    [],
    () =>
      queryReportAPI.load({
        teamId: currentTeamId,
        sortBy: serverSortBy,
        id: queryId,
      }),
    {
      enabled: !!queryId,
      refetchOnWindowFocus: false,
      onError: (error) => handlePageError(error),
    }
  );

  // Used to set host's team in AppContext for RBAC action buttons
  useEffect(() => {
    if (storedQuery?.team_id) {
      const querysTeam = availableTeams?.find(
        (team) => team.id === storedQuery.team_id
      );
      setCurrentTeam(querysTeam);
    }
  }, [storedQuery]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Discover TLS certificates | Queries | Fleet
    if (storedQuery?.name) {
      document.title = `${storedQuery.name} | Queries | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Queries | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedQuery?.name]);

  const onShowQueryModal = () => {
    setShowQueryModal(!showQueryModal);
  };

  const isLoading = isStoredQueryLoading || isQueryReportLoading;
  const isApiError = storedQueryError || queryReportError;
  const isClipped = queryReport?.report_clipped;
  const isLiveQueryDisabled = config?.server_settings.live_query_disabled;

  const renderHeader = () => {
    // Team admins/maintainers can only edit queries assigned to a team
    const canEditQuery =
      isGlobalAdmin ||
      isGlobalMaintainer ||
      (isTeamMaintainerOrTeamAdmin && storedQuery?.team_id);

    const canLiveQuery =
      lastEditedQueryObserverCanRun ||
      isObserverPlus ||
      isGlobalAdmin ||
      isGlobalMaintainer ||
      isTeamMaintainerOrTeamAdmin;

    // Function instead of constant eliminates race condition with filteredQueriesPath
    const backToQueriesPath = () => {
      return (
        filteredQueriesPath ||
        `${PATHS.MANAGE_QUERIES}?${buildQueryStringFromParams({
          team_id: currentTeamId,
        })}`
      );
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
                <Button
                  className={`${baseClass}__show-query-btn`}
                  onClick={onShowQueryModal}
                  variant="text-icon"
                >
                  Show query
                </Button>
                {canEditQuery && (
                  <Button
                    onClick={() => {
                      queryId &&
                        router.push(PATHS.EDIT_QUERY(queryId, currentTeamId));
                    }}
                    className={`${baseClass}__manage-automations button`}
                    variant="brand"
                  >
                    Edit query
                  </Button>
                )}
                {canLiveQuery && (
                  <div
                    className={`button-wrap ${baseClass}__button-wrap--new-query`}
                  >
                    <div
                      data-tip
                      data-for="live-query-button"
                      // Tooltip shows when live queries are globally disabled
                      data-tip-disable={!isLiveQueryDisabled}
                    >
                      <Button
                        className={`${baseClass}__run`}
                        variant="blue-green"
                        onClick={() => {
                          queryId &&
                            router.push(
                              PATHS.LIVE_QUERY(queryId, currentTeamId, hostId)
                            );
                        }}
                        disabled={isLiveQueryDisabled}
                      >
                        Live query
                      </Button>
                    </div>
                    <ReactTooltip
                      className="live-query-button-tooltip"
                      place="top"
                      effect="solid"
                      backgroundColor={COLORS["tooltip-bg"]}
                      id="live-query-button"
                      data-html
                    >
                      Live queries are disabled in organization settings
                    </ReactTooltip>
                  </div>
                )}
              </div>
            </div>
          )}
          {!isLoading && !isApiError && (
            <div className={`${baseClass}__settings`}>
              <div className={`${baseClass}__automations`}>
                <TooltipWrapper
                  tipContent={
                    <>
                      Query automations let you send data to your log <br />
                      destination on a schedule. When automations are <b>
                        on
                      </b>, <br />
                      data is sent according to a query&apos;s frequency.
                    </>
                  }
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
      cta={<CustomLink url={SUPPORT_LINK} text="Get help" newTab />}
    >
      <div>
        <b>Report clipped.</b> A sample of this query&apos;s results is included
        below.
        {
          // Exclude below message for global and team observers/observer+s
          !(
            (currentUser && isGlobalObserver(currentUser)) ||
            isTeamObserver(currentUser, currentTeamId ?? null)
          ) &&
            " You can still use query automations to complete this report in your log destination."
        }
      </div>
    </InfoBanner>
  );

  const renderReport = () => {
    const loggingSnapshot = lastEditedQueryLoggingType === "snapshot";
    const disabledCaching =
      disabledCachingGlobally || lastEditedQueryDiscardData || !loggingSnapshot;
    const emptyCache = (queryReport?.results?.length ?? 0) === 0;

    // Loading state
    if (isLoading) {
      return <Spinner />;
    }

    // Error state
    if (isApiError) {
      return <DataError />;
    }

    // Empty state with varying messages explaining why there's no results
    if (emptyCache || lastEditedQueryDiscardData) {
      return (
        <NoResults
          queryInterval={storedQuery?.interval}
          queryUpdatedAt={storedQuery?.updated_at}
          disabledCaching={disabledCaching}
          disabledCachingGlobally={disabledCachingGlobally}
          discardDataEnabled={lastEditedQueryDiscardData}
          loggingSnapshot={loggingSnapshot}
        />
      );
    }
    return <QueryReport {...{ queryReport, isClipped }} />;
  };

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        {renderHeader()}
        {isClipped && renderClippedBanner()}
        {renderReport()}
        {showQueryModal && (
          <ShowQueryModal
            query={lastEditedQueryBody}
            onCancel={onShowQueryModal}
          />
        )}
      </div>
    </MainContent>
  );
};

export default QueryDetailsPage;
