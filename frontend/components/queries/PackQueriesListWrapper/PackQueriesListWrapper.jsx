import React, { Component } from "react";
import PropTypes from "prop-types";

import simpleSearch from "utilities/simple_search";
import TableContainer from "components/TableContainer";
import helpers from "components/queries/PackQueriesListWrapper/helpers";
import queryInterface from "interfaces/query";
import EmptyPack from "./EmptyPack";
import EmptySearch from "./EmptySearch";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PackQueriesTable/PackQueriesTableConfig";
import AddQueryIcon from "../../../../assets/images/icon-plus-16x16@2x.png";

const baseClass = "pack-queries-list-wrapper";
class PackQueriesListWrapper extends Component {
  static propTypes = {
    onAddScheduledQuery: PropTypes.func,
    onRemoveScheduledQueries: PropTypes.func,
    onScheduledQueryFormSubmit: PropTypes.func,
    scheduledQueries: PropTypes.arrayOf(queryInterface),
    packId: PropTypes.number,
    isLoadingScheduledQueries: PropTypes.bool,
  };

  constructor(props) {
    super(props);

    this.state = {
      querySearchText: "",
      checkedScheduledQueryIDs: [],
    };
  }

  onRemoveScheduledQueries = (selectedItemIds) => {
    const {
      onRemoveScheduledQueries: handleRemoveScheduledQueries,
    } = this.props;

    return handleRemoveScheduledQueries(selectedItemIds);
  };

  // 7/15 ADDED for New table
  // NOTE: this is called once on the initial rendering. The initial render of
  // the TableContainer child component will call this handler.
  onTableQueryChange = (queryData) => {
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
      this.setState({ querySearchText: "" });
      return;
    }

    this.setState({ querySearchText: searchQuery });
  };

  getQueries = () => {
    const { scheduledQueries } = this.props;
    const { querySearchText } = this.state;

    return simpleSearch(querySearchText, scheduledQueries);
  };

  render() {
    const {
      onTableQueryChange,
      onAddScheduledQuery,
      onRemoveScheduledQueries,
      isLoadingScheduledQueries,
      getQueries,
    } = this;
    const { scheduledQueries } = this.props;

    const tableHeaders = generateTableHeaders();
    const tableData = generateDataSet(getQueries());

    return (
      <div className={`${baseClass} body-wrap`}>
        {!scheduledQueries || scheduledQueries.length === 0 ? (
          <EmptyPack />
        ) : (
          <TableContainer
            columns={tableHeaders}
            data={tableData}
            isLoading={isLoadingScheduledQueries}
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
            onActionButtonClick={onAddScheduledQuery}
            onPrimarySelectActionClick={onRemoveScheduledQueries}
            primarySelectActionButtonVariant="text-icon"
            primarySelectActionButtonIcon="close"
            primarySelectActionButtonText={"Remove"}
            searchable
            disablePagination
          />
        )}
      </div>
    );
  }
}

export default ScheduledQueriesListWrapper;
