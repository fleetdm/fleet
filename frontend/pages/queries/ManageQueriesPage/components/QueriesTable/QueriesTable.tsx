/* eslint-disable react/prop-types */
import React, { useContext, useState } from "react";

import { AppContext } from "context/app";
import { IQuery } from "interfaces/query";
import { IEmptyTableProps } from "interfaces/empty_table";
import { ITableQueryData } from "components/TableContainer/TableContainer";

import Button from "components/buttons/Button";
import TableContainer from "components/TableContainer";
import CustomLink from "components/CustomLink";
import EmptyTable from "components/EmptyTable";
import generateTableHeaders from "./QueriesTableConfig";

const baseClass = "queries-table";

interface IQueryTableData extends IQuery {
  performance: string;
  platforms: string[];
}
interface IQueriesTableProps {
  queriesList: IQueryTableData[] | null;
  isLoading: boolean;
  onDeleteQueryClick: (selectedTableQueryIds: number[]) => void;
  onCreateQueryClick: () => void;
  customControl?: () => JSX.Element;
  selectedDropdownFilter: string;
  isOnlyObserver?: boolean;
}

const QueriesTable = ({
  queriesList,
  isLoading,
  onDeleteQueryClick,
  onCreateQueryClick,
  customControl,
  selectedDropdownFilter,
  isOnlyObserver,
}: IQueriesTableProps): JSX.Element | null => {
  const { currentUser } = useContext(AppContext);
  const [searchString, setSearchString] = useState("");

  const handleSearchChange = ({ searchQuery }: ITableQueryData) => {
    setSearchString(searchQuery);
  };

  const emptyState = () => {
    const emptyQueries: IEmptyTableProps = {
      iconName: "empty-queries",
      header: "You don't have any queries",
      info: "A query is a specific question you can ask about your devices.",
    };
    if (searchString) {
      delete emptyQueries.iconName;
      emptyQueries.header = "No queries match the current search criteria.";
      emptyQueries.info =
        "Expecting to see queries? Try again in a few seconds as the system catches up.";
    } else if (!isOnlyObserver) {
      emptyQueries.additionalInfo = (
        <>
          Create a new query, or{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/standard-query-library"
            text="import Fleet’s standard query library"
            newTab
          />
        </>
      );
      emptyQueries.primaryButton = (
        <Button
          variant="brand"
          className={`${baseClass}__create-button`}
          onClick={onCreateQueryClick}
        >
          Create new query
        </Button>
      );
    }

    return emptyQueries;
  };

  const tableHeaders = currentUser && generateTableHeaders(currentUser);

  const searchable = !(queriesList?.length === 0 && searchString === "");

  return tableHeaders && !isLoading ? (
    <div className={`${baseClass}`}>
      <TableContainer
        resultsTitle={"queries"}
        columns={tableHeaders}
        data={queriesList}
        isLoading={isLoading}
        defaultSortHeader={"updated_at"}
        defaultSortDirection={"desc"}
        showMarkAllPages={false}
        isAllPagesSelected={false}
        onQueryChange={handleSearchChange}
        inputPlaceHolder="Search by name"
        searchable={searchable}
        onPrimarySelectActionClick={onDeleteQueryClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={() =>
          EmptyTable({
            iconName: emptyState().iconName,
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
          })
        }
        customControl={searchable ? customControl : undefined}
        isClientSideFilter
        searchQueryColumn="name"
        selectedDropdownFilter={selectedDropdownFilter}
        isClientSidePagination
      />
    </div>
  ) : null;
};

export default QueriesTable;
