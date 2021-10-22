import React, { useContext, useCallback, useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { push } from "react-router-redux";
import { memoize, pick } from "lodash";

import { AppContext } from "context/app";
import { performanceIndicator } from "fleet/helpers";
import { IQuery } from "interfaces/query";
import fleetQueriesAPI from "services/entities/queries";
// @ts-ignore
import queryActions from "redux/nodes/entities/queries/actions"; // TODO: Delete this after redux dependencies have been removed.
// @ts-ignore
import { renderFlash } from "redux/nodes/notifications/actions";
import PATHS from "router/paths";
// @ts-ignore
import sqlTools from "utilities/sql_tools";
import Button from "components/buttons/Button";
// @ts-ignore
import Dropdown from "components/forms/fields/Dropdown";
import Spinner from "components/loaders/Spinner";
import TableDataError from "components/TableDataError";
import QueriesListWrapper from "./components/QueriesListWrapper";
import RemoveQueryModal from "./components/RemoveQueryModal";

const baseClass = "manage-queries-page";
interface IFleetQueriesResponse {
  queries: IQuery[];
}
interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}
interface IQueriesByPlatform extends Record<string, IQueryTableData[]> {
  all: IQueryTableData[];
  darwin: IQueryTableData[];
  linux: IQueryTableData[];
  windows: IQueryTableData[];
}

const PLATFORMS = ["darwin", "linux", "windows"];

const PLATFORM_FILTER_OPTIONS = [
  {
    disabled: false,
    label: "All platforms",
    value: "all",
    helpText: "All queries.",
  },
  {
    disabled: false,
    label: "Linux",
    value: "linux",
    helpText: "Queries that are compatible with Linux operating systems.",
  },
  {
    disabled: false,
    label: "macOS",
    value: "darwin",
    helpText: "Queries that are compatible with macOS operating systems.",
  },
  {
    disabled: false,
    label: "Windows",
    value: "windows",
    helpText: "Queries that are compatible with Windows operating systems.",
  },
];

const memoizedSqlTables = memoize(sqlTools.parseSqlTables);
const getPlatforms = (queryString: string): string[] =>
  sqlTools
    .listCompatiblePlatforms(memoizedSqlTables(queryString))
    .filter((p: string) => PLATFORMS.includes(p));

const enhanceQuery = (q: IQuery) => {
  return {
    ...q,
    performance: performanceIndicator(
      pick(q.stats, ["user_time_p50", "system_time_p50", "total_executions"])
    ),
    platforms: getPlatforms(q.query),
  };
};

