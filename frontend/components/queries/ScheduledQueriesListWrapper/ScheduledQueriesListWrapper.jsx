import React, { Component } from "react";
import PropTypes from "prop-types";

import simpleSearch from "utilities/simple_search";
import TableContainer from "components/TableContainer";
import helpers from "components/queries/ScheduledQueriesListWrapper/helpers";
import queryInterface from "interfaces/query";
import EmptyPack from "./EmptyPack";
import EmptySearch from "./EmptySearch";
import {
  generateTableHeaders,
  generateDataSet,
} from "./PackQueriesTable/PackQueriesTableConfig";
import RemoveQueryModal from "./RemoveQueryModal";

const baseClass = "scheduled-queries-list-wrapper";
class ScheduledQueriesListWrapper extends Component {
  static propTypes = {
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
      onRemoveScheduledQueries,
      isLoadingScheduledQueries,
      getQueries,
    } = this;
    const { scheduledQueries } = this.props;

    const tableHeaders = generateTableHeaders();
    const tableData = generateDataSet(getQueries());

    return (
      <div className={`${baseClass} body-wrap`}>
        <div>
          <h1>Queries</h1>
        </div>
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
