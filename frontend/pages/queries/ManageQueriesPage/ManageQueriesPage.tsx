import React, { useContext, useCallback, useEffect, useState } from "react";
import { useDispatch } from "react-redux";
import { useQuery } from "react-query";
import { push } from "react-router-redux";
import { memoize } from "lodash";

import { AppContext } from "context/app";
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
  platforms?: string[];
}

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
  sqlTools.listCompatiblePlatforms(memoizedSqlTables(queryString));

const ManageQueriesPage = (): JSX.Element => {
  const dispatch = useDispatch();

  const { isOnlyObserver } = useContext(AppContext);

  const [filteredQueries, setFilteredQueries] = useState<IQueryTableData[]>([]);
  const [searchString, setSearchString] = useState<string>("");
  const [selectedPlatform, setSelectedPlatform] = useState<string>("all");
  const [selectedQueryIds, setSelectedQueryIds] = useState<number[]>([]);
  const [showRemoveQueryModal, setShowRemoveQueryModal] = useState<boolean>(
    false
  );
  const [queriesByPlatform, setQueriesByPlatform] = useState<
    Record<string, IQueryTableData[]>
  >({ darwin: [], linux: [], windows: [] });

  const {
    data: fleetQueries,
    error: fleetQueriesError,
    isLoading: isLoadingFleetQueries,
    refetch: refetchFleetQueries,
  } = useQuery<IFleetQueriesResponse, Error, IQueryTableData[]>(
    "fleet queries",
    () => fleetQueriesAPI.loadAll(),
    {
      // refetchOnMount: false,
      // refetchOnReconnect: false,
      // refetchOnWindowFocus: false,
      select: (data: IFleetQueriesResponse) =>
        // data.queries.map((q) => {
        //   return {
        //     ...q,
        //     platforms: getPlatforms(q.query),
        //   };
        // }),
        data.queries,
      // TODO: Try moving queriesByPlatform into the select method
      // onSuccess: (queriesList) => {
      //   setQueriesByPlatform(
      //     queriesList.reduce(
      //       (acc: Record<string, IQueryTableData[]>, q) => {
      //         q.platforms.forEach((p) => acc[p]?.push(q));
      //         return acc;
      //       },
      //       { darwin: [], linux: [], windows: [] }
      //     )
      //   );
      // },
    }
  );

  useEffect(() => {
    let queriesList = fleetQueries;
    // selectedPlatform !== "all"
    //   ? queriesByPlatform[selectedPlatform]
    //   : fleetQueries || [];
    if (searchString) {
      queriesList = queriesList?.filter((q) =>
        q.name.toLowerCase().includes(searchString.toLowerCase())
      );
    }
    setFilteredQueries(queriesList || []);
    // }, [fleetQueries, queriesByPlatform, searchString, selectedPlatform]);
  }, [fleetQueries, searchString]);

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

  // const renderPlatformDropdown = () => {
  //   return fleetQueries?.length ? (
  //     <Dropdown
  //       value={selectedPlatform}
  //       className={`${baseClass}__platform_dropdown`}
  //       options={PLATFORM_FILTER_OPTIONS}
  //       searchable={false}
  //       onChange={setSelectedPlatform}
  //     />
  //   ) : (
  //     <></>
  //   );
  // };

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
          {!isOnlyObserver && !!fleetQueries?.length && (
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
          {isLoadingFleetQueries && <Spinner />}
          {!isLoadingFleetQueries && fleetQueriesError ? (
            <TableDataError />
          ) : (
            <QueriesListWrapper
              queriesList={filteredQueries}
              isLoading={isLoadingFleetQueries}
              onCreateQueryClick={onCreateQueryClick}
              onRemoveQueryClick={onRemoveQueryClick}
              searchable={!!fleetQueries?.length}
              onSearchChange={setSearchString}
              // customControl={renderPlatformDropdown}
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
