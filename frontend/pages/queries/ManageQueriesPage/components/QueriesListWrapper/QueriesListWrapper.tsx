import React, { useCallback, useEffect, useState } from "react";
import { useSelector } from "react-redux";

import { IQuery } from "interfaces/query";
import { IUser } from "interfaces/user";
import permissionUtils from "utilities/permissions";
import TableContainer from "components/TableContainer";
import generateTableHeaders from "./QueriesTableConfig";

const baseClass = "queries-list-wrapper";
const noQueriesClass = "no-queries";

interface IQueriesListWrapperProps {
  onRemoveQueryClick: any;
  queriesList: IQuery[];
}

interface IRootState {
  auth: {
    user: IUser;
  };
  entities: {
    queries: {
      isLoading: boolean;
      data: IQuery[];
    };
  };
}

const QueriesListWrapper = (props: IQueriesListWrapperProps): JSX.Element => {
  const { onRemoveQueryClick, queriesList } = props;

  const loadingTableData = useSelector(
    (state: IRootState) => state.entities.queries.isLoading
  );

  const currentUser = useSelector((state: IRootState) => state.auth.user);
  const isOnlyObserver = permissionUtils.isOnlyObserver(currentUser);

  const [filteredQueries, setFilteredQueries] = useState(queriesList);
  const [searchString, setSearchString] = useState("");

  useEffect(() => {
    setFilteredQueries(queriesList);
  }, [queriesList]);

  useEffect(() => {
    setFilteredQueries(() => {
      return queriesList.filter((query) => {
        return query.name.toLowerCase().includes(searchString.toLowerCase());
      });
    });
  }, [queriesList, searchString, setFilteredQueries]);

  const onQueryChange = useCallback(
    (queryData) => {
      const { searchQuery } = queryData;
      setSearchString(searchQuery);
    },
    [setSearchString]
  );

  const NoQueriesComponent = useCallback(() => {
    return (
      <div className={`${noQueriesClass}`}>
        <div className={`${noQueriesClass}__inner`}>
          {/* <img src={scheduleSvg} alt="No Schedule" /> */}
          <div className={`${noQueriesClass}__inner-text`}>
            {!searchString ? (
              <h2>You don&apos;t have any queries.</h2>
            ) : (
              <h2>No queries match your search.</h2>
            )}
            <p>
              Create a new query, or{" "}
              <a href="https://github.com/fleetdm/fleet/tree/main/docs/1-Using-Fleet/standard-query-library">
                go to GitHub
              </a>{" "}
              to import Fleet’s standard query library.
            </p>
          </div>
        </div>
      </div>
    );
  }, [searchString]);

  const tableHeaders = generateTableHeaders(isOnlyObserver);

  return (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={filteredQueries}
        isLoading={loadingTableData}
        defaultSortHeader={"query"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={onQueryChange}
        inputPlaceHolder="Search by name"
        searchable
        disablePagination
        onPrimarySelectActionClick={onRemoveQueryClick}
        primarySelectActionButtonVariant="text-link"
        primarySelectActionButtonIcon="close"
        primarySelectActionButtonText={"Remove"}
        emptyComponent={NoQueriesComponent}
      />
    </div>
  );
};

export default QueriesListWrapper;
