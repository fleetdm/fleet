import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { includes, sortBy, size } from "lodash";

import queryInterface from "interfaces/query";
import FleetIcon from "components/icons/FleetIcon";
import Checkbox from "components/forms/fields/Checkbox";
import QueriesListRow from "components/queries/QueriesList/QueriesListRow";

const baseClass = "queries-list";

class QueriesList extends Component {
  static propTypes = {
    checkedQueryIDs: PropTypes.arrayOf(PropTypes.number).isRequired,
    isQueriesAvailable: PropTypes.bool,
    onCheckAll: PropTypes.func.isRequired,
    onCheckQuery: PropTypes.func.isRequired,
    onSelectQuery: PropTypes.func.isRequired,
    onDblClickQuery: PropTypes.func,
    queries: PropTypes.arrayOf(queryInterface).isRequired,
    selectedQuery: queryInterface,
    isOnlyObserver: PropTypes.bool,
  };

  static defaultProps = {
    selectedQuery: {},
  };

  constructor(props) {
    super(props);

    this.state = { allQueriesChecked: false };
  }

  isChecked = (query) => {
    const { checkedQueryIDs } = this.props;
    const { allQueriesChecked } = this.state;

    if (allQueriesChecked) {
      return true;
    }

    return includes(checkedQueryIDs, query.id);
  };

  handleCheckAll = (shouldCheckAllQueries) => {
    const { onCheckAll } = this.props;
    const { allQueriesChecked } = this.state;

    this.setState({ allQueriesChecked: !allQueriesChecked });

    return onCheckAll(shouldCheckAllQueries);
  };

  handleCheckQuery = (val, id) => {
    const { allQueriesChecked } = this.state;
    const { onCheckQuery } = this.props;

    if (allQueriesChecked) {
      this.setState({ allQueriesChecked: false });
    }

    onCheckQuery(val, id);
  };

  renderHelpText = () => {
    const { isQueriesAvailable, queries } = this.props;

    if (queries.length) {
      return false;
    }

    if (isQueriesAvailable) {
      return (
        <tr>
          <td colSpan={6}>
            <p>No queries matched your search criteria.</p>
          </td>
        </tr>
      );
    }

    return (
      <tr>
        <td colSpan={6}>
          <p>
            No queries available. Try creating one or get started by&nbsp;
            <a
              href="https://fleetdm.com/queries"
              target="_blank"
              rel="noopener noreferrer"
            >
              importing standard queries <FleetIcon name="external-link" />
            </a>
          </p>
        </td>
      </tr>
    );
  };

  render() {
    const alphaSort = (q) => q.name.toLowerCase();
    const {
      checkedQueryIDs,
      onSelectQuery,
      onDblClickQuery,
      queries,
      selectedQuery,
      isOnlyObserver,
    } = this.props;
    const { allQueriesChecked } = this.state;
    const { renderHelpText, handleCheckAll, handleCheckQuery } = this;
    const sortedQueries = sortBy(queries, [alphaSort]);
    const wrapperClassName = classnames(`${baseClass}__table`, {
      [`${baseClass}__table--query-selected`]: size(checkedQueryIDs),
    });
    return (
      <div className={baseClass}>
        <table className={wrapperClassName}>
          <thead>
            <tr>
              <th>
                <Checkbox
                  name="check-all-queries"
                  onChange={handleCheckAll}
                  value={allQueriesChecked}
                />
              </th>
              <th>Query name</th>
              <th>Description</th>
              {isOnlyObserver ? null : (
                <th className={`${baseClass}__observers-can-run`}>
                  Observers can run
                </th>
              )}
              <th className={`${baseClass}__author-name`}>Author</th>
              <th>Last modified</th>
            </tr>
          </thead>
          <tbody>
            {renderHelpText()}
            {!!sortedQueries.length &&
              sortedQueries.map((query) => {
                return (
                  <QueriesListRow
                    checked={this.isChecked(query)}
                    key={`query-row-${query.id}`}
                    onCheck={handleCheckQuery}
                    onSelect={onSelectQuery}
                    onDoubleClick={onDblClickQuery}
                    query={query}
                    isOnlyObserver={isOnlyObserver}
                    selected={
                      allQueriesChecked || selectedQuery.id === query.id
                    }
                  />
                );
              })}
          </tbody>
        </table>
      </div>
    );
  }
}

export default QueriesList;
