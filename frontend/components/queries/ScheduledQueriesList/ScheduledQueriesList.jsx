import React, { Component } from "react";
import PropTypes from "prop-types";
import classnames from "classnames";
import { includes, sortBy, size } from "lodash";

import queryInterface from "interfaces/query";
import KolideIcon from "components/icons/KolideIcon";
import QueriesListItem from "components/queries/ScheduledQueriesList/ScheduledQueriesListItem";
import Checkbox from "components/forms/fields/Checkbox";

const baseClass = "scheduled-queries-list";

class ScheduledQueriesList extends Component {
  static propTypes = {
    isScheduledQueriesAvailable: PropTypes.bool,
    onCheckAllQueries: PropTypes.func.isRequired,
    onCheckQuery: PropTypes.func.isRequired,
    onSelectQuery: PropTypes.func.isRequired,
    onDblClickQuery: PropTypes.func.isRequired,
    scheduledQueries: PropTypes.arrayOf(queryInterface).isRequired,
    checkedScheduledQueryIDs: PropTypes.arrayOf(PropTypes.number).isRequired,
  };

  constructor(props) {
    super(props);

    this.state = {
      allQueriesSelected: false,
      selectedQueryRowId: null,
    };
  }

  isChecked = (scheduledQuery) => {
    const { checkedScheduledQueryIDs } = this.props;
    const { allQueriesSelected } = this.state;

    if (allQueriesSelected) {
      return true;
    }

    return includes(checkedScheduledQueryIDs, scheduledQuery.id);
  };

  handleSelectAllQueries = (shouldSelectAllQueries) => {
    const { onCheckAllQueries } = this.props;
    const { allQueriesSelected } = this.state;

    this.setState({ allQueriesSelected: !allQueriesSelected });

    return onCheckAllQueries(shouldSelectAllQueries);
  };

  handleSelectQuery = (scheduledQuery) => {
    const { onSelectQuery } = this.props;

    this.setState({ selectedQueryRowId: scheduledQuery.id });

    return onSelectQuery(scheduledQuery);
  };

  handleDblClickQuery = (scheduledQueryId) => {
    const { onDblClickQuery } = this.props;

    return onDblClickQuery(scheduledQueryId);
  };

  renderHelpText = () => {
    const { isScheduledQueriesAvailable, scheduledQueries } = this.props;

    if (scheduledQueries.length) {
      return false;
    }

    if (isScheduledQueriesAvailable) {
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
          <div className={`${baseClass}__first-query`}>
            <h1>
              First let&apos;s <span>add a query</span>.
            </h1>
            <h2>Then we&apos;ll set the following:</h2>
            <p>
              <strong>interval:</strong> the amount of time, in seconds, the
              query waits before running
            </p>
            <p>
              <strong>platform:</strong> the computer platform where this query
              will run (other platforms ignored)
            </p>
            <p>
              <strong>
                minimum <KolideIcon name="osquery" /> version:
              </strong>{" "}
              the minimum required <strong>osqueryd</strong> version installed
              on a host
            </p>
            <p>
              <strong>logging type:</strong>
            </p>
            <ul>
              <li>
                <strong>
                  <KolideIcon name="plus-minus" /> differential:
                </strong>{" "}
                show only what’s added from last run
              </li>
              <li>
                <strong>
                  <KolideIcon name="bold-plus" /> differential (ignore
                  removals):
                </strong>{" "}
                show only what’s been added since the last run
              </li>
              <li>
                <strong>
                  <KolideIcon name="camera" /> snapshot:
                </strong>{" "}
                show everything in its current state
              </li>
            </ul>
          </div>
        </td>
      </tr>
    );
  };

  render() {
    const {
      onCheckQuery,
      scheduledQueries,
      checkedScheduledQueryIDs,
    } = this.props;
    const { allQueriesSelected, selectedQueryRowId } = this.state;
    const {
      renderHelpText,
      handleSelectQuery,
      handleDblClickQuery,
      handleSelectAllQueries,
    } = this;

    const wrapperClassName = classnames(`${baseClass}__table`, {
      [`${baseClass}__table--query-selected`]: size(checkedScheduledQueryIDs),
    });

    return (
      <div className={baseClass}>
        <table className={wrapperClassName}>
          <thead>
            <tr>
              <th>
                <Checkbox
                  name="select-all-scheduled-queries"
                  onChange={handleSelectAllQueries}
                  value={allQueriesSelected}
                />
              </th>
              <th>Query name</th>
              <th>Interval(s)</th>
              <th>Platform</th>
              <th>
                <KolideIcon name="osquery" /> Ver.
              </th>
              <th>Shard</th>
              <th>Logging</th>
            </tr>
          </thead>
          <tbody>
            {renderHelpText()}
            {!!scheduledQueries.length &&
              sortBy(scheduledQueries, ["name"]).map((scheduledQuery) => {
                return (
                  <QueriesListItem
                    checked={this.isChecked(scheduledQuery)}
                    key={`scheduled-query-${scheduledQuery.id}`}
                    onCheck={onCheckQuery}
                    onSelect={handleSelectQuery}
                    onDblClick={handleDblClickQuery}
                    isSelected={selectedQueryRowId === scheduledQuery.id}
                    scheduledQuery={scheduledQuery}
                  />
                );
              })}
          </tbody>
        </table>
      </div>
    );
  }
}

export default ScheduledQueriesList;
