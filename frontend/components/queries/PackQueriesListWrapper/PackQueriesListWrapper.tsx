import React, { Component, useState } from "react";
import PropTypes from "prop-types";
// @ts-ignore
import simpleSearch from "utilities/simple_search";
import TableContainer from "components/TableContainer";
import Button from "components/buttons/Button";
// @ts-ignore
import helpers from "components/queries/PackQueriesListWrapper/helpers";
import { IScheduledQuery } from "interfaces/scheduled_query";
import { IQuery } from "interfaces/query";
import EmptySearch from "./EmptySearch";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PackQueriesTable/PackQueriesTableConfig";
import AddQueryIcon from "../../../../assets/images/icon-plus-16x16@2x.png";

const baseClass = "pack-queries-list-wrapper";

interface IFormData {
  interval: number;
  name?: string;
  shard: number;
  query?: string;
  query_id?: number;
  snapshot: boolean;
  removed: boolean;
  platform: string;
  version: string;
  pack_id: number;
}

interface IPackQueriesListWrapperProps {
  onAddPackQuery: any;
  onEditPackQuery: any;
  onRemovePackQueries: any;
  onPackQueryFormSubmit: (
    formData: IFormData,
    editQuery: IScheduledQuery | undefined
  ) => void;
  scheduledQueries: IScheduledQuery[] | undefined;
  packId: number;
  isLoadingPackQueries: boolean;
}

const PackQueriesListWrapper = ({
  onAddPackQuery,
  onEditPackQuery,
  onRemovePackQueries,
  onPackQueryFormSubmit,
  packId,
  scheduledQueries,
  isLoadingPackQueries,
}: IPackQueriesListWrapperProps): JSX.Element => {
  const [querySearchText, setQuerySearchText] = useState<string>("");
  const [checkedScheduledQueryIDs, setCheckedScheduledQueryIDs] = useState<
    number[]
  >([]);

  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  const onTableQueryChange = (queryData: any) => {
    const {
      pageIndex,
      pageSize,
      searchQuery,
      sortHeader,
      sortDirection,
    } = queryData;
    let sortBy = [];
    if (sortHeader !== "") {
      sortBy = [{ id: sortHeader, direction: sortDirection }];
    }

    if (!searchQuery) {
      setQuerySearchText("");
      return;
    }

    setQuerySearchText(searchQuery);
  };

  const getQueries = () => {
    return simpleSearch(querySearchText, scheduledQueries);
  };

  const onActionSelection = (
    action: string,
    selectedQuery: IScheduledQuery
  ) => {
    switch (action) {
      case "edit":
        onEditPackQuery(selectedQuery);
        break;
      case "remove":
        onRemovePackQueries([selectedQuery.id]);
        break;
      default:
    }
  };

  const tableHeaders = generateTableHeaders(onActionSelection);
  const tableData = generateDataSet(getQueries());

  return (
    <div className={`${baseClass} body-wrap`}>
      {!scheduledQueries?.length ? (
        <div className={`${baseClass}__no-queries`}>
          <p>Your pack has no queries.</p>
          <Button
            onClick={onAddPackQuery}
            variant={"text-icon"}
            className={`${baseClass}__no-queries-action-button`}
          >
            <>
              Add query
              <img src={AddQueryIcon} alt={`Add query icon`} />
            </>
          </Button>
        </div>
      ) : (
        <TableContainer
          columns={tableHeaders}
          data={tableData}
          isLoading={isLoadingPackQueries}
          defaultSortHeader={"name"}
          defaultSortDirection={"asc"}
          inputPlaceHolder={"Search queries"}
          onQueryChange={onTableQueryChange}
          resultsTitle={"queries"}
          emptyComponent={EmptySearch}
          showMarkAllPages={false}
          actionButtonText={"Add query"}
          actionButtonIcon={AddQueryIcon}
          actionButtonVariant={"text-icon"}
          onActionButtonClick={onAddPackQuery}
          onPrimarySelectActionClick={onRemovePackQueries}
          primarySelectActionButtonVariant="text-icon"
          primarySelectActionButtonIcon="close"
          primarySelectActionButtonText={"Remove"}
          searchable
          disablePagination
          isAllPagesSelected={false}
        />
      )}
    </div>
  );
};

export default PackQueriesListWrapper;
