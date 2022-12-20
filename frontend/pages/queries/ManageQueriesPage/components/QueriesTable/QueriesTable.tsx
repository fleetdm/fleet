/* eslint-disable react/prop-types */
import React, { useCallback, useContext, useState } from "react";
import { IconNames } from "components/icons";

import { AppContext } from "context/app";
import { IQuery } from "interfaces/query";
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

interface IEmptyTableProps {
  iconName?: IconNames;
  header?: JSX.Element | string;
  info?: JSX.Element | string;
  additionalInfo?: JSX.Element | string;
  className?: string;
  primaryButton?: JSX.Element;
  secondaryButton?: JSX.Element;
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
    const noQueries: IEmptyTableProps = {
      iconName: "empty-queries",
      header: "You don't have any queries.",
      info: "A query is a specific question you can ask about your devices.",
    };
    if (searchString) {
      noQueries.header = "No queries match the current search criteria.";
      noQueries.info =
        "Expecting to see queries? Try again in a few seconds as the system catches up.";
    }
    if (!isOnlyObserver) {
      noQueries.additionalInfo = (
        <>
          Create a new query, or{" "}
          <CustomLink
            url="https://fleetdm.com/docs/using-fleet/standard-query-library"
            text="import Fleet’s standard query library"
            newTab
          />
        </>
      );
      noQueries.primaryButton = (
        <Button
          variant="brand"
          className={`${baseClass}__create-button`}
          onClick={onCreateQueryClick}
        >
          Create new query
        </Button>
      );
    }

    return noQueries;
  };

  // const NoQueriesComponent = useCallback(() => {
  //   return (
  //     <div className={`${noQueriesClass}`}>
  //       <div className={`${noQueriesClass}__inner`}>
  //         <Icon name="empty-queries" />
  //         <div className={`${noQueriesClass}__inner-text`}>
  //           {searchString ? (
  //             <div className={`${noQueriesClass}__no-results`}>
  //               <h2>No queries match the current search criteria.</h2>
  //               <p>
  //                 Expecting to see queries? Try again in a few seconds as the
  //                 system catches up.
  //               </p>
  //             </div>
  //           ) : (
  //             <div className={`${noQueriesClass}__none-created`}>
  //               <h2>You don&apos;t have any queries.</h2>
  //               <p>
  //                 A query is a specific question you can ask about your devices.
  //               </p>
  //               {!isOnlyObserver && (
  //                 <>
  //                   <p>
  //                     Create a new query, or{" "}
  //                     <CustomLink
  //                       url="https://fleetdm.com/docs/using-fleet/standard-query-library"
  //                       text="import Fleet’s standard query library"
  //                       newTab
  //                     />
  //                   </p>
  //                   <Button
  //                     variant="brand"
  //                     className={`${baseClass}__create-button`}
  //                     onClick={onCreateQueryClick}
  //                   >
  //                     Create new query
  //                   </Button>
  //                 </>
  //               )}
  //             </div>
  //           )}
  //         </div>
  //       </div>
  //     </div>
  //   );
  // }, [searchString, onCreateQueryClick]);

  const tableHeaders = currentUser && generateTableHeaders(currentUser);

  // Queries have not been created
  if (!isLoading && queriesList?.length === 0) {
    return (
      <div className={`${baseClass}`}>
        {EmptyTable({
          iconName: "empty-queries", // TODO: Fix types to use emptyState().iconName
          header: emptyState().header,
          info: emptyState().info,
          additionalInfo: emptyState().additionalInfo,
          primaryButton: emptyState().primaryButton,
        })}
      </div>
    );
  }

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
        searchable={!!queriesList}
        onPrimarySelectActionClick={onDeleteQueryClick}
        primarySelectActionButtonVariant="text-icon"
        primarySelectActionButtonIcon="delete"
        primarySelectActionButtonText={"Delete"}
        emptyComponent={() =>
          EmptyTable({
            iconName: "empty-queries", // TODO: Fix types to use emptyState().iconName
            header: emptyState().header,
            info: emptyState().info,
            additionalInfo: emptyState().additionalInfo,
            primaryButton: emptyState().primaryButton,
          })
        }
        customControl={customControl}
        isClientSideFilter
        searchQueryColumn="name"
        selectedDropdownFilter={selectedDropdownFilter}
        isClientSidePagination
      />
    </div>
  ) : null;
};

export default QueriesTable;
