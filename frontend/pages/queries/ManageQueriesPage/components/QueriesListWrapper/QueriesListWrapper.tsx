/* eslint-disable react/prop-types */
import React, { useCallback, useContext, useState } from "react";

import { AppContext } from "context/app";
import { IQuery } from "interfaces/query";
import { ITableSearchData } from "components/TableContainer/TableContainer";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import generateTableHeaders from "./QueriesTableConfig";

const baseClass = "queries-list-wrapper";
const noQueriesClass = "no-queries";
interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}
interface IQueriesListWrapperProps {
  queriesList: IQueryTableData[] | null;
  isLoading: boolean;
  onRemoveQueryClick: any;
  onCreateQueryClick: () => void;
  searchable: boolean;
  customControl?: () => JSX.Element;
  selectedDropdownFilter: string;
  clearSelectionCount?: number;
  isOnlyObserver?: boolean;
}

const QueriesListWrapper = ({
  queriesList,
  isLoading,
  onRemoveQueryClick,
  onCreateQueryClick,
  searchable,
  customControl,
  selectedDropdownFilter,
  clearSelectionCount,
  isOnlyObserver,
}: IQueriesListWrapperProps): JSX.Element | null => {
  const { currentUser } = useContext(AppContext);
  const [searchString, setSearchString] = useState<string>("");

  const handleSearchChange = ({ searchQuery }: ITableSearchData) => {
    setSearchString(searchQuery);
  };

  const NoQueriesComponent = useCallback(() => {
    return (
      <div className={`${noQueriesClass}`}>
        <div className={`${noQueriesClass}__inner`}>
          <div className={`${noQueriesClass}__inner-text`}>
            {searchString ? (
              <>
                <h2>No queries match the current search criteria.</h2>
                <p>
                  Expecting to see queries? Try again in a few seconds as the
                  system catches up.
                </p>
              </>
            ) : (
              <>
                <h2>You don&apos;t have any queries.</h2>
                <p>
                  A query is a specific question you can ask about your devices.
                </p>
                {!isOnlyObserver && (
                  <>
                    <p>
                      Create a new query, or go to GitHub to{" "}
                      <a href="https://fleetdm.com/docs/using-fleet/standard-query-library">
                        import Fleet’s standard query library
                      </a>
                      .
                    </p>
                    <Button
                      variant="brand"
                      className={`${baseClass}__create-button`}
                      onClick={onCreateQueryClick}
                    >
                      Create new query
                    </Button>
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    );
  }, [searchString, onCreateQueryClick]);

  const tableHeaders = currentUser && generateTableHeaders(currentUser);

  // Queries have not been created
  if (!isLoading && queriesList?.length === 0) {
    return (
      <div className={`${baseClass}`}>
        <NoQueriesComponent />
      </div>
    );
  }

  return tableHeaders && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        columns={tableHeaders}
        data={queriesList}
        isLoading={isLoading}
        selectedDropdownFilter={selectedDropdownFilter}
        clearSelectionCount={clearSelectionCount}
        resultsTitle={"queries"}
        defaultSortHeader={"updated_at"}
        defaultSortDirection={"desc"}
        inputPlaceHolder="Search by name"
        searchable={searchable}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        searchQueryColumn="name"
        showMarkAllPages={false}
        isAllPagesSelected={false}
        isClientSidePagination
        isClientSideFilter
        disablePagination
        onPrimarySelectActionClick={onRemoveQueryClick}
        emptyComponent={NoQueriesComponent}
        customControl={customControl}
        onQueryChange={handleSearchChange}
      />
    </div>
  ) : null;
};

export default QueriesListWrapper;
