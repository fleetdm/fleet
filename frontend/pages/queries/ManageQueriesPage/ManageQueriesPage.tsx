import React, {
  useContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { pick } from "lodash";

import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import { performanceIndicator } from "utilities/helpers";
import { IOsqueryPlatform } from "interfaces/platform";
// TODO: remove old interfaces
import { IQuery, IFleetQueriesResponse } from "interfaces/query";
import {
  IListQueriesResponse,
  ISchedulableQuery,
} from "interfaces/schedulable_query";
import fleetQueriesAPI from "services/entities/queries";
import PATHS from "router/paths";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import checkPlatformCompatibility from "utilities/sql_tools";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import TeamsDropdown from "components/TeamsDropdown";
import useTeamIdParam from "hooks/useTeamIdParam";
import QueriesTable from "./components/QueriesTable";
import DeleteQueryModal from "./components/DeleteQueryModal";

const baseClass = "manage-queries-page";
interface IManageQueriesPageProps {
  router: InjectedRouter; // v3
  location: {
    pathname: string;
    query: {
      platform?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
      team_id?: string;
    };
    search: string;
  };
}

interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}

const getPlatforms = (queryString: string): Array<IOsqueryPlatform | "---"> => {
  const { platforms } = checkPlatformCompatibility(queryString);

  return platforms || [DEFAULT_EMPTY_CELL_VALUE];
};

const enhanceQuery = (q: ISchedulableQuery) => {
  return {
    ...q,
    performance:
      q.interval || q.automations_enabled
        ? performanceIndicator(
            pick(q.stats, [
              "user_time_p50",
              "system_time_p50",
              "total_executions",
            ])
          )
        : "Undetermined",
    platforms: getPlatforms(q.query),
  };
};

