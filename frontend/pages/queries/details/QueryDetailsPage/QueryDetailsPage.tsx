import React, { useContext, useState, useEffect } from "react";
import { useQuery } from "react-query";
import { InjectedRouter, Params } from "react-router/lib/Router";
import { useErrorHandler } from "react-error-boundary";

import PATHS from "router/paths";
import { AppContext } from "context/app";

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
import { getPathWithQueryParams } from "utilities/url";
import useTeamIdParam from "hooks/useTeamIdParam";

import Icon from "components/Icon";
import Spinner from "components/Spinner/Spinner";
import Button from "components/buttons/Button";
import BackButton from "components/BackButton";
import MainContent from "components/MainContent";
import TooltipWrapper from "components/TooltipWrapper/TooltipWrapper";
import TooltipTruncatedText from "components/TooltipTruncatedText";
import QueryAutomationsStatusIndicator from "pages/queries/ManageQueriesPage/components/QueryAutomationsStatusIndicator/QueryAutomationsStatusIndicator";
import DataError from "components/DataError/DataError";
import LogDestinationIndicator from "components/LogDestinationIndicator/LogDestinationIndicator";
import CustomLink from "components/CustomLink";
import InfoBanner from "components/InfoBanner";
import ShowQueryModal from "components/modals/ShowQueryModal";
import PageDescription from "components/PageDescription";
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
      fleet_id?: string;
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
    router.push(PATHS.MANAGE_REPORTS);
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
    isGlobalTechnician,
    isTeamTechnician,
  } = useContext(AppContext);
  const [showQueryModal, setShowQueryModal] = useState(false);
  const [disabledCachingGlobally, setDisabledCachingGlobally] = useState(true);

  useEffect(() => {
    if (config) {
      setDisabledCachingGlobally(config.server_settings.query_reports_disabled);
    }
  }, [config]);

  const {
    isLoading: isStoredQueryLoading,
    data: storedQuery,
    error: storedQueryError,
  } = useQuery<IGetQueryResponse, Error, ISchedulableQuery>(
    ["query", queryId],
    () => queryAPI.load(queryId),
    {
      enabled: !!queryId,
      select: (data) => data.query,
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
    !(storedQuery?.team_id?.toString() === location.query.fleet_id)
  ) {
    router.push(
      getPathWithQueryParams(location.pathname, {
        fleet_id: storedQuery?.team_id?.toString(),
      })
    );
  }

  const discardData = !!storedQuery?.discard_data;
  const loggingSnapshot = storedQuery?.logging === "snapshot";
  const reportCachingDisabled =
    disabledCachingGlobally || discardData || !loggingSnapshot;

  const {
    isLoading: isQueryReportLoading,
    data: queryReport,
    error: queryReportError,
  } = useQuery<IQueryReport, Error, IQueryReport>(
    // Key must include every queryFn parameter; an empty key bled one report's
    // cached rows into another on revisit (and suppressed refetch on sort).
    ["queryReport", queryId, currentTeamId, serverSortBy],
    () =>
      queryReportAPI.load({
        teamId: currentTeamId,
        sortBy: serverSortBy,
        id: queryId,
      }),
    {
      enabled: !!queryId,
      refetchOnWindowFocus: !reportCachingDisabled,
      refetchInterval: (data) =>
        !reportCachingDisabled && data?.results?.length === 0 ? 5000 : false,
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
  }, [storedQuery, availableTeams, setCurrentTeam]);

  // Updates title that shows up on browser tabs
  useEffect(() => {
    // e.g., Discover TLS certificates | Queries | Fleet
    if (storedQuery?.name) {
      document.title = `${storedQuery.name} | Reports | ${DOCUMENT_TITLE_SUFFIX}`;
    } else {
      document.title = `Reports | ${DOCUMENT_TITLE_SUFFIX}`;
    }
  }, [location.pathname, storedQuery?.name]);

  const onShowQueryModal = () => {
    setShowQueryModal(!showQueryModal);
  };

  const isLoading = isStoredQueryLoading || isQueryReportLoading;
  const isApiError = storedQueryError || queryReportError;
  const isClipped = queryReport?.report_clipped;
  const isLiveQueryDisabled = config?.server_settings.live_query_disabled;

  const canLiveQuery =
    storedQuery?.observer_can_run ||
    isObserverPlus ||
    isGlobalAdmin ||
    isGlobalMaintainer ||
    isTeamMaintainerOrTeamAdmin ||
    isGlobalTechnician ||
    isTeamTechnician;

  const canRunLiveReport = canLiveQuery && !isLiveQueryDisabled;

  // Team admins/maintainers can only edit queries assigned to a team
  const canEditQuery =
    isGlobalAdmin ||
    isGlobalMaintainer ||
    (isTeamMaintainerOrTeamAdmin && storedQuery?.team_id);

  const renderHeader = () => {
    // Function instead of constant eliminates race condition with filteredQueriesPath
    const backPath = () => {
      if (hostId)
        return getPathWithQueryParams(
          PATHS.HOST_DETAILS(hostId, currentTeamId)
        );

      if (filteredQueriesPath) return filteredQueriesPath;

      return getPathWithQueryParams(PATHS.MANAGE_REPORTS, {
        fleet_id: currentTeamId,
      });
    };

    return (
      <>
        <div className={`${baseClass}__header-links`}>
          <BackButton
            text={hostId ? "Back to host details" : "Back to reports"}
            path={backPath()}
          />
        </div>
        {!isLoading && !isApiError && (
          <>
            <div className={`${baseClass}__title-bar`}>
              <div className={`${baseClass}__name-description`}>
                <h1 className={`${baseClass}__query-name`}>
                  <TooltipTruncatedText
                    value={storedQuery?.name}
                    fixedPositionStrategy
                  />
                </h1>
              </div>
              <div className={`${baseClass}__action-button-container`}>
                <Button
                  className={`${baseClass}__show-query-btn`}
                  onClick={onShowQueryModal}
                  variant="secondary"
                >
                  Show query
                </Button>
                {canLiveQuery && (
                  <div
                    className={`button-wrap ${baseClass}__button-wrap--new-query`}
                  >
                    <TooltipWrapper
                      tipContent="Live reports are disabled in organization settings."
                      position="top"
                      disableTooltip={!isLiveQueryDisabled}
                      underline={false}
                      showArrow
                    >
                      <div>
                        <Button
                          className={`${baseClass}__run`}
                          variant="secondary"
                          onClick={() => {
                            queryId &&
                              router.push(
                                getPathWithQueryParams(
                                  PATHS.LIVE_REPORT(queryId),
                                  {
                                    host_id: hostId,
                                    fleet_id: currentTeamId,
                                  }
                                )
                              );
                          }}
                          disabled={isLiveQueryDisabled}
                        >
                          Live report <Icon name="run" />
                        </Button>
                      </div>
                    </TooltipWrapper>
                  </div>
                )}
                {canEditQuery && (
                  <Button
                    onClick={() => {
                      queryId &&
                        router.push(
                          getPathWithQueryParams(PATHS.EDIT_REPORT(queryId), {
                            fleet_id: currentTeamId,
                            host_id: hostId,
                          })
                        );
                    }}
                    className={`${baseClass}__manage-automations button`}
                  >
                    Edit report
                  </Button>
                )}
              </div>
            </div>
            <PageDescription
              className={`${baseClass}__query-description`}
              content={storedQuery?.description}
            />
            <div className={`${baseClass}__settings`}>
              <div className={`${baseClass}__automations`}>
                <TooltipWrapper
                  tipContent={
                    <>
                      Report automations let you send data to your log <br />
                      destination on a schedule. When automations are <b>
                        on
                      </b>, <br />
                      data is sent according to a report&apos;s interval.
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
                  filesystemDestination={
                    config?.logging.result.config?.result_log_file
                  }
                  webhookDestination={config?.logging.result.config?.result_url}
                />
              </div>
            </div>
          </>
        )}
      </>
    );
  };

  const renderClippedBanner = () => (
    <InfoBanner
      color="yellow"
      cta={<CustomLink url={SUPPORT_LINK} text="Get help" newTab />}
    >
      <div>
        <b>Report clipped.</b> A sample of this report&apos;s results is
        included below.
        {
          // Exclude below message for global and team observers/observer+s
          !(
            (currentUser && isGlobalObserver(currentUser)) ||
            isTeamObserver(currentUser, currentTeamId ?? null)
          ) &&
            " You can still use automations to complete this report in your log destination."
        }
      </div>
    </InfoBanner>
  );

  const renderReport = () => {
    const emptyCache = (queryReport?.results?.length ?? 0) === 0;

    if (isLoading) {
      return <Spinner />;
    }

    if (isApiError) {
      return <DataError />;
    }

    // Empty state with varying messages explaining why there's no results
    if (emptyCache || discardData) {
      return (
        <NoResults
          queryId={queryId}
          queryInterval={storedQuery?.interval}
          queryUpdatedAt={storedQuery?.updated_at}
          disabledCaching={reportCachingDisabled}
          disabledCachingGlobally={disabledCachingGlobally}
          discardDataEnabled={discardData}
          loggingSnapshot={loggingSnapshot}
          canLiveQuery={canRunLiveReport}
          canEditQuery={!!canEditQuery}
        />
      );
    }
    return (
      <QueryReport
        queryReport={queryReport}
        queryId={queryId}
        queryName={storedQuery?.name}
        isClipped={isClipped}
        canLiveQuery={canRunLiveReport}
      />
    );
  };

  return (
    <MainContent className={baseClass}>
      <>
        {renderHeader()}
        {isClipped && renderClippedBanner()}
        {renderReport()}
        {showQueryModal && (
          <ShowQueryModal
            query={storedQuery?.query}
            onCancel={onShowQueryModal}
          />
        )}
      </>
    </MainContent>
  );
};

export default QueryDetailsPage;