const ManageQueriesPage = (): JSX.Element => {
  const dispatch = useDispatch();

  const { isOnlyObserver } = useContext(AppContext);

  const [filteredQueries, setFilteredQueries] = useState<
    IQueryTableData[] | null
  >(null);
  const [searchString, setSearchString] = useState<string>("");
  const [selectedPlatform, setSelectedPlatform] = useState<string>("all");
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showRemoveQueryModal, setShowRemoveQueryModal] = useState<boolean>(
    false
  );

  const {
    data: fleetQueriesByPlatform,
    error: fleetQueriesError,
    isLoading: isLoadingFleetQueries,
    refetch: refetchFleetQueries,
  } = useQuery<IFleetQueriesResponse, Error, IQueriesByPlatform>(
    "fleet queries",
    () => fleetQueriesAPI.loadAll(),
    {
      // refetchOnMount: false,
      // refetchOnReconnect: false,
      // refetchOnWindowFocus: false,
      select: (data: IFleetQueriesResponse) =>
        data.queries.reduce(
          (dictionary: IQueriesByPlatform, q) => {
            const queryEntry = enhanceQuery(q);
            dictionary.all.push(queryEntry);
            queryEntry.platforms.forEach((platform) =>
              dictionary[platform]?.push(queryEntry)
            );

            return dictionary;
          },
          { all: [], darwin: [], linux: [], windows: [] }
        ),
    }
  );

  useEffect(() => {
    if (!isLoadingFleetQueries && fleetQueriesByPlatform) {
      let queriesList = fleetQueriesByPlatform[selectedPlatform];
      if (searchString) {
        queriesList = queriesList.filter((q) =>
          q.name.toLowerCase().includes(searchString.toLowerCase())
        );
      }
      setFilteredQueries(queriesList);
    }
  }, [
    fleetQueriesByPlatform,
    isLoadingFleetQueries,
    searchString,
    selectedPlatform,
  ]);

  const onCreateQueryClick = () => dispatch(push(PATHS.NEW_QUERY));

  const toggleRemoveQueryModal = useCallback(() => {
    setShowRemoveQueryModal(!showRemoveQueryModal);
  }, [showRemoveQueryModal, setShowRemoveQueryModal]);

  const onRemoveQueryClick = (selectedTableQueryIds: any) => {
    toggleRemoveQueryModal();
    setSelectedQueryIds(selectedTableQueryIds);
  };

  const onRemoveQuerySubmit = useCallback(() => {
    const queryOrQueries = selectedQueryIds.length === 1 ? "query" : "queries";

    const promises = selectedQueryIds.map((id: number) => {
      return fleetQueriesAPI.destroy(id);
    });

    return Promise.all(promises)
      .then(() => {
        dispatch(
          renderFlash("success", `Successfully removed ${queryOrQueries}.`)
        );
        toggleRemoveQueryModal();
      })
      .catch((response) => {
        if (
          response?.errors?.filter((error: Record<string, string>) =>
            error.reason?.includes(
              "the operation violates a foreign key constraint"
            )
          ).length
        ) {
          dispatch(
            renderFlash(
              "error",
              `Could not delete query because this query is used as a policy. First remove the policy and then try deleting the query again.`
            )
          );
        } else {
          dispatch(
            renderFlash(
              "error",
              `Unable to remove ${queryOrQueries}. Please try again.`
            )
          );
        }
      })
      .finally(() => {
        refetchFleetQueries();
        // TODO: Delete this redux action after redux dependencies have been removed (e.g., schedules page
        // depends on redux state for queries).
        dispatch(queryActions.loadAll());
        toggleRemoveQueryModal();
      });
  }, [dispatch, selectedQueryIds]);

  const renderPlatformDropdown = () => {
    return fleetQueriesByPlatform?.all.length ? (
      <Dropdown
        value={selectedPlatform}
        className={`${baseClass}__platform_dropdown`}
        options={PLATFORM_FILTER_OPTIONS}
        searchable={false}
        onChange={setSelectedPlatform}
      />
    ) : (
      <></>
    );
  };

  const isTableDataLoading = isLoadingFleetQueries || filteredQueries === null;

  return (
    <div className={baseClass}>
      <div className={`${baseClass}__wrapper body-wrap`}>
        <div className={`${baseClass}__header-wrap`}>
          <div className={`${baseClass}__header`}>
            <div className={`${baseClass}__text`}>
              <h1 className={`${baseClass}__title`}>
                <span>Queries</span>
              </h1>
              <div className={`${baseClass}__description`}>
                <p>
                  Manage queries to ask specific questions about your devices.
                </p>
              </div>
            </div>
          </div>
          {!isOnlyObserver && !!fleetQueriesByPlatform?.all.length && (
            <div className={`${baseClass}__action-button-container`}>
              <Button
                variant="brand"
                className={`${baseClass}__create-button`}
                onClick={onCreateQueryClick}
              >
                Create new query
              </Button>
            </div>
          )}
        </div>
        <div>
          {isTableDataLoading && !fleetQueriesError && <Spinner />}
          {!isTableDataLoading && fleetQueriesError ? (
            <TableDataError />
          ) : (
            <QueriesListWrapper
              queriesList={filteredQueries}
              isLoading={isTableDataLoading}
              onCreateQueryClick={onCreateQueryClick}
              onRemoveQueryClick={onRemoveQueryClick}
              searchable={!!fleetQueriesByPlatform?.all.length}
              onSearchChange={setSearchString}
              customControl={renderPlatformDropdown}
            />
          )}
        </div>
        {showRemoveQueryModal && (
          <RemoveQueryModal
            onCancel={toggleRemoveQueryModal}
            onSubmit={onRemoveQuerySubmit}
          />
        )}
      </div>
    </div>
  );
};

export default ManageQueriesPage;
