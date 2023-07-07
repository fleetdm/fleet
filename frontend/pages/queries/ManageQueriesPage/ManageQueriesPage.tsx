import React, {
  useContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from "react";
import { RouteProps, InjectedRouter } from "react-router";
import { useQuery } from "react-query";
import { pick } from "lodash";

import { AppContext } from "context/app";
import { TableContext } from "context/table";
import { NotificationContext } from "context/notification";
import { performanceIndicator } from "utilities/helpers";
import { IOsqueryPlatform } from "interfaces/platform";
import { IQuery, IFleetQueriesResponse } from "interfaces/query";
import fleetQueriesAPI from "services/entities/queries";
import PATHS from "router/paths";
import { DEFAULT_EMPTY_CELL_VALUE } from "utilities/constants";
import checkPlatformCompatibility from "utilities/sql_tools";
import Button from "components/buttons/Button";
import Spinner from "components/Spinner";
import TableDataError from "components/DataError";
import MainContent from "components/MainContent";
import QueriesTable from "./components/QueriesTable";
import DeleteQueryModal from "./components/DeleteQueryModal";

const baseClass = "manage-queries-page";
interface IManageQueriesPageProps {
  route: RouteProps;
  router: InjectedRouter; // v3
  location: {
    pathname?: string;
    query: {
      platform?: string;
      page?: string;
      query?: string;
      order_key?: string;
      order_direction?: "asc" | "desc";
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

const enhanceQuery = (q: IQuery) => {
  return {
    ...q,
    performance: performanceIndicator(
      pick(q.stats, ["user_time_p50", "system_time_p50", "total_executions"])
    ),
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
    setFilteredQueriesPath,
    filteredQueriesPath,
  } = useContext(AppContext);

  const { setResetSelectedRows } = useContext(TableContext);
  const { renderFlash } = useContext(NotificationContext);

  const [queriesList, setQueriesList] = useState<IQueryTableData[] | null>(
    null
  );
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showDeleteQueryModal, setShowDeleteQueryModal] = useState(false);
  const [isUpdatingQueries, setIsUpdatingQueries] = useState(false);

  const {
    data: fleetQueries,
    error: fleetQueriesError,
    isFetching: isFetchingFleetQueries,
    refetch: refetchFleetQueries,
  } = useQuery<IFleetQueriesResponse, Error, IQuery[]>(
    "fleet queries by platform",
    () => fleetQueriesAPI.loadAll(),
    {
      refetchOnWindowFocus: false,
      select: (data: IFleetQueriesResponse) => data.queries,
    }
  );

  const enhancedQueriesList = useMemo(() => {
    const enhancedQueries = fleetQueries?.map((q: IQuery) => {
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

  return (
    <MainContent className={baseClass}>
      <div className={`${baseClass}__wrapper`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Queries</span>
              </h1>
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
          <p>Manage queries to ask specific questions about your devices.</p>
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