const ManageQueriesPage = ({
  router,
  location,
}: IManageQueriesPageProps): JSX.Element => {
  const queryParams = location.query;

  const {
    isOnlyObserver,
    isObserverPlus,
    isAnyTeamObserverPlus,
    isOnGlobalTeam,
    setFilteredQueriesPath,
    filteredQueriesPath,
    isPremiumTier,
    isSandboxMode,
  } = useContext(AppContext);

  const { setResetSelectedRows } = useContext(TableContext);
  const { renderFlash } = useContext(NotificationContext);

  const {
    userTeams,
    currentTeamId,
    handleTeamChange,
    teamIdForApi,
    isRouteOk,
  } = useTeamIdParam({
    location,
    router,
    includeAllTeams: true,
    includeNoTeam: false,
  });

  const [queriesList, setQueriesList] = useState<IQueryTableData[] | null>(
    null
  );
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showDeleteQueryModal, setShowDeleteQueryModal] = useState(false);
  const [isUpdatingQueries, setIsUpdatingQueries] = useState(false);

  // TODO: reenable API call once backend work completed

  // const {
  //   data: fleetQueries,
  //   error: fleetQueriesError,
  //   isFetching: isFetchingFleetQueries,
  //   refetch: refetchFleetQueries,
  // } = useQuery<IListQueriesResponse, Error, ISchedulableQuery[]>(
  //   [{ scope: "queries", teamId: teamIdForApi }],
  //   () => fleetQueriesAPI.loadAll(teamIdForApi),
  //   {
  //     refetchOnWindowFocus: false,
  //     enabled: isRouteOk,
  //     select: (data) => data.queries,
  //   }
  // );

  const [
    fleetQueries,
    fleetQueriesError,
    isFetchingFleetQueries,
    refetchFleetQueries,
  ] = useMemo(() => {
    return [
      [
        {
          created_at: "2023-06-08T15:31:35Z",
          updated_at: "2023-06-08T15:31:35Z",
          id: 2,
          name: "test",
          description: "",
          query: "SELECT * FROM osquery_info;",
          team_id: 43,
          platform: "darwin",
          min_osquery_version: "",
          automations_enabled: false,
          logging: "snapshot",
          saved: true,
          // interval: 300,
          interval: 0,
          observer_can_run: false,
          author_id: 1,
          author_name: "Jacob",
          author_email: "jacob@fleetdm.com",
          packs: [],
          stats: {
            system_time_p50: 1,
            // system_time_p95: null,
            user_time_p50: 1,
            // user_time_p95: null,
            total_executions: 1,
          },
        },
      ] as ISchedulableQuery[],
      undefined,
      false,
      () => {
        console.log("got the new queries");
      },
    ];
  }, []);

  const enhancedQueriesList = useMemo(() => {
    const enhancedQueries = fleetQueries?.map((q: ISchedulableQuery) => {
      const query = enhanceQuery(q);
      return query;
    });

    return enhancedQueries || [];
  }, [fleetQueries]);

  useEffect(() => {
    if (!isFetchingFleetQueries && enhancedQueriesList) {
      setQueriesList(enhancedQueriesList);
    }
  }, [enhancedQueriesList, isFetchingFleetQueries]);

  useEffect(() => {
    const path = location.pathname + location.search;
    if (filteredQueriesPath !== path) {
      setFilteredQueriesPath(path);
    }
  }, [location, filteredQueriesPath, setFilteredQueriesPath]);

  const onCreateQueryClick = () => router.push(PATHS.NEW_QUERY);

  const toggleDeleteQueryModal = useCallback(() => {
    setShowDeleteQueryModal(!showDeleteQueryModal);
  }, [showDeleteQueryModal, setShowDeleteQueryModal]);

  const onDeleteQueryClick = (selectedTableQueryIds: number[]) => {
    toggleDeleteQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onDeleteQuerySubmit = useCallback(async () => {
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";

    setIsUpdatingQueries(true);

    const deleteQueries = selectedQueryIds.map((id) =>
      fleetQueriesAPI.destroy(id)
    );

    try {
      await Promise.all(deleteQueries).then(() => {
        renderFlash("success", `Successfully deleted ${queryOrQueries}.`);
        setResetSelectedRows(true);
        refetchFleetQueries();
      });
      renderFlash("success", `Successfully deleted ${queryOrQueries}.`);
    } catch (errorResponse) {
      renderFlash(
        "error",
        `There was an error deleting your ${queryOrQueries}. Please try again later.`
      );
    } finally {
      toggleDeleteQueryModal();
      setIsUpdatingQueries(false);
    }
  }, [refetchFleetQueries, selectedQueryIds, toggleDeleteQueryModal]);

  const isTableDataLoading = isFetchingFleetQueries || queriesList === null;

  const renderHeader = () => {
    if (isPremiumTier) {
      if (userTeams) {
        if (userTeams.length > 1 || isOnGlobalTeam) {
          return (
            <TeamsDropdown
              currentUserTeams={userTeams}
              selectedTeamId={currentTeamId}
              onChange={handleTeamChange}
              isSandboxMode={isSandboxMode}
            />
          );
        } else if (!isOnGlobalTeam && userTeams.length === 1) {
          return <h1>{userTeams[0].name}</h1>;
        }
      }
    }
    return <h1>Queries</h1>;
  };
  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <div className={`${baseClass}__title`}>{renderHeader()}</div>
            </div>
          </div>
          {(!isOnlyObserver || isObserverPlus || isAnyTeamObserverPlus) &&
            !!fleetQueries?.length && (
              <div className={`${baseClass}__action-button-container`}>
                <Button
                  variant="brand"
                  className={`${baseClass}__create-button`}
                  onClick={onCreateQueryClick}
                >
                  Add query
                </Button>
              </div>
            )}
        </div>
        <div className={`${baseClass}__description`}>
          <p>
            Manage and schedule queries to ask questions and collect telemetry
            for all hosts{currentTeamId !== -1 && " assigned to this team"}.
          </p>
        </div>
        <div>
          {isTableDataLoading && !fleetQueriesError && <Spinner />}
          {!isTableDataLoading && fleetQueriesError ? (
            <TableDataError />
          ) : (
            <QueriesTable
              queriesList={queriesList}
              isLoading={isTableDataLoading}
              onCreateQueryClick={onCreateQueryClick}
              onDeleteQueryClick={onDeleteQueryClick}
              isOnlyObserver={isOnlyObserver}
              isObserverPlus={isObserverPlus}
              isAnyTeamObserverPlus={isAnyTeamObserverPlus || false}
              router={router}
              queryParams={queryParams}
            />
          )}
        </div>
        {showDeleteQueryModal && (
          <DeleteQueryModal
            isUpdatingQueries={isUpdatingQueries}
            onCancel={toggleDeleteQueryModal}
            onSubmit={onDeleteQuerySubmit}
          />
        )}
      </div>
    </MainContent>
  );
};

export default ManageQueriesPage;
